package report

import "context"

// DashboardService computes derived dashboard metrics.
type DashboardService struct {
	dashboard  DashboardRepository
}

func NewDashboardService(d DashboardRepository) *DashboardService {
	return &DashboardService{dashboard: d}
}

// CalculateServiceLevel20s computes the percentage of calls answered within 20 seconds.
func CalculateServiceLevel20s(answeredWithin20s, totalOffered int) float64 {
	if totalOffered == 0 {
		return 0
	}
	return float64(answeredWithin20s) / float64(totalOffered) * 100
}

// CalculateAgentUtilization computes the percentage of time an agent is productive.
func CalculateAgentUtilization(talkSec, acwSec, dialingSec, onlineSec int) float64 {
	if onlineSec == 0 {
		return 0
	}
	productive := talkSec + acwSec + dialingSec
	return float64(productive) / float64(onlineSec) * 100
}

// CalculateCallFunnelRatios computes conversion ratios for the call funnel.
func CalculateCallFunnelRatios(funnel *CallFunnel) (ivrRate, answerRate, abandonRate float64) {
	if funnel.TotalInbound == 0 {
		return 0, 0, 0
	}
	ivrRate = float64(funnel.IVRHandled+funnel.RobotHandled) / float64(funnel.TotalInbound) * 100
	answerRate = float64(funnel.ActualAnswered) / float64(funnel.TotalInbound) * 100
	abandonRate = float64(funnel.Abandoned) / float64(funnel.TotalInbound) * 100
	return
}

func (s *DashboardService) GetOverview(ctx context.Context, tenantID int64) (*DashboardOverview, error) {
	return s.dashboard.GetOverview(ctx, tenantID)
}

func (s *DashboardService) GetCallFunnel(ctx context.Context, tenantID int64) (*CallFunnel, error) {
	return s.dashboard.GetCallFunnel(ctx, tenantID)
}

func (s *DashboardService) GetCallTrend(ctx context.Context, tenantID int64, intervalMin int) ([]*CallTrend, error) {
	return s.dashboard.GetCallTrend(ctx, tenantID, intervalMin)
}

func (s *DashboardService) GetAgentStatusList(ctx context.Context, tenantID int64) ([]*AgentStatusSummary, error) {
	return s.dashboard.GetAgentStatusList(ctx, tenantID)
}

// ReportService provides report queries.
type ReportService struct {
	agents       AgentReportRepository
	groupAgents  GroupAgentReportRepository
	skillGroups  SkillGroupReportRepository
	b2b          Back2BackReportRepository
	internal     InternalCallReportRepository
	statusLog    AgentStatusLogRepository
}

func NewReportService(
	agents AgentReportRepository,
	groupAgents GroupAgentReportRepository,
	skillGroups SkillGroupReportRepository,
	b2b Back2BackReportRepository,
	internal InternalCallReportRepository,
	statusLog AgentStatusLogRepository,
) *ReportService {
	return &ReportService{
		agents: agents, groupAgents: groupAgents, skillGroups: skillGroups,
		b2b: b2b, internal: internal, statusLog: statusLog,
	}
}

func (s *ReportService) AgentReport(ctx context.Context, f ReportFilter) ([]*AgentReport, int64, error) {
	return s.agents.Query(ctx, f)
}

func (s *ReportService) GroupAgentReport(ctx context.Context, f ReportFilter) ([]*GroupAgentReport, int64, error) {
	return s.groupAgents.Query(ctx, f)
}

func (s *ReportService) SkillGroupReport(ctx context.Context, f ReportFilter) ([]*SkillGroupReport, int64, error) {
	return s.skillGroups.Query(ctx, f)
}

func (s *ReportService) Back2BackReport(ctx context.Context, f ReportFilter) (*Back2BackReport, error) {
	return s.b2b.Query(ctx, f)
}

func (s *ReportService) InternalCallReport(ctx context.Context, f ReportFilter) (*InternalCallReport, error) {
	return s.internal.Query(ctx, f)
}

func (s *ReportService) AgentStatusLogQuery(ctx context.Context, f ReportFilter, breakReasonCode string) ([]*AgentStatusLog, int64, error) {
	return s.statusLog.Query(ctx, f, breakReasonCode)
}
