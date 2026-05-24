package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/jmoiron/sqlx"
)

type SocialChannelConfigRepo struct{ db *sqlx.DB }

func NewSocialChannelConfigRepo(db *sqlx.DB) *SocialChannelConfigRepo {
	return &SocialChannelConfigRepo{db: db}
}

func (r *SocialChannelConfigRepo) Create(ctx context.Context, c *im.SocialChannelConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO social_channel_configs (id,tenant_id,channel_id,platform,app_id,app_secret,token,encoding_aes_key,webhook_url,is_verified,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.ChannelID, c.Platform, c.AppID, c.AppSecret, c.Token, c.EncodingAESKey, c.WebhookURL, c.IsVerified, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *SocialChannelConfigRepo) GetByChannelID(ctx context.Context, channelID int64) (*im.SocialChannelConfig, error) {
	var c im.SocialChannelConfig
	err := r.db.GetContext(ctx, &c, "SELECT * FROM social_channel_configs WHERE channel_id=?", channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SocialChannelConfigRepo) GetByPlatformAndAppID(ctx context.Context, platform im.SocialPlatform, appID string) (*im.SocialChannelConfig, error) {
	var c im.SocialChannelConfig
	err := r.db.GetContext(ctx, &c, "SELECT * FROM social_channel_configs WHERE platform=? AND app_id=?", platform, appID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SocialChannelConfigRepo) Update(ctx context.Context, c *im.SocialChannelConfig) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE social_channel_configs SET app_id=?,app_secret=?,token=?,encoding_aes_key=?,webhook_url=?,is_verified=?,updated_at=?
		 WHERE id=?`,
		c.AppID, c.AppSecret, c.Token, c.EncodingAESKey, c.WebhookURL, c.IsVerified, c.UpdatedAt, c.ID)
	return err
}

func (r *SocialChannelConfigRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM social_channel_configs WHERE id=?", id)
	return err
}

var _ im.SocialChannelConfigRepository = (*SocialChannelConfigRepo)(nil)
