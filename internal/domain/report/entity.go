package report

import (
	"fmt"
	"time"
)

// DashboardOverview holds real-time aggregated metrics for a tenant.
type DashboardOverview struct {
	TenantID          int64   `json:"tenant_id"`
	TotalCallsToday   int     `json:"total_calls_today"`
	InboundCalls      int     `json:"inbound_calls"`
	OutboundCalls     int     `json:"outbound_calls"`
	ActiveCalls       int     `json:"active_calls"`
	QueuedCalls       int     `json:"queued_calls"`
	AbandonedCalls    int     `json:"abandoned_calls"`
	AnsweredCalls     int     `json:"answered_calls"`
	ServiceLevel20s   float64 `json:"service_level_20s"`
	AvgWaitSec        float64 `json:"avg_wait_sec"`
	AvgTalkSec        float64 `json:"avg_talk_sec"`
	AgentsOnline      int     `json:"agents_online"`
	AgentsIdle        int     `json:"agents_idle"`
	AgentsTalking     int     `json:"agents_talking"`
	AgentsACW         int     `json:"agents_acw"`
	AgentsBreak       int     `json:"agents_break"`
	AgentsDialing     int     `json:"agents_dialing"`
	LongestWaitSec    int     `json:"longest_wait_sec"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// CallFunnel represents the call flow breakdown for the funnel chart.
type CallFunnel struct {
	TotalInbound       int `json:"total_inbound"`
	IVRHandled         int `json:"ivr_handled"`
	RobotHandled       int `json:"robot_handled"`
	TransferToHuman    int `json:"transfer_to_human"`
	FullService        int `json:"full_service"`
	HalfService        int `json:"half_service"`
	DirectTransfer     int `json:"direct_transfer"`
	ActualAnswered     int `json:"actual_answered"`
	Abandoned          int `json:"abandoned"`
}

// CallTrend represents call volume over time intervals.
type CallTrend struct {
	Time      time.Time `json:"time"`
	Inbound   int       `json:"inbound"`
	Outbound  int       `json:"outbound"`
	Answered  int       `json:"answered"`
	Abandoned int       `json:"abandoned"`
}

// AgentStatusSummary for dashboard agent status list.
type AgentStatusSummary struct {
	AgentID     int64  `json:"agent_id" db:"agent_id"`
	AgentName   string `json:"agent_name" db:"agent_name"`
	Status      string `json:"status" db:"status"`
	SubState    string `json:"sub_state" db:"sub_state"`
	WorkMode    string `json:"work_mode" db:"work_mode"`
	DurationSec int    `json:"duration_sec" db:"duration_sec"`
}

// AgentReport represents aggregated metrics for a single agent over a time range.
type AgentReport struct {
	AgentID              int64   `json:"agent_id" db:"agent_id"`
	AgentName            string  `json:"agent_name" db:"agent_name"`
	TotalCalls           int     `json:"total_calls" db:"total_calls"`
	InboundCalls         int     `json:"inbound_calls" db:"inbound_calls"`
	OutboundCalls        int     `json:"outbound_calls" db:"outbound_calls"`
	AnsweredCalls        int     `json:"answered_calls" db:"answered_calls"`
	MissedCalls          int     `json:"missed_calls" db:"missed_calls"`
	TransferredCalls     int     `json:"transferred_calls" db:"transferred_calls"`
	HeldCalls            int     `json:"held_calls" db:"held_calls"`
	AvgTalkDurationSec   float64 `json:"avg_talk_duration_sec" db:"avg_talk_duration_sec"`
	TotalTalkDurationSec int     `json:"total_talk_duration_sec" db:"total_talk_duration_sec"`
	AvgHoldDurationSec   float64 `json:"avg_hold_duration_sec" db:"avg_hold_duration_sec"`
	AvgACWDurationSec    float64 `json:"avg_acw_duration_sec" db:"avg_acw_duration_sec"`
	TotalACWDurationSec  int     `json:"total_acw_duration_sec" db:"total_acw_duration_sec"`
	AvgRingDurationSec   float64 `json:"avg_ring_duration_sec" db:"avg_ring_duration_sec"`
	AvgWaitDurationSec   float64 `json:"avg_wait_duration_sec" db:"avg_wait_duration_sec"`
	FirstCallResolution  float64 `json:"first_call_resolution" db:"first_call_resolution"`
	ServiceLevel20s      float64 `json:"service_level_20s" db:"service_level_20s"`
	AnswerRate           float64 `json:"answer_rate" db:"answer_rate"`
	OnlineTimeSec        int     `json:"online_time_sec" db:"online_time_sec"`
	IdleTimeSec          int     `json:"idle_time_sec" db:"idle_time_sec"`
	TalkTimeSec          int     `json:"talk_time_sec" db:"talk_time_sec"`
	ACWTimeSec           int     `json:"acw_time_sec" db:"acw_time_sec"`
	BreakTimeSec         int     `json:"break_time_sec" db:"break_time_sec"`
	DialingTimeSec       int     `json:"dialing_time_sec" db:"dialing_time_sec"`
	Utilization          float64 `json:"utilization" db:"utilization"`
	AvgSatisfaction      float64 `json:"avg_satisfaction" db:"avg_satisfaction"`
	SatisfactionCount    int     `json:"satisfaction_count" db:"satisfaction_count"`
	CallbackCount        int     `json:"callback_count" db:"callback_count"`
	InternalCallCount    int     `json:"internal_call_count" db:"internal_call_count"`
	DoubleCallCount      int     `json:"double_call_count" db:"double_call_count"`
}

// GroupAgentReport adds skill group dimension to agent report.
type GroupAgentReport struct {
	SkillGroupID   int64  `json:"skill_group_id" db:"skill_group_id"`
	SkillGroupName string `json:"skill_group_name" db:"skill_group_name"`
	AgentReport
}

// SkillGroupReport represents aggregated metrics for a skill group.
type SkillGroupReport struct {
	SkillGroupID       int64   `json:"skill_group_id" db:"skill_group_id"`
	SkillGroupName     string  `json:"skill_group_name" db:"skill_group_name"`
	TotalCalls         int     `json:"total_calls" db:"total_calls"`
	InboundCalls       int     `json:"inbound_calls" db:"inbound_calls"`
	OutboundCalls      int     `json:"outbound_calls" db:"outbound_calls"`
	AnsweredCalls      int     `json:"answered_calls" db:"answered_calls"`
	AbandonedCalls     int     `json:"abandoned_calls" db:"abandoned_calls"`
	QueueTotal         int     `json:"queue_total" db:"queue_total"`
	QueueAbandoned     int     `json:"queue_abandoned" db:"queue_abandoned"`
	RingAbandoned      int     `json:"ring_abandoned" db:"ring_abandoned"`
	ServiceLevel20s    float64 `json:"service_level_20s" db:"service_level_20s"`
	AvgWaitSec         float64 `json:"avg_wait_sec" db:"avg_wait_sec"`
	AvgTalkSec         float64 `json:"avg_talk_sec" db:"avg_talk_sec"`
	AnswerRate         float64 `json:"answer_rate" db:"answer_rate"`
	AgentCount         int     `json:"agent_count" db:"agent_count"`
}

// Back2BackReport for double-call (B2B) calls.
type Back2BackReport struct {
	TotalCalls     int     `json:"total_calls" db:"total_calls"`
	ConnectedCalls int     `json:"connected_calls" db:"connected_calls"`
	ConnectRate    float64 `json:"connect_rate" db:"connect_rate"`
	AvgDurationSec float64 `json:"avg_duration_sec" db:"avg_duration_sec"`
	TotalDuration  int     `json:"total_duration" db:"total_duration"`
}

// InternalCallReport for internal agent-to-agent calls.
type InternalCallReport struct {
	TotalCalls     int     `json:"total_calls" db:"total_calls"`
	ConnectedCalls int     `json:"connected_calls" db:"connected_calls"`
	ConnectRate    float64 `json:"connect_rate" db:"connect_rate"`
	AvgDurationSec float64 `json:"avg_duration_sec" db:"avg_duration_sec"`
	TotalDuration  int     `json:"total_duration" db:"total_duration"`
}

// AgentStatusLog represents a single status change log entry.
type AgentStatusLog struct {
	ID              int64     `json:"id" db:"id"`
	TenantID        int64     `json:"tenant_id" db:"tenant_id"`
	AgentID         int64     `json:"agent_id" db:"agent_id"`
	AgentName       string    `json:"agent_name" db:"agent_name"`
	Status          string    `json:"status" db:"status"`
	SubState        string    `json:"sub_state" db:"sub_state"`
	WorkMode        string    `json:"work_mode" db:"work_mode"`
	BreakReasonCode string    `json:"break_reason_code" db:"break_reason_code"`
	DurationSec     int       `json:"duration_sec" db:"duration_sec"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// CampaignReport represents aggregated metrics for a campaign.
type CampaignReport struct {
	CampaignID     int64   `json:"campaign_id" db:"campaign_id"`
	CampaignName   string  `json:"campaign_name" db:"campaign_name"`
	DialingMode    string  `json:"dialing_mode" db:"dialing_mode"`
	Status         string  `json:"status" db:"status"`
	TotalCases     int     `json:"total_cases" db:"total_cases"`
	CompletedCases int     `json:"completed_cases" db:"completed_cases"`
	SuccessCases   int     `json:"success_cases" db:"success_cases"`
	FailedCases    int     `json:"failed_cases" db:"failed_cases"`
	SkippedCases   int     `json:"skipped_cases" db:"skipped_cases"`
	CompletionRate float64 `json:"completion_rate" db:"completion_rate"`
	SuccessRate    float64 `json:"success_rate" db:"success_rate"`
	AvgDurationSec float64 `json:"avg_duration_sec" db:"avg_duration_sec"`
}

// MaxReportWindow is the maximum allowed time range for a single report query (31 days).
const MaxReportWindow = 31 * 24 * time.Hour

// ReportFilter holds common filter parameters for report queries.
type ReportFilter struct {
	TenantID     int64
	StartTime    time.Time
	EndTime      time.Time
	AgentID      *int64
	SkillGroupID *int64
	CampaignID   *int64
	Offset       int
	Limit        int
}

// Validate checks that the filter has a bounded time window.
func (f *ReportFilter) Validate() error {
	if f.EndTime.Before(f.StartTime) {
		return fmt.Errorf("report: end_time must be after start_time")
	}
	if f.EndTime.Sub(f.StartTime) > MaxReportWindow {
		return fmt.Errorf("report: time range cannot exceed 31 days")
	}
	return nil
}
