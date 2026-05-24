package configuration

import "context"

type BreakReasonRepository interface {
	Create(ctx context.Context, br *BreakReason) error
	GetByID(ctx context.Context, id int64) (*BreakReason, error)
	Update(ctx context.Context, br *BreakReason) error
	List(ctx context.Context, tenantID int64) ([]*BreakReason, error)
	Delete(ctx context.Context, id int64) error
}

type DispositionCodeRepository interface {
	Create(ctx context.Context, dc *DispositionCode) error
	GetByID(ctx context.Context, id int64) (*DispositionCode, error)
	Update(ctx context.Context, dc *DispositionCode) error
	List(ctx context.Context, tenantID int64) ([]*DispositionCode, error)
	Delete(ctx context.Context, id int64) error
}

type CustomFieldDefinitionRepository interface {
	Create(ctx context.Context, cfd *CustomFieldDefinition) error
	GetByID(ctx context.Context, id int64) (*CustomFieldDefinition, error)
	Update(ctx context.Context, cfd *CustomFieldDefinition) error
	List(ctx context.Context, tenantID int64, target CustomFieldTarget) ([]*CustomFieldDefinition, error)
	Delete(ctx context.Context, id int64) error
}

type CallTagRepository interface {
	Create(ctx context.Context, ct *CallTag) error
	GetByID(ctx context.Context, id int64) (*CallTag, error)
	List(ctx context.Context, tenantID int64) ([]*CallTag, error)
	Update(ctx context.Context, ct *CallTag) error
	Delete(ctx context.Context, id int64) error
}
