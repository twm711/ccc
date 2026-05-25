package report

import "context"

// DashboardRepository provides real-time metrics.
type DashboardRepository interface {
	GetOverview(ctx context.Context, tenantID int64) (*DashboardOverview, error)
	UpdateOverview(ctx context.Context, o *DashboardOverview) error
	GetCallFunnel(ctx context.Context, tenantID int64) (*CallFunnel, error)
	GetCallTrend(ctx context.Context, tenantID int64, intervalMin int) ([]*CallTrend, error)
	GetAgentStatusList(ctx context.Context, tenantID int64) ([]*AgentStatusSummary, error)
}

// AgentReportRepository provides agent-level aggregated reports.
type AgentReportRepository interface {
	Query(ctx context.Context, f ReportFilter) ([]*AgentReport, int64, error)
}

// GroupAgentReportRepository provides skill-group×agent reports.
type GroupAgentReportRepository interface {
	Query(ctx context.Context, f ReportFilter) ([]*GroupAgentReport, int64, error)
}

// SkillGroupReportRepository provides skill group level reports.
type SkillGroupReportRepository interface {
	Query(ctx context.Context, f ReportFilter) ([]*SkillGroupReport, int64, error)
}

// Back2BackReportRepository provides B2B call reports.
type Back2BackReportRepository interface {
	Query(ctx context.Context, f ReportFilter) (*Back2BackReport, error)
}

// InternalCallReportRepository provides internal call reports.
type InternalCallReportRepository interface {
	Query(ctx context.Context, f ReportFilter) (*InternalCallReport, error)
}

// AgentStatusLogRepository provides status log queries.
type AgentStatusLogRepository interface {
	Query(ctx context.Context, f ReportFilter, breakReasonCode string) ([]*AgentStatusLog, int64, error)
}

// CampaignReportRepository provides campaign-level aggregated reports.
type CampaignReportRepository interface {
	Query(ctx context.Context, f ReportFilter) ([]*CampaignReport, int64, error)
}
