package testutils

import (
	"context"
	"errors"
	"io"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/frostbyte73/core"
	"github.com/gammazero/deque"
	"golang.org/x/exp/maps"

	"github.com/a4to/conkit-server/pkg/agent"
	"github.com/a4to/conkit-server/pkg/config"
	"github.com/a4to/conkit-server/pkg/routing"
	"github.com/a4to/conkit-server/pkg/service"
	"github.com/a4to/protocol/auth"
	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/utils/events"
	"github.com/a4to/protocol/utils/guid"
	"github.com/a4to/protocol/utils/must"
	"github.com/a4to/protocol/utils/options"
	"github.com/a4to/psrpc"
)

type AgentService interface {
	HandleConnection(context.Context, agent.SignalConn, agent.WorkerProtocolVersion)
	DrainConnections(time.Duration)
}

type TestServer struct {
	AgentService
}

func NewTestServer(bus psrpc.MessageBus) *TestServer {
	localNode, _ := routing.NewLocalNode(nil)
	return NewTestServerWithService(must.Get(service.NewAgentService(
		&config.Config{Region: "test"},
		localNode,
		bus,
		auth.NewSimpleKeyProvider("test", "verysecretsecret"),
	)))
}

func NewTestServerWithService(s AgentService) *TestServer {
	return &TestServer{s}
}

type SimulatedWorkerOptions struct {
	Context            context.Context
	Label              string
	SupportResume      bool
	DefaultJobLoad     float32
	JobLoadThreshold   float32
	DefaultWorkerLoad  float32
	HandleAvailability func(AgentJobRequest)
	HandleAssignment   func(*conkit.Job) JobLoad
}

type SimulatedWorkerOption func(*SimulatedWorkerOptions)

func WithContext(ctx context.Context) SimulatedWorkerOption {
	return func(o *SimulatedWorkerOptions) {
		o.Context = ctx
	}
}

func WithLabel(label string) SimulatedWorkerOption {
	return func(o *SimulatedWorkerOptions) {
		o.Label = label
	}
}

func WithJobAvailabilityHandler(h func(AgentJobRequest)) SimulatedWorkerOption {
	return func(o *SimulatedWorkerOptions) {
		o.HandleAvailability = h
	}
}

func WithJobAssignmentHandler(h func(*conkit.Job) JobLoad) SimulatedWorkerOption {
	return func(o *SimulatedWorkerOptions) {
		o.HandleAssignment = h
	}
}

func WithJobLoad(l JobLoad) SimulatedWorkerOption {
	return WithJobAssignmentHandler(func(j *conkit.Job) JobLoad { return l })
}

func WithDefaultWorkerLoad(load float32) SimulatedWorkerOption {
	return func(o *SimulatedWorkerOptions) {
		o.DefaultWorkerLoad = load
	}
}

func (h *TestServer) SimulateAgentWorker(opts ...SimulatedWorkerOption) *AgentWorker {
	o := &SimulatedWorkerOptions{
		Context:            context.Background(),
		Label:              guid.New("TEST_AGENT_"),
		DefaultJobLoad:     0.1,
		JobLoadThreshold:   0.8,
		DefaultWorkerLoad:  0.0,
		HandleAvailability: func(r AgentJobRequest) { r.Accept() },
		HandleAssignment:   func(j *conkit.Job) JobLoad { return nil },
	}
	options.Apply(o, opts)

	w := &AgentWorker{
		workerMessages:         make(chan *conkit.WorkerMessage, 1),
		jobs:                   map[string]*AgentJob{},
		SimulatedWorkerOptions: o,

		RegisterWorkerResponses: events.NewObserverList[*conkit.RegisterWorkerResponse](),
		AvailabilityRequests:    events.NewObserverList[*conkit.AvailabilityRequest](),
		JobAssignments:          events.NewObserverList[*conkit.JobAssignment](),
		JobTerminations:         events.NewObserverList[*conkit.JobTermination](),
		WorkerPongs:             events.NewObserverList[*conkit.WorkerPong](),
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())

	if o.DefaultWorkerLoad > 0.0 {
		w.sendStatus()
	}

	ctx := service.WithAPIKey(o.Context, &auth.ClaimGrants{}, "test")
	go h.HandleConnection(ctx, w, agent.CurrentProtocol)

	return w
}

func (h *TestServer) Close() {
	h.DrainConnections(1)
}

var _ agent.SignalConn = (*AgentWorker)(nil)

type JobLoad interface {
	Load() float32
}

type AgentJob struct {
	*conkit.Job
	JobLoad
}

type AgentJobRequest struct {
	w *AgentWorker
	*conkit.AvailabilityRequest
}

func (r AgentJobRequest) Accept() {
	identity := guid.New("PI_")
	r.w.SendAvailability(&conkit.AvailabilityResponse{
		JobId:               r.Job.Id,
		Available:           true,
		SupportsResume:      r.w.SupportResume,
		ParticipantName:     identity,
		ParticipantIdentity: identity,
	})
}

func (r AgentJobRequest) Reject() {
	r.w.SendAvailability(&conkit.AvailabilityResponse{
		JobId:     r.Job.Id,
		Available: false,
	})
}

type AgentWorker struct {
	*SimulatedWorkerOptions

	fuse           core.Fuse
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	workerMessages chan *conkit.WorkerMessage
	serverMessages deque.Deque[*conkit.ServerMessage]
	jobs           map[string]*AgentJob

	RegisterWorkerResponses *events.ObserverList[*conkit.RegisterWorkerResponse]
	AvailabilityRequests    *events.ObserverList[*conkit.AvailabilityRequest]
	JobAssignments          *events.ObserverList[*conkit.JobAssignment]
	JobTerminations         *events.ObserverList[*conkit.JobTermination]
	WorkerPongs             *events.ObserverList[*conkit.WorkerPong]
}

func (w *AgentWorker) statusWorker() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for !w.fuse.IsBroken() {
		w.sendStatus()
		<-t.C
	}
}

func (w *AgentWorker) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.fuse.Break()
	return nil
}

func (w *AgentWorker) SetReadDeadline(t time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.fuse.IsBroken() {
		cancel := w.cancel
		if t.IsZero() {
			w.ctx, w.cancel = context.WithCancel(context.Background())
		} else {
			w.ctx, w.cancel = context.WithDeadline(context.Background(), t)
		}
		cancel()
	}
	return nil
}

func (w *AgentWorker) ReadWorkerMessage() (*conkit.WorkerMessage, int, error) {
	for {
		w.mu.Lock()
		ctx := w.ctx
		w.mu.Unlock()

		select {
		case <-w.fuse.Watch():
			return nil, 0, io.EOF
		case <-ctx.Done():
			if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
				return nil, 0, err
			}
		case m := <-w.workerMessages:
			return m, 0, nil
		}
	}
}

func (w *AgentWorker) WriteServerMessage(m *conkit.ServerMessage) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.serverMessages.PushBack(m)
	if w.serverMessages.Len() == 1 {
		go w.handleServerMessages()
	}
	return 0, nil
}

func (w *AgentWorker) handleServerMessages() {
	w.mu.Lock()
	for w.serverMessages.Len() != 0 {
		m := w.serverMessages.Front()
		w.mu.Unlock()

		switch m := m.Message.(type) {
		case *conkit.ServerMessage_Register:
			w.handleRegister(m.Register)
		case *conkit.ServerMessage_Availability:
			w.handleAvailability(m.Availability)
		case *conkit.ServerMessage_Assignment:
			w.handleAssignment(m.Assignment)
		case *conkit.ServerMessage_Termination:
			w.handleTermination(m.Termination)
		case *conkit.ServerMessage_Pong:
			w.handlePong(m.Pong)
		}

		w.mu.Lock()
		w.serverMessages.PopFront()
	}
	w.mu.Unlock()
}

func (w *AgentWorker) handleRegister(m *conkit.RegisterWorkerResponse) {
	w.RegisterWorkerResponses.Emit(m)
}

func (w *AgentWorker) handleAvailability(m *conkit.AvailabilityRequest) {
	w.AvailabilityRequests.Emit(m)
	if w.HandleAvailability != nil {
		w.HandleAvailability(AgentJobRequest{w, m})
	} else {
		AgentJobRequest{w, m}.Accept()
	}
}

func (w *AgentWorker) handleAssignment(m *conkit.JobAssignment) {
	w.JobAssignments.Emit(m)

	var load JobLoad
	if w.HandleAssignment != nil {
		load = w.HandleAssignment(m.Job)
	}

	if load == nil {
		load = NewStableJobLoad(w.DefaultJobLoad)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.jobs[m.Job.Id] = &AgentJob{m.Job, load}
}

func (w *AgentWorker) handleTermination(m *conkit.JobTermination) {
	w.JobTerminations.Emit(m)

	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.jobs, m.JobId)
}

func (w *AgentWorker) handlePong(m *conkit.WorkerPong) {
	w.WorkerPongs.Emit(m)
}

func (w *AgentWorker) sendMessage(m *conkit.WorkerMessage) {
	select {
	case <-w.fuse.Watch():
	case w.workerMessages <- m:
	}
}

func (w *AgentWorker) SendRegister(m *conkit.RegisterWorkerRequest) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_Register{
		Register: m,
	}})
}

func (w *AgentWorker) SendAvailability(m *conkit.AvailabilityResponse) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_Availability{
		Availability: m,
	}})
}

func (w *AgentWorker) SendUpdateWorker(m *conkit.UpdateWorkerStatus) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_UpdateWorker{
		UpdateWorker: m,
	}})
}

func (w *AgentWorker) SendUpdateJob(m *conkit.UpdateJobStatus) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_UpdateJob{
		UpdateJob: m,
	}})
}

func (w *AgentWorker) SendPing(m *conkit.WorkerPing) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_Ping{
		Ping: m,
	}})
}

func (w *AgentWorker) SendSimulateJob(m *conkit.SimulateJobRequest) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_SimulateJob{
		SimulateJob: m,
	}})
}

func (w *AgentWorker) SendMigrateJob(m *conkit.MigrateJobRequest) {
	w.sendMessage(&conkit.WorkerMessage{Message: &conkit.WorkerMessage_MigrateJob{
		MigrateJob: m,
	}})
}

func (w *AgentWorker) sendStatus() {
	w.mu.Lock()
	var load float32
	jobCount := len(w.jobs)

	if len(w.jobs) == 0 {
		load = w.DefaultWorkerLoad
	} else {
		for _, j := range w.jobs {
			load += j.Load()
		}
	}
	w.mu.Unlock()

	status := conkit.WorkerStatus_WS_AVAILABLE
	if load > w.JobLoadThreshold {
		status = conkit.WorkerStatus_WS_FULL
	}

	w.SendUpdateWorker(&conkit.UpdateWorkerStatus{
		Status:   &status,
		Load:     load,
		JobCount: uint32(jobCount),
	})
}

func (w *AgentWorker) Register(agentName string, jobType conkit.JobType) {
	w.SendRegister(&conkit.RegisterWorkerRequest{
		Type:      jobType,
		AgentName: agentName,
	})
	go w.statusWorker()
}

func (w *AgentWorker) SimulateRoomJob(roomName string) {
	w.SendSimulateJob(&conkit.SimulateJobRequest{
		Type: conkit.JobType_JT_ROOM,
		Room: &conkit.Room{
			Sid:  guid.New(guid.RoomPrefix),
			Name: roomName,
		},
	})
}

func (w *AgentWorker) Jobs() []*AgentJob {
	w.mu.Lock()
	defer w.mu.Unlock()
	return maps.Values(w.jobs)
}

type stableJobLoad struct {
	load float32
}

func NewStableJobLoad(load float32) JobLoad {
	return stableJobLoad{load}
}

func (s stableJobLoad) Load() float32 {
	return s.load
}

type periodicJobLoad struct {
	amplitude float64
	period    time.Duration
	epoch     time.Time
}

func NewPeriodicJobLoad(max float32, period time.Duration) JobLoad {
	return periodicJobLoad{
		amplitude: float64(max / 2),
		period:    period,
		epoch:     time.Now().Add(-time.Duration(rand.Int64N(int64(period)))),
	}
}

func (s periodicJobLoad) Load() float32 {
	a := math.Sin(time.Since(s.epoch).Seconds() / s.period.Seconds() * math.Pi * 2)
	return float32(s.amplitude + a*s.amplitude)
}

type uniformRandomJobLoad struct {
	min, max float32
	rng      func() float64
}

func NewUniformRandomJobLoad(min, max float32) JobLoad {
	return uniformRandomJobLoad{min, max, rand.Float64}
}

func NewUniformRandomJobLoadWithRNG(min, max float32, rng *rand.Rand) JobLoad {
	return uniformRandomJobLoad{min, max, rng.Float64}
}

func (s uniformRandomJobLoad) Load() float32 {
	return rand.Float32()*(s.max-s.min) + s.min
}

type normalRandomJobLoad struct {
	mean, stddev float64
	rng          func() float64
}

func NewNormalRandomJobLoad(mean, stddev float64) JobLoad {
	return normalRandomJobLoad{mean, stddev, rand.Float64}
}

func NewNormalRandomJobLoadWithRNG(mean, stddev float64, rng *rand.Rand) JobLoad {
	return normalRandomJobLoad{mean, stddev, rng.Float64}
}

func (s normalRandomJobLoad) Load() float32 {
	u := 1 - s.rng()
	v := s.rng()
	z := math.Sqrt(-2*math.Log(u)) * math.Cos(2*math.Pi*v)
	return float32(max(0, z*s.stddev+s.mean))
}
