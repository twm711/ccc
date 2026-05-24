package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/jmoiron/sqlx"
)

type CallRepo struct {
	db *sqlx.DB
}

func NewCallRepo(db *sqlx.DB) *CallRepo {
	return &CallRepo{db: db}
}

func (r *CallRepo) Create(ctx context.Context, c *call.Call) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO calls (id, tenant_id, direction, call_type, media_type, caller, callee, masked_callee,
		 agent_user_id, skill_group_id, ivr_flow_id, phone_number_id, carrier_id, parent_call_id, campaign_case_id,
		 status, hangup_reason, disposition_code, hold_count, transfer_count, satisfaction_rating,
		 ivr_duration_sec, ring_duration_sec, queue_duration_sec, wait_duration_sec, duration_sec,
		 recording_url, custom_data, started_at, answered_at, ended_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.Direction, c.CallType, c.MediaType, c.Caller, c.Callee, c.MaskedCallee,
		c.AgentUserID, c.SkillGroupID, c.IVRFlowID, c.PhoneNumberID, c.CarrierID, c.ParentCallID, c.CampaignCaseID,
		c.Status, c.HangupReason, c.DispositionCode, c.HoldCount, c.TransferCount, c.SatisfactionRating,
		c.IVRDurationSec, c.RingDurationSec, c.QueueDurationSec, c.WaitDurationSec, c.DurationSec,
		c.RecordingURL, c.CustomData, c.StartedAt, c.AnsweredAt, c.EndedAt)
	return err
}

func (r *CallRepo) GetByID(ctx context.Context, id int64) (*call.Call, error) {
	var c call.Call
	err := r.db.GetContext(ctx, &c, "SELECT * FROM calls WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *CallRepo) Update(ctx context.Context, c *call.Call) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE calls SET status=?, hangup_reason=?, disposition_code=?, agent_user_id=?, skill_group_id=?,
		 hold_count=?, transfer_count=?, satisfaction_rating=?,
		 ivr_duration_sec=?, ring_duration_sec=?, queue_duration_sec=?, wait_duration_sec=?, duration_sec=?,
		 recording_url=?, answered_at=?, ended_at=? WHERE id=?`,
		c.Status, c.HangupReason, c.DispositionCode, c.AgentUserID, c.SkillGroupID,
		c.HoldCount, c.TransferCount, c.SatisfactionRating,
		c.IVRDurationSec, c.RingDurationSec, c.QueueDurationSec, c.WaitDurationSec, c.DurationSec,
		c.RecordingURL, c.AnsweredAt, c.EndedAt, c.ID)
	return err
}

func (r *CallRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*call.Call, int64, error) {
	return r.ListWithFilter(ctx, tenantID, call.CallListFilter{}, offset, limit)
}

func (r *CallRepo) ListWithFilter(ctx context.Context, tenantID int64, filter call.CallListFilter, offset, limit int) ([]*call.Call, int64, error) {
	where := "WHERE tenant_id = ?"
	args := []interface{}{tenantID}

	if filter.Direction != nil {
		where += " AND direction = ?"
		args = append(args, *filter.Direction)
	}
	if filter.CallType != nil {
		where += " AND call_type = ?"
		args = append(args, *filter.CallType)
	}
	if filter.MediaType != nil {
		where += " AND media_type = ?"
		args = append(args, *filter.MediaType)
	}
	if filter.Status != nil {
		where += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.Caller != "" {
		where += " AND caller LIKE ?"
		args = append(args, "%"+filter.Caller+"%")
	}
	if filter.Callee != "" {
		where += " AND callee LIKE ?"
		args = append(args, "%"+filter.Callee+"%")
	}
	if filter.StartFrom != nil {
		where += " AND started_at >= ?"
		args = append(args, *filter.StartFrom)
	}
	if filter.StartTo != nil {
		where += " AND started_at <= ?"
		args = append(args, *filter.StartTo)
	}

	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM calls "+where, args...)

	queryArgs := append(args, limit, offset)
	var calls []*call.Call
	err := r.db.SelectContext(ctx, &calls,
		"SELECT * FROM calls "+where+" ORDER BY started_at DESC LIMIT ? OFFSET ?", queryArgs...)
	return calls, total, err
}

// CallEventRepo

type CallEventRepo struct {
	db *sqlx.DB
}

func NewCallEventRepo(db *sqlx.DB) *CallEventRepo {
	return &CallEventRepo{db: db}
}

func (r *CallEventRepo) Create(ctx context.Context, e *call.CallEvent) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO call_events (id, call_id, tenant_id, event, detail, created_at) VALUES (?,?,?,?,?,?)",
		e.ID, e.CallID, e.TenantID, e.Event, e.Detail, e.CreatedAt)
	return err
}

func (r *CallEventRepo) ListByCallID(ctx context.Context, callID int64) ([]*call.CallEvent, error) {
	var events []*call.CallEvent
	err := r.db.SelectContext(ctx, &events,
		"SELECT * FROM call_events WHERE call_id = ? ORDER BY created_at ASC", callID)
	return events, err
}

// IVRTrackingRepo

type IVRTrackingRepo struct {
	db *sqlx.DB
}

func NewIVRTrackingRepo(db *sqlx.DB) *IVRTrackingRepo {
	return &IVRTrackingRepo{db: db}
}

func (r *IVRTrackingRepo) Create(ctx context.Context, t *call.IVRTracking) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ivr_tracking (id, call_id, tenant_id, ivr_flow_id, node_id, node_type, node_name,
		 variables, exit_name, status_code, entered_at, exited_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.CallID, t.TenantID, t.IVRFlowID, t.NodeID, t.NodeType, t.NodeName,
		t.Variables, t.ExitName, t.StatusCode, t.EnteredAt, t.ExitedAt)
	return err
}

func (r *IVRTrackingRepo) ListByCallID(ctx context.Context, callID int64) ([]*call.IVRTracking, error) {
	var entries []*call.IVRTracking
	err := r.db.SelectContext(ctx, &entries,
		"SELECT * FROM ivr_tracking WHERE call_id = ? ORDER BY entered_at ASC", callID)
	return entries, err
}

// RecordingRepo

type RecordingRepo struct {
	db *sqlx.DB
}

func NewRecordingRepo(db *sqlx.DB) *RecordingRepo {
	return &RecordingRepo{db: db}
}

func (r *RecordingRepo) Create(ctx context.Context, rec *call.Recording) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO recordings (id, tenant_id, call_id, agent_user_id, file_name, file_path, file_size,
		 duration_sec, mime_type, storage_tier, status, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		rec.ID, rec.TenantID, rec.CallID, rec.AgentUserID, rec.FileName, rec.FilePath, rec.FileSize,
		rec.DurationSec, rec.MimeType, rec.StorageTier, rec.Status, rec.CreatedAt)
	return err
}

func (r *RecordingRepo) GetByID(ctx context.Context, id int64) (*call.Recording, error) {
	var rec call.Recording
	err := r.db.GetContext(ctx, &rec, "SELECT * FROM recordings WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &rec, err
}

func (r *RecordingRepo) GetByCallID(ctx context.Context, callID int64) (*call.Recording, error) {
	var rec call.Recording
	err := r.db.GetContext(ctx, &rec, "SELECT * FROM recordings WHERE call_id = ? LIMIT 1", callID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &rec, err
}

func (r *RecordingRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*call.Recording, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM recordings WHERE tenant_id = ?", tenantID)

	var recs []*call.Recording
	err := r.db.SelectContext(ctx, &recs,
		"SELECT * FROM recordings WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return recs, total, err
}

// VoicemailRepo

type VoicemailRepo struct {
	db *sqlx.DB
}

func NewVoicemailRepo(db *sqlx.DB) *VoicemailRepo {
	return &VoicemailRepo{db: db}
}

func (r *VoicemailRepo) Create(ctx context.Context, v *call.Voicemail) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO voicemails (id, tenant_id, call_id, caller, agent_user_id, skill_group_id,
		 file_path, duration_sec, is_read, created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.TenantID, v.CallID, v.Caller, v.AgentUserID, v.SkillGroupID,
		v.FilePath, v.DurationSec, v.IsRead, v.CreatedAt)
	return err
}

func (r *VoicemailRepo) GetByID(ctx context.Context, id int64) (*call.Voicemail, error) {
	var v call.Voicemail
	err := r.db.GetContext(ctx, &v, "SELECT * FROM voicemails WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

func (r *VoicemailRepo) Update(ctx context.Context, v *call.Voicemail) error {
	_, err := r.db.ExecContext(ctx, "UPDATE voicemails SET is_read=? WHERE id=?", v.IsRead, v.ID)
	return err
}

func (r *VoicemailRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*call.Voicemail, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM voicemails WHERE tenant_id = ?", tenantID)

	var vms []*call.Voicemail
	err := r.db.SelectContext(ctx, &vms,
		"SELECT * FROM voicemails WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return vms, total, err
}

func (r *VoicemailRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM voicemails WHERE id = ?", id)
	return err
}
