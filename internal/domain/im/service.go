package im

import (
	"context"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

var validChannelTypes = map[ChannelType]bool{
	ChannelTypeWebWidget: true, ChannelTypeApp: true, ChannelTypeMiniProg: true,
	ChannelTypeDingTalk: true, ChannelTypeEmail: true, ChannelTypeAPI: true,
	ChannelTypeWeChat: true, ChannelTypeWeibo: true,
}

var validContentTypes = map[ContentType]bool{
	ContentTypeText: true, ContentTypeImage: true, ContentTypeFile: true,
	ContentTypeAudio: true, ContentTypeVideo: true, ContentTypeCard: true,
	ContentTypeSystem: true,
}

// IMService handles IM channel, session, and message operations.
type IMService struct {
	channels    IMChannelRepository
	sessions    IMSessionRepository
	messages    IMMessageRepository
	maxChatSlots int
}

func NewIMService(
	channels IMChannelRepository,
	sessions IMSessionRepository,
	messages IMMessageRepository,
	maxChatSlots int,
) *IMService {
	if maxChatSlots <= 0 {
		maxChatSlots = 5
	}
	return &IMService{
		channels:    channels,
		sessions:    sessions,
		messages:    messages,
		maxChatSlots: maxChatSlots,
	}
}

// --- Channel ---

func (s *IMService) CreateChannel(ctx context.Context, tenantID int64, channelType ChannelType, name string, skillGroupID *int64) (*IMChannel, error) {
	if !validChannelTypes[channelType] {
		return nil, ErrInvalidChannelType
	}
	ch := &IMChannel{
		ID:           snowflake.NextID(),
		TenantID:     tenantID,
		ChannelType:  channelType,
		Name:         name,
		SkillGroupID: skillGroupID,
		Status:       ChannelStatusActive,
		CreatedAt:    time.Now(),
	}
	if err := s.channels.Create(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *IMService) GetChannel(ctx context.Context, id int64) (*IMChannel, error) {
	ch, err := s.channels.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}
	return ch, nil
}

func (s *IMService) UpdateChannel(ctx context.Context, ch *IMChannel) error {
	return s.channels.Update(ctx, ch)
}

func (s *IMService) ListChannels(ctx context.Context, tenantID int64) ([]*IMChannel, error) {
	return s.channels.List(ctx, tenantID)
}

// --- Session ---

type CreateSessionInput struct {
	TenantID   int64  `json:"tenant_id"`
	ChannelID  int64  `json:"channel_id"`
	VisitorID  string `json:"visitor_id"`
	CustomerID *int64 `json:"customer_id"`
}

func (s *IMService) CreateSession(ctx context.Context, in CreateSessionInput) (*IMSession, error) {
	ch, err := s.channels.GetByID(ctx, in.ChannelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}
	if ch.Status == ChannelStatusDisabled {
		return nil, ErrChannelDisabled
	}

	now := time.Now()
	sess := &IMSession{
		ID:           snowflake.NextID(),
		TenantID:     in.TenantID,
		ChannelID:    in.ChannelID,
		VisitorID:    in.VisitorID,
		CustomerID:   in.CustomerID,
		SkillGroupID: ch.SkillGroupID,
		Status:       SessionStatusWaiting,
		StartAt:      now,
		CreatedAt:    now,
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *IMService) AssignAgent(ctx context.Context, sessionID int64, agentUserID int64) error {
	sess, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrSessionNotFound
	}
	if sess.Status != SessionStatusWaiting {
		return ErrSessionNotWaiting
	}

	count, err := s.sessions.CountActiveByAgent(ctx, agentUserID)
	if err != nil {
		return err
	}
	if count >= s.maxChatSlots {
		return ErrMaxChatSlots
	}

	sess.AgentUserID = &agentUserID
	sess.Status = SessionStatusActive
	return s.sessions.Update(ctx, sess)
}

func (s *IMService) TransferSession(ctx context.Context, sessionID int64, toAgentUserID int64) error {
	sess, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrSessionNotFound
	}
	if sess.Status != SessionStatusActive {
		return ErrSessionNotActive
	}

	count, err := s.sessions.CountActiveByAgent(ctx, toAgentUserID)
	if err != nil {
		return err
	}
	if count >= s.maxChatSlots {
		return ErrMaxChatSlots
	}

	sess.AgentUserID = &toAgentUserID
	sess.Status = SessionStatusActive
	return s.sessions.Update(ctx, sess)
}

func (s *IMService) CloseSession(ctx context.Context, sessionID int64) error {
	sess, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrSessionNotFound
	}
	if sess.Status == SessionStatusClosed {
		return ErrSessionClosed
	}

	now := time.Now()
	sess.Status = SessionStatusClosed
	sess.EndAt = &now
	return s.sessions.Update(ctx, sess)
}

func (s *IMService) GetSession(ctx context.Context, id int64) (*IMSession, error) {
	sess, err := s.sessions.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

func (s *IMService) ListSessions(ctx context.Context, tenantID int64, offset, limit int) ([]*IMSession, error) {
	return s.sessions.List(ctx, tenantID, offset, limit)
}

// --- Message ---

func (s *IMService) SendMessage(ctx context.Context, sessionID int64, senderType SenderType, senderID string, contentType ContentType, content string) (*IMMessage, error) {
	sess, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	if sess.Status == SessionStatusClosed {
		return nil, ErrSessionClosed
	}
	if content == "" {
		return nil, ErrEmptyMessage
	}
	if !validContentTypes[contentType] {
		return nil, ErrInvalidContentType
	}

	msg := &IMMessage{
		SessionID:   sessionID,
		SenderType:  senderType,
		SenderID:    senderID,
		ContentType: contentType,
		Content:     content,
		CreatedAt:   time.Now(),
	}
	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *IMService) ListMessages(ctx context.Context, sessionID int64, offset, limit int) ([]*IMMessage, error) {
	return s.messages.ListBySession(ctx, sessionID, offset, limit)
}
