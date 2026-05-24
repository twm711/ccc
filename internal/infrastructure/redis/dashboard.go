package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/domain/report"
	"github.com/redis/go-redis/v9"
)

// DashboardRepo implements report.DashboardRepository using Redis for real-time metrics.
type DashboardRepo struct {
	client *redis.Client
}

func NewDashboardRepo(client *redis.Client) *DashboardRepo {
	return &DashboardRepo{client: client}
}

func dashboardKey(tenantID int64) string {
	return fmt.Sprintf("dashboard:%d", tenantID)
}

func (r *DashboardRepo) GetOverview(ctx context.Context, tenantID int64) (*report.DashboardOverview, error) {
	key := dashboardKey(tenantID)
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	o := &report.DashboardOverview{
		TenantID:    tenantID,
		GeneratedAt: time.Now(),
	}

	o.TotalCallsToday = atoi(data["total_calls_today"])
	o.InboundCalls = atoi(data["inbound_calls"])
	o.OutboundCalls = atoi(data["outbound_calls"])
	o.ActiveCalls = atoi(data["active_calls"])
	o.QueuedCalls = atoi(data["queued_calls"])
	o.AbandonedCalls = atoi(data["abandoned_calls"])
	o.AnsweredCalls = atoi(data["answered_calls"])
	o.ServiceLevel20s = atof(data["service_level_20s"])
	o.AvgWaitSec = atof(data["avg_wait_sec"])
	o.AvgTalkSec = atof(data["avg_talk_sec"])
	o.AgentsOnline = atoi(data["agents_online"])
	o.AgentsIdle = atoi(data["agents_idle"])
	o.AgentsTalking = atoi(data["agents_talking"])
	o.AgentsACW = atoi(data["agents_acw"])
	o.AgentsBreak = atoi(data["agents_break"])
	o.AgentsDialing = atoi(data["agents_dialing"])
	o.LongestWaitSec = atoi(data["longest_wait_sec"])

	return o, nil
}

// UpdateOverview writes dashboard metrics to Redis HASH.
func (r *DashboardRepo) UpdateOverview(ctx context.Context, o *report.DashboardOverview) error {
	key := dashboardKey(o.TenantID)
	fields := map[string]interface{}{
		"total_calls_today": o.TotalCallsToday,
		"inbound_calls":     o.InboundCalls,
		"outbound_calls":    o.OutboundCalls,
		"active_calls":      o.ActiveCalls,
		"queued_calls":      o.QueuedCalls,
		"abandoned_calls":   o.AbandonedCalls,
		"answered_calls":    o.AnsweredCalls,
		"service_level_20s": o.ServiceLevel20s,
		"avg_wait_sec":      o.AvgWaitSec,
		"avg_talk_sec":      o.AvgTalkSec,
		"agents_online":     o.AgentsOnline,
		"agents_idle":       o.AgentsIdle,
		"agents_talking":    o.AgentsTalking,
		"agents_acw":        o.AgentsACW,
		"agents_break":      o.AgentsBreak,
		"agents_dialing":    o.AgentsDialing,
		"longest_wait_sec":  o.LongestWaitSec,
	}
	return r.client.HSet(ctx, key, fields).Err()
}

func (r *DashboardRepo) GetCallFunnel(ctx context.Context, tenantID int64) (*report.CallFunnel, error) {
	key := fmt.Sprintf("funnel:%d", tenantID)
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return &report.CallFunnel{
		TotalInbound:    atoi(data["total_inbound"]),
		IVRHandled:      atoi(data["ivr_handled"]),
		RobotHandled:    atoi(data["robot_handled"]),
		TransferToHuman: atoi(data["transfer_to_human"]),
		FullService:     atoi(data["full_service"]),
		HalfService:     atoi(data["half_service"]),
		DirectTransfer:  atoi(data["direct_transfer"]),
		ActualAnswered:  atoi(data["actual_answered"]),
		Abandoned:       atoi(data["abandoned"]),
	}, nil
}

func (r *DashboardRepo) GetCallTrend(ctx context.Context, tenantID int64, intervalMin int) ([]*report.CallTrend, error) {
	if intervalMin <= 0 {
		intervalMin = 30
	}
	now := time.Now()
	var trends []*report.CallTrend
	// Read last 24 hours of trend data in intervals from Redis
	for i := 0; i < 24*60/intervalMin; i++ {
		t := now.Add(-time.Duration(i*intervalMin) * time.Minute)
		slotKey := fmt.Sprintf("trend:%d:%s", tenantID, t.Format("2006010215")+fmt.Sprintf("%02d", t.Minute()/intervalMin*intervalMin))
		data, err := r.client.HGetAll(ctx, slotKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		trends = append(trends, &report.CallTrend{
			Time:      t.Truncate(time.Duration(intervalMin) * time.Minute),
			Inbound:   atoi(data["inbound"]),
			Outbound:  atoi(data["outbound"]),
			Answered:  atoi(data["answered"]),
			Abandoned: atoi(data["abandoned"]),
		})
	}
	// Reverse to chronological order
	for i, j := 0, len(trends)-1; i < j; i, j = i+1, j-1 {
		trends[i], trends[j] = trends[j], trends[i]
	}
	return trends, nil
}

func (r *DashboardRepo) GetAgentStatusList(ctx context.Context, tenantID int64) ([]*report.AgentStatusSummary, error) {
	key := fmt.Sprintf("agent_status:%d", tenantID)
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var list []*report.AgentStatusSummary
	for agentKey, statusJSON := range data {
		agentID, _ := strconv.ParseInt(agentKey, 10, 64)
		var entry struct {
			Name        string `json:"name"`
			Status      string `json:"status"`
			SubState    string `json:"sub_state"`
			WorkMode    string `json:"work_mode"`
			DurationSec int    `json:"duration_sec"`
		}
		if err := json.Unmarshal([]byte(statusJSON), &entry); err != nil {
			continue
		}
		list = append(list, &report.AgentStatusSummary{
			AgentID:     agentID,
			AgentName:   entry.Name,
			Status:      entry.Status,
			SubState:    entry.SubState,
			WorkMode:    entry.WorkMode,
			DurationSec: entry.DurationSec,
		})
	}
	return list, nil
}

func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
