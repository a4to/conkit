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

	"github.com/a4to/conkit-server/pkg/routing"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/rpc"
	"github.com/a4to/protocol/utils/guid"
)

type AgentDispatchService struct {
	agentDispatchClient rpc.TypedAgentDispatchInternalClient
	topicFormatter      rpc.TopicFormatter
	roomAllocator       RoomAllocator
	router              routing.MessageRouter
}

func NewAgentDispatchService(
	agentDispatchClient rpc.TypedAgentDispatchInternalClient,
	topicFormatter rpc.TopicFormatter,
	roomAllocator RoomAllocator,
	router routing.MessageRouter,
) *AgentDispatchService {
	return &AgentDispatchService{
		agentDispatchClient: agentDispatchClient,
		topicFormatter:      topicFormatter,
		roomAllocator:       roomAllocator,
		router:              router,
	}
}

func (ag *AgentDispatchService) CreateDispatch(ctx context.Context, req *conkit.CreateAgentDispatchRequest) (*conkit.AgentDispatch, error) {
	err := EnsureAdminPermission(ctx, conkit.RoomName(req.Room))
	if err != nil {
		return nil, twirpAuthError(err)
	}

	if ag.roomAllocator.AutoCreateEnabled(ctx) {
		err := ag.roomAllocator.SelectRoomNode(ctx, conkit.RoomName(req.Room), "")
		if err != nil {
			return nil, err
		}

		_, err = ag.router.CreateRoom(ctx, &conkit.CreateRoomRequest{Name: req.Room})
		if err != nil {
			return nil, err
		}
	}

	dispatch := &conkit.AgentDispatch{
		Id:        guid.New(guid.AgentDispatchPrefix),
		AgentName: req.AgentName,
		Room:      req.Room,
		Metadata:  req.Metadata,
	}
	return ag.agentDispatchClient.CreateDispatch(ctx, ag.topicFormatter.RoomTopic(ctx, conkit.RoomName(req.Room)), dispatch)
}

func (ag *AgentDispatchService) DeleteDispatch(ctx context.Context, req *conkit.DeleteAgentDispatchRequest) (*conkit.AgentDispatch, error) {
	err := EnsureAdminPermission(ctx, conkit.RoomName(req.Room))
	if err != nil {
		return nil, twirpAuthError(err)
	}

	return ag.agentDispatchClient.DeleteDispatch(ctx, ag.topicFormatter.RoomTopic(ctx, conkit.RoomName(req.Room)), req)
}

func (ag *AgentDispatchService) ListDispatch(ctx context.Context, req *conkit.ListAgentDispatchRequest) (*conkit.ListAgentDispatchResponse, error) {
	err := EnsureAdminPermission(ctx, conkit.RoomName(req.Room))
	if err != nil {
		return nil, twirpAuthError(err)
	}

	return ag.agentDispatchClient.ListDispatch(ctx, ag.topicFormatter.RoomTopic(ctx, conkit.RoomName(req.Room)), req)
}
