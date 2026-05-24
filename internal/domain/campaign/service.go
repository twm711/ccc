package campaign

import (
	"context"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/pkg/snowflake"
)

type CampaignService struct {
	campaigns CampaignRepository
	cases     CampaignCaseRepository
	dncSvc    *integration.DNCService
}

func NewCampaignService(campaigns CampaignRepository, cases CampaignCaseRepository, dncSvc *integration.DNCService) *CampaignService {
	return &CampaignService{campaigns: campaigns, cases: cases, dncSvc: dncSvc}
}

type CreateCampaignInput struct {
	TenantID     int64
	Name         string
	DialingMode  DialingMode
	SkillGroupID int64
	CLIPolicyID  *int64
}

type CaseInput struct {
	PhoneNumber  string
	CustomerName string
	CustomData   string
}

func (s *CampaignService) Create(ctx context.Context, in CreateCampaignInput) (*Campaign, error) {
	switch in.DialingMode {
	case DialingModePredictive, DialingModePreview, DialingModeProgressive, DialingModePower:
	default:
		return nil, ErrInvalidDialingMode
	}

	now := time.Now()
	c := &Campaign{
		ID:           snowflake.NextID(),
		TenantID:     in.TenantID,
		Name:         in.Name,
		DialingMode:  in.DialingMode,
		SkillGroupID: in.SkillGroupID,
		CLIPolicyID:  in.CLIPolicyID,
		Status:       CampaignStatusDraft,
		MaxRetries:   3,
		RetryIntervalSec: 300,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	switch in.DialingMode {
	case DialingModePredictive:
		c.RatioMultiplier = 1.5
		c.MaxAbandonRate = 3.0
		c.ConcurrentLimit = 10
	case DialingModePreview:
		c.PreviewTimeoutSec = 30
		c.RatioMultiplier = 1.0
	case DialingModeProgressive:
		c.RatioMultiplier = 1.0
		c.ConcurrentLimit = 1
	case DialingModePower:
		c.RatioMultiplier = 3.0
		c.ConcurrentLimit = 10
	}

	if err := s.campaigns.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) GetByID(ctx context.Context, id int64) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCampaignNotFound
	}
	return c, nil
}

func (s *CampaignService) Update(ctx context.Context, c *Campaign) error {
	c.UpdatedAt = time.Now()
	return s.campaigns.Update(ctx, c)
}

func (s *CampaignService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*Campaign, int64, error) {
	return s.campaigns.List(ctx, tenantID, offset, limit)
}

func (s *CampaignService) Start(ctx context.Context, id int64) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCampaignNotFound
	}
	if c.Status != CampaignStatusDraft && c.Status != CampaignStatusPaused {
		return nil, ErrCampaignNotDraft
	}

	pending, _, _, err := s.cases.CountByStatus(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	if pending == 0 && c.TotalCases == 0 {
		return nil, ErrCampaignNoCases
	}

	now := time.Now()
	c.Status = CampaignStatusRunning
	c.StartedAt = &now
	c.UpdatedAt = now
	if err := s.campaigns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) Pause(ctx context.Context, id int64) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCampaignNotFound
	}
	if c.Status != CampaignStatusRunning {
		return nil, ErrCampaignNotRunning
	}

	c.Status = CampaignStatusPaused
	c.UpdatedAt = time.Now()
	if err := s.campaigns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) Abort(ctx context.Context, id int64) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCampaignNotFound
	}
	if c.Status != CampaignStatusRunning && c.Status != CampaignStatusPaused {
		return nil, ErrCampaignNotRunning
	}

	now := time.Now()
	c.Status = CampaignStatusAborted
	c.CompletedAt = &now
	c.UpdatedAt = now
	if err := s.campaigns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) ImportCases(ctx context.Context, campaignID int64, inputs []CaseInput) error {
	c, err := s.campaigns.GetByID(ctx, campaignID)
	if err != nil || c == nil {
		return ErrCampaignNotFound
	}

	var blocked map[string]bool
	if s.dncSvc != nil {
		numbers := make([]string, 0, len(inputs))
		for _, in := range inputs {
			if in.PhoneNumber != "" {
				numbers = append(numbers, in.PhoneNumber)
			}
		}
		if len(numbers) > 0 {
			blockedList, _ := s.dncSvc.CheckBatch(ctx, c.TenantID, numbers)
			if len(blockedList) > 0 {
				blocked = make(map[string]bool, len(blockedList))
				for _, n := range blockedList {
					blocked[n] = true
				}
			}
		}
	}

	now := time.Now()
	var validCases []*CampaignCase
	for _, in := range inputs {
		if in.PhoneNumber == "" {
			continue
		}
		if blocked[in.PhoneNumber] {
			continue
		}
		validCases = append(validCases, &CampaignCase{
			ID:           snowflake.NextID(),
			CampaignID:   campaignID,
			TenantID:     c.TenantID,
			PhoneNumber:  in.PhoneNumber,
			CustomerName: in.CustomerName,
			CustomData:   in.CustomData,
			Status:       CaseStatusPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	if len(validCases) > 0 {
		if err := s.cases.BulkCreate(ctx, validCases); err != nil {
			return err
		}
		c.TotalCases += len(validCases)
		c.UpdatedAt = now
		return s.campaigns.Update(ctx, c)
	}
	return nil
}

func (s *CampaignService) ListCases(ctx context.Context, campaignID int64, offset, limit int) ([]*CampaignCase, int64, error) {
	return s.cases.ListByCampaign(ctx, campaignID, offset, limit)
}

func (s *CampaignService) GetCaseByID(ctx context.Context, caseID int64) (*CampaignCase, error) {
	cs, err := s.cases.GetByID(ctx, caseID)
	if err != nil || cs == nil {
		return nil, ErrCaseNotFound
	}
	return cs, nil
}

func (s *CampaignService) GetNextCase(ctx context.Context, campaignID int64) (*CampaignCase, error) {
	return s.cases.GetNextPending(ctx, campaignID)
}

func (s *CampaignService) MarkCaseCompleted(ctx context.Context, caseID int64, dispositionCode string, durationSec int) (*CampaignCase, error) {
	cs, err := s.cases.GetByID(ctx, caseID)
	if err != nil || cs == nil {
		return nil, ErrCaseNotFound
	}

	now := time.Now()
	cs.Status = CaseStatusCompleted
	cs.DispositionCode = dispositionCode
	cs.DurationSec = durationSec
	cs.AttemptCount++
	cs.CompletedAt = &now
	cs.UpdatedAt = now

	if err := s.cases.Update(ctx, cs); err != nil {
		return nil, err
	}

	c, _ := s.campaigns.GetByID(ctx, cs.CampaignID)
	if c != nil {
		c.CompletedCases++
		if durationSec > 0 {
			c.SuccessCases++
		}
		c.UpdatedAt = now
		s.autoComplete(c, now)
		_ = s.campaigns.Update(ctx, c)
	}
	return cs, nil
}

func (s *CampaignService) MarkCaseFailed(ctx context.Context, caseID int64) (*CampaignCase, error) {
	cs, err := s.cases.GetByID(ctx, caseID)
	if err != nil || cs == nil {
		return nil, ErrCaseNotFound
	}

	cs.AttemptCount++
	cs.UpdatedAt = time.Now()

	c, _ := s.campaigns.GetByID(ctx, cs.CampaignID)
	maxRetries := 3
	retryInterval := 300
	if c != nil {
		maxRetries = c.MaxRetries
		retryInterval = c.RetryIntervalSec
	}

	if cs.AttemptCount >= maxRetries {
		cs.Status = CaseStatusFailed
		now := time.Now()
		cs.CompletedAt = &now
		if c != nil {
			c.CompletedCases++
			c.FailedCases++
			c.UpdatedAt = now
			s.autoComplete(c, now)
			_ = s.campaigns.Update(ctx, c)
		}
	} else {
		cs.Status = CaseStatusPending
		next := time.Now().Add(time.Duration(retryInterval) * time.Second)
		cs.NextAttemptAt = &next
	}

	if err := s.cases.Update(ctx, cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (s *CampaignService) autoComplete(c *Campaign, now time.Time) {
	if c.Status == CampaignStatusRunning && c.TotalCases > 0 && c.CompletedCases >= c.TotalCases {
		c.Status = CampaignStatusCompleted
		c.CompletedAt = &now
	}
}
