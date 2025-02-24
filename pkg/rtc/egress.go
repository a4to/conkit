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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/a4to/conkit-server/pkg/rtc/types"
	"github.com/a4to/conkit-server/pkg/telemetry"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/rpc"
	"github.com/a4to/protocol/webhook"
)

type EgressLauncher interface {
	StartEgress(context.Context, *rpc.StartEgressRequest) (*conkit.EgressInfo, error)
}

func StartParticipantEgress(
	ctx context.Context,
	launcher EgressLauncher,
	ts telemetry.TelemetryService,
	opts *conkit.AutoParticipantEgress,
	identity conkit.ParticipantIdentity,
	roomName conkit.RoomName,
	roomID conkit.RoomID,
) error {
	if req, err := startParticipantEgress(ctx, launcher, opts, identity, roomName, roomID); err != nil {
		// send egress failed webhook
		ts.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event: webhook.EventEgressEnded,
			EgressInfo: &conkit.EgressInfo{
				RoomId:   string(roomID),
				RoomName: string(roomName),
				Status:   conkit.EgressStatus_EGRESS_FAILED,
				Error:    err.Error(),
				Request:  &conkit.EgressInfo_Participant{Participant: req},
			},
		})
		return err
	}
	return nil
}

func startParticipantEgress(
	ctx context.Context,
	launcher EgressLauncher,
	opts *conkit.AutoParticipantEgress,
	identity conkit.ParticipantIdentity,
	roomName conkit.RoomName,
	roomID conkit.RoomID,
) (*conkit.ParticipantEgressRequest, error) {
	req := &conkit.ParticipantEgressRequest{
		RoomName:       string(roomName),
		Identity:       string(identity),
		FileOutputs:    opts.FileOutputs,
		SegmentOutputs: opts.SegmentOutputs,
	}

	switch o := opts.Options.(type) {
	case *conkit.AutoParticipantEgress_Preset:
		req.Options = &conkit.ParticipantEgressRequest_Preset{Preset: o.Preset}
	case *conkit.AutoParticipantEgress_Advanced:
		req.Options = &conkit.ParticipantEgressRequest_Advanced{Advanced: o.Advanced}
	}

	if launcher == nil {
		return req, errors.New("egress launcher not found")
	}

	_, err := launcher.StartEgress(ctx, &rpc.StartEgressRequest{
		Request: &rpc.StartEgressRequest_Participant{
			Participant: req,
		},
		RoomId: string(roomID),
	})
	return req, err
}

func StartTrackEgress(
	ctx context.Context,
	launcher EgressLauncher,
	ts telemetry.TelemetryService,
	opts *conkit.AutoTrackEgress,
	track types.MediaTrack,
	roomName conkit.RoomName,
	roomID conkit.RoomID,
) error {
	if req, err := startTrackEgress(ctx, launcher, opts, track, roomName, roomID); err != nil {
		// send egress failed webhook
		ts.NotifyEvent(ctx, &conkit.WebhookEvent{
			Event: webhook.EventEgressEnded,
			EgressInfo: &conkit.EgressInfo{
				RoomId:   string(roomID),
				RoomName: string(roomName),
				Status:   conkit.EgressStatus_EGRESS_FAILED,
				Error:    err.Error(),
				Request:  &conkit.EgressInfo_Track{Track: req},
			},
		})
		return err
	}
	return nil
}

func startTrackEgress(
	ctx context.Context,
	launcher EgressLauncher,
	opts *conkit.AutoTrackEgress,
	track types.MediaTrack,
	roomName conkit.RoomName,
	roomID conkit.RoomID,
) (*conkit.TrackEgressRequest, error) {
	output := &conkit.DirectFileOutput{
		Filepath: getFilePath(opts.Filepath),
	}

	switch out := opts.Output.(type) {
	case *conkit.AutoTrackEgress_Azure:
		output.Output = &conkit.DirectFileOutput_Azure{Azure: out.Azure}
	case *conkit.AutoTrackEgress_Gcp:
		output.Output = &conkit.DirectFileOutput_Gcp{Gcp: out.Gcp}
	case *conkit.AutoTrackEgress_S3:
		output.Output = &conkit.DirectFileOutput_S3{S3: out.S3}
	}

	req := &conkit.TrackEgressRequest{
		RoomName: string(roomName),
		TrackId:  string(track.ID()),
		Output: &conkit.TrackEgressRequest_File{
			File: output,
		},
	}

	if launcher == nil {
		return req, errors.New("egress launcher not found")
	}

	_, err := launcher.StartEgress(ctx, &rpc.StartEgressRequest{
		Request: &rpc.StartEgressRequest_Track{
			Track: req,
		},
		RoomId: string(roomID),
	})
	return req, err
}

func getFilePath(filepath string) string {
	if filepath == "" || strings.HasSuffix(filepath, "/") || strings.Contains(filepath, "{track_id}") {
		return filepath
	}

	idx := strings.Index(filepath, ".")
	if idx == -1 {
		return fmt.Sprintf("%s-{track_id}", filepath)
	} else {
		return fmt.Sprintf("%s-%s%s", filepath[:idx], "{track_id}", filepath[idx:])
	}
}
