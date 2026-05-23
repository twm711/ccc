package call

import (
	"encoding/json"
	"time"
)

type CallDirection string

const (
	DirectionInbound  CallDirection = "inbound"
	DirectionOutbound CallDirection = "outbound"
)

type CallType string

const (
	CallTypeNormal           CallType = "NORMAL"
	CallTypeConsult           CallType = "CONSULT"
	CallTypeTransfer          CallType = "TRANSFER"
	CallTypeMonitor           CallType = "MONITOR"
	CallTypeWhisper           CallType = "WHISPER"
	CallTypeBarge             CallType = "BARGE"
	CallTypeCoach             CallType = "COACH"
	CallTypeIntercept         CallType = "INTERCEPT"
	CallTypeConference        CallType = "CONFERENCE"
	CallTypeInternal          CallType = "INTERNAL"
	CallTypeDoubleCall        CallType = "DOUBLE_CALL"
	CallTypeCallback          CallType = "CALLBACK"
	CallTypePreview           CallType = "PREVIEW"
	CallTypeProgressive       CallType = "PROGRESSIVE"
	CallTypePower             CallType = "POWER"
	CallTypePredictive        CallType = "PREDICTIVE"
)

type CallStatus string

const (
	CallStatusIVR       CallStatus = "ivr"
	CallStatusQueue     CallStatus = "queue"
	CallStatusRinging   CallStatus = "ringing"
	CallStatusActive    CallStatus = "active"
	CallStatusHeld      CallStatus = "held"
	CallStatusCompleted CallStatus = "completed"
	CallStatusAbandoned CallStatus = "abandoned"
	CallStatusFailed    CallStatus = "failed"
)

type HangupReason string

const (
	HangupNormal             HangupReason = "NORMAL"
	HangupBusy               HangupReason = "BUSY"
	HangupNoAnswer           HangupReason = "NO_ANSWER"
	HangupReject             HangupReason = "REJECT"
	HangupAbandon            HangupReason = "ABANDON"
	HangupQueueTimeout       HangupReason = "QUEUE_TIMEOUT"
	HangupQueueOverflow      HangupReason = "QUEUE_OVERFLOW"
	HangupBlacklist          HangupReason = "BLACKLIST"
	HangupOffHours           HangupReason = "OFF_HOURS"
	HangupIVRHangup          HangupReason = "IVR_HANGUP"
	HangupSystemError        HangupReason = "SYSTEM_ERROR"
	HangupTransferFailed     HangupReason = "TRANSFER_FAILED"
	HangupVoicemailCompleted HangupReason = "VOICEMAIL_COMPLETED"
)

type MediaType string

const (
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
)

type Call struct {
	ID                  int64           `db:"id" json:"id"`
	TenantID            int64           `db:"tenant_id" json:"tenant_id"`
	Direction           CallDirection   `db:"direction" json:"direction"`
	CallType            CallType        `db:"call_type" json:"call_type"`
	MediaType           MediaType       `db:"media_type" json:"media_type"`
	Caller              string          `db:"caller" json:"caller"`
	Callee              string          `db:"callee" json:"callee"`
	MaskedCallee        *string         `db:"masked_callee" json:"masked_callee,omitempty"`
	AgentUserID         *int64          `db:"agent_user_id" json:"agent_user_id,omitempty"`
	SkillGroupID        *int64          `db:"skill_group_id" json:"skill_group_id,omitempty"`
	IVRFlowID           *int64          `db:"ivr_flow_id" json:"ivr_flow_id,omitempty"`
	PhoneNumberID       *int64          `db:"phone_number_id" json:"phone_number_id,omitempty"`
	CarrierID           *int64          `db:"carrier_id" json:"carrier_id,omitempty"`
	SIPTrunkID          *int64          `db:"sip_trunk_id" json:"sip_trunk_id,omitempty"`
	ParentCallID        *int64          `db:"parent_call_id" json:"parent_call_id,omitempty"`
	CampaignCaseID      *int64          `db:"campaign_case_id" json:"campaign_case_id,omitempty"`
	Status              CallStatus      `db:"status" json:"status"`
	HangupReason        *HangupReason   `db:"hangup_reason" json:"hangup_reason,omitempty"`
	DispositionCode     *string         `db:"disposition_code" json:"disposition_code,omitempty"`
	HoldCount           int             `db:"hold_count" json:"hold_count"`
	TransferCount       int             `db:"transfer_count" json:"transfer_count"`
	SatisfactionRating  *int            `db:"satisfaction_rating" json:"satisfaction_rating,omitempty"`
	IVRDurationSec      int             `db:"ivr_duration_sec" json:"ivr_duration_sec"`
	RingDurationSec     int             `db:"ring_duration_sec" json:"ring_duration_sec"`
	QueueDurationSec    int             `db:"queue_duration_sec" json:"queue_duration_sec"`
	WaitDurationSec     int             `db:"wait_duration_sec" json:"wait_duration_sec"`
	DurationSec         int             `db:"duration_sec" json:"duration_sec"`
	RecordingURL        *string         `db:"recording_url" json:"recording_url,omitempty"`
	CustomData          json.RawMessage `db:"custom_data" json:"custom_data,omitempty"`
	StartedAt           time.Time       `db:"started_at" json:"started_at"`
	AnsweredAt          *time.Time      `db:"answered_at" json:"answered_at,omitempty"`
	EndedAt             *time.Time      `db:"ended_at" json:"ended_at,omitempty"`
}

type CallEvent struct {
	ID        int64     `db:"id" json:"id"`
	CallID    int64     `db:"call_id" json:"call_id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Event     string    `db:"event" json:"event"`
	Detail    string    `db:"detail" json:"detail"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type IVRTracking struct {
	ID        int64     `db:"id" json:"id"`
	CallID    int64     `db:"call_id" json:"call_id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	IVRFlowID int64     `db:"ivr_flow_id" json:"ivr_flow_id"`
	NodeID    string    `db:"node_id" json:"node_id"`
	NodeType  string    `db:"node_type" json:"node_type"`
	NodeName  string    `db:"node_name" json:"node_name"`
	Variables string    `db:"variables" json:"variables"`
	ExitName  string    `db:"exit_name" json:"exit_name"`
	StatusCode int      `db:"status_code" json:"status_code"`
	EnteredAt time.Time `db:"entered_at" json:"entered_at"`
	ExitedAt  *time.Time `db:"exited_at" json:"exited_at,omitempty"`
}

type Recording struct {
	ID          int64     `db:"id" json:"id"`
	TenantID    int64     `db:"tenant_id" json:"tenant_id"`
	CallID      int64     `db:"call_id" json:"call_id"`
	AgentUserID *int64    `db:"agent_user_id" json:"agent_user_id,omitempty"`
	FileName    string    `db:"file_name" json:"file_name"`
	FilePath    string    `db:"file_path" json:"file_path"`
	FileSize    int64     `db:"file_size" json:"file_size"`
	DurationSec int      `db:"duration_sec" json:"duration_sec"`
	MimeType    string    `db:"mime_type" json:"mime_type"`
	StorageTier string   `db:"storage_tier" json:"storage_tier"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type QueueSnapshot struct {
	ID            int64     `db:"id" json:"id"`
	TenantID      int64     `db:"tenant_id" json:"tenant_id"`
	SkillGroupID  int64     `db:"skill_group_id" json:"skill_group_id"`
	WaitingCount  int       `db:"waiting_count" json:"waiting_count"`
	AvailableAgents int    `db:"available_agents" json:"available_agents"`
	AvgWaitSec    int       `db:"avg_wait_sec" json:"avg_wait_sec"`
	MaxWaitSec    int       `db:"max_wait_sec" json:"max_wait_sec"`
	SnapshotAt    time.Time `db:"snapshot_at" json:"snapshot_at"`
}

type Voicemail struct {
	ID          int64     `db:"id" json:"id"`
	TenantID    int64     `db:"tenant_id" json:"tenant_id"`
	CallID      *int64    `db:"call_id" json:"call_id,omitempty"`
	Caller      string    `db:"caller" json:"caller"`
	AgentUserID *int64    `db:"agent_user_id" json:"agent_user_id,omitempty"`
	SkillGroupID *int64   `db:"skill_group_id" json:"skill_group_id,omitempty"`
	FilePath    string    `db:"file_path" json:"file_path"`
	DurationSec int      `db:"duration_sec" json:"duration_sec"`
	IsRead      bool      `db:"is_read" json:"is_read"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type CallbackRequest struct {
	ID            int64     `db:"id" json:"id"`
	TenantID      int64     `db:"tenant_id" json:"tenant_id"`
	CallID        int64     `db:"call_id" json:"call_id"`
	SkillGroupID  int64     `db:"skill_group_id" json:"skill_group_id"`
	Caller        string    `db:"caller" json:"caller"`
	Status        string    `db:"status" json:"status"`
	ScheduledAt   *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	AttemptCount  int       `db:"attempt_count" json:"attempt_count"`
	LastAttemptAt *time.Time `db:"last_attempt_at" json:"last_attempt_at,omitempty"`
	CompletedAt   *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}
