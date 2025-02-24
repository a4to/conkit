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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"

	"github.com/a4to/conkit-server/pkg/sfu/buffer"
	"github.com/a4to/conkit-server/pkg/sfu/mime"
	"github.com/a4to/conkit-server/pkg/telemetry/telemetryfakes"
	"github.com/a4to/protocol/auth"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/utils"
	"github.com/a4to/protocol/utils/guid"

	"github.com/a4to/conkit-server/pkg/config"
	"github.com/a4to/conkit-server/pkg/routing"
	"github.com/a4to/conkit-server/pkg/routing/routingfakes"
	"github.com/a4to/conkit-server/pkg/rtc/types"
	"github.com/a4to/conkit-server/pkg/rtc/types/typesfakes"
	"github.com/a4to/conkit-server/pkg/testutils"
)

func TestIsReady(t *testing.T) {
	tests := []struct {
		state conkit.ParticipantInfo_State
		ready bool
	}{
		{
			state: conkit.ParticipantInfo_JOINING,
			ready: false,
		},
		{
			state: conkit.ParticipantInfo_JOINED,
			ready: true,
		},
		{
			state: conkit.ParticipantInfo_ACTIVE,
			ready: true,
		},
		{
			state: conkit.ParticipantInfo_DISCONNECTED,
			ready: false,
		},
	}

	for _, test := range tests {
		t.Run(test.state.String(), func(t *testing.T) {
			p := &ParticipantImpl{}
			p.state.Store(test.state)
			require.Equal(t, test.ready, p.IsReady())
		})
	}
}

func TestTrackPublishing(t *testing.T) {
	t.Run("should send the correct events", func(t *testing.T) {
		p := newParticipantForTest("test")
		track := &typesfakes.FakeMediaTrack{}
		track.IDReturns("id")
		published := false
		updated := false
		p.OnTrackUpdated(func(p types.LocalParticipant, track types.MediaTrack) {
			updated = true
		})
		p.OnTrackPublished(func(p types.LocalParticipant, track types.MediaTrack) {
			published = true
		})
		p.UpTrackManager.AddPublishedTrack(track)
		p.handleTrackPublished(track)
		require.True(t, published)
		require.False(t, updated)
		require.Len(t, p.UpTrackManager.publishedTracks, 1)
	})

	t.Run("sends back trackPublished event", func(t *testing.T) {
		p := newParticipantForTest("test")
		sink := p.params.Sink.(*routingfakes.FakeMessageSink)
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:    "cid",
			Name:   "webcam",
			Type:   conkit.TrackType_VIDEO,
			Width:  1024,
			Height: 768,
		})
		require.Equal(t, 1, sink.WriteMessageCallCount())
		res := sink.WriteMessageArgsForCall(0).(*conkit.SignalResponse)
		require.IsType(t, &conkit.SignalResponse_TrackPublished{}, res.Message)
		published := res.Message.(*conkit.SignalResponse_TrackPublished).TrackPublished
		require.Equal(t, "cid", published.Cid)
		require.Equal(t, "webcam", published.Track.Name)
		require.Equal(t, conkit.TrackType_VIDEO, published.Track.Type)
		require.Equal(t, uint32(1024), published.Track.Width)
		require.Equal(t, uint32(768), published.Track.Height)
	})

	t.Run("should not allow adding of duplicate tracks", func(t *testing.T) {
		p := newParticipantForTest("test")
		sink := p.params.Sink.(*routingfakes.FakeMessageSink)
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:  "cid",
			Name: "webcam",
			Type: conkit.TrackType_VIDEO,
		})
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:  "cid",
			Name: "duplicate",
			Type: conkit.TrackType_AUDIO,
		})

		require.Equal(t, 1, sink.WriteMessageCallCount())
	})

	t.Run("should queue adding of duplicate tracks if already published by client id in signalling", func(t *testing.T) {
		p := newParticipantForTest("test")
		sink := p.params.Sink.(*routingfakes.FakeMessageSink)

		track := &typesfakes.FakeLocalMediaTrack{}
		track.SignalCidReturns("cid")
		track.ToProtoReturns(&conkit.TrackInfo{})
		// directly add to publishedTracks without lock - for testing purpose only
		p.UpTrackManager.publishedTracks["cid"] = track

		p.AddTrack(&conkit.AddTrackRequest{
			Cid:  "cid",
			Name: "webcam",
			Type: conkit.TrackType_VIDEO,
		})
		require.Equal(t, 0, sink.WriteMessageCallCount())
		require.Equal(t, 1, len(p.pendingTracks["cid"].trackInfos))

		// add again - it should be added to the queue
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:  "cid",
			Name: "webcam",
			Type: conkit.TrackType_VIDEO,
		})
		require.Equal(t, 0, sink.WriteMessageCallCount())
		require.Equal(t, 2, len(p.pendingTracks["cid"].trackInfos))

		// check SID is the same
		require.Equal(t, p.pendingTracks["cid"].trackInfos[0].Sid, p.pendingTracks["cid"].trackInfos[1].Sid)
	})

	t.Run("should queue adding of duplicate tracks if already published by client id in sdp", func(t *testing.T) {
		p := newParticipantForTest("test")
		sink := p.params.Sink.(*routingfakes.FakeMessageSink)

		track := &typesfakes.FakeLocalMediaTrack{}
		track.ToProtoReturns(&conkit.TrackInfo{})
		track.HasSdpCidCalls(func(s string) bool { return s == "cid" })
		// directly add to publishedTracks without lock - for testing purpose only
		p.UpTrackManager.publishedTracks["cid"] = track

		p.AddTrack(&conkit.AddTrackRequest{
			Cid:  "cid",
			Name: "webcam",
			Type: conkit.TrackType_VIDEO,
		})
		require.Equal(t, 0, sink.WriteMessageCallCount())
		require.Equal(t, 1, len(p.pendingTracks["cid"].trackInfos))

		// add again - it should be added to the queue
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:  "cid",
			Name: "webcam",
			Type: conkit.TrackType_VIDEO,
		})
		require.Equal(t, 0, sink.WriteMessageCallCount())
		require.Equal(t, 2, len(p.pendingTracks["cid"].trackInfos))

		// check SID is the same
		require.Equal(t, p.pendingTracks["cid"].trackInfos[0].Sid, p.pendingTracks["cid"].trackInfos[1].Sid)
	})

	t.Run("should not allow adding disallowed sources", func(t *testing.T) {
		p := newParticipantForTest("test")
		p.SetPermission(&conkit.ParticipantPermission{
			CanPublish: true,
			CanPublishSources: []conkit.TrackSource{
				conkit.TrackSource_CAMERA,
			},
		})
		sink := p.params.Sink.(*routingfakes.FakeMessageSink)
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:    "cid",
			Name:   "webcam",
			Source: conkit.TrackSource_CAMERA,
			Type:   conkit.TrackType_VIDEO,
		})
		require.Equal(t, 1, sink.WriteMessageCallCount())

		p.AddTrack(&conkit.AddTrackRequest{
			Cid:    "cid2",
			Name:   "rejected source",
			Type:   conkit.TrackType_AUDIO,
			Source: conkit.TrackSource_MICROPHONE,
		})
		require.Equal(t, 1, sink.WriteMessageCallCount())
	})
}

func TestOutOfOrderUpdates(t *testing.T) {
	p := newParticipantForTest("test")
	p.updateState(conkit.ParticipantInfo_JOINED)
	p.SetMetadata("initial metadata")
	sink := p.getResponseSink().(*routingfakes.FakeMessageSink)
	pi1 := p.ToProto()
	p.SetMetadata("second update")
	pi2 := p.ToProto()

	require.Greater(t, pi2.Version, pi1.Version)

	// send the second update first
	require.NoError(t, p.SendParticipantUpdate([]*conkit.ParticipantInfo{pi2}))
	require.NoError(t, p.SendParticipantUpdate([]*conkit.ParticipantInfo{pi1}))

	// only sent once, and it's the earlier message
	require.Equal(t, 1, sink.WriteMessageCallCount())
	sent := sink.WriteMessageArgsForCall(0).(*conkit.SignalResponse)
	require.Equal(t, "second update", sent.GetUpdate().Participants[0].Metadata)
}

// after disconnection, things should continue to function and not panic
func TestDisconnectTiming(t *testing.T) {
	t.Run("Negotiate doesn't panic after channel closed", func(t *testing.T) {
		p := newParticipantForTest("test")
		msg := routing.NewMessageChannel(conkit.ConnectionID("test"), routing.DefaultMessageChannelSize)
		p.params.Sink = msg
		go func() {
			for msg := range msg.ReadChan() {
				t.Log("received message from chan", msg)
			}
		}()
		track := &typesfakes.FakeMediaTrack{}
		p.UpTrackManager.AddPublishedTrack(track)
		p.handleTrackPublished(track)

		// close channel and then try to Negotiate
		msg.Close()
	})
}

func TestCorrectJoinedAt(t *testing.T) {
	p := newParticipantForTest("test")
	info := p.ToProto()
	require.NotZero(t, info.JoinedAt)
	require.True(t, time.Now().Unix()-info.JoinedAt <= 1)
}

func TestMuteSetting(t *testing.T) {
	t.Run("can set mute when track is pending", func(t *testing.T) {
		p := newParticipantForTest("test")
		ti := &conkit.TrackInfo{Sid: "testTrack"}
		p.pendingTracks["cid"] = &pendingTrackInfo{trackInfos: []*conkit.TrackInfo{ti}}

		p.SetTrackMuted(conkit.TrackID(ti.Sid), true, false)
		require.True(t, p.pendingTracks["cid"].trackInfos[0].Muted)
	})

	t.Run("can publish a muted track", func(t *testing.T) {
		p := newParticipantForTest("test")
		p.AddTrack(&conkit.AddTrackRequest{
			Cid:   "cid",
			Type:  conkit.TrackType_AUDIO,
			Muted: true,
		})

		_, ti, _, _ := p.getPendingTrack("cid", conkit.TrackType_AUDIO, false)
		require.NotNil(t, ti)
		require.True(t, ti.Muted)
	})
}

func TestSubscriberAsPrimary(t *testing.T) {
	t.Run("protocol 4 uses subs as primary", func(t *testing.T) {
		p := newParticipantForTestWithOpts("test", &participantOpts{
			permissions: &conkit.ParticipantPermission{
				CanSubscribe: true,
				CanPublish:   true,
			},
		})
		require.True(t, p.SubscriberAsPrimary())
	})

	t.Run("protocol 2 uses pub as primary", func(t *testing.T) {
		p := newParticipantForTestWithOpts("test", &participantOpts{
			protocolVersion: 2,
			permissions: &conkit.ParticipantPermission{
				CanSubscribe: true,
				CanPublish:   true,
			},
		})
		require.False(t, p.SubscriberAsPrimary())
	})

	t.Run("publisher only uses pub as primary", func(t *testing.T) {
		p := newParticipantForTestWithOpts("test", &participantOpts{
			permissions: &conkit.ParticipantPermission{
				CanSubscribe: false,
				CanPublish:   true,
			},
		})
		require.False(t, p.SubscriberAsPrimary())

		// ensure that it doesn't change after perms
		p.SetPermission(&conkit.ParticipantPermission{
			CanSubscribe: true,
			CanPublish:   true,
		})
		require.False(t, p.SubscriberAsPrimary())
	})
}

func TestDisableCodecs(t *testing.T) {
	participant := newParticipantForTestWithOpts("123", &participantOpts{
		publisher: false,
		clientConf: &conkit.ClientConfiguration{
			DisabledCodecs: &conkit.DisabledCodecs{
				Codecs: []*conkit.Codec{
					{Mime: "video/h264"},
				},
			},
		},
	})

	participant.SetMigrateState(types.MigrateStateComplete)

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	transceiver, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
	require.NoError(t, err)
	sdp, err := pc.CreateOffer(nil)
	require.NoError(t, err)
	pc.SetLocalDescription(sdp)
	codecs := transceiver.Receiver().GetParameters().Codecs
	var found264 bool
	for _, c := range codecs {
		if mime.IsMimeTypeStringH264(c.MimeType) {
			found264 = true
		}
	}
	require.True(t, found264)

	// negotiated codec should not contain h264
	sink := &routingfakes.FakeMessageSink{}
	participant.SetResponseSink(sink)
	var answer webrtc.SessionDescription
	var answerReceived atomic.Bool
	sink.WriteMessageCalls(func(msg proto.Message) error {
		if res, ok := msg.(*conkit.SignalResponse); ok {
			if res.GetAnswer() != nil {
				answer = FromProtoSessionDescription(res.GetAnswer())
				answerReceived.Store(true)
			}
		}
		return nil
	})
	participant.HandleOffer(sdp)

	testutils.WithTimeout(t, func() string {
		if answerReceived.Load() {
			return ""
		} else {
			return "answer not received"
		}
	})
	require.NoError(t, pc.SetRemoteDescription(answer), answer.SDP, sdp.SDP)

	codecs = transceiver.Receiver().GetParameters().Codecs
	found264 = false
	for _, c := range codecs {
		if mime.IsMimeTypeStringH264(c.MimeType) {
			found264 = true
		}
	}
	require.False(t, found264)
}

func TestDisablePublishCodec(t *testing.T) {
	participant := newParticipantForTestWithOpts("123", &participantOpts{
		publisher: true,
		clientConf: &conkit.ClientConfiguration{
			DisabledCodecs: &conkit.DisabledCodecs{
				Publish: []*conkit.Codec{
					{Mime: "video/h264"},
				},
			},
		},
	})

	for _, codec := range participant.enabledPublishCodecs {
		require.False(t, mime.IsMimeTypeStringH264(codec.Mime))
	}

	sink := &routingfakes.FakeMessageSink{}
	participant.SetResponseSink(sink)
	var publishReceived atomic.Bool
	sink.WriteMessageCalls(func(msg proto.Message) error {
		if res, ok := msg.(*conkit.SignalResponse); ok {
			if published := res.GetTrackPublished(); published != nil {
				publishReceived.Store(true)
				require.NotEmpty(t, published.Track.Codecs)
				require.True(t, mime.IsMimeTypeStringVP8(published.Track.Codecs[0].MimeType))
			}
		}
		return nil
	})

	// simulcast codec response should pick an alternative
	participant.AddTrack(&conkit.AddTrackRequest{
		Cid:  "cid1",
		Type: conkit.TrackType_VIDEO,
		SimulcastCodecs: []*conkit.SimulcastCodec{{
			Codec: "h264",
			Cid:   "cid1",
		}},
	})

	require.Eventually(t, func() bool { return publishReceived.Load() }, 5*time.Second, 10*time.Millisecond)

	// publishing a supported codec should not change
	publishReceived.Store(false)
	sink.WriteMessageCalls(func(msg proto.Message) error {
		if res, ok := msg.(*conkit.SignalResponse); ok {
			if published := res.GetTrackPublished(); published != nil {
				publishReceived.Store(true)
				require.NotEmpty(t, published.Track.Codecs)
				require.True(t, mime.IsMimeTypeStringVP8(published.Track.Codecs[0].MimeType))
			}
		}
		return nil
	})
	participant.AddTrack(&conkit.AddTrackRequest{
		Cid:  "cid2",
		Type: conkit.TrackType_VIDEO,
		SimulcastCodecs: []*conkit.SimulcastCodec{{
			Codec: "vp8",
			Cid:   "cid2",
		}},
	})
	require.Eventually(t, func() bool { return publishReceived.Load() }, 5*time.Second, 10*time.Millisecond)
}

func TestPreferVideoCodecForPublisher(t *testing.T) {
	participant := newParticipantForTestWithOpts("123", &participantOpts{
		publisher: true,
	})
	participant.SetMigrateState(types.MigrateStateComplete)

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	defer pc.Close()

	for i := 0; i < 2; i++ {
		// publish h264 track without client preferred codec
		trackCid := fmt.Sprintf("preferh264video%d", i)
		participant.AddTrack(&conkit.AddTrackRequest{
			Type:   conkit.TrackType_VIDEO,
			Name:   "video",
			Width:  1280,
			Height: 720,
			Source: conkit.TrackSource_CAMERA,
			SimulcastCodecs: []*conkit.SimulcastCodec{
				{
					Codec: "h264",
					Cid:   trackCid,
				},
			},
		})

		track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, trackCid, trackCid)
		require.NoError(t, err)
		transceiver, err := pc.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
		require.NoError(t, err)
		codecs := transceiver.Receiver().GetParameters().Codecs

		if i > 0 {
			// the negotiated codecs order could be updated by first negotiation, reorder to make h264 not preferred
			for mime.IsMimeTypeStringH264(codecs[0].MimeType) {
				codecs = append(codecs[1:], codecs[0])
			}
		}
		// h264 should not be preferred
		require.False(t, mime.IsMimeTypeStringH264(codecs[0].MimeType), "codecs", codecs)

		sdp, err := pc.CreateOffer(nil)
		require.NoError(t, err)
		require.NoError(t, pc.SetLocalDescription(sdp))

		sink := &routingfakes.FakeMessageSink{}
		participant.SetResponseSink(sink)
		var answer webrtc.SessionDescription
		var answerReceived atomic.Bool
		sink.WriteMessageCalls(func(msg proto.Message) error {
			if res, ok := msg.(*conkit.SignalResponse); ok {
				if res.GetAnswer() != nil {
					answer = FromProtoSessionDescription(res.GetAnswer())
					pc.SetRemoteDescription(answer)
					answerReceived.Store(true)
				}
			}
			return nil
		})
		participant.HandleOffer(sdp)

		require.Eventually(t, func() bool { return answerReceived.Load() }, 5*time.Second, 10*time.Millisecond)

		var h264Preferred bool
		parsed, err := answer.Unmarshal()
		require.NoError(t, err)
		var videoSectionIndex int
		for _, m := range parsed.MediaDescriptions {
			if m.MediaName.Media == "video" {
				if videoSectionIndex == i {
					codecs, err := codecsFromMediaDescription(m)
					require.NoError(t, err)
					if mime.IsMimeTypeCodecStringH264(codecs[0].Name) {
						h264Preferred = true
						break
					}
				}
				videoSectionIndex++
			}
		}

		require.Truef(t, h264Preferred, "h264 should be preferred for video section %d, answer sdp: \n%s", i, answer.SDP)
	}
}

func TestPreferAudioCodecForRed(t *testing.T) {
	participant := newParticipantForTestWithOpts("123", &participantOpts{
		publisher: true,
	})
	participant.SetMigrateState(types.MigrateStateComplete)

	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	require.NoError(t, me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: RedCodecCapability,
		PayloadType:        63,
	}, webrtc.RTPCodecTypeAudio))

	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	defer pc.Close()

	for i, disableRed := range []bool{false, true} {
		t.Run(fmt.Sprintf("disableRed=%v", disableRed), func(t *testing.T) {
			trackCid := fmt.Sprintf("audiotrack%d", i)
			participant.AddTrack(&conkit.AddTrackRequest{
				Type:       conkit.TrackType_AUDIO,
				DisableRed: disableRed,
				Cid:        trackCid,
			})
			track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, trackCid, trackCid)
			require.NoError(t, err)
			transceiver, err := pc.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
			require.NoError(t, err)
			codecs := transceiver.Sender().GetParameters().Codecs
			for i, c := range codecs {
				if c.MimeType == "audio/opus" && i != 0 {
					codecs[0], codecs[i] = codecs[i], codecs[0]
					break
				}
			}
			transceiver.SetCodecPreferences(codecs)
			sdp, err := pc.CreateOffer(nil)
			require.NoError(t, err)
			pc.SetLocalDescription(sdp)
			// opus should be preferred
			require.Equal(t, codecs[0].MimeType, "audio/opus", sdp)

			sink := &routingfakes.FakeMessageSink{}
			participant.SetResponseSink(sink)
			var answer webrtc.SessionDescription
			var answerReceived atomic.Bool
			sink.WriteMessageCalls(func(msg proto.Message) error {
				if res, ok := msg.(*conkit.SignalResponse); ok {
					if res.GetAnswer() != nil {
						answer = FromProtoSessionDescription(res.GetAnswer())
						pc.SetRemoteDescription(answer)
						answerReceived.Store(true)
					}
				}
				return nil
			})
			participant.HandleOffer(sdp)

			require.Eventually(t, func() bool { return answerReceived.Load() }, 5*time.Second, 10*time.Millisecond)

			var redPreferred bool
			parsed, err := answer.Unmarshal()
			require.NoError(t, err)
			var audioSectionIndex int
			for _, m := range parsed.MediaDescriptions {
				if m.MediaName.Media == "audio" {
					if audioSectionIndex == i {
						codecs, err := codecsFromMediaDescription(m)
						require.NoError(t, err)
						// nack is always enabled. if red is preferred, server will not generate nack request
						var nackEnabled bool
						for _, c := range codecs {
							if c.Name == "opus" {
								for _, fb := range c.RTCPFeedback {
									if strings.Contains(fb, "nack") {
										nackEnabled = true
										break
									}
								}
							}
						}
						require.True(t, nackEnabled, "nack should be enabled for opus")

						if mime.IsMimeTypeCodecStringRED(codecs[0].Name) {
							redPreferred = true
							break
						}
					}
					audioSectionIndex++
				}
			}
			require.Equalf(t, !disableRed, redPreferred, "offer : \n%s\nanswer sdp: \n%s", sdp, answer.SDP)
		})
	}
}

type participantOpts struct {
	permissions     *conkit.ParticipantPermission
	protocolVersion types.ProtocolVersion
	publisher       bool
	clientConf      *conkit.ClientConfiguration
	clientInfo      *conkit.ClientInfo
}

func newParticipantForTestWithOpts(identity conkit.ParticipantIdentity, opts *participantOpts) *ParticipantImpl {
	if opts == nil {
		opts = &participantOpts{}
	}
	if opts.protocolVersion == 0 {
		opts.protocolVersion = 6
	}
	conf, _ := config.NewConfig("", true, nil, nil)
	// disable mux, it doesn't play too well with unit test
	conf.RTC.TCPPort = 0
	rtcConf, err := NewWebRTCConfig(conf)
	if err != nil {
		panic(err)
	}
	ff := buffer.NewFactoryOfBufferFactory(500, 200)
	rtcConf.SetBufferFactory(ff.CreateBufferFactory())
	grants := &auth.ClaimGrants{
		Video: &auth.VideoGrant{},
	}
	if opts.permissions != nil {
		grants.Video.SetCanPublish(opts.permissions.CanPublish)
		grants.Video.SetCanPublishData(opts.permissions.CanPublishData)
		grants.Video.SetCanSubscribe(opts.permissions.CanSubscribe)
	}

	enabledCodecs := make([]*conkit.Codec, 0, len(conf.Room.EnabledCodecs))
	for _, c := range conf.Room.EnabledCodecs {
		enabledCodecs = append(enabledCodecs, &conkit.Codec{
			Mime:     c.Mime,
			FmtpLine: c.FmtpLine,
		})
	}
	sid := conkit.ParticipantID(guid.New(utils.ParticipantPrefix))
	p, _ := NewParticipant(ParticipantParams{
		SID:                    sid,
		Identity:               identity,
		Config:                 rtcConf,
		Sink:                   &routingfakes.FakeMessageSink{},
		ProtocolVersion:        opts.protocolVersion,
		SessionStartTime:       time.Now(),
		PLIThrottleConfig:      conf.RTC.PLIThrottle,
		Grants:                 grants,
		PublishEnabledCodecs:   enabledCodecs,
		SubscribeEnabledCodecs: enabledCodecs,
		ClientConf:             opts.clientConf,
		ClientInfo:             ClientInfo{ClientInfo: opts.clientInfo},
		Logger:                 LoggerWithParticipant(logger.GetLogger(), identity, sid, false),
		Telemetry:              &telemetryfakes.FakeTelemetryService{},
		VersionGenerator:       utils.NewDefaultTimedVersionGenerator(),
	})
	p.isPublisher.Store(opts.publisher)
	p.updateState(conkit.ParticipantInfo_ACTIVE)

	return p
}

func newParticipantForTest(identity conkit.ParticipantIdentity) *ParticipantImpl {
	return newParticipantForTestWithOpts(identity, nil)
}
