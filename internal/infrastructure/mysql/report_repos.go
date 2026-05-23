package mysql

import (
	"context"
	"fmt"

	"github.com/divord97/ccc/internal/domain/report"
	"github.com/jmoiron/sqlx"
)

// AgentReportRepo implements report.AgentReportRepository.
type AgentReportRepo struct{ db *sqlx.DB }

func NewAgentReportRepo(db *sqlx.DB) *AgentReportRepo { return &AgentReportRepo{db: db} }

func (r *AgentReportRepo) Query(ctx context.Context, f report.ReportFilter) ([]*report.AgentReport, int64, error) {
	query := `SELECT
		c.agent_user_id AS agent_id,
		u.display_name AS agent_name,
		COUNT(*) AS total_calls,
		SUM(CASE WHEN c.direction='inbound' THEN 1 ELSE 0 END) AS inbound_calls,
		SUM(CASE WHEN c.direction='outbound' THEN 1 ELSE 0 END) AS outbound_calls,
		SUM(CASE WHEN c.status='completed' AND c.talk_duration_sec > 0 THEN 1 ELSE 0 END) AS answered_calls,
		SUM(CASE WHEN c.status='completed' AND c.talk_duration_sec = 0 THEN 1 ELSE 0 END) AS missed_calls,
		SUM(CASE WHEN c.transfer_count > 0 THEN 1 ELSE 0 END) AS transferred_calls,
		SUM(CASE WHEN c.hold_count > 0 THEN 1 ELSE 0 END) AS held_calls,
		COALESCE(AVG(c.talk_duration_sec),0) AS avg_talk_duration_sec,
		COALESCE(SUM(c.talk_duration_sec),0) AS total_talk_duration_sec,
		COALESCE(AVG(c.hold_duration_sec),0) AS avg_hold_duration_sec,
		COALESCE(AVG(c.acw_duration_sec),0) AS avg_acw_duration_sec,
		COALESCE(SUM(c.acw_duration_sec),0) AS total_acw_duration_sec,
		COALESCE(AVG(c.ring_duration_sec),0) AS avg_ring_duration_sec,
		COALESCE(AVG(c.wait_duration_sec),0) AS avg_wait_duration_sec
	FROM calls c
	LEFT JOIN users u ON u.id = c.agent_user_id
	WHERE c.tenant_id = ? AND c.created_at BETWEEN ? AND ? AND c.agent_user_id IS NOT NULL`

	args := []interface{}{f.TenantID, f.StartTime, f.EndTime}

	if f.AgentID != nil {
		query += " AND c.agent_user_id = ?"
		args = append(args, *f.AgentID)
	}

	query += " GROUP BY c.agent_user_id, u.display_name ORDER BY total_calls DESC"

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) sub", query)
	var total int64
	_ = r.db.GetContext(ctx, &total, countQuery, args...)

	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	var items []*report.AgentReport
	err := r.db.SelectContext(ctx, &items, query, args...)
	return items, total, err
}

// GroupAgentReportRepo implements report.GroupAgentReportRepository.
type GroupAgentReportRepo struct{ db *sqlx.DB }

func NewGroupAgentReportRepo(db *sqlx.DB) *GroupAgentReportRepo {
	return &GroupAgentReportRepo{db: db}
}

func (r *GroupAgentReportRepo) Query(ctx context.Context, f report.ReportFilter) ([]*report.GroupAgentReport, int64, error) {
	query := `SELECT
		sgm.skill_group_id,
		sg.name AS skill_group_name,
		c.agent_user_id AS agent_id,
		u.display_name AS agent_name,
		COUNT(*) AS total_calls,
		SUM(CASE WHEN c.direction='inbound' THEN 1 ELSE 0 END) AS inbound_calls,
		SUM(CASE WHEN c.direction='outbound' THEN 1 ELSE 0 END) AS outbound_calls,
		SUM(CASE WHEN c.status='completed' AND c.talk_duration_sec > 0 THEN 1 ELSE 0 END) AS answered_calls,
		COALESCE(AVG(c.talk_duration_sec),0) AS avg_talk_duration_sec,
		COALESCE(SUM(c.talk_duration_sec),0) AS total_talk_duration_sec
	FROM calls c
	JOIN skill_group_members sgm ON sgm.agent_id = c.agent_user_id
	JOIN skill_groups sg ON sg.id = sgm.skill_group_id
	LEFT JOIN users u ON u.id = c.agent_user_id
	WHERE c.tenant_id = ? AND c.created_at BETWEEN ? AND ? AND c.agent_user_id IS NOT NULL`

	args := []interface{}{f.TenantID, f.StartTime, f.EndTime}

	if f.SkillGroupID != nil {
		query += " AND sgm.skill_group_id = ?"
		args = append(args, *f.SkillGroupID)
	}

	query += " GROUP BY sgm.skill_group_id, sg.name, c.agent_user_id, u.display_name ORDER BY sg.name, total_calls DESC"

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) sub", query)
	var total int64
	_ = r.db.GetContext(ctx, &total, countQuery, args...)

	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	var items []*report.GroupAgentReport
	err := r.db.SelectContext(ctx, &items, query, args...)
	return items, total, err
}

// SkillGroupReportRepo implements report.SkillGroupReportRepository.
type SkillGroupReportRepo struct{ db *sqlx.DB }

func NewSkillGroupReportRepo(db *sqlx.DB) *SkillGroupReportRepo {
	return &SkillGroupReportRepo{db: db}
}

func (r *SkillGroupReportRepo) Query(ctx context.Context, f report.ReportFilter) ([]*report.SkillGroupReport, int64, error) {
	query := `SELECT
		sg.id AS skill_group_id,
		sg.name AS skill_group_name,
		COUNT(c.id) AS total_calls,
		SUM(CASE WHEN c.direction='inbound' THEN 1 ELSE 0 END) AS inbound_calls,
		SUM(CASE WHEN c.direction='outbound' THEN 1 ELSE 0 END) AS outbound_calls,
		SUM(CASE WHEN c.status='completed' AND c.talk_duration_sec > 0 THEN 1 ELSE 0 END) AS answered_calls,
		SUM(CASE WHEN c.hangup_cause='CALLER_ABANDON' THEN 1 ELSE 0 END) AS abandoned_calls,
		SUM(CASE WHEN c.queue_duration_sec > 0 THEN 1 ELSE 0 END) AS queue_total,
		SUM(CASE WHEN c.queue_duration_sec > 0 AND c.hangup_cause='CALLER_ABANDON' THEN 1 ELSE 0 END) AS queue_abandoned,
		SUM(CASE WHEN c.ring_duration_sec > 0 AND c.hangup_cause='CALLER_ABANDON' THEN 1 ELSE 0 END) AS ring_abandoned,
		COALESCE(AVG(c.wait_duration_sec),0) AS avg_wait_sec,
		COALESCE(AVG(c.talk_duration_sec),0) AS avg_talk_sec
	FROM skill_groups sg
	LEFT JOIN calls c ON c.skill_group_id = sg.id AND c.created_at BETWEEN ? AND ?
	WHERE sg.tenant_id = ?`

	args := []interface{}{f.StartTime, f.EndTime, f.TenantID}

	if f.SkillGroupID != nil {
		query += " AND sg.id = ?"
		args = append(args, *f.SkillGroupID)
	}

	query += " GROUP BY sg.id, sg.name ORDER BY total_calls DESC"

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) sub", query)
	var total int64
	_ = r.db.GetContext(ctx, &total, countQuery, args...)

	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	var items []*report.SkillGroupReport
	err := r.db.SelectContext(ctx, &items, query, args...)
	return items, total, err
}

// Back2BackReportRepo implements report.Back2BackReportRepository.
type Back2BackReportRepo struct{ db *sqlx.DB }

func NewBack2BackReportRepo(db *sqlx.DB) *Back2BackReportRepo {
	return &Back2BackReportRepo{db: db}
}

func (r *Back2BackReportRepo) Query(ctx context.Context, f report.ReportFilter) (*report.Back2BackReport, error) {
	var result report.Back2BackReport
	err := r.db.GetContext(ctx, &result, `SELECT
		COUNT(*) AS total_calls,
		SUM(CASE WHEN talk_duration_sec > 0 THEN 1 ELSE 0 END) AS connected_calls,
		COALESCE(AVG(talk_duration_sec),0) AS avg_duration_sec,
		COALESCE(SUM(talk_duration_sec),0) AS total_duration
	FROM calls WHERE tenant_id = ? AND call_type = 'BACK2BACK' AND created_at BETWEEN ? AND ?`,
		f.TenantID, f.StartTime, f.EndTime)
	if err != nil {
		return nil, err
	}
	if result.TotalCalls > 0 {
		result.ConnectRate = float64(result.ConnectedCalls) / float64(result.TotalCalls) * 100
	}
	return &result, nil
}

// InternalCallReportRepo implements report.InternalCallReportRepository.
type InternalCallReportRepo struct{ db *sqlx.DB }

func NewInternalCallReportRepo(db *sqlx.DB) *InternalCallReportRepo {
	return &InternalCallReportRepo{db: db}
}

func (r *InternalCallReportRepo) Query(ctx context.Context, f report.ReportFilter) (*report.InternalCallReport, error) {
	var result report.InternalCallReport
	err := r.db.GetContext(ctx, &result, `SELECT
		COUNT(*) AS total_calls,
		SUM(CASE WHEN talk_duration_sec > 0 THEN 1 ELSE 0 END) AS connected_calls,
		COALESCE(AVG(talk_duration_sec),0) AS avg_duration_sec,
		COALESCE(SUM(talk_duration_sec),0) AS total_duration
	FROM calls WHERE tenant_id = ? AND call_type = 'INTERNAL' AND created_at BETWEEN ? AND ?`,
		f.TenantID, f.StartTime, f.EndTime)
	if err != nil {
		return nil, err
	}
	if result.TotalCalls > 0 {
		result.ConnectRate = float64(result.ConnectedCalls) / float64(result.TotalCalls) * 100
	}
	return &result, nil
}

// AgentStatusLogRepo implements report.AgentStatusLogRepository.
type AgentStatusLogRepo struct{ db *sqlx.DB }

func NewAgentStatusLogRepo(db *sqlx.DB) *AgentStatusLogRepo {
	return &AgentStatusLogRepo{db: db}
}

func (r *AgentStatusLogRepo) Query(ctx context.Context, f report.ReportFilter, breakReasonCode string) ([]*report.AgentStatusLog, int64, error) {
	query := `SELECT apl.id, apl.tenant_id, apl.agent_id, u.display_name AS agent_name,
		apl.status, COALESCE(apl.sub_state,'') AS sub_state, COALESCE(apl.work_mode,'') AS work_mode,
		COALESCE(apl.break_reason_code,'') AS break_reason_code, apl.duration_sec, apl.created_at
	FROM agent_presence_log apl
	LEFT JOIN users u ON u.id = apl.agent_id
	WHERE apl.tenant_id = ? AND apl.created_at BETWEEN ? AND ?`

	args := []interface{}{f.TenantID, f.StartTime, f.EndTime}

	if f.AgentID != nil {
		query += " AND apl.agent_id = ?"
		args = append(args, *f.AgentID)
	}
	if breakReasonCode != "" {
		query += " AND apl.break_reason_code = ?"
		args = append(args, breakReasonCode)
	}

	query += " ORDER BY apl.created_at DESC"

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) sub", query)
	var total int64
	_ = r.db.GetContext(ctx, &total, countQuery, args...)

	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	var items []*report.AgentStatusLog
	err := r.db.SelectContext(ctx, &items, query, args...)
	return items, total, err
}
