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

package service

import (
	"context"
	"sync"
	"time"

	"github.com/thoas/go-funk"

	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/utils"
)

// encapsulates CRUD operations for room settings
type LocalStore struct {
	// map of roomName => room
	rooms        map[conkit.RoomName]*conkit.Room
	roomInternal map[conkit.RoomName]*conkit.RoomInternal
	// map of roomName => { identity: participant }
	participants map[conkit.RoomName]map[conkit.ParticipantIdentity]*conkit.ParticipantInfo

	agentDispatches map[conkit.RoomName]map[string]*conkit.AgentDispatch
	agentJobs       map[conkit.RoomName]map[string]*conkit.Job

	lock       sync.RWMutex
	globalLock sync.Mutex
}

func NewLocalStore() *LocalStore {
	return &LocalStore{
		rooms:           make(map[conkit.RoomName]*conkit.Room),
		roomInternal:    make(map[conkit.RoomName]*conkit.RoomInternal),
		participants:    make(map[conkit.RoomName]map[conkit.ParticipantIdentity]*conkit.ParticipantInfo),
		agentDispatches: make(map[conkit.RoomName]map[string]*conkit.AgentDispatch),
		agentJobs:       make(map[conkit.RoomName]map[string]*conkit.Job),
		lock:            sync.RWMutex{},
	}
}

func (s *LocalStore) StoreRoom(_ context.Context, room *conkit.Room, internal *conkit.RoomInternal) error {
	if room.CreationTime == 0 {
		now := time.Now()
		room.CreationTime = now.Unix()
		room.CreationTimeMs = now.UnixMilli()
	}
	roomName := conkit.RoomName(room.Name)

	s.lock.Lock()
	s.rooms[roomName] = room
	s.roomInternal[roomName] = internal
	s.lock.Unlock()

	return nil
}

func (s *LocalStore) LoadRoom(_ context.Context, roomName conkit.RoomName, includeInternal bool) (*conkit.Room, *conkit.RoomInternal, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	room := s.rooms[roomName]
	if room == nil {
		return nil, nil, ErrRoomNotFound
	}

	var internal *conkit.RoomInternal
	if includeInternal {
		internal = s.roomInternal[roomName]
	}

	return room, internal, nil
}

func (s *LocalStore) ListRooms(_ context.Context, roomNames []conkit.RoomName) ([]*conkit.Room, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	rooms := make([]*conkit.Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		if roomNames == nil || funk.Contains(roomNames, conkit.RoomName(r.Name)) {
			rooms = append(rooms, r)
		}
	}
	return rooms, nil
}

func (s *LocalStore) DeleteRoom(ctx context.Context, roomName conkit.RoomName) error {
	room, _, err := s.LoadRoom(ctx, roomName, false)
	if err == ErrRoomNotFound {
		return nil
	} else if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.participants, conkit.RoomName(room.Name))
	delete(s.rooms, conkit.RoomName(room.Name))
	delete(s.roomInternal, conkit.RoomName(room.Name))
	delete(s.agentDispatches, conkit.RoomName(room.Name))
	delete(s.agentJobs, conkit.RoomName(room.Name))
	return nil
}

func (s *LocalStore) LockRoom(_ context.Context, _ conkit.RoomName, _ time.Duration) (string, error) {
	// local rooms lock & unlock globally
	s.globalLock.Lock()
	return "", nil
}

func (s *LocalStore) UnlockRoom(_ context.Context, _ conkit.RoomName, _ string) error {
	s.globalLock.Unlock()
	return nil
}

func (s *LocalStore) StoreParticipant(_ context.Context, roomName conkit.RoomName, participant *conkit.ParticipantInfo) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	roomParticipants := s.participants[roomName]
	if roomParticipants == nil {
		roomParticipants = make(map[conkit.ParticipantIdentity]*conkit.ParticipantInfo)
		s.participants[roomName] = roomParticipants
	}
	roomParticipants[conkit.ParticipantIdentity(participant.Identity)] = participant
	return nil
}

func (s *LocalStore) LoadParticipant(_ context.Context, roomName conkit.RoomName, identity conkit.ParticipantIdentity) (*conkit.ParticipantInfo, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	roomParticipants := s.participants[roomName]
	if roomParticipants == nil {
		return nil, ErrParticipantNotFound
	}
	participant := roomParticipants[identity]
	if participant == nil {
		return nil, ErrParticipantNotFound
	}
	return participant, nil
}

func (s *LocalStore) ListParticipants(_ context.Context, roomName conkit.RoomName) ([]*conkit.ParticipantInfo, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	roomParticipants := s.participants[roomName]
	if roomParticipants == nil {
		// empty array
		return nil, nil
	}

	items := make([]*conkit.ParticipantInfo, 0, len(roomParticipants))
	for _, p := range roomParticipants {
		items = append(items, p)
	}

	return items, nil
}

func (s *LocalStore) DeleteParticipant(_ context.Context, roomName conkit.RoomName, identity conkit.ParticipantIdentity) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	roomParticipants := s.participants[roomName]
	if roomParticipants != nil {
		delete(roomParticipants, identity)
	}
	return nil
}

func (s *LocalStore) StoreAgentDispatch(ctx context.Context, dispatch *conkit.AgentDispatch) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	clone := utils.CloneProto(dispatch)
	if clone.State != nil {
		clone.State.Jobs = nil
	}

	roomDispatches := s.agentDispatches[conkit.RoomName(dispatch.Room)]
	if roomDispatches == nil {
		roomDispatches = make(map[string]*conkit.AgentDispatch)
		s.agentDispatches[conkit.RoomName(dispatch.Room)] = roomDispatches
	}

	roomDispatches[clone.Id] = clone
	return nil
}

func (s *LocalStore) DeleteAgentDispatch(ctx context.Context, dispatch *conkit.AgentDispatch) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	roomDispatches := s.agentDispatches[conkit.RoomName(dispatch.Room)]
	if roomDispatches != nil {
		delete(roomDispatches, dispatch.Id)
	}

	return nil
}

func (s *LocalStore) ListAgentDispatches(ctx context.Context, roomName conkit.RoomName) ([]*conkit.AgentDispatch, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	agentDispatches := s.agentDispatches[roomName]
	if agentDispatches == nil {
		return nil, nil
	}
	agentJobs := s.agentJobs[roomName]

	var js []*conkit.Job
	if agentJobs != nil {
		for _, j := range agentJobs {
			js = append(js, utils.CloneProto(j))
		}
	}
	var ds []*conkit.AgentDispatch

	m := make(map[string]*conkit.AgentDispatch)
	for _, d := range agentDispatches {
		clone := utils.CloneProto(d)
		m[d.Id] = clone
		ds = append(ds, clone)
	}

	for _, j := range js {
		d := m[j.DispatchId]
		if d != nil {
			d.State.Jobs = append(d.State.Jobs, utils.CloneProto(j))
		}
	}

	return ds, nil
}

func (s *LocalStore) StoreAgentJob(ctx context.Context, job *conkit.Job) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	clone := utils.CloneProto(job)
	clone.Room = nil
	if clone.Participant != nil {
		clone.Participant = &conkit.ParticipantInfo{
			Identity: clone.Participant.Identity,
		}
	}

	roomJobs := s.agentJobs[conkit.RoomName(job.Room.Name)]
	if roomJobs == nil {
		roomJobs = make(map[string]*conkit.Job)
		s.agentJobs[conkit.RoomName(job.Room.Name)] = roomJobs
	}
	roomJobs[clone.Id] = clone

	return nil
}

func (s *LocalStore) DeleteAgentJob(ctx context.Context, job *conkit.Job) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	roomJobs := s.agentJobs[conkit.RoomName(job.Room.Name)]
	if roomJobs != nil {
		delete(roomJobs, job.Id)
	}

	return nil
}
