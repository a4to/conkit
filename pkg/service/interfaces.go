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
	"time"

	"github.com/a4to/protocol/conkit"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// encapsulates CRUD operations for room settings
//
//counterfeiter:generate . ObjectStore
type ObjectStore interface {
	ServiceStore

	// enable locking on a specific room to prevent race
	// returns a (lock uuid, error)
	LockRoom(ctx context.Context, roomName conkit.RoomName, duration time.Duration) (string, error)
	UnlockRoom(ctx context.Context, roomName conkit.RoomName, uid string) error

	StoreRoom(ctx context.Context, room *conkit.Room, internal *conkit.RoomInternal) error

	StoreParticipant(ctx context.Context, roomName conkit.RoomName, participant *conkit.ParticipantInfo) error
	DeleteParticipant(ctx context.Context, roomName conkit.RoomName, identity conkit.ParticipantIdentity) error
}

//counterfeiter:generate . ServiceStore
type ServiceStore interface {
	LoadRoom(ctx context.Context, roomName conkit.RoomName, includeInternal bool) (*conkit.Room, *conkit.RoomInternal, error)
	DeleteRoom(ctx context.Context, roomName conkit.RoomName) error

	// ListRooms returns currently active rooms. if names is not nil, it'll filter and return
	// only rooms that match
	ListRooms(ctx context.Context, roomNames []conkit.RoomName) ([]*conkit.Room, error)
	LoadParticipant(ctx context.Context, roomName conkit.RoomName, identity conkit.ParticipantIdentity) (*conkit.ParticipantInfo, error)
	ListParticipants(ctx context.Context, roomName conkit.RoomName) ([]*conkit.ParticipantInfo, error)
}

//counterfeiter:generate . EgressStore
type EgressStore interface {
	StoreEgress(ctx context.Context, info *conkit.EgressInfo) error
	LoadEgress(ctx context.Context, egressID string) (*conkit.EgressInfo, error)
	ListEgress(ctx context.Context, roomName conkit.RoomName, active bool) ([]*conkit.EgressInfo, error)
	UpdateEgress(ctx context.Context, info *conkit.EgressInfo) error
}

//counterfeiter:generate . IngressStore
type IngressStore interface {
	StoreIngress(ctx context.Context, info *conkit.IngressInfo) error
	LoadIngress(ctx context.Context, ingressID string) (*conkit.IngressInfo, error)
	LoadIngressFromStreamKey(ctx context.Context, streamKey string) (*conkit.IngressInfo, error)
	ListIngress(ctx context.Context, roomName conkit.RoomName) ([]*conkit.IngressInfo, error)
	UpdateIngress(ctx context.Context, info *conkit.IngressInfo) error
	UpdateIngressState(ctx context.Context, ingressId string, state *conkit.IngressState) error
	DeleteIngress(ctx context.Context, info *conkit.IngressInfo) error
}

//counterfeiter:generate . RoomAllocator
type RoomAllocator interface {
	AutoCreateEnabled(ctx context.Context) bool
	SelectRoomNode(ctx context.Context, roomName conkit.RoomName, nodeID conkit.NodeID) error
	CreateRoom(ctx context.Context, req *conkit.CreateRoomRequest, isExplicit bool) (*conkit.Room, *conkit.RoomInternal, bool, error)
	ValidateCreateRoom(ctx context.Context, roomName conkit.RoomName) error
}

//counterfeiter:generate . SIPStore
type SIPStore interface {
	StoreSIPTrunk(ctx context.Context, info *conkit.SIPTrunkInfo) error
	StoreSIPInboundTrunk(ctx context.Context, info *conkit.SIPInboundTrunkInfo) error
	StoreSIPOutboundTrunk(ctx context.Context, info *conkit.SIPOutboundTrunkInfo) error
	LoadSIPTrunk(ctx context.Context, sipTrunkID string) (*conkit.SIPTrunkInfo, error)
	LoadSIPInboundTrunk(ctx context.Context, sipTrunkID string) (*conkit.SIPInboundTrunkInfo, error)
	LoadSIPOutboundTrunk(ctx context.Context, sipTrunkID string) (*conkit.SIPOutboundTrunkInfo, error)
	ListSIPTrunk(ctx context.Context, opts *conkit.ListSIPTrunkRequest) (*conkit.ListSIPTrunkResponse, error)
	ListSIPInboundTrunk(ctx context.Context, opts *conkit.ListSIPInboundTrunkRequest) (*conkit.ListSIPInboundTrunkResponse, error)
	ListSIPOutboundTrunk(ctx context.Context, opts *conkit.ListSIPOutboundTrunkRequest) (*conkit.ListSIPOutboundTrunkResponse, error)
	DeleteSIPTrunk(ctx context.Context, sipTrunkID string) error

	StoreSIPDispatchRule(ctx context.Context, info *conkit.SIPDispatchRuleInfo) error
	LoadSIPDispatchRule(ctx context.Context, sipDispatchRuleID string) (*conkit.SIPDispatchRuleInfo, error)
	ListSIPDispatchRule(ctx context.Context, opts *conkit.ListSIPDispatchRuleRequest) (*conkit.ListSIPDispatchRuleResponse, error)
	DeleteSIPDispatchRule(ctx context.Context, sipDispatchRuleID string) error
}

//counterfeiter:generate . AgentStore
type AgentStore interface {
	StoreAgentDispatch(ctx context.Context, dispatch *conkit.AgentDispatch) error
	DeleteAgentDispatch(ctx context.Context, dispatch *conkit.AgentDispatch) error
	ListAgentDispatches(ctx context.Context, roomName conkit.RoomName) ([]*conkit.AgentDispatch, error)

	StoreAgentJob(ctx context.Context, job *conkit.Job) error
	DeleteAgentJob(ctx context.Context, job *conkit.Job) error
}
