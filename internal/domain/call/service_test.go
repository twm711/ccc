package call

import (
	"context"
	"testing"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = snowflake.Init(1)
}

func TestCallService_CreateInboundCall(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	ctx := context.Background()

	ivrID := int64(999)
	c, err := svc.CreateInboundCall(ctx, CreateCallInput{
		TenantID:  1,
		Direction: DirectionInbound,
		Caller:    "+8613800001111",
		Callee:    "+862188880001",
		IVRFlowID: &ivrID,
	})

	require.NoError(t, err)
	assert.Equal(t, DirectionInbound, c.Direction)
	assert.Equal(t, CallTypeNormal, c.CallType)
	assert.Equal(t, MediaTypeAudio, c.MediaType)
	assert.Equal(t, CallStatusIVR, c.Status)
	assert.Equal(t, "+8613800001111", c.Caller)
}

func TestCallService_CreateInboundCall_DefaultDirection(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())

	c, err := svc.CreateInboundCall(context.Background(), CreateCallInput{
		TenantID: 1, Caller: "+86138", Callee: "+86021",
	})

	require.NoError(t, err)
	assert.Equal(t, DirectionInbound, c.Direction)
	assert.Equal(t, CallTypeNormal, c.CallType)
}

func TestCallService_RecordIVRTracking_NodeSequence(t *testing.T) {
	trackRepo := NewMockIVRTrackingRepo()
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), trackRepo)
	ctx := context.Background()

	c, _ := svc.CreateInboundCall(ctx, CreateCallInput{
		TenantID: 1, Caller: "a", Callee: "b",
	})

	nodes := []struct {
		nodeID   string
		nodeType string
	}{
		{"start_1", "start"},
		{"play_1", "play"},
		{"dtmf_1", "collect_dtmf"},
		{"transfer_1", "transfer_to_agent"},
	}

	for _, n := range nodes {
		now := time.Now()
		err := svc.RecordIVRTracking(ctx, &IVRTracking{
			CallID:    c.ID,
			TenantID:  c.TenantID,
			IVRFlowID: 100,
			NodeID:    n.nodeID,
			NodeType:  n.nodeType,
			EnteredAt: now,
		})
		require.NoError(t, err)
	}

	tracking, err := svc.GetIVRTracking(ctx, c.ID)
	require.NoError(t, err)
	assert.Len(t, tracking, 4)
	assert.Equal(t, "start_1", tracking[0].NodeID)
	assert.Equal(t, "transfer_to_agent", tracking[3].NodeType)
}

func TestCallService_EndCall_WithHangupReason(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	ctx := context.Background()

	c, _ := svc.CreateInboundCall(ctx, CreateCallInput{
		TenantID: 1, Caller: "a", Callee: "b",
	})

	ended, err := svc.EndCall(ctx, c.ID, HangupNormal)
	require.NoError(t, err)
	assert.Equal(t, CallStatusCompleted, ended.Status)
	assert.NotNil(t, ended.HangupReason)
	assert.Equal(t, HangupNormal, *ended.HangupReason)
	assert.NotNil(t, ended.EndedAt)
}

func TestCallService_EndCall_AlreadyEnded(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	ctx := context.Background()

	c, _ := svc.CreateInboundCall(ctx, CreateCallInput{
		TenantID: 1, Caller: "a", Callee: "b",
	})
	_, _ = svc.EndCall(ctx, c.ID, HangupNormal)

	_, err := svc.EndCall(ctx, c.ID, HangupNormal)
	assert.ErrorIs(t, err, ErrCallAlreadyEnded)
}

func TestCallService_EndCall_NotFound(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())

	_, err := svc.EndCall(context.Background(), 99999, HangupNormal)
	assert.ErrorIs(t, err, ErrCallNotFound)
}

func TestCallService_CalculateDurations(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())

	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	c := &Call{ID: 1, TenantID: 1, StartedAt: base}

	events := []*CallEvent{
		{Event: "call_created", CreatedAt: base},
		{Event: "ivr_completed", CreatedAt: base.Add(15 * time.Second)},
		{Event: "queue_entered", CreatedAt: base.Add(15 * time.Second)},
		{Event: "agent_ringing", CreatedAt: base.Add(25 * time.Second)},
		{Event: "call_answered", CreatedAt: base.Add(30 * time.Second)},
	}

	svc.CalculateDurations(c, events)

	assert.Equal(t, 15, c.IVRDurationSec)
	assert.Equal(t, 10, c.QueueDurationSec)
	assert.Equal(t, 5, c.RingDurationSec)
}

func TestCallService_CreateOutboundCall(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	ctx := context.Background()

	agentID := int64(50)
	c, err := svc.CreateOutboundCall(ctx, CreateCallInput{
		TenantID:    1,
		Caller:      "+862188880001",
		Callee:      "+8613900139000",
		AgentUserID: &agentID,
	})

	require.NoError(t, err)
	assert.Equal(t, DirectionOutbound, c.Direction)
	assert.Equal(t, CallTypeNormal, c.CallType)
	assert.Equal(t, CallStatusRinging, c.Status)
	assert.Equal(t, &agentID, c.AgentUserID)
}

func TestCallService_CreateOutboundCall_MissingCallee(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())

	_, err := svc.CreateOutboundCall(context.Background(), CreateCallInput{
		TenantID: 1, Caller: "+86021",
	})
	assert.ErrorIs(t, err, ErrMissingCallee)
}

func TestCallService_CreateInternalCall(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	ctx := context.Background()

	c, err := svc.CreateInternalCall(ctx, CreateCallInput{
		TenantID: 1,
		Caller:   "agent_001",
		Callee:   "agent_002",
	})

	require.NoError(t, err)
	assert.Equal(t, DirectionOutbound, c.Direction)
	assert.Equal(t, CallTypeInternal, c.CallType)
	assert.Equal(t, CallStatusRinging, c.Status)
}

func TestCallService_ListCalls_Filter(t *testing.T) {
	repo := NewMockCallRepo()
	svc := NewCallService(repo, NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	ctx := context.Background()

	_, _ = svc.CreateInboundCall(ctx, CreateCallInput{TenantID: 1, Caller: "a", Callee: "b"})
	_, _ = svc.CreateOutboundCall(ctx, CreateCallInput{TenantID: 1, Caller: "c", Callee: "d"})
	_, _ = svc.CreateInternalCall(ctx, CreateCallInput{TenantID: 1, Caller: "e", Callee: "f"})

	// Filter outbound only
	dir := DirectionOutbound
	calls, total, err := svc.ListCalls(ctx, 1, CallListFilter{Direction: &dir}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // outbound + internal (both are outbound direction)
	assert.Len(t, calls, 2)

	// Filter internal call type
	ct := CallTypeInternal
	calls, total, err = svc.ListCalls(ctx, 1, CallListFilter{CallType: &ct}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, calls, 1)
	assert.Equal(t, CallTypeInternal, calls[0].CallType)
}

// --- Phase 3 TDD Tests ---

func newServiceWithCallback() (*CallService, *MockCallbackRepo) {
	cbRepo := NewMockCallbackRepo()
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo(), cbRepo)
	return svc, cbRepo
}

func createActiveCall(t *testing.T, svc *CallService) *Call {
	t.Helper()
	c, err := svc.CreateInboundCall(context.Background(), CreateCallInput{
		TenantID: 1, Caller: "a", Callee: "b",
	})
	require.NoError(t, err)
	c.Status = CallStatusActive
	_ = svc.calls.Update(context.Background(), c)
	return c
}

func TestCallService_HoldCall(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	c := createActiveCall(t, svc)

	held, err := svc.HoldCall(context.Background(), c.ID)
	require.NoError(t, err)
	assert.Equal(t, CallStatusHeld, held.Status)
	assert.Equal(t, 1, held.HoldCount)
}

func TestCallService_RetrieveCall(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	c := createActiveCall(t, svc)

	_, _ = svc.HoldCall(context.Background(), c.ID)
	retrieved, err := svc.RetrieveCall(context.Background(), c.ID)
	require.NoError(t, err)
	assert.Equal(t, CallStatusActive, retrieved.Status)
}

func TestCallService_BlindTransfer_ToSkillGroup(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	c := createActiveCall(t, svc)

	sgID := int64(10)
	transferred, err := svc.BlindTransfer(context.Background(), c.ID, TransferTarget{
		Type: "skill_group", SkillGroupID: &sgID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, transferred.TransferCount)
}

func TestCallService_BlindTransfer_ToAgent(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	c := createActiveCall(t, svc)

	agentID := int64(20)
	transferred, err := svc.BlindTransfer(context.Background(), c.ID, TransferTarget{
		Type: "agent", AgentUserID: &agentID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, transferred.TransferCount)
}

func TestCallService_BlindTransfer_ToExternal(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	c := createActiveCall(t, svc)

	transferred, err := svc.BlindTransfer(context.Background(), c.ID, TransferTarget{
		Type: "external", ExternalNum: "+8613800000000",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, transferred.TransferCount)
}

func TestCallService_SendDTMF(t *testing.T) {
	svc := NewCallService(NewMockCallRepo(), NewMockCallEventRepo(), NewMockIVRTrackingRepo())
	c := createActiveCall(t, svc)

	err := svc.SendDTMF(context.Background(), c.ID, "1234#")
	assert.NoError(t, err)
}

func TestCallService_RequestCallback(t *testing.T) {
	svc, _ := newServiceWithCallback()
	ctx := context.Background()

	cb := &CallbackRequest{
		TenantID:     1,
		CallID:       100,
		SkillGroupID: 10,
		Caller:       "+8613800138000",
	}
	err := svc.RequestCallback(ctx, cb)
	require.NoError(t, err)
	assert.Equal(t, "pending", cb.Status)
	assert.NotZero(t, cb.ID)
}

func TestCallService_ExecuteCallback(t *testing.T) {
	svc, _ := newServiceWithCallback()
	ctx := context.Background()

	cb := &CallbackRequest{TenantID: 1, CallID: 100, SkillGroupID: 10, Caller: "+86138"}
	_ = svc.RequestCallback(ctx, cb)

	completed, err := svc.ExecuteCallback(ctx, cb.ID, true)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
	assert.Equal(t, 1, completed.AttemptCount)
	assert.NotNil(t, completed.CompletedAt)
}
