package routing

import "context"

type IVRFlowRepository interface {
	Create(ctx context.Context, f *IVRFlow) error
	GetByID(ctx context.Context, id int64) (*IVRFlow, error)
	GetByCode(ctx context.Context, tenantID int64, code string, version int) (*IVRFlow, error)
	Update(ctx context.Context, f *IVRFlow) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*IVRFlow, int64, error)
	Delete(ctx context.Context, id int64) error
}

type IVRFlowVersionRepository interface {
	Create(ctx context.Context, v *IVRFlowVersion) error
	GetByFlowAndVersion(ctx context.Context, flowID int64, version int) (*IVRFlowVersion, error)
	ListByFlow(ctx context.Context, flowID int64) ([]*IVRFlowVersion, error)
}
