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
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/a4to/conkit-server/pkg/sfu/mime"
	"github.com/a4to/conkit-server/pkg/telemetry/prometheus"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/utils/guid"
	"github.com/a4to/protocol/webhook"
)

func (t *telemetryService) NotifyEvent(ctx context.Context, event *conkit.WebhookEvent) {
	if t.notifier == nil {
		return
	}

	event.CreatedAt = time.Now().Unix()
	event.Id = guid.New("EV_")

	if err := t.notifier.QueueNotify(ctx, event); err != nil {
		logger.Warnw("failed to notify webhook", err, "event", event.Event)
	}
}

func (t *telemetryService) RoomStarted(ctx context.Context, room *conkit.Room) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event: webhook.EventRoomStarted,
			Room:  room,
		})

		t.SendEvent(ctx, &conkit.AnalyticsEvent{
			Type:      conkit.AnalyticsEventType_ROOM_CREATED,
			Timestamp: &timestamppb.Timestamp{Seconds: room.CreationTime},
			Room:      room,
		})
	})
}

func (t *telemetryService) RoomEnded(ctx context.Context, room *conkit.Room) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event: webhook.EventRoomFinished,
			Room:  room,
		})

		t.SendEvent(ctx, &conkit.AnalyticsEvent{
			Type:      conkit.AnalyticsEventType_ROOM_ENDED,
			Timestamp: timestamppb.Now(),
			RoomId:    room.Sid,
			Room:      room,
		})
	})
}

func (t *telemetryService) ParticipantJoined(
	ctx context.Context,
	room *conkit.Room,
	participant *conkit.ParticipantInfo,
	clientInfo *conkit.ClientInfo,
	clientMeta *conkit.AnalyticsClientMeta,
	shouldSendEvent bool,
) {
	t.enqueue(func() {
		_, found := t.getOrCreateWorker(
			ctx,
			conkit.RoomID(room.Sid),
			conkit.RoomName(room.Name),
			conkit.ParticipantID(participant.Sid),
			conkit.ParticipantIdentity(participant.Identity),
		)
		if !found {
			prometheus.IncrementParticipantRtcConnected(1)
			prometheus.AddParticipant()
		}

		if shouldSendEvent {
			ev := newParticipantEvent(conkit.AnalyticsEventType_PARTICIPANT_JOINED, room, participant)
			ev.ClientInfo = clientInfo
			ev.ClientMeta = clientMeta
			t.SendEvent(ctx, ev)
		}
	})
}

func (t *telemetryService) ParticipantActive(
	ctx context.Context,
	room *conkit.Room,
	participant *conkit.ParticipantInfo,
	clientMeta *conkit.AnalyticsClientMeta,
	isMigration bool,
) {
	t.enqueue(func() {
		if !isMigration {
			// consider participant joined only when they became active
			t.NotifyEvent(ctx, &conkit.WebhookEvent{
				Event:       webhook.EventParticipantJoined,
				Room:        room,
				Participant: participant,
			})
		}

		worker, found := t.getOrCreateWorker(
			ctx,
			conkit.RoomID(room.Sid),
			conkit.RoomName(room.Name),
			conkit.ParticipantID(participant.Sid),
			conkit.ParticipantIdentity(participant.Identity),
		)
		if !found {
			// need to also account for participant count
			prometheus.AddParticipant()
		}
		worker.SetConnected()

		ev := newParticipantEvent(conkit.AnalyticsEventType_PARTICIPANT_ACTIVE, room, participant)
		ev.ClientMeta = clientMeta
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) ParticipantResumed(
	ctx context.Context,
	room *conkit.Room,
	participant *conkit.ParticipantInfo,
	nodeID conkit.NodeID,
	reason conkit.ReconnectReason,
) {
	t.enqueue(func() {
		// create a worker if needed.
		//
		// Signalling channel stats collector and media channel stats collector could both call
		// ParticipantJoined and ParticipantLeft.
		//
		// On a resume, the signalling channel collector would call `ParticipantLeft` which would close
		// the corresponding participant's stats worker.
		//
		// So, on a successful resume, create the worker if needed.
		_, found := t.getOrCreateWorker(
			ctx,
			conkit.RoomID(room.Sid),
			conkit.RoomName(room.Name),
			conkit.ParticipantID(participant.Sid),
			conkit.ParticipantIdentity(participant.Identity),
		)
		if !found {
			prometheus.AddParticipant()
		}

		ev := newParticipantEvent(conkit.AnalyticsEventType_PARTICIPANT_RESUMED, room, participant)
		ev.ClientMeta = &conkit.AnalyticsClientMeta{
			Node:            string(nodeID),
			ReconnectReason: reason,
		}
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) ParticipantLeft(ctx context.Context,
	room *conkit.Room,
	participant *conkit.ParticipantInfo,
	shouldSendEvent bool,
) {
	t.enqueue(func() {
		isConnected := false
		if worker, ok := t.getWorker(conkit.ParticipantID(participant.Sid)); ok {
			isConnected = worker.IsConnected()
			if worker.Close() {
				prometheus.SubParticipant()
			}
		}

		if isConnected && shouldSendEvent {
			t.NotifyEvent(ctx, &conkit.WebhookEvent{
				Event:       webhook.EventParticipantLeft,
				Room:        room,
				Participant: participant,
			})

			t.SendEvent(ctx, newParticipantEvent(conkit.AnalyticsEventType_PARTICIPANT_LEFT, room, participant))
		}
	})
}

func (t *telemetryService) TrackPublishRequested(
	ctx context.Context,
	participantID conkit.ParticipantID,
	identity conkit.ParticipantIdentity,
	track *conkit.TrackInfo,
) {
	t.enqueue(func() {
		prometheus.AddPublishAttempt(track.Type.String())
		room := t.getRoomDetails(participantID)
		ev := newTrackEvent(conkit.AnalyticsEventType_TRACK_PUBLISH_REQUESTED, room, participantID, track)
		if ev.Participant != nil {
			ev.Participant.Identity = string(identity)
		}
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackPublished(
	ctx context.Context,
	participantID conkit.ParticipantID,
	identity conkit.ParticipantIdentity,
	track *conkit.TrackInfo,
) {
	t.enqueue(func() {
		prometheus.AddPublishedTrack(track.Type.String())
		prometheus.AddPublishSuccess(track.Type.String())

		room := t.getRoomDetails(participantID)
		participant := &conkit.ParticipantInfo{
			Sid:      string(participantID),
			Identity: string(identity),
		}
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:       webhook.EventTrackPublished,
			Room:        room,
			Participant: participant,
			Track:       track,
		})

		ev := newTrackEvent(conkit.AnalyticsEventType_TRACK_PUBLISHED, room, participantID, track)
		ev.Participant = participant
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackPublishedUpdate(ctx context.Context, participantID conkit.ParticipantID, track *conkit.TrackInfo) {
	t.enqueue(func() {
		room := t.getRoomDetails(participantID)
		t.SendEvent(ctx, newTrackEvent(conkit.AnalyticsEventType_TRACK_PUBLISHED_UPDATE, room, participantID, track))
	})
}

func (t *telemetryService) TrackMaxSubscribedVideoQuality(
	ctx context.Context,
	participantID conkit.ParticipantID,
	track *conkit.TrackInfo,
	mime mime.MimeType,
	maxQuality conkit.VideoQuality,
) {
	t.enqueue(func() {
		room := t.getRoomDetails(participantID)
		ev := newTrackEvent(conkit.AnalyticsEventType_TRACK_MAX_SUBSCRIBED_VIDEO_QUALITY, room, participantID, track)
		ev.MaxSubscribedVideoQuality = maxQuality
		ev.Mime = mime.String()
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackSubscribeRequested(
	ctx context.Context,
	participantID conkit.ParticipantID,
	track *conkit.TrackInfo,
) {
	t.enqueue(func() {
		prometheus.RecordTrackSubscribeAttempt()

		room := t.getRoomDetails(participantID)
		ev := newTrackEvent(conkit.AnalyticsEventType_TRACK_SUBSCRIBE_REQUESTED, room, participantID, track)
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackSubscribed(
	ctx context.Context,
	participantID conkit.ParticipantID,
	track *conkit.TrackInfo,
	publisher *conkit.ParticipantInfo,
	shouldSendEvent bool,
) {
	t.enqueue(func() {
		prometheus.RecordTrackSubscribeSuccess(track.Type.String())

		if !shouldSendEvent {
			return
		}

		room := t.getRoomDetails(participantID)
		ev := newTrackEvent(conkit.AnalyticsEventType_TRACK_SUBSCRIBED, room, participantID, track)
		ev.Publisher = publisher
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackSubscribeFailed(
	ctx context.Context,
	participantID conkit.ParticipantID,
	trackID conkit.TrackID,
	err error,
	isUserError bool,
) {
	t.enqueue(func() {
		prometheus.RecordTrackSubscribeFailure(err, isUserError)

		room := t.getRoomDetails(participantID)
		ev := newTrackEvent(conkit.AnalyticsEventType_TRACK_SUBSCRIBE_FAILED, room, participantID, &conkit.TrackInfo{
			Sid: string(trackID),
		})
		ev.Error = err.Error()
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackUnsubscribed(
	ctx context.Context,
	participantID conkit.ParticipantID,
	track *conkit.TrackInfo,
	shouldSendEvent bool,
) {
	t.enqueue(func() {
		prometheus.RecordTrackUnsubscribed(track.Type.String())

		if shouldSendEvent {
			room := t.getRoomDetails(participantID)
			t.SendEvent(ctx, newTrackEvent(conkit.AnalyticsEventType_TRACK_UNSUBSCRIBED, room, participantID, track))
		}
	})
}

func (t *telemetryService) TrackUnpublished(
	ctx context.Context,
	participantID conkit.ParticipantID,
	identity conkit.ParticipantIdentity,
	track *conkit.TrackInfo,
	shouldSendEvent bool,
) {
	t.enqueue(func() {
		prometheus.SubPublishedTrack(track.Type.String())
		if !shouldSendEvent {
			return
		}

		room := t.getRoomDetails(participantID)
		participant := &conkit.ParticipantInfo{
			Sid:      string(participantID),
			Identity: string(identity),
		}
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:       webhook.EventTrackUnpublished,
			Room:        room,
			Participant: participant,
			Track:       track,
		})

		t.SendEvent(ctx, newTrackEvent(conkit.AnalyticsEventType_TRACK_UNPUBLISHED, room, participantID, track))
	})
}

func (t *telemetryService) TrackMuted(
	ctx context.Context,
	participantID conkit.ParticipantID,
	track *conkit.TrackInfo,
) {
	t.enqueue(func() {
		room := t.getRoomDetails(participantID)
		t.SendEvent(ctx, newTrackEvent(conkit.AnalyticsEventType_TRACK_MUTED, room, participantID, track))
	})
}

func (t *telemetryService) TrackUnmuted(
	ctx context.Context,
	participantID conkit.ParticipantID,
	track *conkit.TrackInfo,
) {
	t.enqueue(func() {
		room := t.getRoomDetails(participantID)
		t.SendEvent(ctx, newTrackEvent(conkit.AnalyticsEventType_TRACK_UNMUTED, room, participantID, track))
	})
}

func (t *telemetryService) TrackPublishRTPStats(
	ctx context.Context,
	participantID conkit.ParticipantID,
	trackID conkit.TrackID,
	mimeType mime.MimeType,
	layer int,
	stats *conkit.RTPStats,
) {
	t.enqueue(func() {
		room := t.getRoomDetails(participantID)
		ev := newRoomEvent(conkit.AnalyticsEventType_TRACK_PUBLISH_STATS, room)
		ev.ParticipantId = string(participantID)
		ev.TrackId = string(trackID)
		ev.Mime = mimeType.String()
		ev.VideoLayer = int32(layer)
		ev.RtpStats = stats
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) TrackSubscribeRTPStats(
	ctx context.Context,
	participantID conkit.ParticipantID,
	trackID conkit.TrackID,
	mimeType mime.MimeType,
	stats *conkit.RTPStats,
) {
	t.enqueue(func() {
		room := t.getRoomDetails(participantID)
		ev := newRoomEvent(conkit.AnalyticsEventType_TRACK_SUBSCRIBE_STATS, room)
		ev.ParticipantId = string(participantID)
		ev.TrackId = string(trackID)
		ev.Mime = mimeType.String()
		ev.RtpStats = stats
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) EgressStarted(ctx context.Context, info *conkit.EgressInfo) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:      webhook.EventEgressStarted,
			EgressInfo: info,
		})

		t.SendEvent(ctx, newEgressEvent(conkit.AnalyticsEventType_EGRESS_STARTED, info))
	})
}

func (t *telemetryService) EgressUpdated(ctx context.Context, info *conkit.EgressInfo) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:      webhook.EventEgressUpdated,
			EgressInfo: info,
		})
		t.SendEvent(ctx, newEgressEvent(conkit.AnalyticsEventType_EGRESS_UPDATED, info))
	})
}

func (t *telemetryService) EgressEnded(ctx context.Context, info *conkit.EgressInfo) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:      webhook.EventEgressEnded,
			EgressInfo: info,
		})

		t.SendEvent(ctx, newEgressEvent(conkit.AnalyticsEventType_EGRESS_ENDED, info))
	})
}

func (t *telemetryService) IngressCreated(ctx context.Context, info *conkit.IngressInfo) {
	t.enqueue(func() {
		t.SendEvent(ctx, newIngressEvent(conkit.AnalyticsEventType_INGRESS_CREATED, info))
	})
}

func (t *telemetryService) IngressDeleted(ctx context.Context, info *conkit.IngressInfo) {
	t.enqueue(func() {
		t.SendEvent(ctx, newIngressEvent(conkit.AnalyticsEventType_INGRESS_DELETED, info))
	})
}

func (t *telemetryService) IngressStarted(ctx context.Context, info *conkit.IngressInfo) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:       webhook.EventIngressStarted,
			IngressInfo: info,
		})

		t.SendEvent(ctx, newIngressEvent(conkit.AnalyticsEventType_INGRESS_STARTED, info))
	})
}

func (t *telemetryService) IngressUpdated(ctx context.Context, info *conkit.IngressInfo) {
	t.enqueue(func() {
		t.SendEvent(ctx, newIngressEvent(conkit.AnalyticsEventType_INGRESS_UPDATED, info))
	})
}

func (t *telemetryService) IngressEnded(ctx context.Context, info *conkit.IngressInfo) {
	t.enqueue(func() {
		t.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event:       webhook.EventIngressEnded,
			IngressInfo: info,
		})

		t.SendEvent(ctx, newIngressEvent(conkit.AnalyticsEventType_INGRESS_ENDED, info))
	})
}

func (t *telemetryService) Report(ctx context.Context, reportInfo *conkit.ReportInfo) {
	t.enqueue(func() {
		ev := &conkit.AnalyticsEvent{
			Type:      conkit.AnalyticsEventType_REPORT,
			Timestamp: timestamppb.Now(),
			Report:    reportInfo,
		}
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) APICall(ctx context.Context, apiCallInfo *conkit.APICallInfo) {
	t.enqueue(func() {
		ev := &conkit.AnalyticsEvent{
			Type:      conkit.AnalyticsEventType_API_CALL,
			Timestamp: timestamppb.Now(),
			ApiCall:   apiCallInfo,
		}
		t.SendEvent(ctx, ev)
	})
}

func (t *telemetryService) Webhook(ctx context.Context, webhookInfo *conkit.WebhookInfo) {
	t.enqueue(func() {
		ev := &conkit.AnalyticsEvent{
			Type:      conkit.AnalyticsEventType_WEBHOOK,
			Timestamp: timestamppb.Now(),
			Webhook:   webhookInfo,
		}
		t.SendEvent(ctx, ev)
	})
}

// returns a conkit.Room with only name and sid filled out
// returns nil if room is not found
func (t *telemetryService) getRoomDetails(participantID conkit.ParticipantID) *conkit.Room {
	if worker, ok := t.getWorker(participantID); ok {
		return &conkit.Room{
			Sid:  string(worker.roomID),
			Name: string(worker.roomName),
		}
	}

	return nil
}

func newRoomEvent(event conkit.AnalyticsEventType, room *conkit.Room) *conkit.AnalyticsEvent {
	ev := &conkit.AnalyticsEvent{
		Type:      event,
		Timestamp: timestamppb.Now(),
	}
	if room != nil {
		ev.Room = room
		ev.RoomId = room.Sid
	}
	return ev
}

func newParticipantEvent(event conkit.AnalyticsEventType, room *conkit.Room, participant *conkit.ParticipantInfo) *conkit.AnalyticsEvent {
	ev := newRoomEvent(event, room)
	if participant != nil {
		ev.ParticipantId = participant.Sid
		ev.Participant = participant
	}
	return ev
}

func newTrackEvent(event conkit.AnalyticsEventType, room *conkit.Room, participantID conkit.ParticipantID, track *conkit.TrackInfo) *conkit.AnalyticsEvent {
	ev := newParticipantEvent(event, room, &conkit.ParticipantInfo{
		Sid: string(participantID),
	})
	if track != nil {
		ev.TrackId = track.Sid
		ev.Track = track
	}
	return ev
}

func newEgressEvent(event conkit.AnalyticsEventType, egress *conkit.EgressInfo) *conkit.AnalyticsEvent {
	return &conkit.AnalyticsEvent{
		Type:      event,
		Timestamp: timestamppb.Now(),
		EgressId:  egress.EgressId,
		RoomId:    egress.RoomId,
		Egress:    egress,
	}
}

func newIngressEvent(event conkit.AnalyticsEventType, ingress *conkit.IngressInfo) *conkit.AnalyticsEvent {
	return &conkit.AnalyticsEvent{
		Type:      event,
		Timestamp: timestamppb.Now(),
		IngressId: ingress.IngressId,
		Ingress:   ingress,
	}
}
