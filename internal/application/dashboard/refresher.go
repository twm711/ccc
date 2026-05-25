package dashboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/domain/report"
	"github.com/rs/zerolog"
)

// refreshConcurrency caps parallel tenant refreshes per tick.
// Tuned for hundreds of tenants per 10s tick without overwhelming MySQL/Redis.
const refreshConcurrency = 8

// AlarmLevel indicates the severity of an SLA alarm.
type AlarmLevel string

const (
	AlarmWarning   AlarmLevel = "warning"
	AlarmCritical  AlarmLevel = "critical"
	AlarmEmergency AlarmLevel = "emergency"
)

// AlarmNotifier receives SLA alarm notifications.
type AlarmNotifier interface {
	NotifyAlarm(ctx context.Context, tenantID int64, level AlarmLevel, metric string, value float64, threshold float64)
}

// Refresher periodically aggregates call and agent data into Redis dashboard snapshots.
type Refresher struct {
	callRepo     call.CallRepository
	presenceRepo identity.AgentPresenceRepository
	tenantRepo   identity.TenantRepository
	dashRepo     report.DashboardRepository
	alarmNotif   AlarmNotifier
	logger       zerolog.Logger
}

func NewRefresher(
	callRepo call.CallRepository,
	presenceRepo identity.AgentPresenceRepository,
	tenantRepo identity.TenantRepository,
	dashRepo report.DashboardRepository,
	logger zerolog.Logger,
) *Refresher {
	return &Refresher{
		callRepo:     callRepo,
		presenceRepo: presenceRepo,
		tenantRepo:   tenantRepo,
		dashRepo:     dashRepo,
		logger:       logger,
	}
}

// SetAlarmNotifier wires an alarm notifier for SLA threshold monitoring.
func (r *Refresher) SetAlarmNotifier(n AlarmNotifier) { r.alarmNotif = n }

// Start runs the refresh loop every 10 seconds until ctx is cancelled.
func (r *Refresher) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refreshAll(ctx)
		}
	}
}

func (r *Refresher) refreshAll(ctx context.Context) {
	tenants, _, err := r.tenantRepo.List(ctx, 0, 1000)
	if err != nil {
		r.logger.Error().Err(err).Msg("dashboard refresher: list tenants failed")
		return
	}

	sem := make(chan struct{}, refreshConcurrency)
	var wg sync.WaitGroup
	for _, t := range tenants {
		wg.Add(1)
		sem <- struct{}{}
		go func(tenantID int64) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := r.refreshTenant(ctx, tenantID); err != nil {
				r.logger.Error().Err(err).Int64("tenant_id", tenantID).Msg("dashboard refresher: refresh failed")
			}
		}(t.ID)
	}
	wg.Wait()
}

func (r *Refresher) refreshTenant(ctx context.Context, tenantID int64) error {
	total, inbound, outbound, answered, abandoned, active, queued, err := r.callRepo.CountTodayByTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	presences, err := r.presenceRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	var online, idle, talking, acw, brk, dialing int
	for _, p := range presences {
		switch p.Status {
		case identity.PresenceOnline, identity.PresenceIdle:
			online++
			idle++
		case identity.PresenceTalking:
			online++
			talking++
		case identity.PresenceACW:
			online++
			acw++
		case identity.PresenceBreak:
			online++
			brk++
		case identity.PresenceDialing:
			online++
			dialing++
		}
	}

	avgWait, within20s, offered, longestWait, _ := r.callRepo.SLATodayByTenant(ctx, tenantID)
	sl20s := report.CalculateServiceLevel20s(within20s, offered)

	overview := &report.DashboardOverview{
		TenantID:        tenantID,
		TotalCallsToday: total,
		InboundCalls:    inbound,
		OutboundCalls:   outbound,
		ActiveCalls:     active,
		QueuedCalls:     queued,
		AbandonedCalls:  abandoned,
		AnsweredCalls:   answered,
		ServiceLevel20s: sl20s,
		AvgWaitSec:      avgWait,
		LongestWaitSec:  longestWait,
		AgentsOnline:    online,
		AgentsIdle:      idle,
		AgentsTalking:   talking,
		AgentsACW:       acw,
		AgentsBreak:     brk,
		AgentsDialing:   dialing,
		GeneratedAt:     time.Now(),
	}

	if err := r.dashRepo.UpdateOverview(ctx, overview); err != nil {
		return err
	}

	r.evaluateAlarms(ctx, overview)
	return nil
}

// evaluateAlarms checks SLA thresholds and fires alarm notifications.
func (r *Refresher) evaluateAlarms(ctx context.Context, o *report.DashboardOverview) {
	if r.alarmNotif == nil {
		return
	}

	// Service Level alarms: <40% emergency, <60% critical, <80% warning.
	if o.ServiceLevel20s > 0 || o.InboundCalls > 0 {
		switch {
		case o.ServiceLevel20s < 40:
			r.alarmNotif.NotifyAlarm(ctx, o.TenantID, AlarmEmergency, "service_level_20s", o.ServiceLevel20s, 40)
		case o.ServiceLevel20s < 60:
			r.alarmNotif.NotifyAlarm(ctx, o.TenantID, AlarmCritical, "service_level_20s", o.ServiceLevel20s, 60)
		case o.ServiceLevel20s < 80:
			r.alarmNotif.NotifyAlarm(ctx, o.TenantID, AlarmWarning, "service_level_20s", o.ServiceLevel20s, 80)
		}
	}

	// Average wait time alarms: >60s critical, >30s warning.
	switch {
	case o.AvgWaitSec > 60:
		r.alarmNotif.NotifyAlarm(ctx, o.TenantID, AlarmCritical, "avg_wait_sec", o.AvgWaitSec, 60)
	case o.AvgWaitSec > 30:
		r.alarmNotif.NotifyAlarm(ctx, o.TenantID, AlarmWarning, "avg_wait_sec", o.AvgWaitSec, 30)
	}

	// Longest wait alarms: >120s critical.
	if o.LongestWaitSec > 120 {
		r.alarmNotif.NotifyAlarm(ctx, o.TenantID, AlarmCritical, "longest_wait_sec", float64(o.LongestWaitSec), 120)
	}
}

// logAlarmNotifier is a default implementation that logs alarms.
type logAlarmNotifier struct {
	logger zerolog.Logger
}

// NewLogAlarmNotifier returns an AlarmNotifier that writes to the logger.
func NewLogAlarmNotifier(logger zerolog.Logger) AlarmNotifier {
	return &logAlarmNotifier{logger: logger}
}

func (n *logAlarmNotifier) NotifyAlarm(_ context.Context, tenantID int64, level AlarmLevel, metric string, value float64, threshold float64) {
	n.logger.Warn().
		Int64("tenant_id", tenantID).
		Str("level", string(level)).
		Str("metric", metric).
		Str("value", fmt.Sprintf("%.2f", value)).
		Str("threshold", fmt.Sprintf("%.0f", threshold)).
		Msg("SLA alarm triggered")
}
