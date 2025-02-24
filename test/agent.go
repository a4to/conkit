// Copyright 2024 LiveKit, Inc.
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

package test

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"

	"github.com/a4to/protocol/conkit"
)

type agentClient struct {
	mu   sync.Mutex
	conn *websocket.Conn

	registered              atomic.Int32
	roomAvailability        atomic.Int32
	roomJobs                atomic.Int32
	publisherAvailability   atomic.Int32
	publisherJobs           atomic.Int32
	participantAvailability atomic.Int32
	participantJobs         atomic.Int32

	requestedJobs chan *conkit.Job

	done chan struct{}
}

func newAgentClient(token string, port uint32) (*agentClient, error) {
	host := fmt.Sprintf("ws://localhost:%d", port)
	u, err := url.Parse(host + "/agent")
	if err != nil {
		return nil, err
	}
	requestHeader := make(http.Header)
	requestHeader.Set("Authorization", "Bearer "+token)

	connectUrl := u.String()
	conn, _, err := websocket.DefaultDialer.Dial(connectUrl, requestHeader)
	if err != nil {
		return nil, err
	}

	return &agentClient{
		conn:          conn,
		requestedJobs: make(chan *conkit.Job, 100),
		done:          make(chan struct{}),
	}, nil
}

func (c *agentClient) Run(jobType conkit.JobType, namespace string) (err error) {
	go c.read()

	switch jobType {
	case conkit.JobType_JT_ROOM:
		err = c.write(&conkit.WorkerMessage{
			Message: &conkit.WorkerMessage_Register{
				Register: &conkit.RegisterWorkerRequest{
					Type:      conkit.JobType_JT_ROOM,
					Version:   "version",
					Namespace: &namespace,
				},
			},
		})

	case conkit.JobType_JT_PUBLISHER:
		err = c.write(&conkit.WorkerMessage{
			Message: &conkit.WorkerMessage_Register{
				Register: &conkit.RegisterWorkerRequest{
					Type:      conkit.JobType_JT_PUBLISHER,
					Version:   "version",
					Namespace: &namespace,
				},
			},
		})

	case conkit.JobType_JT_PARTICIPANT:
		err = c.write(&conkit.WorkerMessage{
			Message: &conkit.WorkerMessage_Register{
				Register: &conkit.RegisterWorkerRequest{
					Type:      conkit.JobType_JT_PARTICIPANT,
					Version:   "version",
					Namespace: &namespace,
				},
			},
		})
	}

	return err
}

func (c *agentClient) read() {
	for {
		select {
		case <-c.done:
			return
		default:
			_, b, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			msg := &conkit.ServerMessage{}
			if err = proto.Unmarshal(b, msg); err != nil {
				return
			}

			switch m := msg.Message.(type) {
			case *conkit.ServerMessage_Assignment:
				go c.handleAssignment(m.Assignment)
			case *conkit.ServerMessage_Availability:
				go c.handleAvailability(m.Availability)
			case *conkit.ServerMessage_Register:
				go c.handleRegister(m.Register)
			}
		}
	}
}

func (c *agentClient) handleAssignment(req *conkit.JobAssignment) {
	switch req.Job.Type {
	case conkit.JobType_JT_ROOM:
		c.roomJobs.Inc()
	case conkit.JobType_JT_PUBLISHER:
		c.publisherJobs.Inc()
	case conkit.JobType_JT_PARTICIPANT:
		c.participantJobs.Inc()
	}
}

func (c *agentClient) handleAvailability(req *conkit.AvailabilityRequest) {
	switch req.Job.Type {
	case conkit.JobType_JT_ROOM:
		c.roomAvailability.Inc()
	case conkit.JobType_JT_PUBLISHER:
		c.publisherAvailability.Inc()
	case conkit.JobType_JT_PARTICIPANT:
		c.participantAvailability.Inc()
	}

	c.requestedJobs <- req.Job

	c.write(&conkit.WorkerMessage{
		Message: &conkit.WorkerMessage_Availability{
			Availability: &conkit.AvailabilityResponse{
				JobId:     req.Job.Id,
				Available: true,
			},
		},
	})
}

func (c *agentClient) handleRegister(req *conkit.RegisterWorkerResponse) {
	c.registered.Inc()
}

func (c *agentClient) write(msg *conkit.WorkerMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return nil
	default:
		b, err := proto.Marshal(msg)
		if err != nil {
			return err
		}

		return c.conn.WriteMessage(websocket.BinaryMessage, b)
	}
}

func (c *agentClient) close() {
	c.mu.Lock()
	close(c.done)
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	_ = c.conn.Close()
	c.mu.Unlock()
}
