package email

import (
	"context"
	"errors"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/rs/zerolog"
)

// InboundInput represents a parsed inbound email webhook payload.
type InboundInput struct {
	TenantID  int64  `json:"tenant_id"`
	ChannelID int64  `json:"channel_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

// Service handles inbound email processing, creating IM sessions and messages.
type Service struct {
	imSvc  *im.IMService
	logger zerolog.Logger
}

func NewService(imSvc *im.IMService, logger zerolog.Logger) *Service {
	return &Service{imSvc: imSvc, logger: logger}
}

// ProcessInbound creates an IM session for an inbound email and adds the email body as the first message.
func (s *Service) ProcessInbound(ctx context.Context, in InboundInput) (*im.IMSession, error) {
	sess, err := s.imSvc.CreateSession(ctx, im.CreateSessionInput{
		TenantID:  in.TenantID,
		ChannelID: in.ChannelID,
		VisitorID: in.From,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("from", in.From).Msg("email: failed to create session")
		return nil, err
	}

	content := in.Subject
	if in.Body != "" {
		content = in.Subject + "\n\n" + in.Body
	}
	if content == "" {
		return nil, errors.New("email has no subject or body")
	}
	_, err = s.imSvc.SendMessage(ctx, sess.ID, im.SenderTypeVisitor, in.From, im.ContentTypeText, content)
	if err != nil {
		s.logger.Error().Err(err).Int64("session_id", sess.ID).Msg("email: failed to create initial message")
		return sess, err
	}

	s.logger.Info().Int64("session_id", sess.ID).Str("from", in.From).Msg("email: inbound processed")
	return sess, nil
}
