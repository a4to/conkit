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
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/a4to/conkit-server/pkg/agent"
	"github.com/a4to/conkit-server/pkg/sfu"
	sutils "github.com/a4to/conkit-server/pkg/utils"
	"github.com/a4to/mediatransportutil/pkg/rtcconfig"
	"github.com/a4to/protocol/auth"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/rpc"
	"github.com/a4to/protocol/utils"
	"github.com/a4to/protocol/utils/guid"
	"github.com/a4to/protocol/utils/must"
	"github.com/a4to/psrpc"
	"github.com/a4to/psrpc/pkg/middleware"

	"github.com/a4to/conkit-server/pkg/clientconfiguration"
	"github.com/a4to/conkit-server/pkg/config"
	"github.com/a4to/conkit-server/pkg/routing"
	"github.com/a4to/conkit-server/pkg/rtc"
	"github.com/a4to/conkit-server/pkg/rtc/types"
	"github.com/a4to/conkit-server/pkg/telemetry"
	"github.com/a4to/conkit-server/pkg/telemetry/prometheus"
	"github.com/a4to/conkit-server/version"
)

const (
	tokenRefreshInterval = 5 * time.Minute
	tokenDefaultTTL      = 10 * time.Minute
)

var affinityEpoch = time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC)

type iceConfigCacheKey struct {
	roomName            conkit.RoomName
	participantIdentity conkit.ParticipantIdentity
}

// RoomManager manages rooms and its interaction with participants.
// It's responsible for creating, deleting rooms, as well as running sessions for participants
type RoomManager struct {
	lock sync.RWMutex

	config            *config.Config
	rtcConfig         *rtc.WebRTCConfig
	serverInfo        *conkit.ServerInfo
	currentNode       routing.LocalNode
	router            routing.Router
	roomAllocator     RoomAllocator
	roomManagerServer rpc.TypedRoomManagerServer
	roomStore         ObjectStore
	telemetry         telemetry.TelemetryService
	clientConfManager clientconfiguration.ClientConfigurationManager
	agentClient       agent.Client
	agentStore        AgentStore
	egressLauncher    rtc.EgressLauncher
	versionGenerator  utils.TimedVersionGenerator
	turnAuthHandler   *TURNAuthHandler
	bus               psrpc.MessageBus

	rooms map[conkit.RoomName]*rtc.Room

	roomServers          utils.MultitonService[rpc.RoomTopic]
	agentDispatchServers utils.MultitonService[rpc.RoomTopic]
	participantServers   utils.MultitonService[rpc.ParticipantTopic]

	iceConfigCache *sutils.IceConfigCache[iceConfigCacheKey]

	forwardStats *sfu.ForwardStats
}

func NewLocalRoomManager(
	conf *config.Config,
	roomStore ObjectStore,
	currentNode routing.LocalNode,
	router routing.Router,
	roomAllocator RoomAllocator,
	telemetry telemetry.TelemetryService,
	clientConfManager clientconfiguration.ClientConfigurationManager,
	agentClient agent.Client,
	agentStore AgentStore,
	egressLauncher rtc.EgressLauncher,
	versionGenerator utils.TimedVersionGenerator,
	turnAuthHandler *TURNAuthHandler,
	bus psrpc.MessageBus,
	forwardStats *sfu.ForwardStats,
) (*RoomManager, error) {
	rtcConf, err := rtc.NewWebRTCConfig(conf)
	if err != nil {
		return nil, err
	}

	r := &RoomManager{
		config:            conf,
		rtcConfig:         rtcConf,
		currentNode:       currentNode,
		router:            router,
		roomAllocator:     roomAllocator,
		roomStore:         roomStore,
		telemetry:         telemetry,
		clientConfManager: clientConfManager,
		egressLauncher:    egressLauncher,
		agentClient:       agentClient,
		agentStore:        agentStore,
		versionGenerator:  versionGenerator,
		turnAuthHandler:   turnAuthHandler,
		bus:               bus,
		forwardStats:      forwardStats,

		rooms: make(map[conkit.RoomName]*rtc.Room),

		iceConfigCache: sutils.NewIceConfigCache[iceConfigCacheKey](0),

		serverInfo: &conkit.ServerInfo{
			Edition:       conkit.ServerInfo_Standard,
			Version:       version.Version,
			Protocol:      types.CurrentProtocol,
			AgentProtocol: agent.CurrentProtocol,
			Region:        conf.Region,
			NodeId:        string(currentNode.NodeID()),
		},
	}

	r.roomManagerServer, err = rpc.NewTypedRoomManagerServer(r, bus, rpc.WithServerLogger(logger.GetLogger()), middleware.WithServerMetrics(rpc.PSRPCMetricsObserver{}), psrpc.WithServerChannelSize(conf.PSRPC.BufferSize))
	if err != nil {
		return nil, err
	}
	if err := r.roomManagerServer.RegisterAllNodeTopics(currentNode.NodeID()); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RoomManager) GetRoom(_ context.Context, roomName conkit.RoomName) *rtc.Room {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.rooms[roomName]
}

// deleteRoom completely deletes all room information, including active sessions, room store, and routing info
func (r *RoomManager) deleteRoom(ctx context.Context, roomName conkit.RoomName) error {
	logger.Infow("deleting room state", "room", roomName)
	r.lock.Lock()
	delete(r.rooms, roomName)
	r.lock.Unlock()

	var err, err2 error
	wg := sync.WaitGroup{}
	wg.Add(2)
	// clear routing information
	go func() {
		defer wg.Done()
		err = r.router.ClearRoomState(ctx, roomName)
	}()
	// also delete room from db
	go func() {
		defer wg.Done()
		err2 = r.roomStore.DeleteRoom(ctx, roomName)
	}()

	wg.Wait()
	if err2 != nil {
		err = err2
	}

	return err
}

func (r *RoomManager) CloseIdleRooms() {
	r.lock.RLock()
	rooms := maps.Values(r.rooms)
	r.lock.RUnlock()

	for _, room := range rooms {
		reason := room.CloseIfEmpty()
		if reason != "" {
			room.Logger.Infow("closing idle room", "reason", reason)
		}
	}
}

func (r *RoomManager) HasParticipants() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for _, room := range r.rooms {
		if len(room.GetParticipants()) != 0 {
			return true
		}
	}
	return false
}

func (r *RoomManager) Stop() {
	// disconnect all clients
	r.lock.RLock()
	rooms := maps.Values(r.rooms)
	r.lock.RUnlock()

	for _, room := range rooms {
		room.Close(types.ParticipantCloseReasonRoomManagerStop)
	}

	r.roomManagerServer.Kill()
	r.roomServers.Kill()
	r.agentDispatchServers.Kill()
	r.participantServers.Kill()

	if r.rtcConfig != nil {
		if r.rtcConfig.UDPMux != nil {
			_ = r.rtcConfig.UDPMux.Close()
		}
		if r.rtcConfig.TCPMuxListener != nil {
			_ = r.rtcConfig.TCPMuxListener.Close()
		}
	}

	r.iceConfigCache.Stop()

	if r.forwardStats != nil {
		r.forwardStats.Stop()
	}
}

func (r *RoomManager) CreateRoom(ctx context.Context, req *conkit.CreateRoomRequest) (*conkit.Room, error) {
	room, err := r.getOrCreateRoom(ctx, req)
	if err != nil {
		return nil, err
	}
	defer room.Release()

	return room.ToProto(), nil
}

// StartSession starts WebRTC session when a new participant is connected, takes place on RTC node
func (r *RoomManager) StartSession(
	ctx context.Context,
	pi routing.ParticipantInit,
	requestSource routing.MessageSource,
	responseSink routing.MessageSink,
	useOneShotSignallingMode bool,
) error {
	sessionStartTime := time.Now()

	createRoom := pi.CreateRoom
	room, err := r.getOrCreateRoom(ctx, createRoom)
	if err != nil {
		return err
	}
	defer room.Release()

	protoRoom, roomInternal := room.ToProto(), room.Internal()

	// only create the room, but don't start a participant session
	if pi.Identity == "" {
		return nil
	}

	// should not error out, error is logged in iceServersForParticipant even if it fails
	// since this is used for TURN server credentials, we don't want to fail the request even if there's no TURN for the session
	apiKey, _, _ := r.getFirstKeyPair()

	participant := room.GetParticipant(pi.Identity)
	if participant != nil {
		// When reconnecting, it means WS has interrupted but underlying peer connection is still ok in this state,
		// we'll keep the participant SID, and just swap the sink for the underlying connection
		if pi.Reconnect {
			if participant.IsClosed() {
				// Send leave request if participant is closed, i. e. handle the case of client trying to resume crossing wires with
				// server closing the participant due to some irrecoverable condition. Such a condition would have triggered
				// a full reconnect when that condition occurred.
				//
				// It is possible that the client did not get that send request. So, send it again.
				logger.Infow("cannot restart a closed participant",
					"room", room.Name(),
					"nodeID", r.currentNode.NodeID(),
					"participant", pi.Identity,
					"reason", pi.ReconnectReason,
				)

				var leave *conkit.LeaveRequest
				pv := types.ProtocolVersion(pi.Client.Protocol)
				if pv.SupportsRegionsInLeaveRequest() {
					leave = &conkit.LeaveRequest{
						Reason: conkit.DisconnectReason_STATE_MISMATCH,
						Action: conkit.LeaveRequest_RECONNECT,
					}
				} else {
					leave = &conkit.LeaveRequest{
						CanReconnect: true,
						Reason:       conkit.DisconnectReason_STATE_MISMATCH,
					}
				}
				_ = responseSink.WriteMessage(&conkit.SignalResponse{
					Message: &conkit.SignalResponse_Leave{
						Leave: leave,
					},
				})
				return errors.New("could not restart closed participant")
			}

			participant.GetLogger().Infow("resuming RTC session",
				"nodeID", r.currentNode.NodeID(),
				"participantInit", &pi,
				"numParticipants", room.GetParticipantCount(),
			)
			iceConfig := r.getIceConfig(room.Name(), participant)
			if err = room.ResumeParticipant(
				participant,
				requestSource,
				responseSink,
				iceConfig,
				r.iceServersForParticipant(
					apiKey,
					participant,
					iceConfig.PreferenceSubscriber == conkit.ICECandidateType_ICT_TLS,
				),
				pi.ReconnectReason,
			); err != nil {
				participant.GetLogger().Warnw("could not resume participant", err)
				return err
			}
			r.telemetry.ParticipantResumed(ctx, room.ToProto(), participant.ToProto(), r.currentNode.NodeID(), pi.ReconnectReason)
			go r.rtcSessionWorker(room, participant, requestSource)
			return nil
		}

		// we need to clean up the existing participant, so a new one can join
		participant.GetLogger().Infow("removing duplicate participant")
		room.RemoveParticipant(participant.Identity(), participant.ID(), types.ParticipantCloseReasonDuplicateIdentity)
	} else if pi.Reconnect {
		// send leave request if participant is trying to reconnect without keep subscribe state
		// but missing from the room
		var leave *conkit.LeaveRequest
		pv := types.ProtocolVersion(pi.Client.Protocol)
		if pv.SupportsRegionsInLeaveRequest() {
			leave = &conkit.LeaveRequest{
				Reason: conkit.DisconnectReason_STATE_MISMATCH,
				Action: conkit.LeaveRequest_RECONNECT,
			}
		} else {
			leave = &conkit.LeaveRequest{
				CanReconnect: true,
				Reason:       conkit.DisconnectReason_STATE_MISMATCH,
			}
		}
		_ = responseSink.WriteMessage(&conkit.SignalResponse{
			Message: &conkit.SignalResponse_Leave{
				Leave: leave,
			},
		})
		return errors.New("could not restart participant")
	}

	sid := conkit.ParticipantID(guid.New(utils.ParticipantPrefix))
	pLogger := rtc.LoggerWithParticipant(
		rtc.LoggerWithRoom(logger.GetLogger(), room.Name(), room.ID()),
		pi.Identity,
		sid,
		false,
	)
	pLogger.Infow("starting RTC session",
		"room", room.Name(),
		"nodeID", r.currentNode.NodeID(),
		"numParticipants", room.GetParticipantCount(),
		"participantInit", &pi,
	)

	clientConf := r.clientConfManager.GetConfiguration(pi.Client)

	pv := types.ProtocolVersion(pi.Client.Protocol)
	rtcConf := *r.rtcConfig
	rtcConf.SetBufferFactory(room.GetBufferFactory())
	if pi.DisableICELite {
		rtcConf.SettingEngine.SetLite(false)
	}
	// default allow forceTCP
	allowFallback := true
	if r.config.RTC.AllowTCPFallback != nil {
		allowFallback = *r.config.RTC.AllowTCPFallback
	}
	// default do not force full reconnect on a publication error
	reconnectOnPublicationError := false
	if r.config.RTC.ReconnectOnPublicationError != nil {
		reconnectOnPublicationError = *r.config.RTC.ReconnectOnPublicationError
	}
	// default do not force full reconnect on a subscription error
	reconnectOnSubscriptionError := false
	if r.config.RTC.ReconnectOnSubscriptionError != nil {
		reconnectOnSubscriptionError = *r.config.RTC.ReconnectOnSubscriptionError
	}
	// default do not force full reconnect on a data channel error
	reconnectOnDataChannelError := false
	if r.config.RTC.ReconnectOnDataChannelError != nil {
		reconnectOnDataChannelError = *r.config.RTC.ReconnectOnDataChannelError
	}
	subscriberAllowPause := r.config.RTC.CongestionControl.AllowPause
	if pi.SubscriberAllowPause != nil {
		subscriberAllowPause = *pi.SubscriberAllowPause
	}
	participant, err = rtc.NewParticipant(rtc.ParticipantParams{
		Identity:                pi.Identity,
		Name:                    pi.Name,
		SID:                     sid,
		Config:                  &rtcConf,
		Sink:                    responseSink,
		AudioConfig:             r.config.Audio,
		VideoConfig:             r.config.Video,
		LimitConfig:             r.config.Limit,
		ProtocolVersion:         pv,
		SessionStartTime:        sessionStartTime,
		Telemetry:               r.telemetry,
		Trailer:                 room.Trailer(),
		PLIThrottleConfig:       r.config.RTC.PLIThrottle,
		CongestionControlConfig: r.config.RTC.CongestionControl,
		PublishEnabledCodecs:    protoRoom.EnabledCodecs,
		SubscribeEnabledCodecs:  protoRoom.EnabledCodecs,
		Grants:                  pi.Grants,
		Reconnect:               pi.Reconnect,
		Logger:                  pLogger,
		ClientConf:              clientConf,
		ClientInfo:              rtc.ClientInfo{ClientInfo: pi.Client},
		Region:                  pi.Region,
		AdaptiveStream:          pi.AdaptiveStream,
		AllowTCPFallback:        allowFallback,
		TURNSEnabled:            r.config.IsTURNSEnabled(),
		GetParticipantInfo: func(pID conkit.ParticipantID) *conkit.ParticipantInfo {
			if p := room.GetParticipantByID(pID); p != nil {
				return p.ToProto()
			}
			return nil
		},
		ReconnectOnPublicationError:  reconnectOnPublicationError,
		ReconnectOnSubscriptionError: reconnectOnSubscriptionError,
		ReconnectOnDataChannelError:  reconnectOnDataChannelError,
		VersionGenerator:             r.versionGenerator,
		TrackResolver:                room.ResolveMediaTrackForSubscriber,
		SubscriberAllowPause:         subscriberAllowPause,
		SubscriptionLimitAudio:       r.config.Limit.SubscriptionLimitAudio,
		SubscriptionLimitVideo:       r.config.Limit.SubscriptionLimitVideo,
		PlayoutDelay:                 roomInternal.GetPlayoutDelay(),
		SyncStreams:                  roomInternal.GetSyncStreams(),
		ForwardStats:                 r.forwardStats,
		MetricConfig:                 r.config.Metric,
		UseOneShotSignallingMode:     useOneShotSignallingMode,
		DataChannelMaxBufferedAmount: r.config.RTC.DataChannelMaxBufferedAmount,
		DatachannelSlowThreshold:     r.config.RTC.DatachannelSlowThreshold,
		FireOnTrackBySdp:             true,
	})
	if err != nil {
		return err
	}
	iceConfig := r.setIceConfig(room.Name(), participant)

	// join room
	opts := rtc.ParticipantOptions{
		AutoSubscribe: pi.AutoSubscribe,
	}
	iceServers := r.iceServersForParticipant(apiKey, participant, iceConfig.PreferenceSubscriber == conkit.ICECandidateType_ICT_TLS)
	if err = room.Join(participant, requestSource, &opts, iceServers); err != nil {
		pLogger.Errorw("could not join room", err)
		_ = participant.Close(true, types.ParticipantCloseReasonJoinFailed, false)
		return err
	}

	participantTopic := rpc.FormatParticipantTopic(room.Name(), participant.Identity())
	participantServer := must.Get(rpc.NewTypedParticipantServer(r, r.bus))
	killParticipantServer := r.participantServers.Replace(participantTopic, participantServer)
	if err := participantServer.RegisterAllParticipantTopics(participantTopic); err != nil {
		killParticipantServer()
		pLogger.Errorw("could not join register participant topic", err)
		_ = participant.Close(true, types.ParticipantCloseReasonMessageBusFailed, false)
		return err
	}

	if err = r.roomStore.StoreParticipant(ctx, room.Name(), participant.ToProto()); err != nil {
		pLogger.Errorw("could not store participant", err)
	}

	persistRoomForParticipantCount := func(proto *conkit.Room) {
		if !participant.Hidden() && !room.IsClosed() {
			err = r.roomStore.StoreRoom(ctx, proto, room.Internal())
			if err != nil {
				logger.Errorw("could not store room", err)
			}
		}
	}

	// update room store with new numParticipants
	persistRoomForParticipantCount(room.ToProto())

	clientMeta := &conkit.AnalyticsClientMeta{Region: r.currentNode.Region(), Node: string(r.currentNode.NodeID())}
	r.telemetry.ParticipantJoined(ctx, protoRoom, participant.ToProto(), pi.Client, clientMeta, true)
	participant.OnClose(func(p types.LocalParticipant) {
		killParticipantServer()

		if err := r.roomStore.DeleteParticipant(ctx, room.Name(), p.Identity()); err != nil {
			pLogger.Errorw("could not delete participant", err)
		}

		// update room store with new numParticipants
		proto := room.ToProto()
		persistRoomForParticipantCount(proto)
		r.telemetry.ParticipantLeft(ctx, proto, p.ToProto(), true)
	})
	participant.OnClaimsChanged(func(participant types.LocalParticipant) {
		pLogger.Debugw("refreshing client token after claims change")
		if err := r.refreshToken(participant); err != nil {
			pLogger.Errorw("could not refresh token", err)
		}
	})
	participant.OnICEConfigChanged(func(participant types.LocalParticipant, iceConfig *conkit.ICEConfig) {
		r.iceConfigCache.Put(iceConfigCacheKey{room.Name(), participant.Identity()}, iceConfig)
	})

	go r.rtcSessionWorker(room, participant, requestSource)
	return nil
}

// create the actual room object, to be used on RTC node
func (r *RoomManager) getOrCreateRoom(ctx context.Context, createRoom *conkit.CreateRoomRequest) (*rtc.Room, error) {
	roomName := conkit.RoomName(createRoom.Name)

	r.lock.RLock()
	lastSeenRoom := r.rooms[roomName]
	r.lock.RUnlock()

	if lastSeenRoom != nil && lastSeenRoom.Hold() {
		return lastSeenRoom, nil
	}

	// create new room, get details first
	ri, internal, created, err := r.roomAllocator.CreateRoom(ctx, createRoom, true)
	if err != nil {
		return nil, err
	}

	r.lock.Lock()

	currentRoom := r.rooms[roomName]
	for currentRoom != lastSeenRoom {
		r.lock.Unlock()
		if currentRoom != nil && currentRoom.Hold() {
			return currentRoom, nil
		}

		lastSeenRoom = currentRoom
		r.lock.Lock()
		currentRoom = r.rooms[roomName]
	}

	// construct ice servers
	newRoom := rtc.NewRoom(ri, internal, *r.rtcConfig, r.config.Room, &r.config.Audio, r.serverInfo, r.telemetry, r.agentClient, r.agentStore, r.egressLauncher)

	roomTopic := rpc.FormatRoomTopic(roomName)
	roomServer := must.Get(rpc.NewTypedRoomServer(r, r.bus))
	killRoomServer := r.roomServers.Replace(roomTopic, roomServer)
	if err := roomServer.RegisterAllRoomTopics(roomTopic); err != nil {
		killRoomServer()
		r.lock.Unlock()
		return nil, err
	}
	agentDispatchServer := must.Get(rpc.NewTypedAgentDispatchInternalServer(r, r.bus))
	killDispServer := r.agentDispatchServers.Replace(roomTopic, agentDispatchServer)
	if err := agentDispatchServer.RegisterAllRoomTopics(roomTopic); err != nil {
		killRoomServer()
		killDispServer()
		r.lock.Unlock()
		return nil, err
	}

	newRoom.OnClose(func() {
		killRoomServer()
		killDispServer()

		roomInfo := newRoom.ToProto()
		r.telemetry.RoomEnded(ctx, roomInfo)
		prometheus.RoomEnded(time.Unix(roomInfo.CreationTime, 0))
		if err := r.deleteRoom(ctx, roomName); err != nil {
			newRoom.Logger.Errorw("could not delete room", err)
		}

		newRoom.Logger.Infow("room closed")
	})

	newRoom.OnRoomUpdated(func() {
		if err := r.roomStore.StoreRoom(ctx, newRoom.ToProto(), newRoom.Internal()); err != nil {
			newRoom.Logger.Errorw("could not handle metadata update", err)
		}
	})

	newRoom.OnParticipantChanged(func(p types.LocalParticipant) {
		if !p.IsDisconnected() {
			if err := r.roomStore.StoreParticipant(ctx, roomName, p.ToProto()); err != nil {
				newRoom.Logger.Errorw("could not handle participant change", err)
			}
		}
	})

	r.rooms[roomName] = newRoom

	r.lock.Unlock()

	newRoom.Hold()

	r.telemetry.RoomStarted(ctx, newRoom.ToProto())
	prometheus.RoomStarted()

	if created && createRoom.GetEgress().GetRoom() != nil {
		// ensure room name matches
		createRoom.Egress.Room.RoomName = createRoom.Name
		_, err = r.egressLauncher.StartEgress(ctx, &rpc.StartEgressRequest{
			Request: &rpc.StartEgressRequest_RoomComposite{
				RoomComposite: createRoom.Egress.Room,
			},
			RoomId: ri.Sid,
		})
		if err != nil {
			newRoom.Release()
			return nil, err
		}
	}

	return newRoom, nil
}

// manages an RTC session for a participant, runs on the RTC node
func (r *RoomManager) rtcSessionWorker(room *rtc.Room, participant types.LocalParticipant, requestSource routing.MessageSource) {
	pLogger := rtc.LoggerWithParticipant(
		rtc.LoggerWithRoom(logger.GetLogger(), room.Name(), room.ID()),
		participant.Identity(),
		participant.ID(),
		false,
	)
	defer func() {
		pLogger.Debugw("RTC session finishing", "connID", requestSource.ConnectionID())
		requestSource.Close()
	}()

	defer func() {
		if r := rtc.Recover(pLogger); r != nil {
			os.Exit(1)
		}
	}()

	// send first refresh for cases when client token is close to expiring
	_ = r.refreshToken(participant)
	tokenTicker := time.NewTicker(tokenRefreshInterval)
	defer tokenTicker.Stop()
	for {
		select {
		case <-participant.Disconnected():
			return
		case <-tokenTicker.C:
			// refresh token with the first API Key/secret pair
			if err := r.refreshToken(participant); err != nil {
				pLogger.Errorw("could not refresh token", err, "connID", requestSource.ConnectionID())
			}
		case obj := <-requestSource.ReadChan():
			if obj == nil {
				if room.GetParticipantRequestSource(participant.Identity()) == requestSource {
					participant.HandleSignalSourceClose()
				}
				return
			}

			req := obj.(*conkit.SignalRequest)
			if err := rtc.HandleParticipantSignal(room, participant, req, pLogger); err != nil {
				// more specific errors are already logged
				// treat errors returned as fatal
				return
			}
		}
	}
}

type participantReq interface {
	GetRoom() string
	GetIdentity() string
}

func (r *RoomManager) roomAndParticipantForReq(ctx context.Context, req participantReq) (*rtc.Room, types.LocalParticipant, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.GetRoom()))
	if room == nil {
		return nil, nil, ErrRoomNotFound
	}

	participant := room.GetParticipant(conkit.ParticipantIdentity(req.GetIdentity()))
	if participant == nil {
		return nil, nil, ErrParticipantNotFound
	}

	return room, participant, nil
}

func (r *RoomManager) RemoveParticipant(ctx context.Context, req *conkit.RoomParticipantIdentity) (*conkit.RemoveParticipantResponse, error) {
	room, participant, err := r.roomAndParticipantForReq(ctx, req)
	if err != nil {
		return nil, err
	}

	participant.GetLogger().Infow("removing participant")
	room.RemoveParticipant(conkit.ParticipantIdentity(req.Identity), "", types.ParticipantCloseReasonServiceRequestRemoveParticipant)
	return &conkit.RemoveParticipantResponse{}, nil
}

func (r *RoomManager) MutePublishedTrack(ctx context.Context, req *conkit.MuteRoomTrackRequest) (*conkit.MuteRoomTrackResponse, error) {
	_, participant, err := r.roomAndParticipantForReq(ctx, req)
	if err != nil {
		return nil, err
	}

	participant.GetLogger().Debugw("setting track muted",
		"trackID", req.TrackSid, "muted", req.Muted)
	if !req.Muted && !r.config.Room.EnableRemoteUnmute {
		participant.GetLogger().Errorw("cannot unmute track, remote unmute is disabled", nil)
		return nil, ErrRemoteUnmuteNoteEnabled
	}
	track := participant.SetTrackMuted(conkit.TrackID(req.TrackSid), req.Muted, true)
	return &conkit.MuteRoomTrackResponse{Track: track}, nil
}

func (r *RoomManager) UpdateParticipant(ctx context.Context, req *conkit.UpdateParticipantRequest) (*conkit.ParticipantInfo, error) {
	_, participant, err := r.roomAndParticipantForReq(ctx, req)
	if err != nil {
		return nil, err
	}

	participant.GetLogger().Debugw("updating participant",
		"metadata", req.Metadata,
		"permission", req.Permission,
		"attributes", req.Attributes,
	)
	if err = participant.CheckMetadataLimits(req.Name, req.Metadata, req.Attributes); err != nil {
		return nil, err
	}

	if req.Name != "" {
		participant.SetName(req.Name)
	}
	if req.Metadata != "" {
		participant.SetMetadata(req.Metadata)
	}
	if req.Attributes != nil {
		participant.SetAttributes(req.Attributes)
	}

	if req.Permission != nil {
		participant.SetPermission(req.Permission)
	}
	return participant.ToProto(), nil
}

func (r *RoomManager) DeleteRoom(ctx context.Context, req *conkit.DeleteRoomRequest) (*conkit.DeleteRoomResponse, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.Room))
	if room == nil {
		// special case of a non-RTC room e.g. room created but no participants joined
		logger.Debugw("Deleting non-rtc room, loading from roomstore")
		err := r.roomStore.DeleteRoom(ctx, conkit.RoomName(req.Room))
		if err != nil {
			logger.Debugw("Error deleting non-rtc room", "err", err)
			return nil, err
		}
	} else {
		room.Logger.Infow("deleting room")
		room.Close(types.ParticipantCloseReasonServiceRequestDeleteRoom)
	}
	return &conkit.DeleteRoomResponse{}, nil
}

func (r *RoomManager) UpdateSubscriptions(ctx context.Context, req *conkit.UpdateSubscriptionsRequest) (*conkit.UpdateSubscriptionsResponse, error) {
	room, participant, err := r.roomAndParticipantForReq(ctx, req)
	if err != nil {
		return nil, err
	}

	participant.GetLogger().Debugw("updating participant subscriptions")
	room.UpdateSubscriptions(
		participant,
		conkit.StringsAsIDs[conkit.TrackID](req.TrackSids),
		req.ParticipantTracks,
		req.Subscribe,
	)
	return &conkit.UpdateSubscriptionsResponse{}, nil
}

func (r *RoomManager) SendData(ctx context.Context, req *conkit.SendDataRequest) (*conkit.SendDataResponse, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.Room))
	if room == nil {
		return nil, ErrRoomNotFound
	}

	room.Logger.Debugw("api send data", "size", len(req.Data))
	room.SendDataPacket(&conkit.DataPacket{
		Kind:                  req.Kind,
		DestinationIdentities: req.DestinationIdentities,
		Value: &conkit.DataPacket_User{
			User: &conkit.UserPacket{
				Payload:               req.Data,
				DestinationSids:       req.DestinationSids,
				DestinationIdentities: req.DestinationIdentities,
				Topic:                 req.Topic,
				Nonce:                 req.Nonce,
			},
		},
	}, req.Kind)
	return &conkit.SendDataResponse{}, nil
}

func (r *RoomManager) UpdateRoomMetadata(ctx context.Context, req *conkit.UpdateRoomMetadataRequest) (*conkit.Room, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.Room))
	if room == nil {
		return nil, ErrRoomNotFound
	}

	room.Logger.Debugw("updating room")
	done := room.SetMetadata(req.Metadata)
	// wait till the update is applied
	<-done
	return room.ToProto(), nil
}

func (r *RoomManager) ListDispatch(ctx context.Context, req *conkit.ListAgentDispatchRequest) (*conkit.ListAgentDispatchResponse, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.Room))
	if room == nil {
		return nil, ErrRoomNotFound
	}

	disp, err := room.GetAgentDispatches(req.DispatchId)
	if err != nil {
		return nil, err
	}

	ret := &conkit.ListAgentDispatchResponse{
		AgentDispatches: disp,
	}

	return ret, nil
}

func (r *RoomManager) CreateDispatch(ctx context.Context, req *conkit.AgentDispatch) (*conkit.AgentDispatch, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.Room))
	if room == nil {
		return nil, ErrRoomNotFound
	}

	disp, err := room.AddAgentDispatch(req)
	if err != nil {
		return nil, err
	}

	return disp, nil
}

func (r *RoomManager) DeleteDispatch(ctx context.Context, req *conkit.DeleteAgentDispatchRequest) (*conkit.AgentDispatch, error) {
	room := r.GetRoom(ctx, conkit.RoomName(req.Room))
	if room == nil {
		return nil, ErrRoomNotFound
	}

	disp, err := room.DeleteAgentDispatch(req.DispatchId)
	if err != nil {
		return nil, err
	}

	return disp, nil
}

func (r *RoomManager) iceServersForParticipant(apiKey string, participant types.LocalParticipant, tlsOnly bool) []*conkit.ICEServer {
	var iceServers []*conkit.ICEServer
	rtcConf := r.config.RTC

	if tlsOnly && r.config.TURN.TLSPort == 0 {
		logger.Warnw("tls only enabled but no turn tls config", nil)
		tlsOnly = false
	}

	hasSTUN := false
	if r.config.TURN.Enabled {
		var urls []string
		if r.config.TURN.UDPPort > 0 && !tlsOnly {
			// UDP TURN is used as STUN
			hasSTUN = true
			urls = append(urls, fmt.Sprintf("turn:%s:%d?transport=udp", r.config.RTC.NodeIP, r.config.TURN.UDPPort))
		}
		if r.config.TURN.TLSPort > 0 {
			urls = append(urls, fmt.Sprintf("turns:%s:443?transport=tcp", r.config.TURN.Domain))
		}
		if len(urls) > 0 {
			username := r.turnAuthHandler.CreateUsername(apiKey, participant.ID())
			password, err := r.turnAuthHandler.CreatePassword(apiKey, participant.ID())
			if err != nil {
				participant.GetLogger().Warnw("could not create turn password", err)
				hasSTUN = false
			} else {
				logger.Infow("created TURN password", "username", username, "password", password)
				iceServers = append(iceServers, &conkit.ICEServer{
					Urls:       urls,
					Username:   username,
					Credential: password,
				})
			}
		}
	}

	if len(rtcConf.TURNServers) > 0 {
		hasSTUN = true
		for _, s := range r.config.RTC.TURNServers {
			scheme := "turn"
			transport := "tcp"
			if s.Protocol == "tls" {
				scheme = "turns"
			} else if s.Protocol == "udp" {
				transport = "udp"
			}
			is := &conkit.ICEServer{
				Urls: []string{
					fmt.Sprintf("%s:%s:%d?transport=%s", scheme, s.Host, s.Port, transport),
				},
				Username:   s.Username,
				Credential: s.Credential,
			}
			iceServers = append(iceServers, is)
		}
	}

	if len(rtcConf.STUNServers) > 0 {
		hasSTUN = true
		iceServers = append(iceServers, iceServerForStunServers(r.config.RTC.STUNServers))
	}

	if !hasSTUN {
		iceServers = append(iceServers, iceServerForStunServers(rtcconfig.DefaultStunServers))
	}
	return iceServers
}

func (r *RoomManager) refreshToken(participant types.LocalParticipant) error {
	key, secret, err := r.getFirstKeyPair()
	if err != nil {
		return err
	}

	grants := participant.ClaimGrants()
	token := auth.NewAccessToken(key, secret)
	token.SetName(grants.Name).
		SetIdentity(string(participant.Identity())).
		SetValidFor(tokenDefaultTTL).
		SetMetadata(grants.Metadata).
		SetAttributes(grants.Attributes).
		SetVideoGrant(grants.Video).
		SetRoomConfig(grants.GetRoomConfiguration()).
		SetRoomPreset(grants.RoomPreset)
	jwt, err := token.ToJWT()
	if err == nil {
		err = participant.SendRefreshToken(jwt)
	}
	if err != nil {
		return err
	}

	return nil
}

func (r *RoomManager) setIceConfig(roomName conkit.RoomName, participant types.LocalParticipant) *conkit.ICEConfig {
	iceConfig := r.getIceConfig(roomName, participant)
	participant.SetICEConfig(iceConfig)
	return iceConfig
}

func (r *RoomManager) getIceConfig(roomName conkit.RoomName, participant types.LocalParticipant) *conkit.ICEConfig {
	return r.iceConfigCache.Get(iceConfigCacheKey{roomName, participant.Identity()})
}

func (r *RoomManager) getFirstKeyPair() (string, string, error) {
	for key, secret := range r.config.Keys {
		return key, secret, nil
	}
	return "", "", errors.New("no API keys configured")
}

// ------------------------------------

func iceServerForStunServers(servers []string) *conkit.ICEServer {
	iceServer := &conkit.ICEServer{}
	for _, stunServer := range servers {
		iceServer.Urls = append(iceServer.Urls, fmt.Sprintf("stun:%s", stunServer))
	}
	return iceServer
}
