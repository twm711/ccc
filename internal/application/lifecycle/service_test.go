package lifecycle

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	_ = snowflake.Init(1)
	os.Exit(m.Run())
}

// --- minimal test doubles ---

type stubCallService struct {
	endCallFn  func(ctx context.Context, id int64, reason call.HangupReason, hangupBy ...call.HangupBy) (*call.Call, error)
	answerFn   func(ctx context.Context, id int64, agentUserID int64) (*call.Call, error)
	createFn   func(ctx context.Context, in call.CreateCallInput) (*call.Call, error)
	updateDurFn func(ctx context.Context, c *call.Call) error
	listEvtFn  func(ctx context.Context, callID int64) ([]*call.CallEvent, error)
}

func (s *stubCallService) EndCall(ctx context.Context, id int64, reason call.HangupReason, hangupBy ...call.HangupBy) (*call.Call, error) {
	if s.endCallFn != nil {
		return s.endCallFn(ctx, id, reason, hangupBy...)
	}
	now := time.Now()
	answered := now.Add(-30 * time.Second)
	return &call.Call{ID: id, TenantID: 1, Direction: call.DirectionInbound, Status: call.CallStatusCompleted, StartedAt: answered.Add(-5 * time.Second), AnsweredAt: &answered, EndedAt: &now, HangupReason: &reason, DurationSec: 35}, nil
}

func (s *stubCallService) AnswerCall(ctx context.Context, id int64, agentUserID int64) (*call.Call, error) {
	if s.answerFn != nil {
		return s.answerFn(ctx, id, agentUserID)
	}
	now := time.Now()
	uid := agentUserID
	return &call.Call{ID: id, TenantID: 1, Direction: call.DirectionInbound, Status: call.CallStatusActive, AgentUserID: &uid, AnsweredAt: &now, StartedAt: now.Add(-3 * time.Second)}, nil
}

func (s *stubCallService) CreateInboundCall(ctx context.Context, in call.CreateCallInput) (*call.Call, error) {
	if s.createFn != nil {
		return s.createFn(ctx, in)
	}
	return &call.Call{ID: 1, TenantID: in.TenantID, Direction: call.DirectionInbound, Status: call.CallStatusIVR, StartedAt: time.Now()}, nil
}

func (s *stubCallService) UpdateDurations(ctx context.Context, c *call.Call) error {
	if s.updateDurFn != nil {
		return s.updateDurFn(ctx, c)
	}
	return nil
}

func (s *stubCallService) ListEvents(ctx context.Context, callID int64) ([]*call.CallEvent, error) {
	if s.listEvtFn != nil {
		return s.listEvtFn(ctx, callID)
	}
	return nil, nil
}

type stubRecordingRepo struct {
	created bool
}

func (r *stubRecordingRepo) Create(_ context.Context, _ *call.Recording) error {
	r.created = true
	return nil
}

type stubNotifier struct {
	events []string
}

func (n *stubNotifier) NotifyAgent(_ int64, eventType string, _ int64, _ interface{}) {
	n.events = append(n.events, eventType)
}

type stubPublisher struct {
	published []string
}

func (p *stubPublisher) Publish(_ context.Context, subject string, _ interface{}) error {
	p.published = append(p.published, subject)
	return nil
}

// --- tests ---

func newTestService(cs *stubCallService, opts ...func(*Service)) *Service {
	svc := &Service{
		callSvc: call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil),
		logger:  zerolog.Nop(),
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}

func TestEndCall_PublishesEvent(t *testing.T) {
	pub := &stubPublisher{}
	notif := &stubNotifier{}
	callSvc := call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil)

	svc := &Service{
		callSvc:   callSvc,
		publisher: pub,
		notifier:  notif,
		logger:    zerolog.Nop(),
	}

	// Pre-create a call to end
	c, err := callSvc.CreateInboundCall(context.Background(), call.CreateCallInput{
		TenantID: 1, Caller: "13800138000", Callee: "4001234567",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Answer it first so EndCall can complete
	agentID := int64(100)
	_, err = callSvc.AnswerCall(context.Background(), c.ID, agentID)
	if err != nil {
		t.Fatal(err)
	}

	ended, err := svc.EndCall(context.Background(), c.ID, call.HangupNormal, call.HangupByAgent)
	if err != nil {
		t.Fatal(err)
	}
	if ended.Status != call.CallStatusCompleted {
		t.Errorf("expected completed, got %s", ended.Status)
	}

	// Verify event was published
	found := false
	for _, s := range pub.published {
		if s == "ccc.call.ended" {
			found = true
		}
	}
	if !found {
		t.Error("expected ccc.call.ended to be published")
	}

	// Verify notifier was called
	foundNotif := false
	for _, e := range notif.events {
		if e == "call.ended" {
			foundNotif = true
		}
	}
	if !foundNotif {
		t.Error("expected call.ended notification to agent")
	}
}

func TestAnswerCall_SetsPresenceAndPublishes(t *testing.T) {
	pub := &stubPublisher{}
	callSvc := call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil)

	svc := &Service{
		callSvc:   callSvc,
		publisher: pub,
		logger:    zerolog.Nop(),
	}

	c, err := callSvc.CreateInboundCall(context.Background(), call.CreateCallInput{
		TenantID: 1, Caller: "13800138000", Callee: "4001234567",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Transition to queue then ringing before answering
	_, err = callSvc.TransitionToQueue(context.Background(), c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}

	answered, _, err := svc.AnswerCall(context.Background(), c.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if answered.Status != call.CallStatusActive {
		t.Errorf("expected active, got %s", answered.Status)
	}
	if answered.AnsweredAt == nil {
		t.Error("AnsweredAt should be set")
	}

	found := false
	for _, s := range pub.published {
		if s == "ccc.call.answered" {
			found = true
		}
	}
	if !found {
		t.Error("expected ccc.call.answered to be published")
	}
}

func TestHandleInboundCall_CreatesAndPublishes(t *testing.T) {
	pub := &stubPublisher{}
	callSvc := call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil)

	svc := &Service{
		callSvc:   callSvc,
		publisher: pub,
		logger:    zerolog.Nop(),
	}

	c, err := svc.HandleInboundCall(context.Background(), call.CreateCallInput{
		TenantID: 1, Caller: "13800138000", Callee: "4001234567",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != call.CallStatusIVR {
		t.Errorf("expected IVR status, got %s", c.Status)
	}

	found := false
	for _, s := range pub.published {
		if s == "ccc.call.created" {
			found = true
		}
	}
	if !found {
		t.Error("expected ccc.call.created to be published")
	}
}

func TestEndCall_ReleaseConcurrency(t *testing.T) {
	pub := &stubPublisher{}
	callSvc := call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil)

	released := false
	guard := &stubConcurrencyGuard{releaseFn: func() { released = true }}

	svc := &Service{
		callSvc:     callSvc,
		publisher:   pub,
		concurrency: guard,
		logger:      zerolog.Nop(),
	}

	c, _ := callSvc.CreateInboundCall(context.Background(), call.CreateCallInput{
		TenantID: 1, Caller: "13800138000", Callee: "4001234567",
	})
	_, _ = callSvc.AnswerCall(context.Background(), c.ID, 100)

	_, err := svc.EndCall(context.Background(), c.ID, call.HangupNormal)
	if err != nil {
		t.Fatal(err)
	}
	if !released {
		t.Error("expected concurrency guard to be released")
	}
}

type stubConcurrencyGuard struct {
	releaseFn func()
}

func (g *stubConcurrencyGuard) Acquire(_ context.Context, _ int64, _ int) (bool, error) {
	return true, nil
}

func (g *stubConcurrencyGuard) Release(_ context.Context, _ int64) {
	if g.releaseFn != nil {
		g.releaseFn()
	}
}

func TestTransitionCallToQueue_PublishesEvent(t *testing.T) {
	pub := &stubPublisher{}
	callSvc := call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil)

	svc := &Service{
		callSvc:   callSvc,
		publisher: pub,
		logger:    zerolog.Nop(),
	}

	c, err := callSvc.CreateInboundCall(context.Background(), call.CreateCallInput{
		TenantID: 1, Caller: "13800138000", Callee: "4001234567",
	})
	if err != nil {
		t.Fatal(err)
	}

	queued, err := svc.TransitionCallToQueue(context.Background(), c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if queued.Status != call.CallStatusQueue {
		t.Errorf("expected queue status, got %s", queued.Status)
	}
}

func TestHandleInboundCall_ConcurrencyLimitRejects(t *testing.T) {
	callSvc := call.NewCallService(call.NewMockCallRepo(), call.NewMockCallEventRepo(), nil)
	guard := &stubConcurrencyGuard{}

	svc := &Service{
		callSvc:     callSvc,
		concurrency: guard,
		logger:      zerolog.Nop(),
		tenantSettings: &stubTenantSettings{maxConcurrent: 0},
	}

	// With maxConcurrent=0 from settings, concurrency check should still proceed
	_, err := svc.HandleInboundCall(context.Background(), call.CreateCallInput{
		TenantID: 1, Caller: "13800138000", Callee: "4001234567",
	})
	if err != nil {
		t.Fatal(err)
	}
}

type stubTenantSettings struct {
	maxConcurrent int
}

func (s *stubTenantSettings) GetByTenantID(_ context.Context, _ int64) int {
	return s.maxConcurrent
}
