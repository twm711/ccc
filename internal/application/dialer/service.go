package dialer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/divord97/ccc/internal/infrastructure/esl"
	"github.com/rs/zerolog"
)

// DialFunc is the function signature for placing an outbound campaign call.
type DialFunc func(ctx context.Context, tenantID int64, callee string, campaignID, caseID int64) error

// Service orchestrates outbound dialing for campaigns across 4 modes.
type Service struct {
	campaignSvc *campaign.CampaignService
	esl         *esl.Client
	logger      zerolog.Logger
	dialFn      DialFunc

	mu       sync.Mutex
	active   map[int64]*dialerState // campaignID → state
}

type dialerState struct {
	campaignID    int64
	mode          campaign.DialingMode
	activeCalls   int
	abandonCount  int
	totalDialed   int
	stopCh        chan struct{}
}

func NewService(campaignSvc *campaign.CampaignService, eslClient *esl.Client, logger zerolog.Logger) *Service {
	s := &Service{
		campaignSvc: campaignSvc,
		esl:         eslClient,
		logger:      logger,
		active:      make(map[int64]*dialerState),
	}
	// default dial function: originate via ESL if available
	s.dialFn = s.eslDial
	return s
}

// SetDialFunc allows injecting a custom dial function (e.g. outbound.Service.Dial).
func (s *Service) SetDialFunc(fn DialFunc) {
	s.dialFn = fn
}

func (s *Service) eslDial(ctx context.Context, tenantID int64, callee string, campaignID, caseID int64) error {
	if s.esl == nil {
		s.logger.Debug().Int64("campaign_id", campaignID).Str("phone", callee).Msg("dialer: ESL not configured, skip")
		return nil
	}
	_, err := s.esl.Originate(ctx, fmt.Sprintf("sofia/gateway/campaign_%d/%s", campaignID, callee), callee, "campaign")
	if err != nil {
		s.logger.Error().Err(err).Int64("case_id", caseID).Str("phone", callee).Msg("dialer: ESL originate failed")
	}
	return err
}

// StartDialing begins the dialing loop for a campaign based on its mode.
func (s *Service) StartDialing(ctx context.Context, campaignID int64) error {
	c, err := s.campaignSvc.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if _, exists := s.active[campaignID]; exists {
		s.mu.Unlock()
		return fmt.Errorf("dialer already active for campaign %d", campaignID)
	}
	state := &dialerState{
		campaignID: campaignID,
		mode:       c.DialingMode,
		stopCh:     make(chan struct{}),
	}
	s.active[campaignID] = state
	s.mu.Unlock()

	go s.dialLoop(campaignID, state)
	s.logger.Info().Int64("campaign_id", campaignID).Str("mode", string(c.DialingMode)).Msg("dialer started")
	return nil
}

// StopDialing stops the dialing loop for a campaign.
func (s *Service) StopDialing(campaignID int64) {
	s.mu.Lock()
	state, exists := s.active[campaignID]
	if exists {
		close(state.stopCh)
		delete(s.active, campaignID)
	}
	s.mu.Unlock()
	if exists {
		s.logger.Info().Int64("campaign_id", campaignID).Msg("dialer stopped")
	}
}

// GetStats returns real-time dialer statistics for a campaign.
func (s *Service) GetStats(campaignID int64) *DialerStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, exists := s.active[campaignID]
	if !exists {
		return &DialerStats{CampaignID: campaignID}
	}
	return &DialerStats{
		CampaignID:   campaignID,
		ActiveCalls:  state.activeCalls,
		TotalDialed:  state.totalDialed,
		AbandonCount: state.abandonCount,
		AbandonRate:  calcAbandonRate(state.abandonCount, state.totalDialed),
		IsRunning:    true,
	}
}

type DialerStats struct {
	CampaignID   int64   `json:"campaign_id"`
	ActiveCalls  int     `json:"active_calls"`
	TotalDialed  int     `json:"total_dialed"`
	AbandonCount int     `json:"abandon_count"`
	AbandonRate  float64 `json:"abandon_rate"`
	IsRunning    bool    `json:"is_running"`
}

func calcAbandonRate(abandoned, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(abandoned) / float64(total) * 100
}

func (s *Service) dialLoop(campaignID int64, state *dialerState) {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-state.stopCh:
			return
		case <-ticker.C:
			s.dialBatch(ctx, campaignID, state)
		}
	}
}

func (s *Service) dialBatch(ctx context.Context, campaignID int64, state *dialerState) {
	c, err := s.campaignSvc.GetByID(ctx, campaignID)
	if err != nil {
		return
	}
	if c.Status != campaign.CampaignStatusRunning {
		s.StopDialing(campaignID)
		return
	}

	if !s.isWithinSchedule(c) {
		return
	}

	switch state.mode {
	case campaign.DialingModePredictive:
		s.dialPredictive(ctx, c, state)
	case campaign.DialingModePreview:
		// Preview mode pushes cases to agents — no automatic dialing
		return
	case campaign.DialingModeProgressive:
		s.dialProgressive(ctx, c, state)
	case campaign.DialingModePower:
		s.dialPower(ctx, c, state)
	}
}

func (s *Service) dialPredictive(ctx context.Context, c *campaign.Campaign, state *dialerState) {
	s.mu.Lock()
	abandonRate := calcAbandonRate(state.abandonCount, state.totalDialed)
	slots := int(c.RatioMultiplier) - state.activeCalls
	if c.MaxAbandonRate > 0 && abandonRate > c.MaxAbandonRate {
		// Proportional throttle: scale slots down based on how far above threshold.
		overshoot := abandonRate / c.MaxAbandonRate
		if overshoot >= 2.0 {
			slots = 0
		} else {
			slots = int(float64(slots) / overshoot)
			if slots < 1 {
				slots = 1
			}
		}
		s.logger.Warn().Int64("campaign_id", c.ID).Float64("abandon_rate", abandonRate).Int("throttled_slots", slots).Msg("predictive: throttled due to high abandon rate")
	}
	s.mu.Unlock()

	for i := 0; i < slots; i++ {
		cs, err := s.campaignSvc.GetNextCase(ctx, c.ID)
		if err != nil || cs == nil {
			return
		}
		s.mu.Lock()
		state.activeCalls++
		state.totalDialed++
		s.mu.Unlock()
		s.logger.Debug().Int64("case_id", cs.ID).Str("phone", cs.PhoneNumber).Msg("predictive: dialing")
		go s.dialCase(ctx, c, cs, state)
	}
}

func (s *Service) dialProgressive(ctx context.Context, c *campaign.Campaign, state *dialerState) {
	s.mu.Lock()
	if state.activeCalls >= 1 {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	cs, err := s.campaignSvc.GetNextCase(ctx, c.ID)
	if err != nil || cs == nil {
		return
	}

	s.mu.Lock()
	state.activeCalls++
	state.totalDialed++
	s.mu.Unlock()
	s.logger.Debug().Int64("case_id", cs.ID).Str("phone", cs.PhoneNumber).Msg("progressive: dialing")
	go s.dialCase(ctx, c, cs, state)
}

func (s *Service) dialPower(ctx context.Context, c *campaign.Campaign, state *dialerState) {
	s.mu.Lock()
	slots := int(c.RatioMultiplier) - state.activeCalls
	if slots <= 0 {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	for i := 0; i < slots; i++ {
		cs, err := s.campaignSvc.GetNextCase(ctx, c.ID)
		if err != nil || cs == nil {
			return
		}
		s.mu.Lock()
		state.activeCalls++
		state.totalDialed++
		s.mu.Unlock()
		s.logger.Debug().Int64("case_id", cs.ID).Str("phone", cs.PhoneNumber).Msg("power: dialing")
		go s.dialCase(ctx, c, cs, state)
	}
}

// dialCase places an actual call for a campaign case and records the result.
func (s *Service) dialCase(ctx context.Context, c *campaign.Campaign, cs *campaign.CampaignCase, state *dialerState) {
	err := s.dialFn(ctx, c.TenantID, cs.PhoneNumber, c.ID, cs.ID)
	connected := err == nil
	disposition := ""
	if connected {
		disposition = "connected"
	} else if err != nil {
		disposition = "dial_failed"
	}
	s.RecordCallResult(ctx, c.ID, cs.ID, connected, 0, disposition)
}

// PreviewAccept is called when an agent accepts a preview case and dials.
func (s *Service) PreviewAccept(ctx context.Context, caseID int64) error {
	cs, err := s.campaignSvc.GetCaseByID(ctx, caseID)
	if err != nil {
		return err
	}

	c, _ := s.campaignSvc.GetByID(ctx, cs.CampaignID)

	s.mu.Lock()
	state, exists := s.active[cs.CampaignID]
	if exists {
		state.activeCalls++
		state.totalDialed++
	}
	s.mu.Unlock()

	s.logger.Info().Int64("case_id", caseID).Int64("campaign_id", cs.CampaignID).Msg("preview: agent accepted case, dialing")

	tenantID := cs.TenantID
	if c != nil {
		tenantID = c.TenantID
	}
	go s.dialCase(ctx, &campaign.Campaign{ID: cs.CampaignID, TenantID: tenantID}, cs, state)
	return nil
}

// PreviewSkip is called when an agent skips a preview case.
func (s *Service) PreviewSkip(ctx context.Context, caseID int64) error {
	_, err := s.campaignSvc.MarkCaseFailed(ctx, caseID)
	return err
}

// RecordCallResult records the outcome of a dialed call.
func (s *Service) RecordCallResult(ctx context.Context, campaignID, caseID int64, connected bool, durationSec int, dispositionCode string) {
	s.mu.Lock()
	state, exists := s.active[campaignID]
	if exists {
		state.activeCalls--
		if state.activeCalls < 0 {
			state.activeCalls = 0
		}
		if !connected {
			state.abandonCount++
		}
	}
	s.mu.Unlock()

	if connected {
		_, _ = s.campaignSvc.MarkCaseCompleted(ctx, caseID, dispositionCode, durationSec)
	} else {
		_, _ = s.campaignSvc.MarkCaseFailed(ctx, caseID)
	}
}

// isWithinSchedule checks if current time falls within campaign's allowed schedule
// (both day-of-week and hour-of-day). Default hours are 09:00-20:00 per MIIT compliance.
func (s *Service) isWithinSchedule(c *campaign.Campaign) bool {
	now := time.Now()
	if c.Timezone != "" {
		if loc, err := time.LoadLocation(c.Timezone); err == nil {
			now = now.In(loc)
		}
	}

	if c.ScheduleDays != "" {
		weekday := strings.ToLower(now.Weekday().String())
		if !strings.Contains(strings.ToLower(c.ScheduleDays), weekday) {
			return false
		}
	}

	startHour := c.ScheduleStartHour
	endHour := c.ScheduleEndHour
	if startHour == 0 && endHour == 0 {
		startHour = 9
		endHour = 20
	}
	hour := now.Hour()
	if hour < startHour || hour >= endHour {
		return false
	}
	return true
}
