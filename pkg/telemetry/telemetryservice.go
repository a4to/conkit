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

package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/a4to/conkit-server/pkg/sfu/mime"
	"github.com/a4to/conkit-server/pkg/utils"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/webhook"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . TelemetryService
type TelemetryService interface {
	// TrackStats is called periodically for each track in both directions (published/subscribed)
	TrackStats(key StatsKey, stat *conkit.AnalyticsStat)

	// events
	RoomStarted(ctx context.Context, room *conkit.Room)
	RoomEnded(ctx context.Context, room *conkit.Room)
	// ParticipantJoined - a participant establishes signal connection to a room
	ParticipantJoined(ctx context.Context, room *conkit.Room, participant *conkit.ParticipantInfo, clientInfo *conkit.ClientInfo, clientMeta *conkit.AnalyticsClientMeta, shouldSendEvent bool)
	// ParticipantActive - a participant establishes media connection
	ParticipantActive(ctx context.Context, room *conkit.Room, participant *conkit.ParticipantInfo, clientMeta *conkit.AnalyticsClientMeta, isMigration bool)
	// ParticipantResumed - there has been an ICE restart or connection resume attempt, and we've received their signal connection
	ParticipantResumed(ctx context.Context, room *conkit.Room, participant *conkit.ParticipantInfo, nodeID conkit.NodeID, reason conkit.ReconnectReason)
	// ParticipantLeft - the participant leaves the room, only sent if ParticipantActive has been called before
	ParticipantLeft(ctx context.Context, room *conkit.Room, participant *conkit.ParticipantInfo, shouldSendEvent bool)
	// TrackPublishRequested - a publication attempt has been received
	TrackPublishRequested(ctx context.Context, participantID conkit.ParticipantID, identity conkit.ParticipantIdentity, track *conkit.TrackInfo)
	// TrackPublished - a publication attempt has been successful
	TrackPublished(ctx context.Context, participantID conkit.ParticipantID, identity conkit.ParticipantIdentity, track *conkit.TrackInfo)
	// TrackUnpublished - a participant unpublished a track
	TrackUnpublished(ctx context.Context, participantID conkit.ParticipantID, identity conkit.ParticipantIdentity, track *conkit.TrackInfo, shouldSendEvent bool)
	// TrackSubscribeRequested - a participant requested to subscribe to a track
	TrackSubscribeRequested(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo)
	// TrackSubscribed - a participant subscribed to a track successfully
	TrackSubscribed(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo, publisher *conkit.ParticipantInfo, shouldSendEvent bool)
	// TrackUnsubscribed - a participant unsubscribed from a track successfully
	TrackUnsubscribed(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo, shouldSendEvent bool)
	// TrackSubscribeFailed - failure to subscribe to a track
	TrackSubscribeFailed(ctx context.Context, participantID conkit.ParticipantID, trackID conkit.TrackID, err error, isUserError bool)
	// TrackMuted - the publisher has muted the Track
	TrackMuted(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo)
	// TrackUnmuted - the publisher has muted the Track
	TrackUnmuted(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo)
	// TrackPublishedUpdate - track metadata has been updated
	TrackPublishedUpdate(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo)
	// TrackMaxSubscribedVideoQuality - publisher is notified of the max quality subscribers desire
	TrackMaxSubscribedVideoQuality(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo, mime mime.MimeType, maxQuality conkit.VideoQuality)
	TrackPublishRTPStats(ctx context.Context, participantID conkit.ParticipantID, trackID conkit.TrackID, mimeType mime.MimeType, layer int, stats *conkit.RTPStats)
	TrackSubscribeRTPStats(ctx context.Context, participantID conkit.ParticipantID, trackID conkit.TrackID, mimeType mime.MimeType, stats *conkit.RTPStats)
	EgressStarted(ctx context.Context, info *conkit.EgressInfo)
	EgressUpdated(ctx context.Context, info *conkit.EgressInfo)
	EgressEnded(ctx context.Context, info *conkit.EgressInfo)
	IngressCreated(ctx context.Context, info *conkit.IngressInfo)
	IngressDeleted(ctx context.Context, info *conkit.IngressInfo)
	IngressStarted(ctx context.Context, info *conkit.IngressInfo)
	IngressUpdated(ctx context.Context, info *conkit.IngressInfo)
	IngressEnded(ctx context.Context, info *conkit.IngressInfo)
	LocalRoomState(ctx context.Context, info *conkit.AnalyticsNodeRooms)
	Report(ctx context.Context, reportInfo *conkit.ReportInfo)
	APICall(ctx context.Context, apiCallInfo *conkit.APICallInfo)
	Webhook(ctx context.Context, webhookInfo *conkit.WebhookInfo)

	// helpers
	AnalyticsService
	NotifyEvent(ctx context.Context, event *conkit.WebhookEvent)
	FlushStats()
}

const (
	workerCleanupWait = 3 * time.Minute
	jobsQueueMinSize  = 2048

	telemetryStatsUpdateInterval         = time.Second * 30
	telemetryNonMediaStatsUpdateInterval = time.Second * 30
)

type telemetryService struct {
	AnalyticsService

	notifier  webhook.QueuedNotifier
	jobsQueue *utils.OpsQueue

	workersMu  sync.RWMutex
	workers    map[conkit.ParticipantID]*StatsWorker
	workerList *StatsWorker

	flushMu sync.Mutex
}

func NewTelemetryService(notifier webhook.QueuedNotifier, analytics AnalyticsService) TelemetryService {
	t := &telemetryService{
		AnalyticsService: analytics,
		notifier:         notifier,
		jobsQueue: utils.NewOpsQueue(utils.OpsQueueParams{
			Name:        "telemetry",
			MinSize:     jobsQueueMinSize,
			FlushOnStop: true,
			Logger:      logger.GetLogger(),
		}),
		workers: make(map[conkit.ParticipantID]*StatsWorker),
	}
	if t.notifier != nil {
		t.notifier.RegisterProcessedHook(func(ctx context.Context, whi *conkit.WebhookInfo) {
			t.Webhook(ctx, whi)
		})
	}

	t.jobsQueue.Start()
	go t.run()

	return t
}

func (t *telemetryService) FlushStats() {
	t.flushMu.Lock()
	defer t.flushMu.Unlock()

	t.workersMu.RLock()
	worker := t.workerList
	t.workersMu.RUnlock()

	now := time.Now()
	var prev, reap *StatsWorker
	for worker != nil {
		next := worker.next
		if closed := worker.Flush(now); closed {
			if prev == nil {
				// this worker was at the head of the list
				t.workersMu.Lock()
				p := &t.workerList
				for *p != worker {
					// new workers have been added. scan until we find the one
					// immediately before this
					prev = *p
					p = &prev.next
				}
				*p = worker.next
				t.workersMu.Unlock()
			} else {
				prev.next = worker.next
			}

			worker.next = reap
			reap = worker
		} else {
			prev = worker
		}
		worker = next
	}

	if reap != nil {
		t.workersMu.Lock()
		for reap != nil {
			if reap == t.workers[reap.participantID] {
				delete(t.workers, reap.participantID)
			}
			reap = reap.next
		}
		t.workersMu.Unlock()
	}
}

func (t *telemetryService) run() {
	for range time.Tick(telemetryStatsUpdateInterval) {
		t.FlushStats()
	}
}

func (t *telemetryService) enqueue(op func()) {
	t.jobsQueue.Enqueue(op)
}

func (t *telemetryService) getWorker(participantID conkit.ParticipantID) (worker *StatsWorker, ok bool) {
	t.workersMu.RLock()
	defer t.workersMu.RUnlock()

	worker, ok = t.workers[participantID]
	return
}

func (t *telemetryService) getOrCreateWorker(
	ctx context.Context,
	roomID conkit.RoomID,
	roomName conkit.RoomName,
	participantID conkit.ParticipantID,
	participantIdentity conkit.ParticipantIdentity,
) (*StatsWorker, bool) {
	t.workersMu.Lock()
	defer t.workersMu.Unlock()

	worker, ok := t.workers[participantID]
	if ok && !worker.Closed() {
		return worker, true
	}

	existingIsConnected := false
	if ok {
		existingIsConnected = worker.IsConnected()
	}

	worker = newStatsWorker(
		ctx,
		t,
		roomID,
		roomName,
		participantID,
		participantIdentity,
	)
	if existingIsConnected {
		worker.SetConnected()
	}

	t.workers[participantID] = worker

	worker.next = t.workerList
	t.workerList = worker

	return worker, false
}

func (t *telemetryService) LocalRoomState(ctx context.Context, info *conkit.AnalyticsNodeRooms) {
	t.enqueue(func() {
		t.SendNodeRoomStates(ctx, info)
	})
}
