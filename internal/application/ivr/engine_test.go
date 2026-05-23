package ivr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/divord97/ccc/internal/domain/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeGraph(nodes []routing.FlowNode) *routing.FlowGraph {
	return &routing.FlowGraph{Nodes: nodes}
}

func TestEngine_SimpleFlow_StartPlayEnd(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, map[string]string{"caller": "+86138"})

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{"variables":{"greeting":"hello"}}`), Exits: map[string]string{"default": "p"}},
		{ID: "p", Type: routing.NodePlay, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "e"}},
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Len(t, sess.History, 3)
	assert.Equal(t, "hello", sess.Variables["greeting"])
	assert.Equal(t, "+86138", sess.Variables["caller"])
}

func TestEngine_BranchCondition(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, map[string]string{"level": "vip"})

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "b"}},
		{ID: "b", Type: routing.NodeBranch, Config: json.RawMessage(`{
			"conditions": [
				{"name": "vip_path", "variable": "level", "operator": "eq", "value": "vip"},
				{"name": "normal_path", "variable": "level", "operator": "eq", "value": "normal"}
			]
		}`), Exits: map[string]string{"vip_path": "vip_end", "normal_path": "normal_end", "default": "normal_end"}},
		{ID: "vip_end", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
		{ID: "normal_end", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	// Should take vip_path
	assert.Equal(t, "vip_path", sess.History[1].ExitName)
	assert.Equal(t, "vip_end", sess.History[2].NodeID)
}

func TestEngine_BranchDefaultPath(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, map[string]string{"level": "unknown"})

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "b"}},
		{ID: "b", Type: routing.NodeBranch, Config: json.RawMessage(`{
			"conditions": [{"name": "match", "variable": "level", "operator": "eq", "value": "vip"}]
		}`), Exits: map[string]string{"match": "e1", "default": "e2"}},
		{ID: "e1", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
		{ID: "e2", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Equal(t, "e2", sess.History[2].NodeID)
}

func TestEngine_SetVariable(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, nil)

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "sv"}},
		{ID: "sv", Type: routing.NodeSetVariable, Config: json.RawMessage(`{
			"assignments": [{"name": "foo", "value": "bar"}, {"name": "x", "value": "42"}]
		}`), Exits: map[string]string{"default": "e"}},
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Equal(t, "bar", sess.Variables["foo"])
	assert.Equal(t, "42", sess.Variables["x"])
}

func TestEngine_HangupReason(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, nil)

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "hr"}},
		{ID: "hr", Type: routing.NodeHangupReason, Config: json.RawMessage(`{"reason":"OFF_HOURS"}`), Exits: map[string]string{"default": "e"}},
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Equal(t, "OFF_HOURS", sess.Variables["hangup_reason"])
}

func TestEngine_CollectDTMF(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, nil)

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "dtmf"}},
		{ID: "dtmf", Type: routing.NodeCollectDTMF, Config: json.RawMessage(`{"min_digits":1,"max_digits":4}`), Exits: map[string]string{"success": "e", "failure": "e"}},
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Contains(t, sess.Variables, "dtmf_input")
}

func TestEngine_TransferToAgent(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, nil)

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "ta"}},
		{ID: "ta", Type: routing.NodeTransferToAgent, Config: json.RawMessage(`{"skill_group_id":"sg1","routing_strategy":"longest_idle"}`), Exits: map[string]string{"success": "e", "failure": "e"}},
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Equal(t, "sg1", sess.Variables["transfer_skill_group"])
	assert.Equal(t, "longest_idle", sess.Variables["transfer_strategy"])
}

func TestEngine_FullFlow_AllNodeTypes(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, nil)

	// Linear chain through all 20 node types
	g := makeGraph([]routing.FlowNode{
		{ID: "n01", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n02"}},
		{ID: "n02", Type: routing.NodePlay, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n03"}},
		{ID: "n03", Type: routing.NodeCollectDTMF, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n04"}},
		{ID: "n04", Type: routing.NodeBranch, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n05"}},
		{ID: "n05", Type: routing.NodeTransferToAgent, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n06"}},
		{ID: "n06", Type: routing.NodeTransferToExternal, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n07"}},
		{ID: "n07", Type: routing.NodeBlindTransfer, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n08"}},
		{ID: "n08", Type: routing.NodeSetVariable, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n09"}},
		{ID: "n09", Type: routing.NodeVoicemail, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n10"}},
		{ID: "n10", Type: routing.NodeHangupReason, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n11"}},
		{ID: "n11", Type: routing.NodeFunction, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n12"}},
		{ID: "n12", Type: routing.NodeHTTPRequest, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n13"}},
		{ID: "n13", Type: routing.NodeJSONParser, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n14"}},
		{ID: "n14", Type: routing.NodeSMS, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n15"}},
		{ID: "n15", Type: routing.NodeSatisfactionRating, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n16"}},
		{ID: "n16", Type: routing.NodeASR, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n17"}},
		{ID: "n17", Type: routing.NodeSubFlow, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n18"}},
		{ID: "n18", Type: routing.NodeDigitalEmployee, Config: json.RawMessage(`{}`), Exits: map[string]string{"success": "n19"}},
		{ID: "n19", Type: routing.NodeCallback, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "n20"}},
		{ID: "n20", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	require.NoError(t, err)
	assert.Len(t, sess.History, 20)
}

func TestEngine_NoStartNode_Error(t *testing.T) {
	e := DefaultEngine()
	sess := NewSession(1, 1, 1, nil)

	g := makeGraph([]routing.FlowNode{
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	assert.Error(t, err)
}

func TestEngine_MissingHandler_Error(t *testing.T) {
	e := NewEngine() // empty engine, no handlers registered
	sess := NewSession(1, 1, 1, nil)

	g := makeGraph([]routing.FlowNode{
		{ID: "s", Type: routing.NodeStart, Config: json.RawMessage(`{}`), Exits: map[string]string{"default": "e"}},
		{ID: "e", Type: routing.NodeEnd, Config: json.RawMessage(`{}`), Exits: map[string]string{}},
	})

	err := e.Execute(context.Background(), sess, g)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler")
}
