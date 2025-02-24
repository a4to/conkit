// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dynacast

import (
	"sync"
	"time"

	"github.com/bep/debounce"
	"golang.org/x/exp/maps"

	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"

	"github.com/a4to/conkit-server/pkg/rtc/types"
	"github.com/a4to/conkit-server/pkg/sfu/mime"
	"github.com/a4to/conkit-server/pkg/utils"
)

type DynacastManagerParams struct {
	DynacastPauseDelay time.Duration
	Logger             logger.Logger
}

type DynacastManager struct {
	params DynacastManagerParams

	lock                          sync.RWMutex
	regressedCodec                map[mime.MimeType]struct{}
	dynacastQuality               map[mime.MimeType]*DynacastQuality
	maxSubscribedQuality          map[mime.MimeType]conkit.VideoQuality
	committedMaxSubscribedQuality map[mime.MimeType]conkit.VideoQuality

	maxSubscribedQualityDebounce        func(func())
	maxSubscribedQualityDebouncePending bool

	qualityNotifyOpQueue *utils.OpsQueue

	isClosed bool

	onSubscribedMaxQualityChange func(subscribedQualities []*conkit.SubscribedCodec, maxSubscribedQualities []types.SubscribedCodecQuality)
}

func NewDynacastManager(params DynacastManagerParams) *DynacastManager {
	if params.Logger == nil {
		params.Logger = logger.GetLogger()
	}
	d := &DynacastManager{
		params:                        params,
		regressedCodec:                make(map[mime.MimeType]struct{}),
		dynacastQuality:               make(map[mime.MimeType]*DynacastQuality),
		maxSubscribedQuality:          make(map[mime.MimeType]conkit.VideoQuality),
		committedMaxSubscribedQuality: make(map[mime.MimeType]conkit.VideoQuality),
		qualityNotifyOpQueue: utils.NewOpsQueue(utils.OpsQueueParams{
			Name:        "quality-notify",
			MinSize:     64,
			FlushOnStop: true,
			Logger:      params.Logger,
		}),
	}
	if params.DynacastPauseDelay > 0 {
		d.maxSubscribedQualityDebounce = debounce.New(params.DynacastPauseDelay)
	}
	d.qualityNotifyOpQueue.Start()
	return d
}

func (d *DynacastManager) OnSubscribedMaxQualityChange(f func(subscribedQualities []*conkit.SubscribedCodec, maxSubscribedQualities []types.SubscribedCodecQuality)) {
	d.lock.Lock()
	d.onSubscribedMaxQualityChange = f
	d.lock.Unlock()
}

func (d *DynacastManager) AddCodec(mime mime.MimeType) {
	d.getOrCreateDynacastQuality(mime)
}

func (d *DynacastManager) HandleCodecRegression(fromMime, toMime mime.MimeType) {
	fromDq := d.getOrCreateDynacastQuality(fromMime)

	d.lock.Lock()
	if d.isClosed {
		d.lock.Unlock()
		return
	}

	if fromDq == nil {
		// should not happen as we have added the codec on setup receiver
		d.params.Logger.Warnw("regression from codec not found", nil, "mime", fromMime)
		d.lock.Unlock()
		return
	}
	d.regressedCodec[fromMime] = struct{}{}
	d.maxSubscribedQuality[fromMime] = conkit.VideoQuality_OFF

	// if the new codec is not added, notify the publisher to start publishing
	if _, ok := d.maxSubscribedQuality[toMime]; !ok {
		d.maxSubscribedQuality[toMime] = conkit.VideoQuality_HIGH
	}

	d.lock.Unlock()
	d.update(false)

	fromDq.Stop()
	fromDq.RegressTo(d.getOrCreateDynacastQuality(toMime))
}

func (d *DynacastManager) Restart() {
	d.lock.Lock()
	d.committedMaxSubscribedQuality = make(map[mime.MimeType]conkit.VideoQuality)

	dqs := d.getDynacastQualitiesLocked()
	d.lock.Unlock()

	for _, dq := range dqs {
		dq.Restart()
	}
}

func (d *DynacastManager) Close() {
	d.qualityNotifyOpQueue.Stop()

	d.lock.Lock()
	dqs := d.getDynacastQualitiesLocked()
	d.dynacastQuality = make(map[mime.MimeType]*DynacastQuality)

	d.isClosed = true
	d.lock.Unlock()

	for _, dq := range dqs {
		dq.Stop()
	}
}

// THere are situations like track unmute or streaming from a different node
// where subscribed quality needs to sent to the provider immediately.
// This bypasses any debouncing and forces a subscribed quality update
// with immediate effect.
func (d *DynacastManager) ForceUpdate() {
	d.update(true)
}

// It is possible for tracks to be in pending close state. When track
// is waiting to be closed, a node is not streaming a track. This can
// be used to force an update announcing that subscribed quality is OFF,
// i.e. indicating not pulling track any more.
func (d *DynacastManager) ForceQuality(quality conkit.VideoQuality) {
	d.lock.Lock()
	defer d.lock.Unlock()

	for mime := range d.committedMaxSubscribedQuality {
		d.committedMaxSubscribedQuality[mime] = quality
	}

	d.enqueueSubscribedQualityChange()
}

func (d *DynacastManager) NotifySubscriberMaxQuality(subscriberID conkit.ParticipantID, mime mime.MimeType, quality conkit.VideoQuality) {
	dq := d.getOrCreateDynacastQuality(mime)
	if dq != nil {
		dq.NotifySubscriberMaxQuality(subscriberID, quality)
	}
}

func (d *DynacastManager) NotifySubscriberNodeMaxQuality(nodeID conkit.NodeID, qualities []types.SubscribedCodecQuality) {
	for _, quality := range qualities {
		dq := d.getOrCreateDynacastQuality(quality.CodecMime)
		if dq != nil {
			dq.NotifySubscriberNodeMaxQuality(nodeID, quality.Quality)
		}
	}
}

func (d *DynacastManager) getOrCreateDynacastQuality(mimeType mime.MimeType) *DynacastQuality {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.isClosed || mimeType == mime.MimeTypeUnknown {
		return nil
	}

	if dq := d.dynacastQuality[mimeType]; dq != nil {
		return dq
	}

	dq := NewDynacastQuality(DynacastQualityParams{
		MimeType: mimeType,
		Logger:   d.params.Logger,
	})
	dq.OnSubscribedMaxQualityChange(d.updateMaxQualityForMime)
	dq.Start()

	d.dynacastQuality[mimeType] = dq
	return dq
}

func (d *DynacastManager) getDynacastQualitiesLocked() []*DynacastQuality {
	return maps.Values(d.dynacastQuality)
}

func (d *DynacastManager) updateMaxQualityForMime(mime mime.MimeType, maxQuality conkit.VideoQuality) {
	d.lock.Lock()
	if _, ok := d.regressedCodec[mime]; !ok {
		d.maxSubscribedQuality[mime] = maxQuality
	}
	d.lock.Unlock()

	d.update(false)
}

func (d *DynacastManager) update(force bool) {
	d.lock.Lock()

	d.params.Logger.Debugw("processing quality change",
		"force", force,
		"committedMaxSubscribedQuality", d.committedMaxSubscribedQuality,
		"maxSubscribedQuality", d.maxSubscribedQuality,
	)

	if len(d.maxSubscribedQuality) == 0 {
		// no mime has been added, nothing to update
		d.lock.Unlock()
		return
	}

	// add or remove of a mime triggers an update
	changed := len(d.maxSubscribedQuality) != len(d.committedMaxSubscribedQuality)
	downgradesOnly := !changed
	if !changed {
		for mime, quality := range d.maxSubscribedQuality {
			if cq, ok := d.committedMaxSubscribedQuality[mime]; ok {
				if cq != quality {
					changed = true
				}

				if (cq == conkit.VideoQuality_OFF && quality != conkit.VideoQuality_OFF) || (cq != conkit.VideoQuality_OFF && quality != conkit.VideoQuality_OFF && cq < quality) {
					downgradesOnly = false
				}
			}
		}
	}

	if !force {
		if !changed {
			d.lock.Unlock()
			return
		}

		if downgradesOnly && d.maxSubscribedQualityDebounce != nil {
			if !d.maxSubscribedQualityDebouncePending {
				d.params.Logger.Debugw("debouncing quality downgrade",
					"committedMaxSubscribedQuality", d.committedMaxSubscribedQuality,
					"maxSubscribedQuality", d.maxSubscribedQuality,
				)
				d.maxSubscribedQualityDebounce(func() {
					d.update(true)
				})
				d.maxSubscribedQualityDebouncePending = true
			} else {
				d.params.Logger.Debugw("quality downgrade waiting for debounce",
					"committedMaxSubscribedQuality", d.committedMaxSubscribedQuality,
					"maxSubscribedQuality", d.maxSubscribedQuality,
				)
			}
			d.lock.Unlock()
			return
		}
	}

	// clear debounce on send
	if d.maxSubscribedQualityDebounce != nil {
		d.maxSubscribedQualityDebounce(func() {})
		d.maxSubscribedQualityDebouncePending = false
	}

	d.params.Logger.Debugw("committing quality change",
		"force", force,
		"committedMaxSubscribedQuality", d.committedMaxSubscribedQuality,
		"maxSubscribedQuality", d.maxSubscribedQuality,
	)

	// commit change
	d.committedMaxSubscribedQuality = make(map[mime.MimeType]conkit.VideoQuality, len(d.maxSubscribedQuality))
	for mime, quality := range d.maxSubscribedQuality {
		d.committedMaxSubscribedQuality[mime] = quality
	}

	d.enqueueSubscribedQualityChange()
	d.lock.Unlock()
}

func (d *DynacastManager) enqueueSubscribedQualityChange() {
	if d.isClosed || d.onSubscribedMaxQualityChange == nil {
		return
	}

	subscribedCodecs := make([]*conkit.SubscribedCodec, 0, len(d.committedMaxSubscribedQuality))
	maxSubscribedQualities := make([]types.SubscribedCodecQuality, 0, len(d.committedMaxSubscribedQuality))
	for mime, quality := range d.committedMaxSubscribedQuality {
		maxSubscribedQualities = append(maxSubscribedQualities, types.SubscribedCodecQuality{
			CodecMime: mime,
			Quality:   quality,
		})

		if quality == conkit.VideoQuality_OFF {
			subscribedCodecs = append(subscribedCodecs, &conkit.SubscribedCodec{
				Codec: mime.String(),
				Qualities: []*conkit.SubscribedQuality{
					{Quality: conkit.VideoQuality_LOW, Enabled: false},
					{Quality: conkit.VideoQuality_MEDIUM, Enabled: false},
					{Quality: conkit.VideoQuality_HIGH, Enabled: false},
				},
			})
		} else {
			var subscribedQualities []*conkit.SubscribedQuality
			for q := conkit.VideoQuality_LOW; q <= conkit.VideoQuality_HIGH; q++ {
				subscribedQualities = append(subscribedQualities, &conkit.SubscribedQuality{
					Quality: q,
					Enabled: q <= quality,
				})
			}
			subscribedCodecs = append(subscribedCodecs, &conkit.SubscribedCodec{
				Codec:     mime.String(),
				Qualities: subscribedQualities,
			})
		}
	}

	d.params.Logger.Debugw(
		"subscribedMaxQualityChange",
		"subscribedCodecs", subscribedCodecs,
		"maxSubscribedQualities", maxSubscribedQualities,
	)
	d.qualityNotifyOpQueue.Enqueue(func() {
		d.onSubscribedMaxQualityChange(subscribedCodecs, maxSubscribedQualities)
	})
}
