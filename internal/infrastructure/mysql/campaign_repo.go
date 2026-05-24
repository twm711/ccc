package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/jmoiron/sqlx"
)

type CampaignRepo struct {
	db *sqlx.DB
}

func NewCampaignRepo(db *sqlx.DB) *CampaignRepo {
	return &CampaignRepo{db: db}
}

func (r *CampaignRepo) Create(ctx context.Context, c *campaign.Campaign) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO campaigns (id, tenant_id, name, dialing_mode, skill_group_id, cli_policy_id,
		 status, ratio_multiplier, max_abandon_rate, preview_timeout_sec, concurrent_limit,
		 max_retries, retry_interval_sec, total_cases, completed_cases, success_cases, failed_cases,
		 created_at, updated_at, started_at, completed_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.Name, c.DialingMode, c.SkillGroupID, c.CLIPolicyID,
		c.Status, c.RatioMultiplier, c.MaxAbandonRate, c.PreviewTimeoutSec, c.ConcurrentLimit,
		c.MaxRetries, c.RetryIntervalSec, c.TotalCases, c.CompletedCases, c.SuccessCases, c.FailedCases,
		c.CreatedAt, c.UpdatedAt, c.StartedAt, c.CompletedAt)
	return err
}

func (r *CampaignRepo) GetByID(ctx context.Context, id int64) (*campaign.Campaign, error) {
	var c campaign.Campaign
	err := r.db.GetContext(ctx, &c, "SELECT * FROM campaigns WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *CampaignRepo) Update(ctx context.Context, c *campaign.Campaign) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE campaigns SET name=?, status=?, ratio_multiplier=?, max_abandon_rate=?,
		 preview_timeout_sec=?, concurrent_limit=?, max_retries=?, retry_interval_sec=?,
		 total_cases=?, completed_cases=?, success_cases=?, failed_cases=?,
		 updated_at=?, started_at=?, completed_at=?
		 WHERE id=?`,
		c.Name, c.Status, c.RatioMultiplier, c.MaxAbandonRate,
		c.PreviewTimeoutSec, c.ConcurrentLimit, c.MaxRetries, c.RetryIntervalSec,
		c.TotalCases, c.CompletedCases, c.SuccessCases, c.FailedCases,
		c.UpdatedAt, c.StartedAt, c.CompletedAt, c.ID)
	return err
}

func (r *CampaignRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*campaign.Campaign, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM campaigns WHERE tenant_id = ?", tenantID)
	var items []*campaign.Campaign
	err := r.db.SelectContext(ctx, &items, "SELECT * FROM campaigns WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", tenantID, limit, offset)
	return items, total, err
}

type CampaignCaseRepo struct {
	db *sqlx.DB
}

func NewCampaignCaseRepo(db *sqlx.DB) *CampaignCaseRepo {
	return &CampaignCaseRepo{db: db}
}

func (r *CampaignCaseRepo) Create(ctx context.Context, c *campaign.CampaignCase) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO campaign_cases (id, campaign_id, tenant_id, customer_name, phone_number, custom_data,
		 status, attempt_count, agent_user_id, duration_sec, disposition_code,
		 next_attempt_at, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.CampaignID, c.TenantID, c.CustomerName, c.PhoneNumber, c.CustomData,
		c.Status, c.AttemptCount, c.AgentUserID, c.DurationSec, c.DispositionCode,
		c.NextAttemptAt, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CampaignCaseRepo) GetByID(ctx context.Context, id int64) (*campaign.CampaignCase, error) {
	var c campaign.CampaignCase
	err := r.db.GetContext(ctx, &c, "SELECT * FROM campaign_cases WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *CampaignCaseRepo) Update(ctx context.Context, c *campaign.CampaignCase) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE campaign_cases SET status=?, attempt_count=?, agent_user_id=?,
		 duration_sec=?, disposition_code=?, next_attempt_at=?, updated_at=?
		 WHERE id=?`,
		c.Status, c.AttemptCount, c.AgentUserID,
		c.DurationSec, c.DispositionCode, c.NextAttemptAt, c.UpdatedAt, c.ID)
	return err
}

func (r *CampaignCaseRepo) BulkCreate(ctx context.Context, cases []*campaign.CampaignCase) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	for _, c := range cases {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO campaign_cases (id, campaign_id, tenant_id, customer_name, phone_number, custom_data,
			 status, attempt_count, agent_user_id, duration_sec, disposition_code,
			 next_attempt_at, created_at, updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			c.ID, c.CampaignID, c.TenantID, c.CustomerName, c.PhoneNumber, c.CustomData,
			c.Status, c.AttemptCount, c.AgentUserID, c.DurationSec, c.DispositionCode,
			c.NextAttemptAt, c.CreatedAt, c.UpdatedAt)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *CampaignCaseRepo) ListByCampaign(ctx context.Context, campaignID int64, offset, limit int) ([]*campaign.CampaignCase, int64, error) {
	var total int64
	_ = r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM campaign_cases WHERE campaign_id = ?", campaignID)
	var items []*campaign.CampaignCase
	err := r.db.SelectContext(ctx, &items, "SELECT * FROM campaign_cases WHERE campaign_id = ? ORDER BY id LIMIT ? OFFSET ?", campaignID, limit, offset)
	return items, total, err
}

func (r *CampaignCaseRepo) GetNextPending(ctx context.Context, campaignID int64) (*campaign.CampaignCase, error) {
	var c campaign.CampaignCase
	err := r.db.GetContext(ctx, &c,
		"SELECT * FROM campaign_cases WHERE campaign_id = ? AND status = 'pending' AND (next_attempt_at IS NULL OR next_attempt_at <= NOW()) ORDER BY id LIMIT 1", campaignID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *CampaignCaseRepo) CountByStatus(ctx context.Context, campaignID int64) (pending, completed, failed int, err error) {
	type counts struct {
		Pending   int `db:"pending"`
		Completed int `db:"completed"`
		Failed    int `db:"failed"`
	}
	var c counts
	err = r.db.GetContext(ctx, &c,
		`SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS completed,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed
		FROM campaign_cases WHERE campaign_id = ?`, campaignID)
	return c.Pending, c.Completed, c.Failed, err
}
