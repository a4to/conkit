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

package types

import (
	"fmt"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"

	"github.com/a4to/protocol/auth"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/utils"

	"github.com/a4to/conkit-server/pkg/routing"
	"github.com/a4to/conkit-server/pkg/sfu"
	"github.com/a4to/conkit-server/pkg/sfu/buffer"
	"github.com/a4to/conkit-server/pkg/sfu/mime"
	"github.com/a4to/conkit-server/pkg/sfu/pacer"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . WebsocketClient
type WebsocketClient interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	SetReadDeadline(deadline time.Time) error
	Close() error
}

type AddSubscriberParams struct {
	AllTracks bool
	TrackIDs  []conkit.TrackID
}

// ---------------------------------------------

type MigrateState int32

const (
	MigrateStateInit MigrateState = iota
	MigrateStateSync
	MigrateStateComplete
)

func (m MigrateState) String() string {
	switch m {
	case MigrateStateInit:
		return "MIGRATE_STATE_INIT"
	case MigrateStateSync:
		return "MIGRATE_STATE_SYNC"
	case MigrateStateComplete:
		return "MIGRATE_STATE_COMPLETE"
	default:
		return fmt.Sprintf("%d", int(m))
	}
}

// ---------------------------------------------

type SubscribedCodecQuality struct {
	CodecMime mime.MimeType
	Quality   conkit.VideoQuality
}

// ---------------------------------------------

type ParticipantCloseReason int

const (
	ParticipantCloseReasonNone ParticipantCloseReason = iota
	ParticipantCloseReasonClientRequestLeave
	ParticipantCloseReasonRoomManagerStop
	ParticipantCloseReasonVerifyFailed
	ParticipantCloseReasonJoinFailed
	ParticipantCloseReasonJoinTimeout
	ParticipantCloseReasonMessageBusFailed
	ParticipantCloseReasonPeerConnectionDisconnected
	ParticipantCloseReasonDuplicateIdentity
	ParticipantCloseReasonMigrationComplete
	ParticipantCloseReasonStale
	ParticipantCloseReasonServiceRequestRemoveParticipant
	ParticipantCloseReasonServiceRequestDeleteRoom
	ParticipantCloseReasonSimulateMigration
	ParticipantCloseReasonSimulateNodeFailure
	ParticipantCloseReasonSimulateServerLeave
	ParticipantCloseReasonSimulateLeaveRequest
	ParticipantCloseReasonNegotiateFailed
	ParticipantCloseReasonMigrationRequested
	ParticipantCloseReasonPublicationError
	ParticipantCloseReasonSubscriptionError
	ParticipantCloseReasonDataChannelError
	ParticipantCloseReasonMigrateCodecMismatch
	ParticipantCloseReasonSignalSourceClose
	ParticipantCloseReasonRoomClosed
	ParticipantCloseReasonUserUnavailable
	ParticipantCloseReasonUserRejected
)

func (p ParticipantCloseReason) String() string {
	switch p {
	case ParticipantCloseReasonNone:
		return "NONE"
	case ParticipantCloseReasonClientRequestLeave:
		return "CLIENT_REQUEST_LEAVE"
	case ParticipantCloseReasonRoomManagerStop:
		return "ROOM_MANAGER_STOP"
	case ParticipantCloseReasonVerifyFailed:
		return "VERIFY_FAILED"
	case ParticipantCloseReasonJoinFailed:
		return "JOIN_FAILED"
	case ParticipantCloseReasonJoinTimeout:
		return "JOIN_TIMEOUT"
	case ParticipantCloseReasonMessageBusFailed:
		return "MESSAGE_BUS_FAILED"
	case ParticipantCloseReasonPeerConnectionDisconnected:
		return "PEER_CONNECTION_DISCONNECTED"
	case ParticipantCloseReasonDuplicateIdentity:
		return "DUPLICATE_IDENTITY"
	case ParticipantCloseReasonMigrationComplete:
		return "MIGRATION_COMPLETE"
	case ParticipantCloseReasonStale:
		return "STALE"
	case ParticipantCloseReasonServiceRequestRemoveParticipant:
		return "SERVICE_REQUEST_REMOVE_PARTICIPANT"
	case ParticipantCloseReasonServiceRequestDeleteRoom:
		return "SERVICE_REQUEST_DELETE_ROOM"
	case ParticipantCloseReasonSimulateMigration:
		return "SIMULATE_MIGRATION"
	case ParticipantCloseReasonSimulateNodeFailure:
		return "SIMULATE_NODE_FAILURE"
	case ParticipantCloseReasonSimulateServerLeave:
		return "SIMULATE_SERVER_LEAVE"
	case ParticipantCloseReasonSimulateLeaveRequest:
		return "SIMULATE_LEAVE_REQUEST"
	case ParticipantCloseReasonNegotiateFailed:
		return "NEGOTIATE_FAILED"
	case ParticipantCloseReasonMigrationRequested:
		return "MIGRATION_REQUESTED"
	case ParticipantCloseReasonPublicationError:
		return "PUBLICATION_ERROR"
	case ParticipantCloseReasonSubscriptionError:
		return "SUBSCRIPTION_ERROR"
	case ParticipantCloseReasonDataChannelError:
		return "DATA_CHANNEL_ERROR"
	case ParticipantCloseReasonMigrateCodecMismatch:
		return "MIGRATE_CODEC_MISMATCH"
	case ParticipantCloseReasonSignalSourceClose:
		return "SIGNAL_SOURCE_CLOSE"
	case ParticipantCloseReasonRoomClosed:
		return "ROOM_CLOSED"
	case ParticipantCloseReasonUserUnavailable:
		return "USER_UNAVAILABLE"
	case ParticipantCloseReasonUserRejected:
		return "USER_REJECTED"
	default:
		return fmt.Sprintf("%d", int(p))
	}
}

func (p ParticipantCloseReason) ToDisconnectReason() conkit.DisconnectReason {
	switch p {
	case ParticipantCloseReasonClientRequestLeave, ParticipantCloseReasonSimulateLeaveRequest:
		return conkit.DisconnectReason_CLIENT_INITIATED
	case ParticipantCloseReasonRoomManagerStop:
		return conkit.DisconnectReason_SERVER_SHUTDOWN
	case ParticipantCloseReasonVerifyFailed, ParticipantCloseReasonJoinFailed, ParticipantCloseReasonJoinTimeout, ParticipantCloseReasonMessageBusFailed:
		// expected to be connected but is not
		return conkit.DisconnectReason_JOIN_FAILURE
	case ParticipantCloseReasonPeerConnectionDisconnected:
		return conkit.DisconnectReason_STATE_MISMATCH
	case ParticipantCloseReasonDuplicateIdentity, ParticipantCloseReasonStale:
		return conkit.DisconnectReason_DUPLICATE_IDENTITY
	case ParticipantCloseReasonMigrationRequested, ParticipantCloseReasonMigrationComplete, ParticipantCloseReasonSimulateMigration:
		return conkit.DisconnectReason_MIGRATION
	case ParticipantCloseReasonServiceRequestRemoveParticipant:
		return conkit.DisconnectReason_PARTICIPANT_REMOVED
	case ParticipantCloseReasonServiceRequestDeleteRoom:
		return conkit.DisconnectReason_ROOM_DELETED
	case ParticipantCloseReasonSimulateNodeFailure, ParticipantCloseReasonSimulateServerLeave:
		return conkit.DisconnectReason_SERVER_SHUTDOWN
	case ParticipantCloseReasonNegotiateFailed, ParticipantCloseReasonPublicationError, ParticipantCloseReasonSubscriptionError, ParticipantCloseReasonDataChannelError, ParticipantCloseReasonMigrateCodecMismatch:
		return conkit.DisconnectReason_STATE_MISMATCH
	case ParticipantCloseReasonSignalSourceClose:
		return conkit.DisconnectReason_SIGNAL_CLOSE
	case ParticipantCloseReasonRoomClosed:
		return conkit.DisconnectReason_ROOM_CLOSED
	case ParticipantCloseReasonUserUnavailable:
		return conkit.DisconnectReason_USER_UNAVAILABLE
	case ParticipantCloseReasonUserRejected:
		return conkit.DisconnectReason_USER_REJECTED
	default:
		// the other types will map to unknown reason
		return conkit.DisconnectReason_UNKNOWN_REASON
	}
}

// ---------------------------------------------

type SignallingCloseReason int

const (
	SignallingCloseReasonUnknown SignallingCloseReason = iota
	SignallingCloseReasonMigration
	SignallingCloseReasonResume
	SignallingCloseReasonTransportFailure
	SignallingCloseReasonFullReconnectPublicationError
	SignallingCloseReasonFullReconnectSubscriptionError
	SignallingCloseReasonFullReconnectDataChannelError
	SignallingCloseReasonFullReconnectNegotiateFailed
	SignallingCloseReasonParticipantClose
	SignallingCloseReasonDisconnectOnResume
	SignallingCloseReasonDisconnectOnResumeNoMessages
)

func (s SignallingCloseReason) String() string {
	switch s {
	case SignallingCloseReasonUnknown:
		return "UNKNOWN"
	case SignallingCloseReasonMigration:
		return "MIGRATION"
	case SignallingCloseReasonResume:
		return "RESUME"
	case SignallingCloseReasonTransportFailure:
		return "TRANSPORT_FAILURE"
	case SignallingCloseReasonFullReconnectPublicationError:
		return "FULL_RECONNECT_PUBLICATION_ERROR"
	case SignallingCloseReasonFullReconnectSubscriptionError:
		return "FULL_RECONNECT_SUBSCRIPTION_ERROR"
	case SignallingCloseReasonFullReconnectDataChannelError:
		return "FULL_RECONNECT_DATA_CHANNEL_ERROR"
	case SignallingCloseReasonFullReconnectNegotiateFailed:
		return "FULL_RECONNECT_NEGOTIATE_FAILED"
	case SignallingCloseReasonParticipantClose:
		return "PARTICIPANT_CLOSE"
	case SignallingCloseReasonDisconnectOnResume:
		return "DISCONNECT_ON_RESUME"
	case SignallingCloseReasonDisconnectOnResumeNoMessages:
		return "DISCONNECT_ON_RESUME_NO_MESSAGES"
	default:
		return fmt.Sprintf("%d", int(s))
	}
}

// ---------------------------------------------

//counterfeiter:generate . Participant
type Participant interface {
	ID() conkit.ParticipantID
	Identity() conkit.ParticipantIdentity
	State() conkit.ParticipantInfo_State
	ConnectedAt() time.Time
	CloseReason() ParticipantCloseReason
	Kind() conkit.ParticipantInfo_Kind
	IsRecorder() bool
	IsDependent() bool
	IsAgent() bool

	CanSkipBroadcast() bool
	Version() utils.TimedVersion
	ToProto() *conkit.ParticipantInfo

	IsPublisher() bool
	GetPublishedTrack(trackID conkit.TrackID) MediaTrack
	GetPublishedTracks() []MediaTrack
	RemovePublishedTrack(track MediaTrack, isExpectedToResume bool, shouldClose bool)

	GetAudioLevel() (smoothedLevel float64, active bool)

	// HasPermission checks permission of the subscriber by identity. Returns true if subscriber is allowed to subscribe
	// to the track with trackID
	HasPermission(trackID conkit.TrackID, subIdentity conkit.ParticipantIdentity) bool

	// permissions
	Hidden() bool

	Close(sendLeave bool, reason ParticipantCloseReason, isExpectedToResume bool) error

	SubscriptionPermission() (*conkit.SubscriptionPermission, utils.TimedVersion)

	// updates from remotes
	UpdateSubscriptionPermission(
		subscriptionPermission *conkit.SubscriptionPermission,
		timedVersion utils.TimedVersion,
		resolverBySid func(participantID conkit.ParticipantID) LocalParticipant,
	) error

	DebugInfo() map[string]interface{}

	OnMetrics(callback func(Participant, *conkit.DataPacket))
}

// -------------------------------------------------------

type AddTrackParams struct {
	Stereo bool
	Red    bool
}

//counterfeiter:generate . LocalParticipant
type LocalParticipant interface {
	Participant

	ToProtoWithVersion() (*conkit.ParticipantInfo, utils.TimedVersion)

	// getters
	GetTrailer() []byte
	GetLogger() logger.Logger
	GetAdaptiveStream() bool
	ProtocolVersion() ProtocolVersion
	SupportsSyncStreamID() bool
	SupportsTransceiverReuse() bool
	IsClosed() bool
	IsReady() bool
	IsDisconnected() bool
	Disconnected() <-chan struct{}
	IsIdle() bool
	SubscriberAsPrimary() bool
	GetClientInfo() *conkit.ClientInfo
	GetClientConfiguration() *conkit.ClientConfiguration
	GetBufferFactory() *buffer.Factory
	GetPlayoutDelayConfig() *conkit.PlayoutDelay
	GetPendingTrack(trackID conkit.TrackID) *conkit.TrackInfo
	GetICEConnectionInfo() []*ICEConnectionInfo
	HasConnected() bool
	GetEnabledPublishCodecs() []*conkit.Codec

	SetResponseSink(sink routing.MessageSink)
	CloseSignalConnection(reason SignallingCloseReason)
	UpdateLastSeenSignal()
	SetSignalSourceValid(valid bool)
	HandleSignalSourceClose()

	// updates
	CheckMetadataLimits(name string, metadata string, attributes map[string]string) error
	SetName(name string)
	SetMetadata(metadata string)
	SetAttributes(attributes map[string]string)
	UpdateAudioTrack(update *conkit.UpdateLocalAudioTrack) error
	UpdateVideoTrack(update *conkit.UpdateLocalVideoTrack) error

	// permissions
	ClaimGrants() *auth.ClaimGrants
	SetPermission(permission *conkit.ParticipantPermission) bool
	CanPublish() bool
	CanPublishSource(source conkit.TrackSource) bool
	CanSubscribe() bool
	CanPublishData() bool

	// PeerConnection
	AddICECandidate(candidate webrtc.ICECandidateInit, target conkit.SignalTarget)
	HandleOffer(sdp webrtc.SessionDescription) error
	GetAnswer() (webrtc.SessionDescription, error)
	AddTrack(req *conkit.AddTrackRequest)
	SetTrackMuted(trackID conkit.TrackID, muted bool, fromAdmin bool) *conkit.TrackInfo

	HandleAnswer(sdp webrtc.SessionDescription)
	Negotiate(force bool)
	ICERestart(iceConfig *conkit.ICEConfig)
	AddTrackLocal(trackLocal webrtc.TrackLocal, params AddTrackParams) (*webrtc.RTPSender, *webrtc.RTPTransceiver, error)
	AddTransceiverFromTrackLocal(trackLocal webrtc.TrackLocal, params AddTrackParams) (*webrtc.RTPSender, *webrtc.RTPTransceiver, error)
	RemoveTrackLocal(sender *webrtc.RTPSender) error

	WriteSubscriberRTCP(pkts []rtcp.Packet) error

	// subscriptions
	SubscribeToTrack(trackID conkit.TrackID)
	UnsubscribeFromTrack(trackID conkit.TrackID)
	UpdateSubscribedTrackSettings(trackID conkit.TrackID, settings *conkit.UpdateTrackSettings)
	GetSubscribedTracks() []SubscribedTrack
	IsTrackNameSubscribed(publisherIdentity conkit.ParticipantIdentity, trackName string) bool
	Verify() bool
	VerifySubscribeParticipantInfo(pID conkit.ParticipantID, version uint32)
	// WaitUntilSubscribed waits until all subscriptions have been settled, or if the timeout
	// has been reached. If the timeout expires, it will return an error.
	WaitUntilSubscribed(timeout time.Duration) error
	StopAndGetSubscribedTracksForwarderState() map[conkit.TrackID]*conkit.RTPForwarderState
	SupportsCodecChange() bool

	// returns list of participant identities that the current participant is subscribed to
	GetSubscribedParticipants() []conkit.ParticipantID
	IsSubscribedTo(sid conkit.ParticipantID) bool

	GetConnectionQuality() *conkit.ConnectionQualityInfo

	// server sent messages
	SendJoinResponse(joinResponse *conkit.JoinResponse) error
	SendParticipantUpdate(participants []*conkit.ParticipantInfo) error
	SendSpeakerUpdate(speakers []*conkit.SpeakerInfo, force bool) error
	SendDataPacket(kind conkit.DataPacket_Kind, encoded []byte) error
	SendRoomUpdate(room *conkit.Room) error
	SendConnectionQualityUpdate(update *conkit.ConnectionQualityUpdate) error
	SubscriptionPermissionUpdate(publisherID conkit.ParticipantID, trackID conkit.TrackID, allowed bool)
	SendRefreshToken(token string) error
	SendRequestResponse(requestResponse *conkit.RequestResponse) error
	HandleReconnectAndSendResponse(reconnectReason conkit.ReconnectReason, reconnectResponse *conkit.ReconnectResponse) error
	IssueFullReconnect(reason ParticipantCloseReason)

	// callbacks
	OnStateChange(func(p LocalParticipant, state conkit.ParticipantInfo_State))
	OnMigrateStateChange(func(p LocalParticipant, migrateState MigrateState))
	// OnTrackPublished - remote added a track
	OnTrackPublished(func(LocalParticipant, MediaTrack))
	// OnTrackUpdated - one of its publishedTracks changed in status
	OnTrackUpdated(callback func(LocalParticipant, MediaTrack))
	// OnTrackUnpublished - a track was unpublished
	OnTrackUnpublished(callback func(LocalParticipant, MediaTrack))
	// OnParticipantUpdate - metadata or permission is updated
	OnParticipantUpdate(callback func(LocalParticipant))
	OnDataPacket(callback func(LocalParticipant, conkit.DataPacket_Kind, *conkit.DataPacket))
	OnSubscribeStatusChanged(fn func(publisherID conkit.ParticipantID, subscribed bool))
	OnClose(callback func(LocalParticipant))
	OnClaimsChanged(callback func(LocalParticipant))

	HandleReceiverReport(dt *sfu.DownTrack, report *rtcp.ReceiverReport)

	// session migration
	MaybeStartMigration(force bool, onStart func()) bool
	NotifyMigration()
	SetMigrateState(s MigrateState)
	MigrateState() MigrateState
	SetMigrateInfo(
		previousOffer, previousAnswer *webrtc.SessionDescription,
		mediaTracks []*conkit.TrackPublishedResponse,
		dataChannels []*conkit.DataChannelInfo,
	)
	IsReconnect() bool

	UpdateMediaRTT(rtt uint32)
	UpdateSignalingRTT(rtt uint32)

	CacheDownTrack(trackID conkit.TrackID, rtpTransceiver *webrtc.RTPTransceiver, downTrackState sfu.DownTrackState)
	UncacheDownTrack(rtpTransceiver *webrtc.RTPTransceiver)
	GetCachedDownTrack(trackID conkit.TrackID) (*webrtc.RTPTransceiver, sfu.DownTrackState)

	SetICEConfig(iceConfig *conkit.ICEConfig)
	GetICEConfig() *conkit.ICEConfig
	OnICEConfigChanged(callback func(participant LocalParticipant, iceConfig *conkit.ICEConfig))

	UpdateSubscribedQuality(nodeID conkit.NodeID, trackID conkit.TrackID, maxQualities []SubscribedCodecQuality) error
	UpdateMediaLoss(nodeID conkit.NodeID, trackID conkit.TrackID, fractionalLoss uint32) error

	// down stream bandwidth management
	SetSubscriberAllowPause(allowPause bool)
	SetSubscriberChannelCapacity(channelCapacity int64)

	GetPacer() pacer.Pacer

	GetDisableSenderReportPassThrough() bool

	HandleMetrics(senderParticipantID conkit.ParticipantID, batch *conkit.MetricsBatch) error
}

// Room is a container of participants, and can provide room-level actions
//
//counterfeiter:generate . Room
type Room interface {
	Name() conkit.RoomName
	ID() conkit.RoomID
	RemoveParticipant(identity conkit.ParticipantIdentity, pID conkit.ParticipantID, reason ParticipantCloseReason)
	UpdateSubscriptions(participant LocalParticipant, trackIDs []conkit.TrackID, participantTracks []*conkit.ParticipantTracks, subscribe bool)
	UpdateSubscriptionPermission(participant LocalParticipant, permissions *conkit.SubscriptionPermission) error
	SyncState(participant LocalParticipant, state *conkit.SyncState) error
	SimulateScenario(participant LocalParticipant, scenario *conkit.SimulateScenario) error
	ResolveMediaTrackForSubscriber(sub LocalParticipant, trackID conkit.TrackID) MediaResolverResult
	GetLocalParticipants() []LocalParticipant
	IsDataMessageUserPacketDuplicate(ip *conkit.UserPacket) bool
}

// MediaTrack represents a media track
//
//counterfeiter:generate . MediaTrack
type MediaTrack interface {
	ID() conkit.TrackID
	Kind() conkit.TrackType
	Name() string
	Source() conkit.TrackSource
	Stream() string

	UpdateTrackInfo(ti *conkit.TrackInfo)
	UpdateAudioTrack(update *conkit.UpdateLocalAudioTrack)
	UpdateVideoTrack(update *conkit.UpdateLocalVideoTrack)
	ToProto() *conkit.TrackInfo

	PublisherID() conkit.ParticipantID
	PublisherIdentity() conkit.ParticipantIdentity
	PublisherVersion() uint32

	IsMuted() bool
	SetMuted(muted bool)

	IsSimulcast() bool

	GetAudioLevel() (level float64, active bool)

	Close(isExpectedToResume bool)
	IsOpen() bool

	// callbacks
	AddOnClose(func(isExpectedToResume bool))

	// subscribers
	AddSubscriber(participant LocalParticipant) (SubscribedTrack, error)
	RemoveSubscriber(participantID conkit.ParticipantID, isExpectedToResume bool)
	IsSubscriber(subID conkit.ParticipantID) bool
	RevokeDisallowedSubscribers(allowedSubscriberIdentities []conkit.ParticipantIdentity) []conkit.ParticipantIdentity
	GetAllSubscribers() []conkit.ParticipantID
	GetNumSubscribers() int
	OnTrackSubscribed()

	// returns quality information that's appropriate for width & height
	GetQualityForDimension(width, height uint32) conkit.VideoQuality

	// returns temporal layer that's appropriate for fps
	GetTemporalLayerForSpatialFps(spatial int32, fps uint32, mime mime.MimeType) int32

	Receivers() []sfu.TrackReceiver
	ClearAllReceivers(isExpectedToResume bool)

	IsEncrypted() bool
}

//counterfeiter:generate . LocalMediaTrack
type LocalMediaTrack interface {
	MediaTrack

	Restart()

	SignalCid() string
	HasSdpCid(cid string) bool

	GetConnectionScoreAndQuality() (float32, conkit.ConnectionQuality)
	GetTrackStats() *conkit.RTPStats

	SetRTT(rtt uint32)

	NotifySubscriberNodeMaxQuality(nodeID conkit.NodeID, qualities []SubscribedCodecQuality)
	NotifySubscriberNodeMediaLoss(nodeID conkit.NodeID, fractionalLoss uint8)
}

//counterfeiter:generate . SubscribedTrack
type SubscribedTrack interface {
	AddOnBind(f func(error))
	IsBound() bool
	Close(isExpectedToResume bool)
	OnClose(f func(isExpectedToResume bool))
	ID() conkit.TrackID
	PublisherID() conkit.ParticipantID
	PublisherIdentity() conkit.ParticipantIdentity
	PublisherVersion() uint32
	SubscriberID() conkit.ParticipantID
	SubscriberIdentity() conkit.ParticipantIdentity
	Subscriber() LocalParticipant
	DownTrack() *sfu.DownTrack
	MediaTrack() MediaTrack
	RTPSender() *webrtc.RTPSender
	IsMuted() bool
	SetPublisherMuted(muted bool)
	UpdateSubscriberSettings(settings *conkit.UpdateTrackSettings, isImmediate bool)
	// selects appropriate video layer according to subscriber preferences
	UpdateVideoLayer()
	NeedsNegotiation() bool
}

type ChangeNotifier interface {
	AddObserver(key string, onChanged func())
	RemoveObserver(key string)
	HasObservers() bool
	NotifyChanged()
}

type MediaResolverResult struct {
	TrackChangedNotifier ChangeNotifier
	TrackRemovedNotifier ChangeNotifier
	Track                MediaTrack
	// is permission given to the requesting participant
	HasPermission     bool
	PublisherID       conkit.ParticipantID
	PublisherIdentity conkit.ParticipantIdentity
}

// MediaTrackResolver locates a specific media track for a subscriber
type MediaTrackResolver func(LocalParticipant, conkit.TrackID) MediaResolverResult

// Supervisor/operation monitor related definitions
type OperationMonitorEvent int

const (
	OperationMonitorEventPublisherPeerConnectionConnected OperationMonitorEvent = iota
	OperationMonitorEventAddPendingPublication
	OperationMonitorEventSetPublicationMute
	OperationMonitorEventSetPublishedTrack
	OperationMonitorEventClearPublishedTrack
)

func (o OperationMonitorEvent) String() string {
	switch o {
	case OperationMonitorEventPublisherPeerConnectionConnected:
		return "PUBLISHER_PEER_CONNECTION_CONNECTED"
	case OperationMonitorEventAddPendingPublication:
		return "ADD_PENDING_PUBLICATION"
	case OperationMonitorEventSetPublicationMute:
		return "SET_PUBLICATION_MUTE"
	case OperationMonitorEventSetPublishedTrack:
		return "SET_PUBLISHED_TRACK"
	case OperationMonitorEventClearPublishedTrack:
		return "CLEAR_PUBLISHED_TRACK"
	default:
		return fmt.Sprintf("%d", int(o))
	}
}

type OperationMonitorData interface{}

type OperationMonitor interface {
	PostEvent(ome OperationMonitorEvent, omd OperationMonitorData)
	Check() error
	IsIdle() bool
}
