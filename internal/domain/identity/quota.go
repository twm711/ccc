package identity

import (
	"context"
	"fmt"
)

// StorageUsageProvider returns the current storage usage in MB for a tenant.
type StorageUsageProvider interface {
	UsageMB(ctx context.Context, tenantID int64) (int64, error)
}

// AIUsageProvider returns the current daily AI call count for a tenant.
type AIUsageProvider interface {
	TodayCount(ctx context.Context, tenantID int64) (int, error)
}

// QuotaChecker validates resource usage against tenant quotas.
type QuotaChecker struct {
	settings TenantSettingsRepository
	storage  StorageUsageProvider
	ai       AIUsageProvider
}

// NewQuotaChecker creates a new quota checker.
func NewQuotaChecker(settings TenantSettingsRepository) *QuotaChecker {
	return &QuotaChecker{settings: settings}
}

// SetStorageProvider sets the storage usage provider.
func (q *QuotaChecker) SetStorageProvider(p StorageUsageProvider) { q.storage = p }

// SetAIProvider sets the AI usage provider.
func (q *QuotaChecker) SetAIProvider(p AIUsageProvider) { q.ai = p }

// QuotaStatus represents the result of a quota check.
type QuotaStatus struct {
	Resource string `json:"resource"`
	Limit    int64  `json:"limit"`
	Used     int64  `json:"used"`
	Exceeded bool   `json:"exceeded"`
}

// CheckStorage validates storage quota for a tenant.
func (q *QuotaChecker) CheckStorage(ctx context.Context, tenantID int64) (*QuotaStatus, error) {
	if q.storage == nil {
		return &QuotaStatus{Resource: "storage_mb"}, nil
	}
	s, err := q.settings.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("quota: get settings: %w", err)
	}
	if s.StorageQuotaMB == 0 {
		return &QuotaStatus{Resource: "storage_mb"}, nil
	}
	used, err := q.storage.UsageMB(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("quota: get storage usage: %w", err)
	}
	return &QuotaStatus{
		Resource: "storage_mb",
		Limit:    s.StorageQuotaMB,
		Used:     used,
		Exceeded: used >= s.StorageQuotaMB,
	}, nil
}

// CheckAICalls validates AI call quota for a tenant.
func (q *QuotaChecker) CheckAICalls(ctx context.Context, tenantID int64) (*QuotaStatus, error) {
	if q.ai == nil {
		return &QuotaStatus{Resource: "ai_calls_per_day"}, nil
	}
	s, err := q.settings.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("quota: get settings: %w", err)
	}
	if s.AICallQuotaPerDay == 0 {
		return &QuotaStatus{Resource: "ai_calls_per_day"}, nil
	}
	used, err := q.ai.TodayCount(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("quota: get AI usage: %w", err)
	}
	return &QuotaStatus{
		Resource: "ai_calls_per_day",
		Limit:    int64(s.AICallQuotaPerDay),
		Used:     int64(used),
		Exceeded: used >= s.AICallQuotaPerDay,
	}, nil
}
