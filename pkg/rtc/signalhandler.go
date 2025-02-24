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

package rtc

import (
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"

	"github.com/a4to/conkit-server/pkg/rtc/types"
)

func HandleParticipantSignal(room types.Room, participant types.LocalParticipant, req *conkit.SignalRequest, pLogger logger.Logger) error {
	participant.UpdateLastSeenSignal()

	switch msg := req.GetMessage().(type) {
	case *conkit.SignalRequest_Offer:
		participant.HandleOffer(FromProtoSessionDescription(msg.Offer))

	case *conkit.SignalRequest_Answer:
		participant.HandleAnswer(FromProtoSessionDescription(msg.Answer))

	case *conkit.SignalRequest_Trickle:
		candidateInit, err := FromProtoTrickle(msg.Trickle)
		if err != nil {
			pLogger.Warnw("could not decode trickle", err)
			return nil
		}
		participant.AddICECandidate(candidateInit, msg.Trickle.Target)

	case *conkit.SignalRequest_AddTrack:
		pLogger.Debugw("add track request", "trackID", msg.AddTrack.Cid)
		participant.AddTrack(msg.AddTrack)

	case *conkit.SignalRequest_Mute:
		participant.SetTrackMuted(conkit.TrackID(msg.Mute.Sid), msg.Mute.Muted, false)

	case *conkit.SignalRequest_Subscription:
		// allow participant to indicate their interest in the subscription
		// permission check happens later in SubscriptionManager
		room.UpdateSubscriptions(
			participant,
			conkit.StringsAsIDs[conkit.TrackID](msg.Subscription.TrackSids),
			msg.Subscription.ParticipantTracks,
			msg.Subscription.Subscribe,
		)

	case *conkit.SignalRequest_TrackSetting:
		for _, sid := range conkit.StringsAsIDs[conkit.TrackID](msg.TrackSetting.TrackSids) {
			participant.UpdateSubscribedTrackSettings(sid, msg.TrackSetting)
		}

	case *conkit.SignalRequest_Leave:
		reason := types.ParticipantCloseReasonClientRequestLeave
		switch msg.Leave.Reason {
		case conkit.DisconnectReason_CLIENT_INITIATED:
			reason = types.ParticipantCloseReasonClientRequestLeave
		case conkit.DisconnectReason_USER_UNAVAILABLE:
			reason = types.ParticipantCloseReasonUserUnavailable
		case conkit.DisconnectReason_USER_REJECTED:
			reason = types.ParticipantCloseReasonUserRejected
		}
		pLogger.Debugw("client leaving room", "reason", reason)
		room.RemoveParticipant(participant.Identity(), participant.ID(), reason)

	case *conkit.SignalRequest_SubscriptionPermission:
		err := room.UpdateSubscriptionPermission(participant, msg.SubscriptionPermission)
		if err != nil {
			pLogger.Warnw("could not update subscription permission", err,
				"permissions", msg.SubscriptionPermission)
		}

	case *conkit.SignalRequest_SyncState:
		err := room.SyncState(participant, msg.SyncState)
		if err != nil {
			pLogger.Warnw("could not sync state", err,
				"state", msg.SyncState)
		}

	case *conkit.SignalRequest_Simulate:
		err := room.SimulateScenario(participant, msg.Simulate)
		if err != nil {
			pLogger.Warnw("could not simulate scenario", err,
				"simulate", msg.Simulate)
		}

	case *conkit.SignalRequest_PingReq:
		if msg.PingReq.Rtt > 0 {
			participant.UpdateSignalingRTT(uint32(msg.PingReq.Rtt))
		}

	case *conkit.SignalRequest_UpdateMetadata:
		requestResponse := &conkit.RequestResponse{
			RequestId: msg.UpdateMetadata.RequestId,
			Reason:    conkit.RequestResponse_OK,
		}
		if participant.ClaimGrants().Video.GetCanUpdateOwnMetadata() {
			if err := participant.CheckMetadataLimits(
				msg.UpdateMetadata.Name,
				msg.UpdateMetadata.Metadata,
				msg.UpdateMetadata.Attributes,
			); err == nil {
				if msg.UpdateMetadata.Name != "" {
					participant.SetName(msg.UpdateMetadata.Name)
				}
				if msg.UpdateMetadata.Metadata != "" {
					participant.SetMetadata(msg.UpdateMetadata.Metadata)
				}
				if msg.UpdateMetadata.Attributes != nil {
					participant.SetAttributes(msg.UpdateMetadata.Attributes)
				}
			} else {
				pLogger.Warnw("could not update metadata", err)

				switch err {
				case ErrNameExceedsLimits:
					requestResponse.Reason = conkit.RequestResponse_LIMIT_EXCEEDED
					requestResponse.Message = "exceeds name length limit"

				case ErrMetadataExceedsLimits:
					requestResponse.Reason = conkit.RequestResponse_LIMIT_EXCEEDED
					requestResponse.Message = "exceeds metadata size limit"

				case ErrAttributesExceedsLimits:
					requestResponse.Reason = conkit.RequestResponse_LIMIT_EXCEEDED
					requestResponse.Message = "exceeds attributes size limit"
				}

			}
		} else {
			requestResponse.Reason = conkit.RequestResponse_NOT_ALLOWED
			requestResponse.Message = "does not have permission to update own metadata"
		}
		participant.SendRequestResponse(requestResponse)

	case *conkit.SignalRequest_UpdateAudioTrack:
		if err := participant.UpdateAudioTrack(msg.UpdateAudioTrack); err != nil {
			pLogger.Warnw("could not update audio track", err, "update", msg.UpdateAudioTrack)
		}

	case *conkit.SignalRequest_UpdateVideoTrack:
		if err := participant.UpdateVideoTrack(msg.UpdateVideoTrack); err != nil {
			pLogger.Warnw("could not update video track", err, "update", msg.UpdateVideoTrack)
		}
	}

	return nil
}
