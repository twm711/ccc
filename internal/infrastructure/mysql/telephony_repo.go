package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/telephony"
	"github.com/jmoiron/sqlx"
)

// CarrierRepo

type CarrierRepo struct {
	db *sqlx.DB
}

func NewCarrierRepo(db *sqlx.DB) *CarrierRepo {
	return &CarrierRepo{db: db}
}

func (r *CarrierRepo) Create(ctx context.Context, c *telephony.Carrier) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO carriers (id, tenant_id, name, protocol, host, port, status, max_channels, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.Name, c.Protocol, c.Host, c.Port, c.Status, c.MaxChannels, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CarrierRepo) GetByID(ctx context.Context, id int64) (*telephony.Carrier, error) {
	var c telephony.Carrier
	err := r.db.GetContext(ctx, &c, "SELECT * FROM carriers WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *CarrierRepo) Update(ctx context.Context, c *telephony.Carrier) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE carriers SET name=?, protocol=?, host=?, port=?, status=?, max_channels=?, updated_at=? WHERE id=?",
		c.Name, c.Protocol, c.Host, c.Port, c.Status, c.MaxChannels, c.UpdatedAt, c.ID)
	return err
}

func (r *CarrierRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*telephony.Carrier, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM carriers WHERE tenant_id = ?", tenantID)

	var carriers []*telephony.Carrier
	err := r.db.SelectContext(ctx, &carriers,
		"SELECT * FROM carriers WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return carriers, total, err
}

// SIPTrunkRepo

type SIPTrunkRepo struct {
	db *sqlx.DB
}

func NewSIPTrunkRepo(db *sqlx.DB) *SIPTrunkRepo {
	return &SIPTrunkRepo{db: db}
}

func (r *SIPTrunkRepo) Create(ctx context.Context, t *telephony.SIPTrunk) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sip_trunks (id, tenant_id, carrier_id, name, username, password, domain, transport, codecs, max_channels, status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.TenantID, t.CarrierID, t.Name, t.Username, t.Password, t.Domain, t.Transport, t.Codecs, t.MaxChannels, t.Status, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *SIPTrunkRepo) GetByID(ctx context.Context, id int64) (*telephony.SIPTrunk, error) {
	var t telephony.SIPTrunk
	err := r.db.GetContext(ctx, &t, "SELECT * FROM sip_trunks WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (r *SIPTrunkRepo) Update(ctx context.Context, t *telephony.SIPTrunk) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sip_trunks SET name=?, username=?, password=?, domain=?, transport=?, codecs=?, max_channels=?, status=?, updated_at=? WHERE id=?`,
		t.Name, t.Username, t.Password, t.Domain, t.Transport, t.Codecs, t.MaxChannels, t.Status, t.UpdatedAt, t.ID)
	return err
}

func (r *SIPTrunkRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*telephony.SIPTrunk, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM sip_trunks WHERE tenant_id = ?", tenantID)

	var trunks []*telephony.SIPTrunk
	err := r.db.SelectContext(ctx, &trunks,
		"SELECT * FROM sip_trunks WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return trunks, total, err
}

// PhoneNumberRepo

type PhoneNumberRepo struct {
	db *sqlx.DB
}

func NewPhoneNumberRepo(db *sqlx.DB) *PhoneNumberRepo {
	return &PhoneNumberRepo{db: db}
}

func (r *PhoneNumberRepo) Create(ctx context.Context, p *telephony.PhoneNumber) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO phone_numbers (id, tenant_id, number, display_name, usage, sip_trunk_id, ivr_flow_id, skill_group_id, status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.TenantID, p.Number, p.DisplayName, p.Usage, p.SIPTrunkID, p.IVRFlowID, p.SkillGroupID, p.Status, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *PhoneNumberRepo) GetByID(ctx context.Context, id int64) (*telephony.PhoneNumber, error) {
	var p telephony.PhoneNumber
	err := r.db.GetContext(ctx, &p, "SELECT * FROM phone_numbers WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (r *PhoneNumberRepo) GetByNumber(ctx context.Context, tenantID int64, number string) (*telephony.PhoneNumber, error) {
	var p telephony.PhoneNumber
	err := r.db.GetContext(ctx, &p, "SELECT * FROM phone_numbers WHERE tenant_id = ? AND number = ?", tenantID, number)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (r *PhoneNumberRepo) Update(ctx context.Context, p *telephony.PhoneNumber) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE phone_numbers SET display_name=?, usage=?, sip_trunk_id=?, ivr_flow_id=?, skill_group_id=?, status=?, updated_at=? WHERE id=?`,
		p.DisplayName, p.Usage, p.SIPTrunkID, p.IVRFlowID, p.SkillGroupID, p.Status, p.UpdatedAt, p.ID)
	return err
}

func (r *PhoneNumberRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*telephony.PhoneNumber, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM phone_numbers WHERE tenant_id = ?", tenantID)

	var numbers []*telephony.PhoneNumber
	err := r.db.SelectContext(ctx, &numbers,
		"SELECT * FROM phone_numbers WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return numbers, total, err
}

// CallNumberTagRepo

type CallNumberTagRepo struct {
	db *sqlx.DB
}

func NewCallNumberTagRepo(db *sqlx.DB) *CallNumberTagRepo {
	return &CallNumberTagRepo{db: db}
}

func (r *CallNumberTagRepo) Create(ctx context.Context, t *telephony.CallNumberTag) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO call_number_tags (id, tenant_id, number, tag, source, created_at) VALUES (?,?,?,?,?,?)",
		t.ID, t.TenantID, t.Number, t.Tag, t.Source, t.CreatedAt)
	return err
}

func (r *CallNumberTagRepo) ListByNumber(ctx context.Context, tenantID int64, number string) ([]*telephony.CallNumberTag, error) {
	var tags []*telephony.CallNumberTag
	err := r.db.SelectContext(ctx, &tags,
		"SELECT * FROM call_number_tags WHERE tenant_id = ? AND number = ?", tenantID, number)
	return tags, err
}

func (r *CallNumberTagRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*telephony.CallNumberTag, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM call_number_tags WHERE tenant_id = ?", tenantID)

	var tags []*telephony.CallNumberTag
	err := r.db.SelectContext(ctx, &tags,
		"SELECT * FROM call_number_tags WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return tags, total, err
}

func (r *CallNumberTagRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM call_number_tags WHERE id = ?", id)
	return err
}

// AutoTagRuleRepo

type AutoTagRuleRepo struct {
	db *sqlx.DB
}

func NewAutoTagRuleRepo(db *sqlx.DB) *AutoTagRuleRepo {
	return &AutoTagRuleRepo{db: db}
}

func (r *AutoTagRuleRepo) Create(ctx context.Context, rule *telephony.AutoTagRule) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO auto_tag_rules (id, tenant_id, name, match_type, match_value, tag, priority, is_active, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		rule.ID, rule.TenantID, rule.Name, rule.MatchType, rule.MatchValue, rule.Tag, rule.Priority, rule.IsActive, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (r *AutoTagRuleRepo) GetByID(ctx context.Context, id int64) (*telephony.AutoTagRule, error) {
	var rule telephony.AutoTagRule
	err := r.db.GetContext(ctx, &rule, "SELECT * FROM auto_tag_rules WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &rule, err
}

func (r *AutoTagRuleRepo) Update(ctx context.Context, rule *telephony.AutoTagRule) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE auto_tag_rules SET name=?, match_type=?, match_value=?, tag=?, priority=?, is_active=?, updated_at=? WHERE id=?",
		rule.Name, rule.MatchType, rule.MatchValue, rule.Tag, rule.Priority, rule.IsActive, rule.UpdatedAt, rule.ID)
	return err
}

func (r *AutoTagRuleRepo) ListActive(ctx context.Context, tenantID int64) ([]*telephony.AutoTagRule, error) {
	var rules []*telephony.AutoTagRule
	err := r.db.SelectContext(ctx, &rules,
		"SELECT * FROM auto_tag_rules WHERE tenant_id = ? AND is_active = 1 ORDER BY priority ASC", tenantID)
	return rules, err
}

func (r *AutoTagRuleRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*telephony.AutoTagRule, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM auto_tag_rules WHERE tenant_id = ?", tenantID)

	var rules []*telephony.AutoTagRule
	err := r.db.SelectContext(ctx, &rules,
		"SELECT * FROM auto_tag_rules WHERE tenant_id = ? ORDER BY priority ASC LIMIT ? OFFSET ?",
		tenantID, limit, offset)
	return rules, total, err
}

func (r *AutoTagRuleRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM auto_tag_rules WHERE id = ?", id)
	return err
}
