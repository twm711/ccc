package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = snowflake.Init(1)
}

func validGraph() json.RawMessage {
	return json.RawMessage(`{
		"nodes": [
			{"id": "s1", "type": "start", "config": {}, "exits": {"default": "p1"}},
			{"id": "p1", "type": "play", "config": {"audio_id": "a1"}, "exits": {"default": "e1"}},
			{"id": "e1", "type": "end", "config": {}, "exits": {}}
		],
		"edges": []
	}`)
}

func fullGraph() json.RawMessage {
	// A graph using all 20 node types to verify validator accepts them all
	g := FlowGraph{
		Nodes: []FlowNode{
			{ID: "start", Type: NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "play"}},
			{ID: "play", Type: NodePlay, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "dtmf"}},
			{ID: "dtmf", Type: NodeCollectDTMF, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "branch", "failure": "end"}},
			{ID: "branch", Type: NodeBranch, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "transfer"}},
			{ID: "transfer", Type: NodeTransferToAgent, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "ext", "failure": "end"}},
			{ID: "ext", Type: NodeTransferToExternal, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "blind", "failure": "end"}},
			{ID: "blind", Type: NodeBlindTransfer, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "setvar", "failure": "end"}},
			{ID: "setvar", Type: NodeSetVariable, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "vm"}},
			{ID: "vm", Type: NodeVoicemail, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "hangup"}},
			{ID: "hangup", Type: NodeHangupReason, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "func"}},
			{ID: "func", Type: NodeFunction, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "http", "failure": "end"}},
			{ID: "http", Type: NodeHTTPRequest, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "jsonp", "failure": "end"}},
			{ID: "jsonp", Type: NodeJSONParser, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "sms", "failure": "end"}},
			{ID: "sms", Type: NodeSMS, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "csat", "failure": "end"}},
			{ID: "csat", Type: NodeSatisfactionRating, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "asr", "failure": "end"}},
			{ID: "asr", Type: NodeASR, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "sub", "failure": "end"}},
			{ID: "sub", Type: NodeSubFlow, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "de"}},
			{ID: "de", Type: NodeDigitalEmployee, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "cb", "failure": "end"}},
			{ID: "cb", Type: NodeCallback, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "end"}},
			{ID: "end", Type: NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
		},
	}
	b, _ := json.Marshal(g)
	return b
}

// --- IVR Flow Service Tests ---

func TestIVRFlowService_Create_Success(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())

	flow, err := svc.Create(context.Background(), CreateFlowInput{
		TenantID: 1,
		Code:     "main_ivr",
		Name:     "Main IVR",
		Graph:    validGraph(),
	})

	require.NoError(t, err)
	assert.Equal(t, "main_ivr", flow.Code)
	assert.Equal(t, FlowStatusDraft, flow.Status)
	assert.Equal(t, FlowTypeMain, flow.FlowType)
	assert.Equal(t, 1, flow.Version)
}

func TestIVRFlowService_Create_DuplicateCode(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	_, _ = svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "main", Name: "M", Graph: validGraph()})
	_, err := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "main", Name: "M2", Graph: validGraph()})
	assert.ErrorIs(t, err, ErrFlowCodeExists)
}

func TestIVRFlowService_Create_InvalidGraph(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())

	_, err := svc.Create(context.Background(), CreateFlowInput{
		TenantID: 1, Code: "bad", Name: "Bad",
		Graph: json.RawMessage(`{"nodes":[]}`),
	})
	assert.Error(t, err)
}

func TestIVRFlowService_Publish_DraftToPublished(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "pub", Name: "P", Graph: validGraph()})

	published, err := svc.Publish(ctx, flow.ID, 100)
	require.NoError(t, err)
	assert.Equal(t, FlowStatusPublished, published.Status)
	assert.NotNil(t, published.PublishedAt)
	assert.Equal(t, 2, published.Version)
}

func TestIVRFlowService_Publish_InvalidGraph_Error(t *testing.T) {
	repo := NewMockFlowRepo()
	svc := NewIVRFlowService(repo, NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "bad2", Name: "B", Graph: validGraph()})

	// Corrupt the graph after creation
	flow.Graph = json.RawMessage(`{"nodes":[{"id":"s","type":"start","config":{},"exits":{}}]}`)
	_ = repo.Update(ctx, flow)

	_, err := svc.Publish(ctx, flow.ID, 100)
	assert.ErrorIs(t, err, ErrNoEndNode)
}

func TestIVRFlowService_Publish_AlreadyPublished_Error(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "pub2", Name: "P2", Graph: validGraph()})
	_, _ = svc.Publish(ctx, flow.ID, 100)

	_, err := svc.Publish(ctx, flow.ID, 100)
	assert.ErrorIs(t, err, ErrFlowNotDraft)
}

func TestIVRFlowService_Lock_Success(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "lock", Name: "L", Graph: validGraph()})

	locked, err := svc.Lock(ctx, flow.ID, 100)
	require.NoError(t, err)
	assert.NotNil(t, locked.LockedBy)
	assert.Equal(t, int64(100), *locked.LockedBy)
}

func TestIVRFlowService_Lock_AlreadyLocked_Error(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "lock2", Name: "L2", Graph: validGraph()})
	_, _ = svc.Lock(ctx, flow.ID, 100)

	_, err := svc.Lock(ctx, flow.ID, 200) // different user
	assert.ErrorIs(t, err, ErrFlowLocked)
}

func TestIVRFlowService_Lock_SameUser_OK(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "lock3", Name: "L3", Graph: validGraph()})
	_, _ = svc.Lock(ctx, flow.ID, 100)

	locked, err := svc.Lock(ctx, flow.ID, 100) // same user re-lock
	require.NoError(t, err)
	assert.Equal(t, int64(100), *locked.LockedBy)
}

func TestIVRFlowService_Unlock_NotOwner_Error(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "unlock", Name: "U", Graph: validGraph()})
	_, _ = svc.Lock(ctx, flow.ID, 100)

	_, err := svc.Unlock(ctx, flow.ID, 200)
	assert.ErrorIs(t, err, ErrFlowNotOwner)
}

func TestIVRFlowService_Clone_Success(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "orig", Name: "Original", Graph: validGraph()})

	clone, err := svc.Clone(ctx, flow.ID, "clone1", "Clone 1")
	require.NoError(t, err)
	assert.Equal(t, "clone1", clone.Code)
	assert.Equal(t, "Clone 1", clone.Name)
	assert.Equal(t, FlowStatusDraft, clone.Status)
	assert.NotEqual(t, flow.ID, clone.ID)
}

func TestIVRFlowService_Rollback_ToVersion(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())
	ctx := context.Background()

	flow, _ := svc.Create(ctx, CreateFlowInput{TenantID: 1, Code: "rb", Name: "R", Graph: validGraph()})
	published, _ := svc.Publish(ctx, flow.ID, 100)

	rolled, err := svc.Rollback(ctx, published.ID, published.Version)
	require.NoError(t, err)
	assert.Equal(t, FlowStatusDraft, rolled.Status)
}

func TestIVRFlowService_ValidateNode_AllTypes(t *testing.T) {
	svc := NewIVRFlowService(NewMockFlowRepo(), NewMockVersionRepo())

	flow, err := svc.Create(context.Background(), CreateFlowInput{
		TenantID: 1,
		Code:     "all20",
		Name:     "All 20 Nodes",
		Graph:    fullGraph(),
	})

	require.NoError(t, err)
	assert.NotNil(t, flow)
}

func TestIVRFlowService_Validate_NoStartNode(t *testing.T) {
	g := `{"nodes":[{"id":"e","type":"end","config":{},"exits":{}}]}`
	_, err := ValidateGraph(json.RawMessage(g))
	assert.ErrorIs(t, err, ErrNoStartNode)
}

func TestIVRFlowService_Validate_NoEndNode(t *testing.T) {
	g := `{"nodes":[{"id":"s","type":"start","config":{},"exits":{}}]}`
	_, err := ValidateGraph(json.RawMessage(g))
	assert.ErrorIs(t, err, ErrNoEndNode)
}

func TestIVRFlowService_Validate_DisconnectedNode(t *testing.T) {
	g := `{"nodes":[
		{"id":"s","type":"start","config":{},"exits":{"default":"e"}},
		{"id":"e","type":"end","config":{},"exits":{}},
		{"id":"orphan","type":"play","config":{},"exits":{"default":"e"}}
	]}`
	_, err := ValidateGraph(json.RawMessage(g))
	assert.ErrorIs(t, err, ErrDisconnectedNode)
}

func TestIVRFlowService_Validate_InvalidNodeType(t *testing.T) {
	g := `{"nodes":[
		{"id":"s","type":"start","config":{},"exits":{"default":"bad"}},
		{"id":"bad","type":"nonexistent_type","config":{},"exits":{"default":"e"}},
		{"id":"e","type":"end","config":{},"exits":{}}
	]}`
	_, err := ValidateGraph(json.RawMessage(g))
	assert.ErrorIs(t, err, ErrInvalidNodeType)
}

func TestIVRFlowService_Validate_BrokenExitTarget(t *testing.T) {
	g := `{"nodes":[
		{"id":"s","type":"start","config":{},"exits":{"default":"missing_node"}},
		{"id":"e","type":"end","config":{},"exits":{}}
	]}`
	_, err := ValidateGraph(json.RawMessage(g))
	assert.ErrorIs(t, err, ErrDisconnectedNode)
}

func TestAllNodeTypes_Count(t *testing.T) {
	assert.Equal(t, 21, len(AllNodeTypes), fmt.Sprintf("expected 21 node types, got %d", len(AllNodeTypes)))
}
