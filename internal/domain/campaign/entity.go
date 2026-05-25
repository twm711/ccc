package campaign

import (
	"time"
)

type DialingMode string

const (
	DialingModePredictive  DialingMode = "predictive"
	DialingModePreview     DialingMode = "preview"
	DialingModeProgressive DialingMode = "progressive"
	DialingModePower       DialingMode = "power"
)

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusRunning   CampaignStatus = "running"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusAborted   CampaignStatus = "aborted"
)

type CaseStatus string

const (
	CaseStatusPending   CaseStatus = "pending"
	CaseStatusDialing   CaseStatus = "dialing"
	CaseStatusCompleted CaseStatus = "completed"
	CaseStatusFailed    CaseStatus = "failed"
	CaseStatusSkipped   CaseStatus = "skipped"
)

type Campaign struct {
	ID                int64          `db:"id" json:"id"`
	TenantID          int64          `db:"tenant_id" json:"tenant_id"`
	Name              string         `db:"name" json:"name"`
	DialingMode       DialingMode    `db:"dialing_mode" json:"dialing_mode"`
	SkillGroupID      int64          `db:"skill_group_id" json:"skill_group_id"`
	CLIPolicyID       *int64         `db:"cli_policy_id" json:"cli_policy_id,omitempty"`
	Status            CampaignStatus `db:"status" json:"status"`
	RatioMultiplier   float64        `db:"ratio_multiplier" json:"ratio_multiplier"`
	MaxAbandonRate    float64        `db:"max_abandon_rate" json:"max_abandon_rate"`
	PreviewTimeoutSec int            `db:"preview_timeout_sec" json:"preview_timeout_sec"`
	ConcurrentLimit   int            `db:"concurrent_limit" json:"concurrent_limit"`
	MaxRetries        int            `db:"max_retries" json:"max_retries"`
	RetryIntervalSec  int            `db:"retry_interval_sec" json:"retry_interval_sec"`
	Timezone          string         `db:"timezone" json:"timezone"`
	ScheduleDays      string         `db:"schedule_days" json:"schedule_days"`
	ScheduleStartHour int            `db:"schedule_start_hour" json:"schedule_start_hour"`
	ScheduleEndHour   int            `db:"schedule_end_hour" json:"schedule_end_hour"`
	TotalCases        int            `db:"total_cases" json:"total_cases"`
	CompletedCases    int            `db:"completed_cases" json:"completed_cases"`
	SuccessCases      int            `db:"success_cases" json:"success_cases"`
	FailedCases       int            `db:"failed_cases" json:"failed_cases"`
	StartedAt         *time.Time     `db:"started_at" json:"started_at,omitempty"`
	CompletedAt       *time.Time     `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

type CampaignCase struct {
	ID              int64      `db:"id" json:"id"`
	CampaignID      int64      `db:"campaign_id" json:"campaign_id"`
	TenantID        int64      `db:"tenant_id" json:"tenant_id"`
	PhoneNumber     string     `db:"phone_number" json:"phone_number"`
	CustomerName    string     `db:"customer_name" json:"customer_name"`
	CustomData      string     `db:"custom_data" json:"custom_data"`
	Status          CaseStatus `db:"status" json:"status"`
	AttemptCount    int        `db:"attempt_count" json:"attempt_count"`
	AgentUserID     *int64     `db:"agent_user_id" json:"agent_user_id,omitempty"`
	CallID          *int64     `db:"call_id" json:"call_id,omitempty"`
	DurationSec     int        `db:"duration_sec" json:"duration_sec"`
	DispositionCode string     `db:"disposition_code" json:"disposition_code"`
	NextAttemptAt   *time.Time `db:"next_attempt_at" json:"next_attempt_at,omitempty"`
	CompletedAt     *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}
