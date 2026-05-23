package screenpop

import (
	"context"
	"fmt"
	"strings"

	"github.com/divord97/ccc/internal/domain/integration"
)

type Service struct {
	configs integration.ScreenPopConfigRepository
}

func NewService(configs integration.ScreenPopConfigRepository) *Service {
	return &Service{configs: configs}
}

type CallInfo struct {
	CallID       int64
	Caller       string
	Callee       string
	Direction    string
	SkillGroupID *int64
	AgentUserID  *int64
}

// BuildURLs generates screen pop URLs by substituting call variables into templates.
func (s *Service) BuildURLs(ctx context.Context, tenantID int64, info CallInfo) ([]string, error) {
	configs, _, err := s.configs.List(ctx, tenantID, 0, 5)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, cfg := range configs {
		if !cfg.IsActive {
			continue
		}
		url := s.substitute(cfg.URLTemplate, info)
		urls = append(urls, url)
	}
	return urls, nil
}

func (s *Service) substitute(tmpl string, info CallInfo) string {
	r := strings.NewReplacer(
		"${call_id}", fmt.Sprintf("%d", info.CallID),
		"${caller}", info.Caller,
		"${callee}", info.Callee,
		"${direction}", info.Direction,
	)
	if info.SkillGroupID != nil {
		r = strings.NewReplacer(
			"${call_id}", fmt.Sprintf("%d", info.CallID),
			"${caller}", info.Caller,
			"${callee}", info.Callee,
			"${direction}", info.Direction,
			"${skill_group_id}", fmt.Sprintf("%d", *info.SkillGroupID),
		)
	}
	return r.Replace(tmpl)
}
