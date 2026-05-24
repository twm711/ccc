package im

import "context"

type SocialChannelConfigRepository interface {
	Create(ctx context.Context, c *SocialChannelConfig) error
	GetByChannelID(ctx context.Context, channelID int64) (*SocialChannelConfig, error)
	GetByPlatformAndAppID(ctx context.Context, platform SocialPlatform, appID string) (*SocialChannelConfig, error)
	Update(ctx context.Context, c *SocialChannelConfig) error
	Delete(ctx context.Context, id int64) error
}
