package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/jmoiron/sqlx"
)

type QueueSnapshotRepo struct{ db *sqlx.DB }

func NewQueueSnapshotRepo(db *sqlx.DB) *QueueSnapshotRepo { return &QueueSnapshotRepo{db: db} }

func (r *QueueSnapshotRepo) Create(ctx context.Context, s *call.QueueSnapshot) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO queue_snapshots (id, tenant_id, skill_group_id, waiting_count, available_agents, avg_wait_sec, max_wait_sec, snapshot_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TenantID, s.SkillGroupID, s.WaitingCount, s.AvailableAgents, s.AvgWaitSec, s.MaxWaitSec, s.SnapshotAt)
	return err
}

func (r *QueueSnapshotRepo) GetLatest(ctx context.Context, tenantID, skillGroupID int64) (*call.QueueSnapshot, error) {
	var s call.QueueSnapshot
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM queue_snapshots WHERE tenant_id = ? AND skill_group_id = ? ORDER BY snapshot_at DESC LIMIT 1`,
		tenantID, skillGroupID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
