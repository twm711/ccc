package mysql

import (
	"context"

	"github.com/divord97/ccc/internal/domain/configuration"
	"github.com/divord97/ccc/internal/domain/operation"
	"github.com/jmoiron/sqlx"
)

// --- BreakReasonRepo ---

type BreakReasonRepo struct{ db *sqlx.DB }

func NewBreakReasonRepo(db *sqlx.DB) *BreakReasonRepo { return &BreakReasonRepo{db: db} }

func (r *BreakReasonRepo) Create(ctx context.Context, br *configuration.BreakReason) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO break_reasons (id, tenant_id, code, name, is_system, sort_order, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		br.ID, br.TenantID, br.Code, br.Name, br.IsSystem, br.SortOrder, br.Enabled, br.CreatedAt, br.UpdatedAt)
	return err
}

func (r *BreakReasonRepo) GetByID(ctx context.Context, id int64) (*configuration.BreakReason, error) {
	var br configuration.BreakReason
	err := r.db.GetContext(ctx, &br, `SELECT * FROM break_reasons WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &br, nil
}

func (r *BreakReasonRepo) Update(ctx context.Context, br *configuration.BreakReason) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE break_reasons SET code=?, name=?, is_system=?, sort_order=?, enabled=?, updated_at=? WHERE id=?`,
		br.Code, br.Name, br.IsSystem, br.SortOrder, br.Enabled, br.UpdatedAt, br.ID)
	return err
}

func (r *BreakReasonRepo) List(ctx context.Context, tenantID int64) ([]*configuration.BreakReason, error) {
	var items []*configuration.BreakReason
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM break_reasons WHERE tenant_id = ? ORDER BY sort_order ASC`, tenantID)
	return items, err
}

func (r *BreakReasonRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM break_reasons WHERE id = ?`, id)
	return err
}

// --- DispositionCodeRepo ---

type DispositionCodeRepo struct{ db *sqlx.DB }

func NewDispositionCodeRepo(db *sqlx.DB) *DispositionCodeRepo {
	return &DispositionCodeRepo{db: db}
}

func (r *DispositionCodeRepo) Create(ctx context.Context, dc *configuration.DispositionCode) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO disposition_codes (id, tenant_id, code, name, category, sort_order, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		dc.ID, dc.TenantID, dc.Code, dc.Name, dc.Category, dc.SortOrder, dc.Enabled, dc.CreatedAt, dc.UpdatedAt)
	return err
}

func (r *DispositionCodeRepo) GetByID(ctx context.Context, id int64) (*configuration.DispositionCode, error) {
	var dc configuration.DispositionCode
	err := r.db.GetContext(ctx, &dc, `SELECT * FROM disposition_codes WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &dc, nil
}

func (r *DispositionCodeRepo) Update(ctx context.Context, dc *configuration.DispositionCode) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE disposition_codes SET code=?, name=?, category=?, sort_order=?, enabled=?, updated_at=? WHERE id=?`,
		dc.Code, dc.Name, dc.Category, dc.SortOrder, dc.Enabled, dc.UpdatedAt, dc.ID)
	return err
}

func (r *DispositionCodeRepo) List(ctx context.Context, tenantID int64) ([]*configuration.DispositionCode, error) {
	var items []*configuration.DispositionCode
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM disposition_codes WHERE tenant_id = ? ORDER BY sort_order ASC`, tenantID)
	return items, err
}

func (r *DispositionCodeRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM disposition_codes WHERE id = ?`, id)
	return err
}

// --- CallTagRepo (configuration.CallTagRepository) ---

type CallTagDefRepo struct{ db *sqlx.DB }

func NewCallTagDefRepo(db *sqlx.DB) *CallTagDefRepo { return &CallTagDefRepo{db: db} }

func (r *CallTagDefRepo) Create(ctx context.Context, ct *configuration.CallTag) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO call_tag_definitions (id, tenant_id, name, color, created_at) VALUES (?, ?, ?, ?, ?)`,
		ct.ID, ct.TenantID, ct.Name, ct.Color, ct.CreatedAt)
	return err
}

func (r *CallTagDefRepo) GetByID(ctx context.Context, id int64) (*configuration.CallTag, error) {
	var ct configuration.CallTag
	err := r.db.GetContext(ctx, &ct, `SELECT * FROM call_tag_definitions WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &ct, nil
}

func (r *CallTagDefRepo) List(ctx context.Context, tenantID int64) ([]*configuration.CallTag, error) {
	var items []*configuration.CallTag
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM call_tag_definitions WHERE tenant_id = ? ORDER BY created_at DESC`, tenantID)
	return items, err
}

func (r *CallTagDefRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM call_tag_definitions WHERE id = ?`, id)
	return err
}

// --- AudioFileRepo ---

type AudioFileRepo struct{ db *sqlx.DB }

func NewAudioFileRepo(db *sqlx.DB) *AudioFileRepo { return &AudioFileRepo{db: db} }

func (r *AudioFileRepo) Create(ctx context.Context, af *operation.AudioFile) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audio_files (id, tenant_id, name, file_name, category, file_path, file_size, duration, mime_type, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		af.ID, af.TenantID, af.Name, af.FileName, af.Category, af.FilePath, af.FileSize, af.Duration, af.MimeType, af.CreatedAt)
	return err
}

func (r *AudioFileRepo) GetByID(ctx context.Context, id int64) (*operation.AudioFile, error) {
	var af operation.AudioFile
	err := r.db.GetContext(ctx, &af, `SELECT * FROM audio_files WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &af, nil
}

func (r *AudioFileRepo) List(ctx context.Context, tenantID int64, category operation.AudioCategory) ([]*operation.AudioFile, error) {
	if category != "" {
		var items []*operation.AudioFile
		err := r.db.SelectContext(ctx, &items,
			`SELECT * FROM audio_files WHERE tenant_id = ? AND category = ? ORDER BY created_at DESC`,
			tenantID, category)
		return items, err
	}
	var items []*operation.AudioFile
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM audio_files WHERE tenant_id = ? ORDER BY created_at DESC`, tenantID)
	return items, err
}

func (r *AudioFileRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM audio_files WHERE id = ?`, id)
	return err
}

// --- BusinessHoursRepo ---

type BusinessHoursRepo struct{ db *sqlx.DB }

func NewBusinessHoursRepo(db *sqlx.DB) *BusinessHoursRepo { return &BusinessHoursRepo{db: db} }

func (r *BusinessHoursRepo) Create(ctx context.Context, bh *operation.BusinessHours) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO business_hours (id, tenant_id, name, is_default, timezone, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		bh.ID, bh.TenantID, bh.Name, bh.IsDefault, bh.Timezone, bh.CreatedAt, bh.UpdatedAt)
	return err
}

func (r *BusinessHoursRepo) GetByID(ctx context.Context, id int64) (*operation.BusinessHours, error) {
	var bh operation.BusinessHours
	err := r.db.GetContext(ctx, &bh, `SELECT * FROM business_hours WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &bh, nil
}

func (r *BusinessHoursRepo) Update(ctx context.Context, bh *operation.BusinessHours) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE business_hours SET name=?, is_default=?, timezone=?, updated_at=? WHERE id=?`,
		bh.Name, bh.IsDefault, bh.Timezone, bh.UpdatedAt, bh.ID)
	return err
}

func (r *BusinessHoursRepo) List(ctx context.Context, tenantID int64) ([]*operation.BusinessHours, error) {
	var items []*operation.BusinessHours
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM business_hours WHERE tenant_id = ? ORDER BY is_default DESC, name ASC`, tenantID)
	return items, err
}

func (r *BusinessHoursRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM business_hours WHERE id = ?`, id)
	return err
}

// --- BusinessHoursScheduleRepo ---

type BusinessHoursScheduleRepo struct{ db *sqlx.DB }

func NewBusinessHoursScheduleRepo(db *sqlx.DB) *BusinessHoursScheduleRepo {
	return &BusinessHoursScheduleRepo{db: db}
}

func (r *BusinessHoursScheduleRepo) Create(ctx context.Context, s *operation.BusinessHoursSchedule) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO business_hours_schedules (id, business_hours_id, day_type, day_of_week, specific_date, start_time, end_time, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.BusinessHoursID, s.DayType, s.DayOfWeek, s.SpecificDate, s.StartTime, s.EndTime, s.CreatedAt)
	return err
}

func (r *BusinessHoursScheduleRepo) GetByBusinessHoursID(ctx context.Context, bhID int64) ([]*operation.BusinessHoursSchedule, error) {
	var items []*operation.BusinessHoursSchedule
	err := r.db.SelectContext(ctx, &items,
		`SELECT * FROM business_hours_schedules WHERE business_hours_id = ? ORDER BY day_of_week, start_time`, bhID)
	return items, err
}

func (r *BusinessHoursScheduleRepo) DeleteByBusinessHoursID(ctx context.Context, bhID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM business_hours_schedules WHERE business_hours_id = ?`, bhID)
	return err
}
