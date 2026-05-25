package imrouter

import (
	"context"
	"fmt"

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

// AutoRouteIMSession picks an idle agent from the session's skill group and assigns them.
// Accepts *im.IMSession directly so it satisfies the email.SessionRouter interface.
func (s *Service) AutoRouteSession(ctx context.Context, sess *im.IMSession) error {
	if sess.SkillGroupID == nil {
		return fmt.Errorf("im router: session %d has no skill group", sess.ID)
	}
	return s.autoRouteBySkillGroup(ctx, sess.ID, *sess.SkillGroupID)
}

// autoRouteBySkillGroup picks an idle agent from the given skill group and assigns them.
func (s *Service) autoRouteBySkillGroup(ctx context.Context, sessionID int64, skillGroupID int64) error {
	members, err := s.skillGroupSvc.GetMembers(ctx, skillGroupID)
	if err != nil {
		return fmt.Errorf("im router: list members: %w", err)
	}

	for _, m := range members {
		p, err := s.presenceSvc.GetPresence(ctx, m.AgentID)
		if err != nil || p == nil {
			continue
		}
		if p.Status != identity.PresenceIdle {
			continue
		}
		if err := s.imSvc.AssignAgent(ctx, sessionID, m.AgentID); err != nil {
			continue
		}
		s.logger.Info().Int64("session_id", sessionID).Int64("agent", m.AgentID).Int64("skill_group", skillGroupID).Msg("im: auto-routed session to idle agent")
		return nil
	}

	s.logger.Warn().Int64("session_id", sessionID).Int64("skill_group", skillGroupID).Msg("im: no idle agent available for auto-routing")
	return fmt.Errorf("im router: no idle agent in skill group %d", skillGroupID)
}
