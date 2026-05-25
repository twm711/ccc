package report

import (
	"context"
	"math"
)

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

// WallboardData provides a flattened, big-screen-friendly view of KPIs
// designed for lobby display / wallboard mode (large font, key metrics only).
type WallboardData struct {
	TenantID       int64   `json:"tenant_id"`
	CallsToday     int     `json:"calls_today"`
	ActiveCalls    int     `json:"active_calls"`
	QueuedCalls    int     `json:"queued_calls"`
	AbandonedCalls int     `json:"abandoned_calls"`
	ServiceLevel   float64 `json:"service_level"` // SL20s %
	AvgWaitSec     float64 `json:"avg_wait_sec"`
	LongestWaitSec int     `json:"longest_wait_sec"`
	AbandonRate    float64 `json:"abandon_rate"` // %
	AgentsOnline   int     `json:"agents_online"`
	AgentsIdle     int     `json:"agents_idle"`
	AgentsBusy     int     `json:"agents_busy"` // talking + dialing
}

// GetWallboard returns a wallboard-optimized snapshot from the dashboard overview.
func (s *DashboardService) GetWallboard(ctx context.Context, tenantID int64) (*WallboardData, error) {
	o, err := s.dashboard.GetOverview(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return &WallboardData{TenantID: tenantID}, nil
	}
	var abandonRate float64
	if o.InboundCalls > 0 {
		abandonRate = float64(o.AbandonedCalls) / float64(o.InboundCalls) * 100
	}
	return &WallboardData{
		TenantID:       tenantID,
		CallsToday:     o.TotalCallsToday,
		ActiveCalls:    o.ActiveCalls,
		QueuedCalls:    o.QueuedCalls,
		AbandonedCalls: o.AbandonedCalls,
		ServiceLevel:   o.ServiceLevel20s,
		AvgWaitSec:     o.AvgWaitSec,
		LongestWaitSec: o.LongestWaitSec,
		AbandonRate:    abandonRate,
		AgentsOnline:   o.AgentsOnline,
		AgentsIdle:     o.AgentsIdle,
		AgentsBusy:     o.AgentsTalking + o.AgentsDialing,
	}, nil
}

// ReportService provides report queries.
type ReportService struct {
	agents       AgentReportRepository
	groupAgents  GroupAgentReportRepository
	skillGroups  SkillGroupReportRepository
	b2b          Back2BackReportRepository
	internal     InternalCallReportRepository
	statusLog    AgentStatusLogRepository
	campaigns    CampaignReportRepository
}

func NewReportService(
	agents AgentReportRepository,
	groupAgents GroupAgentReportRepository,
	skillGroups SkillGroupReportRepository,
	b2b Back2BackReportRepository,
	internal InternalCallReportRepository,
	statusLog AgentStatusLogRepository,
	campaigns CampaignReportRepository,
) *ReportService {
	return &ReportService{
		agents: agents, groupAgents: groupAgents, skillGroups: skillGroups,
		b2b: b2b, internal: internal, statusLog: statusLog, campaigns: campaigns,
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

func (s *ReportService) CampaignReport(ctx context.Context, f ReportFilter) ([]*CampaignReport, int64, error) {
	return s.campaigns.Query(ctx, f)
}

// --- WFM: Erlang C traffic forecasting ---

// ErlangCInput contains parameters for Erlang C staffing calculation.
type ErlangCInput struct {
	CallsPerHour   float64 // λ: arrival rate
	AvgHandleSec   float64 // average handle time in seconds
	ServiceLevelPct float64 // target SL% (e.g. 80)
	TargetAnswerSec float64 // target answer time in seconds (e.g. 20)
}

// ErlangCResult contains the staffing recommendation.
type ErlangCResult struct {
	TrafficIntensity float64 // A = λ * AHT / 3600
	MinAgents        int     // minimum agents to handle load
	RecommendedAgents int    // agents needed to meet SL target
	ExpectedSL       float64 // expected service level with recommended agents
}

// ErlangC computes the recommended number of agents using the Erlang C formula.
func ErlangC(in ErlangCInput) ErlangCResult {
	if in.CallsPerHour <= 0 || in.AvgHandleSec <= 0 {
		return ErlangCResult{}
	}
	a := in.CallsPerHour * in.AvgHandleSec / 3600.0 // traffic intensity (Erlangs)
	minN := int(math.Ceil(a))
	if minN < 1 {
		minN = 1
	}

	for n := minN; n <= minN+200; n++ {
		ec := erlangCProb(a, n)
		sl := 1.0 - ec*math.Exp(-float64(n-int(math.Ceil(a)))*in.TargetAnswerSec/in.AvgHandleSec)
		if sl < 0 {
			sl = 0
		}
		if sl*100 >= in.ServiceLevelPct {
			return ErlangCResult{
				TrafficIntensity:  a,
				MinAgents:         minN,
				RecommendedAgents: n,
				ExpectedSL:        math.Round(sl*10000) / 100,
			}
		}
	}
	return ErlangCResult{TrafficIntensity: a, MinAgents: minN, RecommendedAgents: minN + 200}
}

// erlangCProb computes the Erlang C probability (probability of queuing).
func erlangCProb(a float64, n int) float64 {
	if n <= 0 || a <= 0 {
		return 0
	}
	// Compute A^N / N! iteratively to avoid overflow
	sumTerm := 1.0 // k=0 term
	aN_Nfact := 1.0
	for k := 1; k <= n; k++ {
		aN_Nfact *= a / float64(k)
		if k < n {
			sumTerm += aN_Nfact
		}
	}
	last := aN_Nfact * float64(n) / (float64(n) - a)
	if float64(n) <= a {
		return 1.0
	}
	return last / (sumTerm + last)
}
