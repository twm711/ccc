package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/routing"
	"github.com/jmoiron/sqlx"
)

type IVRFlowRepo struct {
	db *sqlx.DB
}

func NewIVRFlowRepo(db *sqlx.DB) *IVRFlowRepo {
	return &IVRFlowRepo{db: db}
}

func (r *IVRFlowRepo) Create(ctx context.Context, f *routing.IVRFlow) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ivr_flows (id, tenant_id, code, name, flow_type, version, graph, status, locked_by, locked_at, published_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.TenantID, f.Code, f.Name, f.FlowType, f.Version, f.Graph, f.Status,
		f.LockedBy, f.LockedAt, f.PublishedAt, f.CreatedAt, f.UpdatedAt)
	return err
}

func (r *IVRFlowRepo) GetByID(ctx context.Context, id int64) (*routing.IVRFlow, error) {
	var f routing.IVRFlow
	err := r.db.GetContext(ctx, &f, "SELECT * FROM ivr_flows WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

func (r *IVRFlowRepo) GetByCode(ctx context.Context, tenantID int64, code string, version int) (*routing.IVRFlow, error) {
	var f routing.IVRFlow
	err := r.db.GetContext(ctx, &f, "SELECT * FROM ivr_flows WHERE tenant_id = ? AND code = ?", tenantID, code)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

func (r *IVRFlowRepo) Update(ctx context.Context, f *routing.IVRFlow) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ivr_flows SET name=?, version=?, graph=?, status=?, locked_by=?, locked_at=?, published_at=?, updated_at=? WHERE id=?`,
		f.Name, f.Version, f.Graph, f.Status, f.LockedBy, f.LockedAt, f.PublishedAt, f.UpdatedAt, f.ID)
	return err
}

func (r *IVRFlowRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*routing.IVRFlow, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM ivr_flows WHERE tenant_id = ?", tenantID)

	var flows []*routing.IVRFlow
	err := r.db.SelectContext(ctx, &flows, "SELECT * FROM ivr_flows WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", tenantID, limit, offset)
	return flows, total, err
}

func (r *IVRFlowRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ivr_flows WHERE id = ?", id)
	return err
}

type IVRFlowVersionRepo struct {
	db *sqlx.DB
}

func NewIVRFlowVersionRepo(db *sqlx.DB) *IVRFlowVersionRepo {
	return &IVRFlowVersionRepo{db: db}
}

func (r *IVRFlowVersionRepo) Create(ctx context.Context, v *routing.IVRFlowVersion) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ivr_flow_versions (id, ivr_flow_id, tenant_id, version, graph, description, published_by, published_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.IVRFlowID, v.TenantID, v.Version, v.Graph, v.Description, v.PublishedBy, v.PublishedAt)
	return err
}

func (r *IVRFlowVersionRepo) GetByFlowAndVersion(ctx context.Context, flowID int64, version int) (*routing.IVRFlowVersion, error) {
	var v routing.IVRFlowVersion
	err := r.db.GetContext(ctx, &v, "SELECT * FROM ivr_flow_versions WHERE ivr_flow_id = ? AND version = ?", flowID, version)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

func (r *IVRFlowVersionRepo) ListByFlow(ctx context.Context, flowID int64) ([]*routing.IVRFlowVersion, error) {
	var versions []*routing.IVRFlowVersion
	err := r.db.SelectContext(ctx, &versions, "SELECT * FROM ivr_flow_versions WHERE ivr_flow_id = ? ORDER BY version DESC", flowID)
	return versions, err
}
