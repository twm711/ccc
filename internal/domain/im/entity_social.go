package im

import "time"

const (
	ChannelTypeWeChat ChannelType = "wechat"
	ChannelTypeWeibo  ChannelType = "weibo"
)

type SocialPlatform string

const (
	PlatformWeChat SocialPlatform = "wechat"
	PlatformWeibo  SocialPlatform = "weibo"
)

// SocialChannelConfig stores platform-specific credentials for a social channel.
type SocialChannelConfig struct {
	ID             int64          `db:"id" json:"id"`
	TenantID       int64          `db:"tenant_id" json:"tenant_id"`
	ChannelID      int64          `db:"channel_id" json:"channel_id"`
	Platform       SocialPlatform `db:"platform" json:"platform"`
	AppID          string         `db:"app_id" json:"app_id"`
	AppSecret      string         `db:"app_secret" json:"-"`
	Token          string         `db:"token" json:"-"`
	EncodingAESKey string         `db:"encoding_aes_key" json:"-"`
	WebhookURL     string         `db:"webhook_url" json:"webhook_url"`
	IsVerified     bool           `db:"is_verified" json:"is_verified"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`
}

// SocialMessage represents a normalized inbound message from a social platform.
type SocialMessage struct {
	Platform    SocialPlatform
	OpenID      string
	ContentType ContentType
	Content     string
	MediaURL    string
	MsgID       string
	Timestamp   int64
}
