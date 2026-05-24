package outbound

import (
	"context"
	"fmt"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/internal/domain/telephony"
	"github.com/divord97/ccc/internal/infrastructure/esl"
)

type Service struct {
	callSvc    *call.CallService
	routingSvc *telephony.RoutingService
	cliSvc     *telephony.CLIPolicyService
	dncSvc     *integration.DNCService
	eslClient  *esl.Client
}

func NewService(
	callSvc *call.CallService,
	routingSvc *telephony.RoutingService,
	cliSvc *telephony.CLIPolicyService,
	dncSvc *integration.DNCService,
	eslClient *esl.Client,
) *Service {
	return &Service{
		callSvc:    callSvc,
		routingSvc: routingSvc,
		cliSvc:     cliSvc,
		dncSvc:     dncSvc,
		eslClient:  eslClient,
	}
}

type DialRequest struct {
	TenantID       int64
	AgentUserID    int64
	Callee         string
	MediaType      call.MediaType
	CLIPolicyID    *int64
	CampaignCaseID *int64
}

// Dial orchestrates an outbound call: DNC check → route match → CLI select → ESL originate.
func (s *Service) Dial(ctx context.Context, req DialRequest) (*call.Call, error) {
	// 1. DNC check
	if err := s.dncSvc.CheckDNC(ctx, req.TenantID, req.Callee); err != nil {
		return nil, fmt.Errorf("DNC blocked: %w", err)
	}

	// 2. Route matching
	rule, err := s.routingSvc.MatchRule(ctx, req.TenantID, req.Callee)
	if err != nil {
		return nil, fmt.Errorf("routing: %w", err)
	}

	// 3. CLI selection
	cli, err := s.cliSvc.SelectCLI(ctx, req.TenantID, req.CLIPolicyID, req.Callee)
	if err != nil {
		return nil, fmt.Errorf("CLI select: %w", err)
	}
	if cli == nil {
		return nil, fmt.Errorf("CLI select: no phone number available")
	}

	// 4. Create call record
	agentID := req.AgentUserID
	input := call.CreateCallInput{
		TenantID:       req.TenantID,
		MediaType:      req.MediaType,
		Caller:         cli.Number,
		Callee:         req.Callee,
		AgentUserID:    &agentID,
		PhoneNumberID:  &cli.ID,
		CampaignCaseID: req.CampaignCaseID,
	}
	if rule != nil {
		input.SIPTrunkID = &rule.SIPTrunkID
	}
	c, err := s.callSvc.CreateOutboundCall(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create call: %w", err)
	}

	return c, nil
}

type InternalDialRequest struct {
	TenantID      int64
	CallerAgentID int64
	CalleeAgentID int64
	CallerExt     string
	CalleeExt     string
	MediaType     call.MediaType
}

// DialInternal creates an internal (agent-to-agent) call.
func (s *Service) DialInternal(ctx context.Context, req InternalDialRequest) (*call.Call, error) {
	agentID := req.CallerAgentID
	c, err := s.callSvc.CreateInternalCall(ctx, call.CreateCallInput{
		TenantID:    req.TenantID,
		MediaType:   req.MediaType,
		Caller:      req.CallerExt,
		Callee:      req.CalleeExt,
		AgentUserID: &agentID,
	})
	if err != nil {
		return nil, err
	}

	// ESL bridge internal call
	if s.eslClient != nil {
		go func() {
			_, _ = s.eslClient.Originate(context.Background(), fmt.Sprintf("user/%s", req.CalleeExt), req.CallerExt, "default")
		}()
	}

	return c, nil
}
