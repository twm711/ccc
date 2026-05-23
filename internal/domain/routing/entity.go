package routing

import (
	"encoding/json"
	"time"
)

type FlowStatus string

const (
	FlowStatusDraft              FlowStatus = "draft"
	FlowStatusPublishing         FlowStatus = "publishing"
	FlowStatusPublished          FlowStatus = "published"
	FlowStatusEditing            FlowStatus = "editing"
	FlowStatusPublishedWithDraft FlowStatus = "published_with_draft"
	FlowStatusFailed             FlowStatus = "failed"
	FlowStatusArchived           FlowStatus = "archived"
)

type FlowType string

const (
	FlowTypeMain   FlowType = "main"
	FlowTypeSub    FlowType = "sub"
	FlowTypeSurvey FlowType = "survey"
)

type NodeType string

const (
	NodeStart              NodeType = "start"
	NodePlay               NodeType = "play"
	NodeCollectDTMF        NodeType = "collect_dtmf"
	NodeBranch             NodeType = "branch"
	NodeTransferToAgent    NodeType = "transfer_to_agent"
	NodeTransferToExternal NodeType = "transfer_to_external"
	NodeBlindTransfer      NodeType = "blind_transfer"
	NodeSetVariable        NodeType = "set_variable"
	NodeVoicemail          NodeType = "voicemail"
	NodeHangupReason       NodeType = "hangup_reason"
	NodeFunction           NodeType = "function"
	NodeHTTPRequest        NodeType = "http_request"
	NodeJSONParser         NodeType = "json_parser"
	NodeSMS                NodeType = "sms"
	NodeSatisfactionRating NodeType = "satisfaction_rating"
	NodeASR                NodeType = "asr"
	NodeSubFlow            NodeType = "sub_flow"
	NodeDigitalEmployee    NodeType = "digital_employee"
	NodeCallback           NodeType = "callback"
	NodeEnd                NodeType = "end"
)

var AllNodeTypes = []NodeType{
	NodeStart, NodePlay, NodeCollectDTMF, NodeBranch,
	NodeTransferToAgent, NodeTransferToExternal, NodeBlindTransfer,
	NodeSetVariable, NodeVoicemail, NodeHangupReason,
	NodeFunction, NodeHTTPRequest, NodeJSONParser, NodeSMS,
	NodeSatisfactionRating, NodeASR, NodeSubFlow, NodeDigitalEmployee,
	NodeCallback, NodeEnd,
}

type IVRFlow struct {
	ID          int64          `db:"id" json:"id"`
	TenantID    int64          `db:"tenant_id" json:"tenant_id"`
	Code        string         `db:"code" json:"code"`
	Name        string         `db:"name" json:"name"`
	FlowType    FlowType       `db:"flow_type" json:"flow_type"`
	Version     int            `db:"version" json:"version"`
	Graph       json.RawMessage `db:"graph" json:"graph"`
	Status      FlowStatus     `db:"status" json:"status"`
	LockedBy    *int64         `db:"locked_by" json:"locked_by,omitempty"`
	LockedAt    *time.Time     `db:"locked_at" json:"locked_at,omitempty"`
	PublishedAt *time.Time     `db:"published_at" json:"published_at,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}

type IVRFlowVersion struct {
	ID          int64           `db:"id" json:"id"`
	IVRFlowID   int64           `db:"ivr_flow_id" json:"ivr_flow_id"`
	TenantID    int64           `db:"tenant_id" json:"tenant_id"`
	Version     int             `db:"version" json:"version"`
	Graph       json.RawMessage `db:"graph" json:"graph"`
	Description string          `db:"description" json:"description"`
	PublishedBy *int64          `db:"published_by" json:"published_by,omitempty"`
	PublishedAt time.Time       `db:"published_at" json:"published_at"`
}

// FlowGraph is the parsed IVR flow graph structure.
type FlowGraph struct {
	Nodes []FlowNode `json:"nodes"`
	Edges []FlowEdge `json:"edges"`
}

type FlowNode struct {
	ID     string            `json:"id"`
	Type   NodeType          `json:"type"`
	Config json.RawMessage   `json:"config"`
	Exits  map[string]string `json:"exits"`
}

type FlowEdge struct {
	Source     string `json:"source"`
	SourcePort string `json:"source_port"`
	Target     string `json:"target"`
}
