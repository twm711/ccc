package dashboard

import (
	"context"
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

// Refresher periodically aggregates call and agent data into Redis dashboard snapshots.
type Refresher struct {
	callRepo    call.CallRepository
	presenceRepo identity.AgentPresenceRepository
	tenantRepo  identity.TenantRepository
	dashRepo    report.DashboardRepository
	logger      zerolog.Logger
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

	overview := &report.DashboardOverview{
		TenantID:        tenantID,
		TotalCallsToday: total,
		InboundCalls:    inbound,
		OutboundCalls:   outbound,
		ActiveCalls:     active,
		QueuedCalls:     queued,
		AbandonedCalls:  abandoned,
		AnsweredCalls:   answered,
		AgentsOnline:    online,
		AgentsIdle:      idle,
		AgentsTalking:   talking,
		AgentsACW:       acw,
		AgentsBreak:     brk,
		AgentsDialing:   dialing,
		GeneratedAt:     time.Now(),
	}

	return r.dashRepo.UpdateOverview(ctx, overview)
}
