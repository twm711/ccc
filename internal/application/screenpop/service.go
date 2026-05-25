package screenpop

import (
	"context"
	"fmt"
	"strings"

	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/domain/integration"
)

// IVRContextLoader returns the IVR session variables captured before transfer.
type IVRContextLoader interface {
	Load(ctx context.Context, callID int64) (map[string]string, error)
}

type Service struct {
	configs    integration.ScreenPopConfigRepository
	customers  *crm.CustomerService
	ivrContext IVRContextLoader
}

func NewService(configs integration.ScreenPopConfigRepository, customers *crm.CustomerService) *Service {
	return &Service{configs: configs, customers: customers}
}

// SetIVRContextLoader enables the screen pop to include the caller's IVR
// context (DTMF selections, captured variables). Optional — pop still works
// without it, just minus the IVR breadcrumbs.
func (s *Service) SetIVRContextLoader(l IVRContextLoader) {
	s.ivrContext = l
}

type CallInfo struct {
	CallID       int64
	Caller       string
	Callee       string
	Direction    string
	SkillGroupID *int64
	AgentUserID  *int64
}

type ScreenPopData struct {
	URLs         []string                   `json:"urls"`
	Customer     *crm.Customer              `json:"customer,omitempty"`
	Phones       []*crm.CustomerPhone       `json:"phones,omitempty"`
	Interactions []*crm.CustomerInteraction `json:"interactions,omitempty"`
	IVRContext   map[string]string          `json:"ivr_context,omitempty"`
}

// BuildScreenPop generates screen pop data: URLs + customer match + history.
func (s *Service) BuildScreenPop(ctx context.Context, tenantID int64, info CallInfo) (*ScreenPopData, error) {
	data := &ScreenPopData{}

	// Build URLs from templates
	configs, _, err := s.configs.List(ctx, tenantID, 0, 5)
	if err != nil {
		return nil, err
	}
	for _, cfg := range configs {
		if !cfg.IsActive {
			continue
		}
		url := s.substitute(cfg.URLTemplate, info)
		data.URLs = append(data.URLs, url)
	}

	// IVR context: DTMFs, prompts, captured variables before transfer.
	if s.ivrContext != nil && info.CallID > 0 {
		if vars, err := s.ivrContext.Load(ctx, info.CallID); err == nil && len(vars) > 0 {
			data.IVRContext = vars
		}
	}

	// Match customer by phone number
	if s.customers != nil {
		phone := info.Caller
		if info.Direction == "outbound" {
			phone = info.Callee
		}
		customer, err := s.customers.FindByPhone(ctx, tenantID, phone)
		if err == nil && customer != nil {
			data.Customer = customer
			phones, _ := s.customers.ListPhones(ctx, customer.ID)
			data.Phones = phones
			interactions, _ := s.customers.ListInteractions(ctx, customer.ID, 0, 10)
			data.Interactions = interactions
		}
	}

	return data, nil
}

// BuildURLs generates screen pop URLs (backward compatible).
func (s *Service) BuildURLs(ctx context.Context, tenantID int64, info CallInfo) ([]string, error) {
	data, err := s.BuildScreenPop(ctx, tenantID, info)
	if err != nil {
		return nil, err
	}
	return data.URLs, nil
}

func (s *Service) substitute(tmpl string, info CallInfo) string {
	pairs := []string{
		"${call_id}", fmt.Sprintf("%d", info.CallID),
		"${caller}", info.Caller,
		"${callee}", info.Callee,
		"${direction}", info.Direction,
	}
	if info.SkillGroupID != nil {
		pairs = append(pairs, "${skill_group_id}", fmt.Sprintf("%d", *info.SkillGroupID))
	}
	if info.AgentUserID != nil {
		pairs = append(pairs, "${agent_user_id}", fmt.Sprintf("%d", *info.AgentUserID))
	}
	return strings.NewReplacer(pairs...).Replace(tmpl)
}
