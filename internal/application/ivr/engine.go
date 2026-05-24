package ivr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/divord97/ccc/internal/domain/routing"
	"github.com/divord97/ccc/internal/infrastructure/esl"
)

// Engine interprets an IVR FlowGraph DAG node-by-node.
type Engine struct {
	handlers   map[routing.NodeType]NodeHandler
	flowLoader FlowLoader
}

// NodeHandler processes a single IVR node and returns the exit name to follow.
type NodeHandler interface {
	Handle(ctx context.Context, sess *Session, node routing.FlowNode) (exitName string, err error)
}

// Session holds per-call IVR execution state.
type Session struct {
	CallID    int64
	TenantID  int64
	FlowID    int64
	CallUUID  string // FreeSWITCH call UUID
	ESL       *esl.Client
	Variables map[string]string
	History   []NodeVisit
}

type NodeVisit struct {
	NodeID   string
	NodeType routing.NodeType
	ExitName string
}

func NewEngine() *Engine {
	return &Engine{handlers: make(map[routing.NodeType]NodeHandler)}
}

func (e *Engine) RegisterHandler(nodeType routing.NodeType, h NodeHandler) {
	e.handlers[nodeType] = h
}

// Execute runs the IVR flow graph to completion.
func (e *Engine) Execute(ctx context.Context, sess *Session, graph *routing.FlowGraph) error {
	nodeMap := make(map[string]routing.FlowNode, len(graph.Nodes))
	var startNode *routing.FlowNode
	for i := range graph.Nodes {
		n := &graph.Nodes[i]
		nodeMap[n.ID] = *n
		if n.Type == routing.NodeStart {
			startNode = n
		}
	}
	if startNode == nil {
		return routing.ErrNoStartNode
	}

	current := *startNode
	maxSteps := 1000
	for step := 0; step < maxSteps; step++ {
		if current.Type == routing.NodeEnd {
			sess.History = append(sess.History, NodeVisit{
				NodeID: current.ID, NodeType: current.Type, ExitName: "",
			})
			return nil
		}

		handler, ok := e.handlers[current.Type]
		if !ok {
			return fmt.Errorf("no handler for node type %s", current.Type)
		}

		exitName, err := handler.Handle(ctx, sess, current)
		if err != nil {
			return fmt.Errorf("node %s (%s): %w", current.ID, current.Type, err)
		}

		sess.History = append(sess.History, NodeVisit{
			NodeID: current.ID, NodeType: current.Type, ExitName: exitName,
		})

		nextID, ok := current.Exits[exitName]
		if !ok {
			if def, hasDef := current.Exits["default"]; hasDef {
				nextID = def
			} else {
				return fmt.Errorf("node %s: no exit %q and no default", current.ID, exitName)
			}
		}

		next, exists := nodeMap[nextID]
		if !exists {
			return fmt.Errorf("node %s: exit target %s not found", current.ID, nextID)
		}
		current = next
	}
	return fmt.Errorf("exceeded max IVR steps (%d)", maxSteps)
}

// NewSession creates a new IVR execution session.
func NewSession(callID, tenantID, flowID int64, callUUID string, eslClient *esl.Client, sysVars map[string]string) *Session {
	vars := make(map[string]string)
	for k, v := range sysVars {
		vars[k] = v
	}
	return &Session{
		CallID:    callID,
		TenantID:  tenantID,
		FlowID:    flowID,
		CallUUID:  callUUID,
		ESL:       eslClient,
		Variables: vars,
	}
}

// FlowLoader retrieves a parsed FlowGraph by flow ID.
type FlowLoader func(ctx context.Context, flowID int64) (*routing.FlowGraph, error)

// ExecuteFlow loads a flow by ID and executes it within the given session.
func (e *Engine) ExecuteFlow(ctx context.Context, sess *Session, flowID int64) error {
	if e.flowLoader == nil {
		return fmt.Errorf("ivr: no flow loader configured")
	}
	graph, err := e.flowLoader(ctx, flowID)
	if err != nil {
		return err
	}
	return e.Execute(ctx, sess, graph)
}

// DefaultEngine returns an engine with all built-in node handlers registered.
func DefaultEngine(eslClient *esl.Client, flowLoader FlowLoader) *Engine {
	e := NewEngine()
	e.flowLoader = flowLoader
	e.RegisterHandler(routing.NodeStart, &StartHandler{})
	e.RegisterHandler(routing.NodePlay, &PlayHandler{})
	e.RegisterHandler(routing.NodeCollectDTMF, &CollectDTMFHandler{})
	e.RegisterHandler(routing.NodeBranch, &BranchHandler{})
	e.RegisterHandler(routing.NodeSetVariable, &SetVariableHandler{})
	e.RegisterHandler(routing.NodeHangupReason, &HangupReasonHandler{})
	e.RegisterHandler(routing.NodeEnd, &EndHandler{})
	e.RegisterHandler(routing.NodeFunction, &FunctionHandler{})
	e.RegisterHandler(routing.NodeHTTPRequest, &HTTPRequestHandler{})
	e.RegisterHandler(routing.NodeJSONParser, &JSONParserHandler{})
	e.RegisterHandler(routing.NodeSMS, &SMSHandler{})
	e.RegisterHandler(routing.NodeSatisfactionRating, &SatisfactionRatingHandler{})
	e.RegisterHandler(routing.NodeASR, &ASRHandler{})
	e.RegisterHandler(routing.NodeVoicemail, &VoicemailHandler{})
	e.RegisterHandler(routing.NodeTransferToAgent, &TransferToAgentHandler{})
	e.RegisterHandler(routing.NodeTransferToExternal, &TransferToExternalHandler{})
	e.RegisterHandler(routing.NodeBlindTransfer, &BlindTransferHandler{})
	e.RegisterHandler(routing.NodeSubFlow, &SubFlowHandler{engine: e, flowLoader: flowLoader})
	e.RegisterHandler(routing.NodeDigitalEmployee, &DigitalEmployeeHandler{})
	e.RegisterHandler(routing.NodeCallback, &CallbackHandler{})
	return e
}

// parseConfig unmarshals node config into target struct.
func parseConfig(raw json.RawMessage, target interface{}) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}
