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

	"github.com/dennwc/iters"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/rpc"
	"github.com/a4to/protocol/sip"
	"github.com/a4to/protocol/utils"
	"github.com/a4to/protocol/utils/guid"
	"github.com/a4to/psrpc"

	"github.com/a4to/conkit-server/pkg/config"
	"github.com/a4to/conkit-server/pkg/telemetry"
)

type SIPService struct {
	conf        *config.SIPConfig
	nodeID      conkit.NodeID
	bus         psrpc.MessageBus
	psrpcClient rpc.SIPClient
	store       SIPStore
	roomService conkit.RoomService
}

func NewSIPService(
	conf *config.SIPConfig,
	nodeID conkit.NodeID,
	bus psrpc.MessageBus,
	psrpcClient rpc.SIPClient,
	store SIPStore,
	rs conkit.RoomService,
	ts telemetry.TelemetryService,
) *SIPService {
	return &SIPService{
		conf:        conf,
		nodeID:      nodeID,
		bus:         bus,
		psrpcClient: psrpcClient,
		store:       store,
		roomService: rs,
	}
}

func (s *SIPService) CreateSIPTrunk(ctx context.Context, req *conkit.CreateSIPTrunkRequest) (*conkit.SIPTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if len(req.InboundNumbersRegex) != 0 {
		return nil, twirp.NewError(twirp.InvalidArgument, "Trunks with InboundNumbersRegex are deprecated. Use InboundNumbers instead.")
	}

	// Keep ID empty, so that validation can print "<new>" instead of a non-existent ID in the error.
	info := &conkit.SIPTrunkInfo{
		InboundAddresses: req.InboundAddresses,
		OutboundAddress:  req.OutboundAddress,
		OutboundNumber:   req.OutboundNumber,
		InboundNumbers:   req.InboundNumbers,
		InboundUsername:  req.InboundUsername,
		InboundPassword:  req.InboundPassword,
		OutboundUsername: req.OutboundUsername,
		OutboundPassword: req.OutboundPassword,
		Name:             req.Name,
		Metadata:         req.Metadata,
	}

	// Validate all trunks including the new one first.
	it, err := ListSIPInboundTrunk(ctx, s.store, &conkit.ListSIPInboundTrunkRequest{}, info.AsInbound())
	if err != nil {
		return nil, err
	}
	defer it.Close()
	if err = sip.ValidateTrunksIter(it); err != nil {
		return nil, err
	}

	// Now we can generate ID and store.
	info.SipTrunkId = guid.New(utils.SIPTrunkPrefix)
	if err := s.store.StoreSIPTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) CreateSIPInboundTrunk(ctx context.Context, req *conkit.CreateSIPInboundTrunkRequest) (*conkit.SIPInboundTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, twirp.WrapError(twirp.NewError(twirp.InvalidArgument, err.Error()), err)
	}

	info := req.Trunk
	if info.SipTrunkId != "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "trunk ID must be empty")
	}
	AppendLogFields(ctx, "trunk", logger.Proto(req.Trunk))

	// Keep ID empty still, so that validation can print "<new>" instead of a non-existent ID in the error.

	// Validate all trunks including the new one first.
	it, err := ListSIPInboundTrunk(ctx, s.store, &conkit.ListSIPInboundTrunkRequest{
		Numbers: req.GetTrunk().GetNumbers(),
	}, info)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	if err = sip.ValidateTrunksIter(it); err != nil {
		return nil, err
	}

	// Now we can generate ID and store.
	info.SipTrunkId = guid.New(utils.SIPTrunkPrefix)
	if err := s.store.StoreSIPInboundTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) CreateSIPOutboundTrunk(ctx context.Context, req *conkit.CreateSIPOutboundTrunkRequest) (*conkit.SIPOutboundTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, twirp.WrapError(twirp.NewError(twirp.InvalidArgument, err.Error()), err)
	}

	info := req.Trunk
	if info.SipTrunkId != "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "trunk ID must be empty")
	}
	AppendLogFields(ctx, "trunk", logger.Proto(req.Trunk))

	// No additional validation needed for outbound.
	info.SipTrunkId = guid.New(utils.SIPTrunkPrefix)
	if err := s.store.StoreSIPOutboundTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) GetSIPInboundTrunk(ctx context.Context, req *conkit.GetSIPInboundTrunkRequest) (*conkit.GetSIPInboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if req.SipTrunkId == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "trunk ID is required")
	}
	AppendLogFields(ctx, "trunkID", req.SipTrunkId)

	trunk, err := s.store.LoadSIPInboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		return nil, err
	}

	return &conkit.GetSIPInboundTrunkResponse{Trunk: trunk}, nil
}

func (s *SIPService) GetSIPOutboundTrunk(ctx context.Context, req *conkit.GetSIPOutboundTrunkRequest) (*conkit.GetSIPOutboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if req.SipTrunkId == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "trunk ID is required")
	}
	AppendLogFields(ctx, "trunkID", req.SipTrunkId)

	trunk, err := s.store.LoadSIPOutboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		return nil, err
	}

	return &conkit.GetSIPOutboundTrunkResponse{Trunk: trunk}, nil
}

// deprecated: ListSIPTrunk will be removed in the future
func (s *SIPService) ListSIPTrunk(ctx context.Context, req *conkit.ListSIPTrunkRequest) (*conkit.ListSIPTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	it := conkit.ListPageIter(s.store.ListSIPTrunk, req)
	defer it.Close()

	items, err := iters.AllPages(ctx, it)
	if err != nil {
		return nil, err
	}
	return &conkit.ListSIPTrunkResponse{Items: items}, nil
}

func ListSIPInboundTrunk(ctx context.Context, s SIPStore, req *conkit.ListSIPInboundTrunkRequest, add ...*conkit.SIPInboundTrunkInfo) (iters.Iter[*conkit.SIPInboundTrunkInfo], error) {
	if s == nil {
		return nil, ErrSIPNotConnected
	}
	pages := conkit.ListPageIter(s.ListSIPInboundTrunk, req)
	it := iters.PagesAsIter(ctx, pages)
	if len(add) != 0 {
		it = iters.MultiIter(true, it, iters.Slice(add))
	}
	return it, nil
}

func ListSIPOutboundTrunk(ctx context.Context, s SIPStore, req *conkit.ListSIPOutboundTrunkRequest, add ...*conkit.SIPOutboundTrunkInfo) (iters.Iter[*conkit.SIPOutboundTrunkInfo], error) {
	if s == nil {
		return nil, ErrSIPNotConnected
	}
	pages := conkit.ListPageIter(s.ListSIPOutboundTrunk, req)
	it := iters.PagesAsIter(ctx, pages)
	if len(add) != 0 {
		it = iters.MultiIter(true, it, iters.Slice(add))
	}
	return it, nil
}

func (s *SIPService) ListSIPInboundTrunk(ctx context.Context, req *conkit.ListSIPInboundTrunkRequest) (*conkit.ListSIPInboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	it, err := ListSIPInboundTrunk(ctx, s.store, req)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	items, err := iters.All(it)
	if err != nil {
		return nil, err
	}
	return &conkit.ListSIPInboundTrunkResponse{Items: items}, nil
}

func (s *SIPService) ListSIPOutboundTrunk(ctx context.Context, req *conkit.ListSIPOutboundTrunkRequest) (*conkit.ListSIPOutboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	it, err := ListSIPOutboundTrunk(ctx, s.store, req)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	items, err := iters.All(it)
	if err != nil {
		return nil, err
	}
	return &conkit.ListSIPOutboundTrunkResponse{Items: items}, nil
}

func (s *SIPService) DeleteSIPTrunk(ctx context.Context, req *conkit.DeleteSIPTrunkRequest) (*conkit.SIPTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if req.SipTrunkId == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "trunk ID is required")
	}

	AppendLogFields(ctx, "trunkID", req.SipTrunkId)
	if err := s.store.DeleteSIPTrunk(ctx, req.SipTrunkId); err != nil {
		return nil, err
	}

	return &conkit.SIPTrunkInfo{SipTrunkId: req.SipTrunkId}, nil
}

func (s *SIPService) CreateSIPDispatchRule(ctx context.Context, req *conkit.CreateSIPDispatchRuleRequest) (*conkit.SIPDispatchRuleInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, twirp.WrapError(twirp.NewError(twirp.InvalidArgument, err.Error()), err)
	}

	AppendLogFields(ctx,
		"request", logger.Proto(req),
		"trunkID", req.TrunkIds,
	)
	// Keep ID empty, so that validation can print "<new>" instead of a non-existent ID in the error.
	info := &conkit.SIPDispatchRuleInfo{
		Rule:            req.Rule,
		TrunkIds:        req.TrunkIds,
		InboundNumbers:  req.InboundNumbers,
		HidePhoneNumber: req.HidePhoneNumber,
		Name:            req.Name,
		Metadata:        req.Metadata,
		Attributes:      req.Attributes,
		RoomConfig:      req.RoomConfig,
	}

	// Validate all rules including the new one first.
	it, err := ListSIPDispatchRule(ctx, s.store, &conkit.ListSIPDispatchRuleRequest{
		TrunkIds: req.TrunkIds,
	}, info)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	if _, err = sip.ValidateDispatchRulesIter(it); err != nil {
		return nil, err
	}

	// Now we can generate ID and store.
	info.SipDispatchRuleId = guid.New(utils.SIPDispatchRulePrefix)
	if err := s.store.StoreSIPDispatchRule(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func ListSIPDispatchRule(ctx context.Context, s SIPStore, req *conkit.ListSIPDispatchRuleRequest, add ...*conkit.SIPDispatchRuleInfo) (iters.Iter[*conkit.SIPDispatchRuleInfo], error) {
	if s == nil {
		return nil, ErrSIPNotConnected
	}
	pages := conkit.ListPageIter(s.ListSIPDispatchRule, req)
	it := iters.PagesAsIter(ctx, pages)
	if len(add) != 0 {
		it = iters.MultiIter(true, it, iters.Slice(add))
	}
	return it, nil
}

func (s *SIPService) ListSIPDispatchRule(ctx context.Context, req *conkit.ListSIPDispatchRuleRequest) (*conkit.ListSIPDispatchRuleResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	it, err := ListSIPDispatchRule(ctx, s.store, req)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	items, err := iters.All(it)
	if err != nil {
		return nil, err
	}
	return &conkit.ListSIPDispatchRuleResponse{Items: items}, nil
}

func (s *SIPService) DeleteSIPDispatchRule(ctx context.Context, req *conkit.DeleteSIPDispatchRuleRequest) (*conkit.SIPDispatchRuleInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if req.SipDispatchRuleId == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "dispatch rule ID is required")
	}

	info, err := s.store.LoadSIPDispatchRule(ctx, req.SipDispatchRuleId)
	if err != nil {
		return nil, err
	}

	if err = s.store.DeleteSIPDispatchRule(ctx, info.SipDispatchRuleId); err != nil {
		return nil, err
	}

	return info, nil
}

func (s *SIPService) CreateSIPParticipant(ctx context.Context, req *conkit.CreateSIPParticipantRequest) (*conkit.SIPParticipantInfo, error) {
	unlikelyLogger := logger.GetLogger().WithUnlikelyValues(
		"room", req.RoomName,
		"sipTrunk", req.SipTrunkId,
		"toUser", req.SipCallTo,
		"participant", req.ParticipantIdentity,
	)
	AppendLogFields(ctx,
		"room", req.RoomName,
		"participant", req.ParticipantIdentity,
		"toUser", req.SipCallTo,
		"trunkID", req.SipTrunkId,
	)
	ireq, err := s.CreateSIPParticipantRequest(ctx, req, "", "", "", "")
	if err != nil {
		unlikelyLogger.Errorw("cannot create sip participant request", err)
		return nil, err
	}
	unlikelyLogger = unlikelyLogger.WithValues(
		"callID", ireq.SipCallId,
		"fromUser", ireq.Number,
		"toHost", ireq.Address,
	)
	AppendLogFields(ctx,
		"callID", ireq.SipCallId,
		"fromUser", ireq.Number,
		"toHost", ireq.Address,
	)

	// CreateSIPParticipant will wait for LiveKit Participant to be created and that can take some time.
	// Thus, we must set a higher deadline for it, if it's not set already.
	timeout := 30 * time.Second
	if req.WaitUntilAnswered {
		timeout = 80 * time.Second
	}
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	} else {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	resp, err := s.psrpcClient.CreateSIPParticipant(ctx, "", ireq, psrpc.WithRequestTimeout(timeout))
	if err != nil {
		unlikelyLogger.Errorw("cannot create sip participant", err)
		return nil, err
	}
	return &conkit.SIPParticipantInfo{
		ParticipantId:       resp.ParticipantId,
		ParticipantIdentity: resp.ParticipantIdentity,
		RoomName:            req.RoomName,
		SipCallId:           ireq.SipCallId,
	}, nil
}

func (s *SIPService) CreateSIPParticipantRequest(ctx context.Context, req *conkit.CreateSIPParticipantRequest, projectID, host, wsUrl, token string) (*rpc.InternalCreateSIPParticipantRequest, error) {
	if err := EnsureSIPCallPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	callID := sip.NewCallID()
	log := logger.GetLogger().WithUnlikelyValues(
		"callID", callID,
		"room", req.RoomName,
		"sipTrunk", req.SipTrunkId,
		"toUser", req.SipCallTo,
	)
	if projectID != "" {
		log = log.WithValues("projectID", projectID)
	}

	trunk, err := s.store.LoadSIPOutboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		log.Errorw("cannot get trunk to update sip participant", err)
		return nil, err
	}
	return rpc.NewCreateSIPParticipantRequest(projectID, callID, host, wsUrl, token, req, trunk)
}

func (s *SIPService) TransferSIPParticipant(ctx context.Context, req *conkit.TransferSIPParticipantRequest) (*emptypb.Empty, error) {
	log := logger.GetLogger().WithUnlikelyValues(
		"room", req.RoomName,
		"participant", req.ParticipantIdentity,
		"transferTo", req.TransferTo,
		"playDialtone", req.PlayDialtone,
	)
	AppendLogFields(ctx,
		"room", req.RoomName,
		"participant", req.ParticipantIdentity,
		"transferTo", req.TransferTo,
		"playDialtone", req.PlayDialtone,
	)

	ireq, err := s.transferSIPParticipantRequest(ctx, req)
	if err != nil {
		log.Errorw("cannot create transfer sip participant request", err)
		return nil, err
	}

	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	} else {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	_, err = s.psrpcClient.TransferSIPParticipant(ctx, ireq.SipCallId, ireq, psrpc.WithRequestTimeout(timeout))
	if err != nil {
		log.Errorw("cannot transfer sip participant", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *SIPService) transferSIPParticipantRequest(ctx context.Context, req *conkit.TransferSIPParticipantRequest) (*rpc.InternalTransferSIPParticipantRequest, error) {
	if req.RoomName == "" {
		return nil, psrpc.NewErrorf(psrpc.InvalidArgument, "Missing room name")
	}

	if req.ParticipantIdentity == "" {
		return nil, psrpc.NewErrorf(psrpc.InvalidArgument, "Missing participant identity")
	}

	if err := EnsureSIPCallPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if err := EnsureAdminPermission(ctx, conkit.RoomName(req.RoomName)); err != nil {
		return nil, twirpAuthError(err)
	}

	resp, err := s.roomService.GetParticipant(ctx, &conkit.RoomParticipantIdentity{
		Room:     req.RoomName,
		Identity: req.ParticipantIdentity,
	})

	if err != nil {
		return nil, err
	}

	callID, ok := resp.Attributes[conkit.AttrSIPCallID]
	if !ok {
		return nil, psrpc.NewErrorf(psrpc.InvalidArgument, "no SIP session associated with participant")
	}

	return &rpc.InternalTransferSIPParticipantRequest{
		SipCallId:    callID,
		TransferTo:   req.TransferTo,
		PlayDialtone: req.PlayDialtone,
	}, nil
}
