package redis

import (
	"context"
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

func (r *DashboardRepo) GetCallFunnel(_ context.Context, _ int64) (*report.CallFunnel, error) {
	// In production, this would aggregate from Redis or MySQL.
	return &report.CallFunnel{}, nil
}

func (r *DashboardRepo) GetCallTrend(_ context.Context, _ int64, _ int) ([]*report.CallTrend, error) {
	return nil, nil
}

func (r *DashboardRepo) GetAgentStatusList(_ context.Context, _ int64) ([]*report.AgentStatusSummary, error) {
	return nil, nil
}

func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
