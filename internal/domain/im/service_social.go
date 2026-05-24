package im

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

var (
	ErrSocialConfigNotFound    = errors.New("social channel config not found")
	ErrSocialAppIDEmpty        = errors.New("app_id is required")
	ErrSocialAppSecretEmpty    = errors.New("app_secret is required")
	ErrSocialTokenEmpty        = errors.New("webhook token is required")
	ErrSocialInvalidPlatform   = errors.New("invalid social platform")
	ErrSocialSignatureInvalid  = errors.New("webhook signature verification failed")
	ErrSocialOpenIDEmpty       = errors.New("open_id is required")
	ErrSocialMessageEmpty      = errors.New("message content is required")
)

var validPlatforms = map[SocialPlatform]bool{
	PlatformWeChat: true,
	PlatformWeibo:  true,
}

// SocialChannelService manages social channel configs and message processing.
type SocialChannelService struct {
	configs  SocialChannelConfigRepository
	channels IMChannelRepository
	sessions IMSessionRepository
	messages IMMessageRepository
}

func NewSocialChannelService(
	configs SocialChannelConfigRepository,
	channels IMChannelRepository,
	sessions IMSessionRepository,
	messages IMMessageRepository,
) *SocialChannelService {
	return &SocialChannelService{
		configs:  configs,
		channels: channels,
		sessions: sessions,
		messages: messages,
	}
}

// CreateConfig creates a social channel config linked to an existing IM channel.
type CreateSocialConfigInput struct {
	TenantID       int64          `json:"tenant_id"`
	ChannelID      int64          `json:"channel_id"`
	Platform       SocialPlatform `json:"platform"`
	AppID          string         `json:"app_id"`
	AppSecret      string         `json:"app_secret"`
	Token          string         `json:"token"`
	EncodingAESKey string         `json:"encoding_aes_key"`
}

func (s *SocialChannelService) CreateConfig(ctx context.Context, in CreateSocialConfigInput) (*SocialChannelConfig, error) {
	if !validPlatforms[in.Platform] {
		return nil, ErrSocialInvalidPlatform
	}
	if in.AppID == "" {
		return nil, ErrSocialAppIDEmpty
	}
	if in.AppSecret == "" {
		return nil, ErrSocialAppSecretEmpty
	}
	if in.Token == "" {
		return nil, ErrSocialTokenEmpty
	}

	ch, err := s.channels.GetByID(ctx, in.ChannelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}

	now := time.Now()
	cfg := &SocialChannelConfig{
		ID:             snowflake.NextID(),
		TenantID:       in.TenantID,
		ChannelID:      in.ChannelID,
		Platform:       in.Platform,
		AppID:          in.AppID,
		AppSecret:      in.AppSecret,
		Token:          in.Token,
		EncodingAESKey: in.EncodingAESKey,
		IsVerified:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.configs.Create(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *SocialChannelService) GetConfig(ctx context.Context, channelID int64) (*SocialChannelConfig, error) {
	cfg, err := s.configs.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrSocialConfigNotFound
	}
	return cfg, nil
}

func (s *SocialChannelService) DeleteConfig(ctx context.Context, id int64) error {
	return s.configs.Delete(ctx, id)
}

// VerifyWeChatSignature validates a WeChat server verification request.
// WeChat sends signature=sha1(sort(token, timestamp, nonce)).
func (s *SocialChannelService) VerifyWeChatSignature(token, timestamp, nonce, signature string) bool {
	strs := []string{token, timestamp, nonce}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	return expected == signature
}

// VerifyWeiboSignature validates a Weibo webhook signature (simplified HMAC check).
func (s *SocialChannelService) VerifyWeiboSignature(appSecret, body, signature string) bool {
	h := sha1.New()
	h.Write([]byte(appSecret + body))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	return expected == signature
}

// ProcessInboundMessage converts a social platform message into an IM session + message.
func (s *SocialChannelService) ProcessInboundMessage(ctx context.Context, channelID int64, msg SocialMessage) (*IMSession, *IMMessage, error) {
	if msg.OpenID == "" {
		return nil, nil, ErrSocialOpenIDEmpty
	}
	if msg.Content == "" && msg.MediaURL == "" {
		return nil, nil, ErrSocialMessageEmpty
	}

	ch, err := s.channels.GetByID(ctx, channelID)
	if err != nil {
		return nil, nil, err
	}
	if ch == nil {
		return nil, nil, ErrChannelNotFound
	}
	if ch.Status == ChannelStatusDisabled {
		return nil, nil, ErrChannelDisabled
	}

	now := time.Now()
	sess := &IMSession{
		ID:           snowflake.NextID(),
		TenantID:     ch.TenantID,
		ChannelID:    channelID,
		VisitorID:    msg.OpenID,
		SkillGroupID: ch.SkillGroupID,
		Status:       SessionStatusWaiting,
		StartAt:      now,
		CreatedAt:    now,
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, nil, err
	}

	ct := msg.ContentType
	if ct == "" {
		ct = ContentTypeText
	}
	content := msg.Content
	if content == "" {
		content = msg.MediaURL
	}

	imMsg := &IMMessage{
		ID:          snowflake.NextID(),
		SessionID:   sess.ID,
		SenderType:  SenderTypeVisitor,
		SenderID:    msg.OpenID,
		ContentType: ct,
		Content:     content,
		CreatedAt:   now,
	}
	if err := s.messages.Create(ctx, imMsg); err != nil {
		return nil, nil, err
	}

	return sess, imMsg, nil
}

// MarkVerified marks a social channel config as verified after webhook handshake.
func (s *SocialChannelService) MarkVerified(ctx context.Context, channelID int64) error {
	cfg, err := s.configs.GetByChannelID(ctx, channelID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return ErrSocialConfigNotFound
	}
	cfg.IsVerified = true
	cfg.UpdatedAt = time.Now()
	return s.configs.Update(ctx, cfg)
}
