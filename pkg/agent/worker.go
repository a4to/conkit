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

package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	pagent "github.com/a4to/protocol/agent"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/rpc"
	"github.com/a4to/protocol/utils"
	"github.com/a4to/protocol/utils/guid"
	"github.com/a4to/psrpc"
)

var (
	ErrUnimplementedWrorkerSignal = errors.New("unimplemented worker signal")
	ErrUnknownWorkerSignal        = errors.New("unknown worker signal")
	ErrUnknownJobType             = errors.New("unknown job type")
	ErrJobNotFound                = psrpc.NewErrorf(psrpc.NotFound, "no running job for given jobID")
	ErrWorkerClosed               = errors.New("worker closed")
	ErrWorkerNotAvailable         = errors.New("worker not available")
	ErrAvailabilityTimeout        = errors.New("agent worker availability timeout")
	ErrDuplicateJobAssignment     = errors.New("duplicate job assignment")
)

type WorkerProtocolVersion int

const CurrentProtocol = 1

const (
	RegisterTimeout  = 10 * time.Second
	AssignJobTimeout = 10 * time.Second
)

type SignalConn interface {
	WriteServerMessage(msg *conkit.ServerMessage) (int, error)
	ReadWorkerMessage() (*conkit.WorkerMessage, int, error)
	SetReadDeadline(time.Time) error
	Close() error
}

func JobStatusIsEnded(s conkit.JobStatus) bool {
	return s == conkit.JobStatus_JS_SUCCESS || s == conkit.JobStatus_JS_FAILED
}

type WorkerSignalHandler interface {
	HandleRegister(*conkit.RegisterWorkerRequest) error
	HandleAvailability(*conkit.AvailabilityResponse) error
	HandleUpdateJob(*conkit.UpdateJobStatus) error
	HandleSimulateJob(*conkit.SimulateJobRequest) error
	HandlePing(*conkit.WorkerPing) error
	HandleUpdateWorker(*conkit.UpdateWorkerStatus) error
	HandleMigrateJob(*conkit.MigrateJobRequest) error
}

func DispatchWorkerSignal(req *conkit.WorkerMessage, h WorkerSignalHandler) error {
	switch m := req.Message.(type) {
	case *conkit.WorkerMessage_Register:
		return h.HandleRegister(m.Register)
	case *conkit.WorkerMessage_Availability:
		return h.HandleAvailability(m.Availability)
	case *conkit.WorkerMessage_UpdateJob:
		return h.HandleUpdateJob(m.UpdateJob)
	case *conkit.WorkerMessage_SimulateJob:
		return h.HandleSimulateJob(m.SimulateJob)
	case *conkit.WorkerMessage_Ping:
		return h.HandlePing(m.Ping)
	case *conkit.WorkerMessage_UpdateWorker:
		return h.HandleUpdateWorker(m.UpdateWorker)
	case *conkit.WorkerMessage_MigrateJob:
		return h.HandleMigrateJob(m.MigrateJob)
	default:
		return ErrUnknownWorkerSignal
	}
}

var _ WorkerSignalHandler = (*UnimplementedWorkerSignalHandler)(nil)

type UnimplementedWorkerSignalHandler struct{}

func (UnimplementedWorkerSignalHandler) HandleRegister(*conkit.RegisterWorkerRequest) error {
	return fmt.Errorf("%w: Register", ErrUnimplementedWrorkerSignal)
}
func (UnimplementedWorkerSignalHandler) HandleAvailability(*conkit.AvailabilityResponse) error {
	return fmt.Errorf("%w: Availability", ErrUnimplementedWrorkerSignal)
}
func (UnimplementedWorkerSignalHandler) HandleUpdateJob(*conkit.UpdateJobStatus) error {
	return fmt.Errorf("%w: UpdateJob", ErrUnimplementedWrorkerSignal)
}
func (UnimplementedWorkerSignalHandler) HandleSimulateJob(*conkit.SimulateJobRequest) error {
	return fmt.Errorf("%w: SimulateJob", ErrUnimplementedWrorkerSignal)
}
func (UnimplementedWorkerSignalHandler) HandlePing(*conkit.WorkerPing) error {
	return fmt.Errorf("%w: Ping", ErrUnimplementedWrorkerSignal)
}
func (UnimplementedWorkerSignalHandler) HandleUpdateWorker(*conkit.UpdateWorkerStatus) error {
	return fmt.Errorf("%w: UpdateWorker", ErrUnimplementedWrorkerSignal)
}
func (UnimplementedWorkerSignalHandler) HandleMigrateJob(*conkit.MigrateJobRequest) error {
	return fmt.Errorf("%w: MigrateJob", ErrUnimplementedWrorkerSignal)
}

type WorkerPingHandler struct {
	UnimplementedWorkerSignalHandler
	conn SignalConn
}

func (h WorkerPingHandler) HandlePing(ping *conkit.WorkerPing) error {
	_, err := h.conn.WriteServerMessage(&conkit.ServerMessage{
		Message: &conkit.ServerMessage_Pong{
			Pong: &conkit.WorkerPong{
				LastTimestamp: ping.Timestamp,
				Timestamp:     time.Now().UnixMilli(),
			},
		},
	})
	return err
}

type WorkerRegistration struct {
	Protocol    WorkerProtocolVersion
	ID          string
	Version     string
	AgentName   string
	Namespace   string
	JobType     conkit.JobType
	Permissions *conkit.ParticipantPermission
}

var _ WorkerSignalHandler = (*WorkerRegisterer)(nil)

type WorkerRegisterer struct {
	WorkerPingHandler
	serverInfo *conkit.ServerInfo
	protocol   WorkerProtocolVersion
	deadline   time.Time

	registration WorkerRegistration
	registered   bool
}

func NewWorkerRegisterer(conn SignalConn, serverInfo *conkit.ServerInfo, protocol WorkerProtocolVersion) *WorkerRegisterer {
	return &WorkerRegisterer{
		WorkerPingHandler: WorkerPingHandler{conn: conn},
		serverInfo:        serverInfo,
		protocol:          protocol,
		deadline:          time.Now().Add(RegisterTimeout),
	}
}

func (h *WorkerRegisterer) Deadline() time.Time {
	return h.deadline
}

func (h *WorkerRegisterer) Registration() WorkerRegistration {
	return h.registration
}

func (h *WorkerRegisterer) Registered() bool {
	return h.registered
}

func (h *WorkerRegisterer) HandleRegister(req *conkit.RegisterWorkerRequest) error {
	if !conkit.IsJobType(req.GetType()) {
		return ErrUnknownJobType
	}

	permissions := req.AllowedPermissions
	if permissions == nil {
		permissions = &conkit.ParticipantPermission{
			CanSubscribe:      true,
			CanPublish:        true,
			CanPublishData:    true,
			CanUpdateMetadata: true,
		}
	}

	h.registration = WorkerRegistration{
		Protocol:    h.protocol,
		ID:          guid.New(guid.AgentWorkerPrefix),
		Version:     req.Version,
		AgentName:   req.AgentName,
		Namespace:   req.GetNamespace(),
		JobType:     req.GetType(),
		Permissions: permissions,
	}
	h.registered = true

	_, err := h.conn.WriteServerMessage(&conkit.ServerMessage{
		Message: &conkit.ServerMessage_Register{
			Register: &conkit.RegisterWorkerResponse{
				WorkerId:   h.registration.ID,
				ServerInfo: h.serverInfo,
			},
		},
	})
	return err
}

var _ WorkerSignalHandler = (*Worker)(nil)

type Worker struct {
	WorkerPingHandler
	WorkerRegistration

	apiKey    string
	apiSecret string
	logger    logger.Logger

	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	mu     sync.Mutex
	load   float32
	status conkit.WorkerStatus

	runningJobs  map[conkit.JobID]*conkit.Job
	availability map[conkit.JobID]chan *conkit.AvailabilityResponse
}

func NewWorker(
	registration WorkerRegistration,
	apiKey string,
	apiSecret string,
	conn SignalConn,
	logger logger.Logger,
) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		WorkerPingHandler:  WorkerPingHandler{conn: conn},
		WorkerRegistration: registration,
		apiKey:             apiKey,
		apiSecret:          apiSecret,
		logger: logger.WithValues(
			"workerID", registration.ID,
			"agentName", registration.AgentName,
			"jobType", registration.JobType.String(),
		),

		ctx:    ctx,
		cancel: cancel,
		closed: make(chan struct{}),

		runningJobs:  make(map[conkit.JobID]*conkit.Job),
		availability: make(map[conkit.JobID]chan *conkit.AvailabilityResponse),
	}
}

func (w *Worker) sendRequest(req *conkit.ServerMessage) {
	if _, err := w.conn.WriteServerMessage(req); err != nil {
		w.logger.Warnw("error writing to websocket", err)
	}
}

func (w *Worker) Status() conkit.WorkerStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

func (w *Worker) Load() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.load
}

func (w *Worker) Logger() logger.Logger {
	return w.logger
}

func (w *Worker) RunningJobs() map[conkit.JobID]*conkit.Job {
	w.mu.Lock()
	defer w.mu.Unlock()
	jobs := make(map[conkit.JobID]*conkit.Job, len(w.runningJobs))
	for k, v := range w.runningJobs {
		jobs[k] = v
	}
	return jobs
}

func (w *Worker) RunningJobCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.runningJobs)
}

func (w *Worker) GetJobState(jobID conkit.JobID) (*conkit.JobState, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	j, ok := w.runningJobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	return utils.CloneProto(j.State), nil
}

func (w *Worker) AssignJob(ctx context.Context, job *conkit.Job) (*conkit.JobState, error) {
	availCh := make(chan *conkit.AvailabilityResponse, 1)
	job = utils.CloneProto(job)
	jobID := conkit.JobID(job.Id)

	w.mu.Lock()
	if _, ok := w.availability[jobID]; ok {
		w.mu.Unlock()
		return nil, ErrDuplicateJobAssignment
	}

	w.availability[jobID] = availCh
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.availability, jobID)
		w.mu.Unlock()
	}()

	if job.State == nil {
		job.State = &conkit.JobState{}
	}
	now := time.Now()
	job.State.UpdatedAt = now.UnixNano()
	job.State.StartedAt = now.UnixNano()
	job.State.Status = conkit.JobStatus_JS_RUNNING

	w.sendRequest(&conkit.ServerMessage{Message: &conkit.ServerMessage_Availability{
		Availability: &conkit.AvailabilityRequest{Job: job},
	}})

	timeout := time.NewTimer(AssignJobTimeout)
	defer timeout.Stop()

	// See handleAvailability for the response
	select {
	case res := <-availCh:
		if !res.Available {
			return nil, ErrWorkerNotAvailable
		}

		job.State.ParticipantIdentity = res.ParticipantIdentity

		token, err := pagent.BuildAgentToken(
			w.apiKey,
			w.apiSecret,
			job.Room.Name,
			res.ParticipantIdentity,
			res.ParticipantName,
			res.ParticipantMetadata,
			res.ParticipantAttributes,
			w.Permissions,
		)
		if err != nil {
			w.logger.Errorw("failed to build agent token", err)
			return nil, err
		}

		// In OSS, Url is nil, and the used API Key is the same as the one used to connect the worker
		w.sendRequest(&conkit.ServerMessage{Message: &conkit.ServerMessage_Assignment{
			Assignment: &conkit.JobAssignment{Job: job, Url: nil, Token: token},
		}})

		state := utils.CloneProto(job.State)

		w.mu.Lock()
		w.runningJobs[jobID] = job
		w.mu.Unlock()

		// TODO sweep jobs that are never started. We can't do this until all SDKs actually update the the JOB state

		return state, nil
	case <-timeout.C:
		return nil, ErrAvailabilityTimeout
	case <-w.ctx.Done():
		return nil, ErrWorkerClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (w *Worker) TerminateJob(jobID conkit.JobID, reason rpc.JobTerminateReason) (*conkit.JobState, error) {
	w.mu.Lock()
	_, ok := w.runningJobs[jobID]
	w.mu.Unlock()

	if !ok {
		return nil, ErrJobNotFound
	}

	w.sendRequest(&conkit.ServerMessage{Message: &conkit.ServerMessage_Termination{
		Termination: &conkit.JobTermination{
			JobId: string(jobID),
		},
	}})

	status := conkit.JobStatus_JS_SUCCESS
	errorStr := ""
	if reason == rpc.JobTerminateReason_AGENT_LEFT_ROOM {
		status = conkit.JobStatus_JS_FAILED
		errorStr = "agent worker left the room"
	}

	return w.UpdateJobStatus(&conkit.UpdateJobStatus{
		JobId:  string(jobID),
		Status: status,
		Error:  errorStr,
	})
}

func (w *Worker) UpdateMetadata(metadata string) {
	w.logger.Debugw("worker metadata updated", nil, "metadata", metadata)
}

func (w *Worker) IsClosed() bool {
	select {
	case <-w.closed:
		return true
	default:
		return false
	}
}

func (w *Worker) Close() {
	w.mu.Lock()
	if w.IsClosed() {
		w.mu.Unlock()
		return
	}

	w.logger.Infow("closing worker", "workerID", w.ID, "jobType", w.JobType, "agentName", w.AgentName)

	close(w.closed)
	w.cancel()
	_ = w.conn.Close()
	w.mu.Unlock()
}

func (w *Worker) HandleAvailability(res *conkit.AvailabilityResponse) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	jobID := conkit.JobID(res.JobId)
	availCh, ok := w.availability[jobID]
	if !ok {
		w.logger.Warnw("received availability response for unknown job", nil, "jobID", jobID)
		return nil
	}

	availCh <- res
	delete(w.availability, jobID)

	return nil
}

func (w *Worker) HandleUpdateJob(update *conkit.UpdateJobStatus) error {
	_, err := w.UpdateJobStatus(update)
	if err != nil {
		// treating this as a debug message only
		// this can happen if the Room closes first, which would delete the agent dispatch
		// that would mark the job as successful. subsequent updates from the same worker
		// would not be able to find the same jobID.
		w.logger.Debugw("received job update for unknown job", "jobID", update.JobId)
	}
	return nil
}

func (w *Worker) UpdateJobStatus(update *conkit.UpdateJobStatus) (*conkit.JobState, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	jobID := conkit.JobID(update.JobId)
	job, ok := w.runningJobs[jobID]
	if !ok {
		return nil, psrpc.NewErrorf(psrpc.NotFound, "received job update for unknown job")
	}

	now := time.Now()
	job.State.UpdatedAt = now.UnixNano()

	if job.State.Status == conkit.JobStatus_JS_PENDING && update.Status != conkit.JobStatus_JS_PENDING {
		job.State.StartedAt = now.UnixNano()
	}

	job.State.Status = update.Status
	job.State.Error = update.Error

	if JobStatusIsEnded(update.Status) {
		job.State.EndedAt = now.UnixNano()
		delete(w.runningJobs, jobID)

		w.logger.Infow("job ended", "jobID", update.JobId, "status", update.Status, "error", update.Error)
	}

	return proto.Clone(job.State).(*conkit.JobState), nil
}

func (w *Worker) HandleSimulateJob(simulate *conkit.SimulateJobRequest) error {
	jobType := conkit.JobType_JT_ROOM
	if simulate.Participant != nil {
		jobType = conkit.JobType_JT_PUBLISHER
	}

	job := &conkit.Job{
		Id:          guid.New(guid.AgentJobPrefix),
		Type:        jobType,
		Room:        simulate.Room,
		Participant: simulate.Participant,
		Namespace:   w.Namespace,
		AgentName:   w.AgentName,
	}

	go func() {
		_, err := w.AssignJob(w.ctx, job)
		if err != nil {
			w.logger.Errorw("unable to simulate job", err, "jobID", job.Id)
		}
	}()

	return nil
}

func (w *Worker) HandleUpdateWorker(update *conkit.UpdateWorkerStatus) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if status := update.Status; status != nil && w.status != *status {
		w.status = *status
		w.Logger().Debugw("worker status changed", "status", w.status)
	}
	w.load = update.GetLoad()

	return nil
}

func (w *Worker) HandleMigrateJob(req *conkit.MigrateJobRequest) error {
	// TODO(theomonnom): On OSS this is not implemented
	// We could maybe just move a specific job to another worker
	return nil
}
