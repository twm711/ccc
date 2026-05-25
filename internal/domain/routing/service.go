package routing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

type IVRFlowService struct {
	flows    IVRFlowRepository
	versions IVRFlowVersionRepository
}

func NewIVRFlowService(fr IVRFlowRepository, vr IVRFlowVersionRepository) *IVRFlowService {
	return &IVRFlowService{flows: fr, versions: vr}
}

type CreateFlowInput struct {
	TenantID int64
	Code     string
	Name     string
	FlowType FlowType
	Graph    json.RawMessage
}

func (s *IVRFlowService) Create(ctx context.Context, in CreateFlowInput) (*IVRFlow, error) {
	existing, _ := s.flows.GetByCode(ctx, in.TenantID, in.Code, 1)
	if existing != nil {
		return nil, ErrFlowCodeExists
	}

	if _, err := ValidateGraph(in.Graph); err != nil {
		return nil, err
	}

	now := time.Now()
	ft := in.FlowType
	if ft == "" {
		ft = FlowTypeMain
	}
	f := &IVRFlow{
		ID:        snowflake.NextID(),
		TenantID:  in.TenantID,
		Code:      in.Code,
		Name:      in.Name,
		FlowType:  ft,
		Version:   1,
		Graph:     in.Graph,
		Status:    FlowStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.flows.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *IVRFlowService) GetByID(ctx context.Context, id int64) (*IVRFlow, error) {
	return s.flows.GetByID(ctx, id)
}

func (s *IVRFlowService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*IVRFlow, int64, error) {
	return s.flows.List(ctx, tenantID, offset, limit)
}

func (s *IVRFlowService) Publish(ctx context.Context, id int64, userID int64) (*IVRFlow, error) {
	f, err := s.flows.GetByID(ctx, id)
	if err != nil || f == nil {
		return nil, ErrFlowNotFound
	}

	if f.Status != FlowStatusDraft && f.Status != FlowStatusPublishedWithDraft {
		return nil, ErrFlowNotDraft
	}

	if _, err := ValidateGraph(f.Graph); err != nil {
		return nil, err
	}

	now := time.Now()
	f.Status = FlowStatusPublished
	f.PublishedAt = &now
	f.UpdatedAt = now
	f.Version++

	if err := s.flows.Update(ctx, f); err != nil {
		return nil, err
	}

	v := &IVRFlowVersion{
		IVRFlowID:   f.ID,
		TenantID:    f.TenantID,
		Version:     f.Version,
		Graph:       f.Graph,
		PublishedBy: &userID,
		PublishedAt: now,
	}
	_ = s.versions.Create(ctx, v)

	return f, nil
}

const lockTTL = 30 * time.Minute

func (s *IVRFlowService) Lock(ctx context.Context, id int64, userID int64) (*IVRFlow, error) {
	f, err := s.flows.GetByID(ctx, id)
	if err != nil || f == nil {
		return nil, ErrFlowNotFound
	}

	now := time.Now()

	// Allow re-lock if the current lock has expired.
	if f.LockedBy != nil && *f.LockedBy != userID {
		if f.LockExpiresAt == nil || now.Before(*f.LockExpiresAt) {
			return nil, ErrFlowLocked
		}
	}

	expires := now.Add(lockTTL)
	f.LockedBy = &userID
	f.LockedAt = &now
	f.LockExpiresAt = &expires
	f.UpdatedAt = now

	if err := s.flows.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// RefreshLock extends the lock expiry for the current lock owner.
func (s *IVRFlowService) RefreshLock(ctx context.Context, id int64, userID int64) (*IVRFlow, error) {
	f, err := s.flows.GetByID(ctx, id)
	if err != nil || f == nil {
		return nil, ErrFlowNotFound
	}
	if f.LockedBy == nil || *f.LockedBy != userID {
		return nil, ErrFlowNotOwner
	}
	now := time.Now()
	expires := now.Add(lockTTL)
	f.LockExpiresAt = &expires
	f.UpdatedAt = now
	if err := s.flows.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *IVRFlowService) Unlock(ctx context.Context, id int64, userID int64) (*IVRFlow, error) {
	f, err := s.flows.GetByID(ctx, id)
	if err != nil || f == nil {
		return nil, ErrFlowNotFound
	}

	if f.LockedBy == nil {
		return nil, ErrFlowNotLocked
	}
	if *f.LockedBy != userID {
		return nil, ErrFlowNotOwner
	}

	f.LockedBy = nil
	f.LockedAt = nil
	f.LockExpiresAt = nil
	f.UpdatedAt = time.Now()

	if err := s.flows.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *IVRFlowService) Clone(ctx context.Context, id int64, newCode, newName string) (*IVRFlow, error) {
	f, err := s.flows.GetByID(ctx, id)
	if err != nil || f == nil {
		return nil, ErrFlowNotFound
	}

	existing, _ := s.flows.GetByCode(ctx, f.TenantID, newCode, 1)
	if existing != nil {
		return nil, ErrFlowCodeExists
	}

	now := time.Now()
	clone := &IVRFlow{
		ID:        snowflake.NextID(),
		TenantID:  f.TenantID,
		Code:      newCode,
		Name:      newName,
		FlowType:  f.FlowType,
		Version:   1,
		Graph:     f.Graph,
		Status:    FlowStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.flows.Create(ctx, clone); err != nil {
		return nil, err
	}
	return clone, nil
}

func (s *IVRFlowService) Rollback(ctx context.Context, id int64, version int) (*IVRFlow, error) {
	f, err := s.flows.GetByID(ctx, id)
	if err != nil || f == nil {
		return nil, ErrFlowNotFound
	}

	v, err := s.versions.GetByFlowAndVersion(ctx, id, version)
	if err != nil || v == nil {
		return nil, ErrVersionNotFound
	}

	f.Graph = v.Graph
	f.Status = FlowStatusDraft
	f.UpdatedAt = time.Now()

	if err := s.flows.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *IVRFlowService) GetVersions(ctx context.Context, flowID int64) ([]*IVRFlowVersion, error) {
	return s.versions.ListByFlow(ctx, flowID)
}
