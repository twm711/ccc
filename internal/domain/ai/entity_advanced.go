package ai

import "time"

// ─── 1. Communication Agent (通信智能体) ───

type AgentMode string

const (
	AgentModeInbound  AgentMode = "inbound"
	AgentModeOutbound AgentMode = "outbound"
)

type AgentSessionStatus string

const (
	AgentSessionActive    AgentSessionStatus = "active"
	AgentSessionCompleted AgentSessionStatus = "completed"
	AgentSessionTransfer  AgentSessionStatus = "transferred"
	AgentSessionFailed    AgentSessionStatus = "failed"
)

// CommAgent is an LLM-powered communication agent that autonomously handles calls.
type CommAgent struct {
	ID              int64     `db:"id" json:"id"`
	TenantID        int64     `db:"tenant_id" json:"tenant_id"`
	DigitalEmployeeID int64   `db:"digital_employee_id" json:"digital_employee_id"`
	Name            string    `db:"name" json:"name"`
	Mode            AgentMode `db:"mode" json:"mode"`
	SystemPrompt    string    `db:"system_prompt" json:"system_prompt"`
	MaxTurns        int       `db:"max_turns" json:"max_turns"`
	TransferSkillGroupID *int64 `db:"transfer_skill_group_id" json:"transfer_skill_group_id,omitempty"`
	LLMModelID      *int64    `db:"llm_model_id" json:"llm_model_id,omitempty"`
	IsActive        bool      `db:"is_active" json:"is_active"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// CommAgentSession tracks a single autonomous conversation session.
type CommAgentSession struct {
	ID           int64              `db:"id" json:"id"`
	TenantID     int64              `db:"tenant_id" json:"tenant_id"`
	CommAgentID  int64              `db:"comm_agent_id" json:"comm_agent_id"`
	CallID       int64              `db:"call_id" json:"call_id"`
	Status       AgentSessionStatus `db:"status" json:"status"`
	TurnCount    int                `db:"turn_count" json:"turn_count"`
	Transcript   string             `db:"transcript" json:"transcript"`
	Summary      string             `db:"summary" json:"summary"`
	TransferredTo *int64            `db:"transferred_to" json:"transferred_to,omitempty"`
	StartedAt    time.Time          `db:"started_at" json:"started_at"`
	EndedAt      *time.Time         `db:"ended_at" json:"ended_at,omitempty"`
}

// ─── 2. Voice Cloning (声纹复刻) ───

type VoiceProfileStatus string

const (
	VoiceProfilePending   VoiceProfileStatus = "pending"
	VoiceProfileTraining  VoiceProfileStatus = "training"
	VoiceProfileReady     VoiceProfileStatus = "ready"
	VoiceProfileFailed    VoiceProfileStatus = "failed"
)

// VoiceProfile represents a custom TTS voice clone.
type VoiceProfile struct {
	ID              int64              `db:"id" json:"id"`
	TenantID        int64              `db:"tenant_id" json:"tenant_id"`
	Name            string             `db:"name" json:"name"`
	SampleAudioURL  string             `db:"sample_audio_url" json:"sample_audio_url"`
	ProviderJobID   string             `db:"provider_job_id" json:"provider_job_id,omitempty"`
	ProviderVoiceID string             `db:"provider_voice_id" json:"provider_voice_id"`
	Status          VoiceProfileStatus `db:"status" json:"status"`
	Language        string             `db:"language" json:"language"`
	CreatedAt       time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `db:"updated_at" json:"updated_at"`
}

// ─── 3. Conversation Analytics (智能对话分析) ───

type AnalysisType string

const (
	AnalysisTypeIntent    AnalysisType = "intent_mining"
	AnalysisTypeSOP       AnalysisType = "sop_discovery"
	AnalysisTypeSalesTalk AnalysisType = "sales_script"
	AnalysisTypeTopic     AnalysisType = "topic_cluster"
)

type AnalysisTaskStatus string

const (
	AnalysisTaskPending    AnalysisTaskStatus = "pending"
	AnalysisTaskRunning    AnalysisTaskStatus = "running"
	AnalysisTaskCompleted  AnalysisTaskStatus = "completed"
	AnalysisTaskFailed     AnalysisTaskStatus = "failed"
)

// ConversationAnalysisTask is a batch analysis job over historical transcripts.
type ConversationAnalysisTask struct {
	ID           int64              `db:"id" json:"id"`
	TenantID     int64              `db:"tenant_id" json:"tenant_id"`
	Name         string             `db:"name" json:"name"`
	Type         AnalysisType       `db:"type" json:"type"`
	DateFrom     string             `db:"date_from" json:"date_from"`
	DateTo       string             `db:"date_to" json:"date_to"`
	TotalCalls   int                `db:"total_calls" json:"total_calls"`
	ProcessedCalls int              `db:"processed_calls" json:"processed_calls"`
	Status       AnalysisTaskStatus `db:"status" json:"status"`
	ResultJSON   string             `db:"result_json" json:"result_json"` // JSON array of findings
	CreatedAt    time.Time          `db:"created_at" json:"created_at"`
	CompletedAt  *time.Time         `db:"completed_at" json:"completed_at,omitempty"`
}

// ─── 4. Training System (智能培训) ───

type CourseStatus string

const (
	CourseStatusDraft     CourseStatus = "draft"
	CourseStatusPublished CourseStatus = "published"
	CourseStatusArchived  CourseStatus = "archived"
)

type ExamStatus string

const (
	ExamStatusPending  ExamStatus = "pending"
	ExamStatusPassed   ExamStatus = "passed"
	ExamStatusFailed   ExamStatus = "failed"
)

// TrainingCourse is a knowledge course for agent training.
type TrainingCourse struct {
	ID          int64        `db:"id" json:"id"`
	TenantID    int64        `db:"tenant_id" json:"tenant_id"`
	Title       string       `db:"title" json:"title"`
	Description string       `db:"description" json:"description"`
	ContentJSON string       `db:"content_json" json:"content_json"` // structured course content
	PassScore   int          `db:"pass_score" json:"pass_score"`
	Status      CourseStatus `db:"status" json:"status"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updated_at"`
}

// TrainingExam records an agent's exam attempt.
type TrainingExam struct {
	ID         int64      `db:"id" json:"id"`
	TenantID   int64      `db:"tenant_id" json:"tenant_id"`
	CourseID   int64      `db:"course_id" json:"course_id"`
	AgentID    int64      `db:"agent_id" json:"agent_id"`
	Score      int        `db:"score" json:"score"`
	MaxScore   int        `db:"max_score" json:"max_score"`
	Status     ExamStatus `db:"status" json:"status"`
	AnswersJSON string    `db:"answers_json" json:"answers_json"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// SimulatedCall is an AI-powered practice call for agent training.
type SimulatedCall struct {
	ID          int64     `db:"id" json:"id"`
	TenantID    int64     `db:"tenant_id" json:"tenant_id"`
	AgentID     int64     `db:"agent_id" json:"agent_id"`
	ScenarioID  int64     `db:"scenario_id" json:"scenario_id"` // links to DEScene
	Transcript  string    `db:"transcript" json:"transcript"`
	AIFeedback  string    `db:"ai_feedback" json:"ai_feedback"`
	Score       int       `db:"score" json:"score"`
	DurationSec int       `db:"duration_sec" json:"duration_sec"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// ─── 5. Ring Analysis (彩铃识别) ───

type RingDetectionResult string

const (
	RingDetectionHuman     RingDetectionResult = "human"
	RingDetectionVoicemail RingDetectionResult = "voicemail"
	RingDetectionBusy      RingDetectionResult = "busy"
	RingDetectionNoAnswer  RingDetectionResult = "no_answer"
	RingDetectionFax       RingDetectionResult = "fax"
	RingDetectionUnknown   RingDetectionResult = "unknown"
)

// RingAnalysisConfig stores per-tenant ring analysis settings.
type RingAnalysisConfig struct {
	ID            int64     `db:"id" json:"id"`
	TenantID      int64     `db:"tenant_id" json:"tenant_id"`
	Enabled       bool      `db:"enabled" json:"enabled"`
	DetectionTimeoutMs int  `db:"detection_timeout_ms" json:"detection_timeout_ms"`
	HangUpOnVoicemail bool  `db:"hangup_on_voicemail" json:"hangup_on_voicemail"`
	HangUpOnFax       bool  `db:"hangup_on_fax" json:"hangup_on_fax"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// RingAnalysisLog records a ring analysis event for an outbound call.
type RingAnalysisLog struct {
	ID         int64               `db:"id" json:"id"`
	TenantID   int64               `db:"tenant_id" json:"tenant_id"`
	CallID     int64               `db:"call_id" json:"call_id"`
	Result     RingDetectionResult `db:"result" json:"result"`
	Confidence float64             `db:"confidence" json:"confidence"`
	DurationMs int                 `db:"duration_ms" json:"duration_ms"`
	CreatedAt  time.Time           `db:"created_at" json:"created_at"`
}

// ─── 6. Full-Duplex Interaction (全双工交互) ───

// FullDuplexConfig stores per-tenant full-duplex interaction settings.
type FullDuplexConfig struct {
	ID                  int64     `db:"id" json:"id"`
	TenantID            int64     `db:"tenant_id" json:"tenant_id"`
	Enabled             bool      `db:"enabled" json:"enabled"`
	InterruptionEnabled bool      `db:"interruption_enabled" json:"interruption_enabled"`
	SilenceThresholdMs  int       `db:"silence_threshold_ms" json:"silence_threshold_ms"`
	InterruptionSensitivity float64 `db:"interruption_sensitivity" json:"interruption_sensitivity"` // 0.0-1.0
	VoiceContinuity     bool      `db:"voice_continuity" json:"voice_continuity"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}
