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

package supervisor

import (
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/a4to/conkit-server/pkg/rtc/types"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
)

const (
	monitorInterval = 1 * time.Second
)

type ParticipantSupervisorParams struct {
	Logger logger.Logger
}

type trackMonitor struct {
	opMon types.OperationMonitor
	err   error
}

type ParticipantSupervisor struct {
	params ParticipantSupervisorParams

	lock                 sync.RWMutex
	isPublisherConnected bool
	publications         map[conkit.TrackID]*trackMonitor

	isStopped atomic.Bool

	onPublicationError func(trackID conkit.TrackID)
}

func NewParticipantSupervisor(params ParticipantSupervisorParams) *ParticipantSupervisor {
	p := &ParticipantSupervisor{
		params:       params,
		publications: make(map[conkit.TrackID]*trackMonitor),
	}

	go p.checkState()

	return p
}

func (p *ParticipantSupervisor) Stop() {
	p.isStopped.Store(true)
}

func (p *ParticipantSupervisor) OnPublicationError(f func(trackID conkit.TrackID)) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.onPublicationError = f
}

func (p *ParticipantSupervisor) getOnPublicationError() func(trackID conkit.TrackID) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.onPublicationError
}

func (p *ParticipantSupervisor) SetPublisherPeerConnectionConnected(isConnected bool) {
	p.lock.Lock()
	p.isPublisherConnected = isConnected

	for _, pm := range p.publications {
		pm.opMon.PostEvent(types.OperationMonitorEventPublisherPeerConnectionConnected, p.isPublisherConnected)
	}
	p.lock.Unlock()
}

func (p *ParticipantSupervisor) AddPublication(trackID conkit.TrackID) {
	p.lock.Lock()
	pm, ok := p.publications[trackID]
	if !ok {
		pm = &trackMonitor{
			opMon: NewPublicationMonitor(
				PublicationMonitorParams{
					TrackID:                   trackID,
					IsPeerConnectionConnected: p.isPublisherConnected,
					Logger:                    p.params.Logger,
				},
			),
		}
		p.publications[trackID] = pm
	}
	pm.opMon.PostEvent(types.OperationMonitorEventAddPendingPublication, nil)
	p.lock.Unlock()
}

func (p *ParticipantSupervisor) SetPublicationMute(trackID conkit.TrackID, isMuted bool) {
	p.lock.Lock()
	pm, ok := p.publications[trackID]
	if ok {
		pm.opMon.PostEvent(types.OperationMonitorEventSetPublicationMute, isMuted)
	}
	p.lock.Unlock()
}

func (p *ParticipantSupervisor) SetPublishedTrack(trackID conkit.TrackID, pubTrack types.LocalMediaTrack) {
	p.lock.RLock()
	pm, ok := p.publications[trackID]
	if ok {
		pm.opMon.PostEvent(types.OperationMonitorEventSetPublishedTrack, pubTrack)
	}
	p.lock.RUnlock()
}

func (p *ParticipantSupervisor) ClearPublishedTrack(trackID conkit.TrackID, pubTrack types.LocalMediaTrack) {
	p.lock.RLock()
	pm, ok := p.publications[trackID]
	if ok {
		pm.opMon.PostEvent(types.OperationMonitorEventClearPublishedTrack, pubTrack)
	}
	p.lock.RUnlock()
}

func (p *ParticipantSupervisor) checkState() {
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for !p.isStopped.Load() {
		<-ticker.C

		p.checkPublications()
	}
}

func (p *ParticipantSupervisor) checkPublications() {
	var erroredPublications []conkit.TrackID
	var removablePublications []conkit.TrackID
	p.lock.RLock()
	for trackID, pm := range p.publications {
		if err := pm.opMon.Check(); err != nil {
			if pm.err == nil {
				p.params.Logger.Errorw("supervisor error on publication", err, "trackID", trackID)
				pm.err = err
				erroredPublications = append(erroredPublications, trackID)
			}
		} else {
			if pm.err != nil {
				p.params.Logger.Infow("supervisor publication recovered", "trackID", trackID)
				pm.err = err
			}
			if pm.opMon.IsIdle() {
				removablePublications = append(removablePublications, trackID)
			}
		}
	}
	p.lock.RUnlock()

	p.lock.Lock()
	for _, trackID := range removablePublications {
		delete(p.publications, trackID)
	}
	p.lock.Unlock()

	if onPublicationError := p.getOnPublicationError(); onPublicationError != nil {
		for _, trackID := range erroredPublications {
			onPublicationError(trackID)
		}
	}
}
