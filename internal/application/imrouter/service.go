package imrouter

import (
	"context"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/domain/im"
	"github.com/rs/zerolog"
)

// Service handles IM session routing: assigns waiting sessions to available agents.
type Service struct {
	imSvc          *im.IMService
	presenceSvc    *identity.AgentPresenceService
	skillGroupSvc  *identity.SkillGroupService
	logger         zerolog.Logger
}

func NewService(
	imSvc *im.IMService,
	presenceSvc *identity.AgentPresenceService,
	skillGroupSvc *identity.SkillGroupService,
	logger zerolog.Logger,
) *Service {
	return &Service{
		imSvc:         imSvc,
		presenceSvc:   presenceSvc,
		skillGroupSvc: skillGroupSvc,
		logger:        logger,
	}
}

// RouteSession attempts to assign an agent to a waiting session based on its skill group.
func (s *Service) RouteSession(ctx context.Context, sessionID int64, agentUserID int64) error {
	if err := s.imSvc.AssignAgent(ctx, sessionID, agentUserID); err != nil {
		s.logger.Warn().Err(err).Int64("session_id", sessionID).Int64("agent", agentUserID).Msg("im: assign agent failed")
		return err
	}
	s.logger.Info().Int64("session_id", sessionID).Int64("agent", agentUserID).Msg("im: session assigned to agent")
	return nil
}
