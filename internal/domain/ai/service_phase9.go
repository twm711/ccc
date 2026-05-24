package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/divord97/ccc/pkg/snowflake"
)

// DigitalEmployeeService manages digital employee (AI bot) entities and scenes.
type DigitalEmployeeService struct {
	employees DigitalEmployeeRepository
	scenes    DigitalEmployeeSceneRepository
}

func NewDigitalEmployeeService(employees DigitalEmployeeRepository, scenes DigitalEmployeeSceneRepository) *DigitalEmployeeService {
	return &DigitalEmployeeService{employees: employees, scenes: scenes}
}

type CreateDigitalEmployeeInput struct {
	TenantID    int64  `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AvatarURL   string `json:"avatar_url"`
}

func (s *DigitalEmployeeService) Create(ctx context.Context, in CreateDigitalEmployeeInput) (*DigitalEmployee, error) {
	now := time.Now()
	de := &DigitalEmployee{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		Name:        in.Name,
		Description: in.Description,
		AvatarURL:   in.AvatarURL,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.employees.Create(ctx, de); err != nil {
		return nil, err
	}
	return de, nil
}

func (s *DigitalEmployeeService) GetByID(ctx context.Context, id int64) (*DigitalEmployee, error) {
	de, err := s.employees.GetByID(ctx, id)
	if err != nil || de == nil {
		return nil, ErrDigitalEmployeeNotFound
	}
	return de, nil
}

func (s *DigitalEmployeeService) Update(ctx context.Context, de *DigitalEmployee) error {
	de.UpdatedAt = time.Now()
	return s.employees.Update(ctx, de)
}

func (s *DigitalEmployeeService) List(ctx context.Context, tenantID int64) ([]*DigitalEmployee, error) {
	return s.employees.List(ctx, tenantID)
}

type CreateSceneInput struct {
	DigitalEmployeeID  int64  `json:"digital_employee_id"`
	TenantID           int64  `json:"tenant_id"`
	Name               string `json:"name"`
	Intents            string `json:"intents"`
	FAQs               string `json:"faqs"`
	TransferSkillGroup *int64 `json:"transfer_skill_group"`
}

func (s *DigitalEmployeeService) CreateScene(ctx context.Context, in CreateSceneInput) (*DigitalEmployeeScene, error) {
	if in.Intents != "" && !json.Valid([]byte(in.Intents)) {
		return nil, ErrInvalidIntentConfig
	}
	now := time.Now()
	scene := &DigitalEmployeeScene{
		ID:                 snowflake.NextID(),
		DigitalEmployeeID:  in.DigitalEmployeeID,
		TenantID:           in.TenantID,
		Name:               in.Name,
		Intents:            in.Intents,
		FAQs:               in.FAQs,
		TransferSkillGroup: in.TransferSkillGroup,
		Status:             SceneStatusDraft,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.scenes.Create(ctx, scene); err != nil {
		return nil, err
	}
	return scene, nil
}

func (s *DigitalEmployeeService) PublishScene(ctx context.Context, sceneID int64) (*DigitalEmployeeScene, error) {
	scene, err := s.scenes.GetByID(ctx, sceneID)
	if err != nil || scene == nil {
		return nil, ErrSceneNotFound
	}
	if scene.Status == SceneStatusPublished {
		return nil, ErrSceneAlreadyPublished
	}
	scene.Status = SceneStatusPublished
	scene.UpdatedAt = time.Now()
	if err := s.scenes.Update(ctx, scene); err != nil {
		return nil, err
	}
	return scene, nil
}

func (s *DigitalEmployeeService) ListScenes(ctx context.Context, digitalEmployeeID int64) ([]*DigitalEmployeeScene, error) {
	return s.scenes.List(ctx, digitalEmployeeID)
}

// IntentMatchResult represents the result of intent matching.
type IntentMatchResult struct {
	Matched    bool   `json:"matched"`
	IntentName string `json:"intent_name,omitempty"`
	Response   string `json:"response,omitempty"`
	Transfer   bool   `json:"transfer"`
}

// IntentConfig represents a single intent configuration entry.
type IntentConfig struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Response string   `json:"response"`
	Transfer bool     `json:"transfer"`
}

// MatchIntent checks user input against a scene's intent configuration.
func (s *DigitalEmployeeService) MatchIntent(ctx context.Context, sceneID int64, userInput string) (*IntentMatchResult, error) {
	scene, err := s.scenes.GetByID(ctx, sceneID)
	if err != nil || scene == nil {
		return nil, ErrSceneNotFound
	}

	var intents []IntentConfig
	if scene.Intents != "" {
		if err := json.Unmarshal([]byte(scene.Intents), &intents); err != nil {
			return &IntentMatchResult{Matched: false}, nil
		}
	}

	for _, intent := range intents {
		for _, kw := range intent.Keywords {
			if containsIgnoreCase(userInput, kw) {
				return &IntentMatchResult{
					Matched:    true,
					IntentName: intent.Name,
					Response:   intent.Response,
					Transfer:   intent.Transfer,
				}, nil
			}
		}
	}

	return &IntentMatchResult{Matched: false}, nil
}

// containsIgnoreCase checks if s contains substr (case-insensitive, Unicode-safe).
func containsIgnoreCase(s, substr string) bool {
	return len(substr) > 0 && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// QALLMProvider is an optional LLM interface for LLM-type QA rules.
type QALLMProvider interface {
	QAInspectLLM(ctx context.Context, transcript, prompt string) (float64, string, error)
}

// QualityInspectionService manages QA rules, schemes, and results.
type QualityInspectionService struct {
	rules   QARuleRepository
	schemes QASchemeRepository
	results QAResultRepository
	llm     QALLMProvider
}

func NewQualityInspectionService(rules QARuleRepository, schemes QASchemeRepository, results QAResultRepository) *QualityInspectionService {
	return &QualityInspectionService{rules: rules, schemes: schemes, results: results}
}

// SetLLMProvider sets an LLM provider for LLM-type QA rules.
func (s *QualityInspectionService) SetLLMProvider(p QALLMProvider) {
	s.llm = p
}

type CreateQARuleInput struct {
	TenantID int64  `json:"tenant_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Config   string `json:"config"`
	Severity string `json:"severity"`
}

func (s *QualityInspectionService) CreateRule(ctx context.Context, in CreateQARuleInput) (*QARule, error) {
	if !IsValidQARuleType(in.Type) {
		return nil, ErrInvalidQARuleType
	}
	if in.Severity == "" {
		in.Severity = "warning"
	}
	now := time.Now()
	rule := &QARule{
		ID:        snowflake.NextID(),
		TenantID:  in.TenantID,
		Name:      in.Name,
		Type:      in.Type,
		Config:    in.Config,
		Severity:  in.Severity,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.rules.Create(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *QualityInspectionService) GetRule(ctx context.Context, id int64) (*QARule, error) {
	r, err := s.rules.GetByID(ctx, id)
	if err != nil || r == nil {
		return nil, ErrQARuleNotFound
	}
	return r, nil
}

func (s *QualityInspectionService) UpdateRule(ctx context.Context, rule *QARule) error {
	rule.UpdatedAt = time.Now()
	return s.rules.Update(ctx, rule)
}

func (s *QualityInspectionService) DeleteRule(ctx context.Context, id int64) error {
	return s.rules.Delete(ctx, id)
}

func (s *QualityInspectionService) ListRules(ctx context.Context, tenantID int64) ([]*QARule, error) {
	return s.rules.List(ctx, tenantID)
}

type CreateQASchemeInput struct {
	TenantID  int64              `json:"tenant_id"`
	Name      string             `json:"name"`
	RuleIDs   []SchemeRuleWeight `json:"rule_ids"`
	IsDefault bool               `json:"is_default"`
}

func (s *QualityInspectionService) CreateScheme(ctx context.Context, in CreateQASchemeInput) (*QAScheme, error) {
	ruleJSON, err := json.Marshal(in.RuleIDs)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	scheme := &QAScheme{
		ID:        snowflake.NextID(),
		TenantID:  in.TenantID,
		Name:      in.Name,
		RuleIDs:   string(ruleJSON),
		IsDefault: in.IsDefault,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.schemes.Create(ctx, scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func (s *QualityInspectionService) GetScheme(ctx context.Context, id int64) (*QAScheme, error) {
	scheme, err := s.schemes.GetByID(ctx, id)
	if err != nil || scheme == nil {
		return nil, ErrQASchemeNotFound
	}
	return scheme, nil
}

func (s *QualityInspectionService) UpdateScheme(ctx context.Context, scheme *QAScheme) error {
	scheme.UpdatedAt = time.Now()
	return s.schemes.Update(ctx, scheme)
}

func (s *QualityInspectionService) DeleteScheme(ctx context.Context, id int64) error {
	return s.schemes.Delete(ctx, id)
}

func (s *QualityInspectionService) ListSchemes(ctx context.Context, tenantID int64) ([]*QAScheme, error) {
	return s.schemes.List(ctx, tenantID)
}

// RunInspection runs quality inspection on a transcript using a scheme.
func (s *QualityInspectionService) RunInspection(ctx context.Context, tenantID, callID, schemeID int64, transcript string) (*QAResult, error) {
	if transcript == "" {
		return nil, ErrEmptyTranscript
	}

	scheme, err := s.schemes.GetByID(ctx, schemeID)
	if err != nil || scheme == nil {
		return nil, ErrQASchemeNotFound
	}

	var weights []SchemeRuleWeight
	if err := json.Unmarshal([]byte(scheme.RuleIDs), &weights); err != nil {
		return nil, err
	}

	ruleIDs := make([]int64, len(weights))
	for i, w := range weights {
		ruleIDs[i] = w.RuleID
	}
	rules, err := s.rules.ListByIDs(ctx, ruleIDs)
	if err != nil {
		return nil, err
	}

	ruleMap := make(map[int64]*QARule, len(rules))
	for _, r := range rules {
		ruleMap[r.ID] = r
	}

	var ruleResults []QARuleResult
	var totalScore, totalWeight float64

	for _, w := range weights {
		rule, ok := ruleMap[w.RuleID]
		if !ok {
			continue
		}
		rr := evaluateRule(ctx, rule, transcript, s.llm)
		ruleResults = append(ruleResults, rr)
		totalScore += rr.Score * w.Weight
		totalWeight += w.Weight
	}

	finalScore := float64(0)
	if totalWeight > 0 {
		finalScore = totalScore / totalWeight
	}

	detailsJSON, _ := json.Marshal(ruleResults)

	now := time.Now()
	result := &QAResult{
		ID:        snowflake.NextID(),
		TenantID:  tenantID,
		CallID:    callID,
		SchemeID:  schemeID,
		Score:     finalScore,
		Details:   string(detailsJSON),
		Status:    QAResultStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.results.Create(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

// evaluateRule evaluates a single QA rule against a transcript.
func evaluateRule(ctx context.Context, rule *QARule, transcript string, llm QALLMProvider) QARuleResult {
	rr := QARuleResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		RuleType: rule.Type,
	}

	switch rule.Type {
	case QARuleTypeKeyword:
		return evaluateKeywordRule(rule, transcript)
	case QARuleTypeRegex:
		return evaluateRegexRule(rule, transcript)
	case QARuleTypeSilence:
		return evaluateSilenceRule(rule, transcript)
	case QARuleTypeSpeed:
		return evaluateSpeedRule(rule, transcript)
	case QARuleTypeInterruption:
		return evaluateInterruptionRule(rule, transcript)
	case QARuleTypeEnergy:
		return evaluateEnergyRule(rule, transcript)
	case QARuleTypeDuration:
		return evaluateDurationRule(rule, transcript)
	case QARuleTypeEntity:
		return evaluateEntityRule(rule, transcript)
	case QARuleTypeRole:
		return evaluateRoleRule(rule, transcript)
	case QARuleTypeAbnormalHangup:
		return evaluateAbnormalHangupRule(rule, transcript)
	case QARuleTypeLLM:
		return evaluateLLMRule(ctx, rule, transcript, llm)
	default:
		rr.Passed = true
		rr.Score = 100
		rr.Detail = "unknown rule type, passed by default"
	}
	return rr
}

// KeywordRuleConfig represents the JSON config for a keyword rule.
type KeywordRuleConfig struct {
	Keywords []string `json:"keywords"`
	Require  string   `json:"require"` // "present" or "absent"
}

func evaluateKeywordRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		RuleType: rule.Type,
	}

	var cfg KeywordRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Passed = false
		rr.Score = 0
		rr.Detail = "invalid keyword config"
		return rr
	}

	for _, kw := range cfg.Keywords {
		found := containsIgnoreCase(transcript, kw)
		if cfg.Require == "absent" && found {
			rr.Passed = false
			rr.Score = 0
			rr.Detail = "forbidden keyword found: " + kw
			return rr
		}
		if cfg.Require == "present" && found {
			rr.Passed = true
			rr.Score = 100
			rr.Detail = "required keyword found: " + kw
			return rr
		}
	}

	if cfg.Require == "present" {
		rr.Passed = false
		rr.Score = 0
		rr.Detail = "required keyword not found"
	} else {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = "no forbidden keywords found"
	}
	return rr
}

// RegexRuleConfig represents the JSON config for a regex rule.
type RegexRuleConfig struct {
	Pattern string `json:"pattern"`
	Require string `json:"require"` // "match" or "no_match"
}

func evaluateRegexRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg RegexRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid regex config"
		return rr
	}

	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		rr.Detail = "invalid regex pattern: " + err.Error()
		return rr
	}

	matched := re.MatchString(transcript)
	if cfg.Require == "no_match" {
		rr.Passed = !matched
		if rr.Passed {
			rr.Score = 100
			rr.Detail = "pattern correctly absent"
		} else {
			rr.Detail = "forbidden pattern matched: " + re.FindString(transcript)
		}
	} else {
		rr.Passed = matched
		if rr.Passed {
			rr.Score = 100
			rr.Detail = "required pattern matched"
		} else {
			rr.Detail = "required pattern not found"
		}
	}
	return rr
}

// SilenceRuleConfig specifies silence detection thresholds.
// The transcript is expected to contain metadata markers like [silence:5.2s].
type SilenceRuleConfig struct {
	MaxSilenceSec float64 `json:"max_silence_sec"`
}

func evaluateSilenceRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg SilenceRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid silence config"
		return rr
	}
	if cfg.MaxSilenceSec <= 0 {
		cfg.MaxSilenceSec = 5.0
	}

	// Parse [silence:Xs] markers from transcript metadata.
	re := regexp.MustCompile(`\[silence:([\d.]+)s\]`)
	matches := re.FindAllStringSubmatch(transcript, -1)
	var maxFound float64
	for _, m := range matches {
		var dur float64
		if _, err := fmt.Sscanf(m[1], "%f", &dur); err == nil && dur > maxFound {
			maxFound = dur
		}
	}

	if maxFound > cfg.MaxSilenceSec {
		rr.Detail = fmt.Sprintf("long silence detected: %.1fs (max: %.1fs)", maxFound, cfg.MaxSilenceSec)
	} else {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = fmt.Sprintf("no excessive silence (max found: %.1fs, limit: %.1fs)", maxFound, cfg.MaxSilenceSec)
	}
	return rr
}

// SpeedRuleConfig specifies speaking speed thresholds.
// The transcript is expected to contain metadata like [speed:180wpm] or uses word count / duration.
type SpeedRuleConfig struct {
	MinWordsPerMin float64 `json:"min_words_per_min"`
	MaxWordsPerMin float64 `json:"max_words_per_min"`
}

func evaluateSpeedRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg SpeedRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid speed config"
		return rr
	}

	// Try to parse [speed:Xwpm] marker.
	re := regexp.MustCompile(`\[speed:([\d.]+)wpm\]`)
	var speed float64
	var hasSpeed bool
	if m := re.FindStringSubmatch(transcript); len(m) > 1 {
		fmt.Sscanf(m[1], "%f", &speed)
		hasSpeed = true
	} else {
		// Try to compute from character count and [duration:Xs] marker.
		durRe := regexp.MustCompile(`\[duration:([\d.]+)s\]`)
		if dm := durRe.FindStringSubmatch(transcript); len(dm) > 1 {
			var durSec float64
			fmt.Sscanf(dm[1], "%f", &durSec)
			if durSec > 0 {
				charCount := utf8.RuneCountInString(transcript)
				speed = float64(charCount) / (durSec / 60)
				hasSpeed = true
			}
		}
	}

	if !hasSpeed {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = "no speed or duration metadata available, skipped"
		return rr
	}

	if cfg.MaxWordsPerMin > 0 && speed > cfg.MaxWordsPerMin {
		rr.Detail = fmt.Sprintf("speaking too fast: %.0f wpm (max: %.0f)", speed, cfg.MaxWordsPerMin)
	} else if cfg.MinWordsPerMin > 0 && speed < cfg.MinWordsPerMin {
		rr.Detail = fmt.Sprintf("speaking too slow: %.0f wpm (min: %.0f)", speed, cfg.MinWordsPerMin)
	} else {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = fmt.Sprintf("speaking speed normal: %.0f wpm", speed)
	}
	return rr
}

// InterruptionRuleConfig specifies interruption detection.
// Expects [interruption] markers in transcript.
type InterruptionRuleConfig struct {
	MaxInterruptions int `json:"max_interruptions"`
}

func evaluateInterruptionRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg InterruptionRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid interruption config"
		return rr
	}
	if cfg.MaxInterruptions <= 0 {
		cfg.MaxInterruptions = 3
	}

	count := strings.Count(transcript, "[interruption]")
	if count > cfg.MaxInterruptions {
		rr.Detail = fmt.Sprintf("too many interruptions: %d (max: %d)", count, cfg.MaxInterruptions)
	} else {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = fmt.Sprintf("interruptions within limit: %d (max: %d)", count, cfg.MaxInterruptions)
	}
	return rr
}

// EnergyRuleConfig specifies volume/energy thresholds.
// Expects [energy:X] markers in transcript (0-100 scale).
type EnergyRuleConfig struct {
	MinEnergy float64 `json:"min_energy"`
	MaxEnergy float64 `json:"max_energy"`
}

func evaluateEnergyRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg EnergyRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid energy config"
		return rr
	}

	re := regexp.MustCompile(`\[energy:([\d.]+)\]`)
	matches := re.FindAllStringSubmatch(transcript, -1)
	if len(matches) == 0 {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = "no energy data available, passed by default"
		return rr
	}

	var total, maxE float64
	for _, m := range matches {
		var e float64
		fmt.Sscanf(m[1], "%f", &e)
		total += e
		if e > maxE {
			maxE = e
		}
	}
	avg := total / float64(len(matches))

	if cfg.MaxEnergy > 0 && maxE > cfg.MaxEnergy {
		rr.Detail = fmt.Sprintf("peak volume too high: %.0f (max: %.0f)", maxE, cfg.MaxEnergy)
	} else if cfg.MinEnergy > 0 && avg < cfg.MinEnergy {
		rr.Detail = fmt.Sprintf("average volume too low: %.0f (min: %.0f)", avg, cfg.MinEnergy)
	} else {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = fmt.Sprintf("volume normal (avg: %.0f, peak: %.0f)", avg, maxE)
	}
	return rr
}

// DurationRuleConfig specifies call duration thresholds in seconds.
// Expects [duration:Xs] marker in transcript.
type DurationRuleConfig struct {
	MinDurationSec float64 `json:"min_duration_sec"`
	MaxDurationSec float64 `json:"max_duration_sec"`
}

func evaluateDurationRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg DurationRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid duration config"
		return rr
	}

	re := regexp.MustCompile(`\[duration:([\d.]+)s\]`)
	var duration float64
	if m := re.FindStringSubmatch(transcript); len(m) > 1 {
		fmt.Sscanf(m[1], "%f", &duration)
	}

	if cfg.MaxDurationSec > 0 && duration > cfg.MaxDurationSec {
		rr.Detail = fmt.Sprintf("call too long: %.0fs (max: %.0fs)", duration, cfg.MaxDurationSec)
	} else if cfg.MinDurationSec > 0 && duration < cfg.MinDurationSec {
		rr.Detail = fmt.Sprintf("call too short: %.0fs (min: %.0fs)", duration, cfg.MinDurationSec)
	} else {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = fmt.Sprintf("call duration normal: %.0fs", duration)
	}
	return rr
}

// EntityRuleConfig specifies required named entities in the transcript.
type EntityRuleConfig struct {
	Entities []string `json:"entities"` // e.g. ["order_number", "phone", "name"]
	Require  string   `json:"require"`  // "all" or "any"
}

func evaluateEntityRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg EntityRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid entity config"
		return rr
	}

	entityPatterns := map[string]*regexp.Regexp{
		"phone":        regexp.MustCompile(`1[3-9]\d{9}`),
		"email":        regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		"order_number": regexp.MustCompile(`(?i)(订单|order)\s*[号:：#]?\s*[A-Z0-9]{6,}`),
		"id_card":      regexp.MustCompile(`\d{17}[\dXx]`),
		"name":         regexp.MustCompile(`(?:我叫|姓名[是：:]\s*|先生|女士|您好\s*)[\p{Han}]{2,4}`),
	}

	var found, missing []string
	for _, entity := range cfg.Entities {
		pat, ok := entityPatterns[entity]
		if !ok {
			// Treat unknown entity as keyword search.
			if containsIgnoreCase(transcript, entity) {
				found = append(found, entity)
			} else {
				missing = append(missing, entity)
			}
			continue
		}
		if pat.MatchString(transcript) {
			found = append(found, entity)
		} else {
			missing = append(missing, entity)
		}
	}

	requireAll := cfg.Require != "any"
	if requireAll {
		rr.Passed = len(missing) == 0
	} else {
		rr.Passed = len(found) > 0
	}

	if rr.Passed {
		rr.Score = 100
		rr.Detail = fmt.Sprintf("entities found: %s", strings.Join(found, ", "))
	} else {
		rr.Detail = fmt.Sprintf("entities missing: %s", strings.Join(missing, ", "))
	}
	return rr
}

// RoleRuleConfig checks if agent followed specific role-based requirements.
type RoleRuleConfig struct {
	RequireGreeting bool     `json:"require_greeting"`
	RequireClosing  bool     `json:"require_closing"`
	RequiredPhrases []string `json:"required_phrases"`
}

func evaluateRoleRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg RoleRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid role config"
		return rr
	}

	var issues []string

	if cfg.RequireGreeting {
		greetingPatterns := []string{"您好", "你好", "欢迎", "感谢来电", "感谢致电"}
		hasGreeting := false
		for _, g := range greetingPatterns {
			if containsIgnoreCase(transcript, g) {
				hasGreeting = true
				break
			}
		}
		if !hasGreeting {
			issues = append(issues, "missing greeting")
		}
	}

	if cfg.RequireClosing {
		closingPatterns := []string{"再见", "再会", "感谢", "祝您", "还有什么"}
		hasClosure := false
		for _, c := range closingPatterns {
			if containsIgnoreCase(transcript, c) {
				hasClosure = true
				break
			}
		}
		if !hasClosure {
			issues = append(issues, "missing closing")
		}
	}

	for _, phrase := range cfg.RequiredPhrases {
		if !containsIgnoreCase(transcript, phrase) {
			issues = append(issues, "missing phrase: "+phrase)
		}
	}

	if len(issues) == 0 {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = "all role requirements met"
	} else {
		rr.Detail = strings.Join(issues, "; ")
	}
	return rr
}

// AbnormalHangupRuleConfig checks for abnormal call termination.
// Expects [hangup:abnormal] or [hangup:normal] marker.
type AbnormalHangupRuleConfig struct {
	CheckMarker bool `json:"check_marker"`
}

func evaluateAbnormalHangupRule(rule *QARule, transcript string) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	if strings.Contains(transcript, "[hangup:abnormal]") {
		rr.Detail = "abnormal hangup detected"
		return rr
	}

	// Heuristic: if the call ends abruptly without closing phrases, flag it.
	closingPatterns := []string{"再见", "再会", "拜拜", "结束", "挂了"}
	lines := strings.Split(transcript, "\n")
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		for _, c := range closingPatterns {
			if containsIgnoreCase(lastLine, c) {
				rr.Passed = true
				rr.Score = 100
				rr.Detail = "normal hangup detected"
				return rr
			}
		}
	}

	// No closing marker, no explicit closing phrase — default to pass with note.
	rr.Passed = true
	rr.Score = 80
	rr.Detail = "no abnormal hangup marker, but closing phrase not found"
	return rr
}

// LLMRuleConfig specifies LLM-based inspection parameters.
type LLMRuleConfig struct {
	Prompt       string  `json:"prompt"`
	PassThreshold float64 `json:"pass_threshold"`
}

func evaluateLLMRule(ctx context.Context, rule *QARule, transcript string, llm QALLMProvider) QARuleResult {
	rr := QARuleResult{RuleID: rule.ID, RuleName: rule.Name, RuleType: rule.Type}

	var cfg LLMRuleConfig
	if err := json.Unmarshal([]byte(rule.Config), &cfg); err != nil {
		rr.Detail = "invalid LLM rule config"
		return rr
	}
	if cfg.PassThreshold <= 0 {
		cfg.PassThreshold = 60
	}

	if llm == nil {
		rr.Passed = true
		rr.Score = 100
		rr.Detail = "LLM provider not configured, skipped"
		return rr
	}

	score, detail, err := llm.QAInspectLLM(ctx, transcript, cfg.Prompt)
	if err != nil {
		rr.Detail = "LLM inspection error: " + err.Error()
		return rr
	}

	rr.Score = score
	rr.Passed = score >= cfg.PassThreshold
	rr.Detail = detail
	return rr
}

// Appeal submits an appeal for a QA result.
func (s *QualityInspectionService) Appeal(ctx context.Context, resultID int64, note string) (*QAResult, error) {
	result, err := s.results.GetByID(ctx, resultID)
	if err != nil || result == nil {
		return nil, ErrQAResultNotFound
	}
	if result.Status != QAResultStatusCompleted {
		return nil, ErrQAResultNotAppealable
	}
	result.Status = QAResultStatusAppealed
	result.AppealNote = note
	result.UpdatedAt = time.Now()
	if err := s.results.Update(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Review completes a review of an appealed QA result.
func (s *QualityInspectionService) Review(ctx context.Context, resultID int64, reviewerID int64, note string, newScore float64) (*QAResult, error) {
	result, err := s.results.GetByID(ctx, resultID)
	if err != nil || result == nil {
		return nil, ErrQAResultNotFound
	}
	if result.Status != QAResultStatusAppealed {
		return nil, ErrQAResultNotReviewable
	}
	result.Status = QAResultStatusReviewed
	result.ReviewerID = &reviewerID
	result.ReviewNote = note
	result.Score = newScore
	result.UpdatedAt = time.Now()
	if err := s.results.Update(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *QualityInspectionService) GetResult(ctx context.Context, id int64) (*QAResult, error) {
	r, err := s.results.GetByID(ctx, id)
	if err != nil || r == nil {
		return nil, ErrQAResultNotFound
	}
	return r, nil
}

func (s *QualityInspectionService) ListResults(ctx context.Context, tenantID int64, offset, limit int) ([]*QAResult, error) {
	return s.results.List(ctx, tenantID, offset, limit)
}

// ASRHotwordsService manages ASR custom hotword vocabularies.
type ASRHotwordsService struct {
	repo ASRHotwordsRepository
}

func NewASRHotwordsService(repo ASRHotwordsRepository) *ASRHotwordsService {
	return &ASRHotwordsService{repo: repo}
}

type CreateASRHotwordsInput struct {
	TenantID int64  `json:"tenant_id"`
	Name     string `json:"name"`
	Words    string `json:"words"`
}

func (s *ASRHotwordsService) Create(ctx context.Context, in CreateASRHotwordsInput) (*ASRHotwords, error) {
	now := time.Now()
	h := &ASRHotwords{
		ID:        snowflake.NextID(),
		TenantID:  in.TenantID,
		Name:      in.Name,
		Words:     in.Words,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *ASRHotwordsService) GetByID(ctx context.Context, id int64) (*ASRHotwords, error) {
	h, err := s.repo.GetByID(ctx, id)
	if err != nil || h == nil {
		return nil, ErrASRHotwordsNotFound
	}
	return h, nil
}

func (s *ASRHotwordsService) Update(ctx context.Context, h *ASRHotwords) error {
	h.UpdatedAt = time.Now()
	return s.repo.Update(ctx, h)
}

func (s *ASRHotwordsService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *ASRHotwordsService) List(ctx context.Context, tenantID int64) ([]*ASRHotwords, error) {
	return s.repo.List(ctx, tenantID)
}

// PerformanceScorecardService manages agent performance scorecards.
type PerformanceScorecardService struct {
	repo PerformanceScorecardRepository
}

func NewPerformanceScorecardService(repo PerformanceScorecardRepository) *PerformanceScorecardService {
	return &PerformanceScorecardService{repo: repo}
}

type GenerateScorecardInput struct {
	TenantID        int64   `json:"tenant_id"`
	AgentID         int64   `json:"agent_id"`
	Period          string  `json:"period"`
	TotalCalls      int     `json:"total_calls"`
	AvgHandleTime   float64 `json:"avg_handle_time"`
	AvgQAScore      float64 `json:"avg_qa_score"`
	CSATScore       float64 `json:"csat_score"`
	FirstCallResolv float64 `json:"first_call_resolution"`
	Adherence       float64 `json:"adherence"`
}

func (s *PerformanceScorecardService) Generate(ctx context.Context, in GenerateScorecardInput) (*PerformanceScorecard, error) {
	// Weighted overall score: QA 30%, CSAT 30%, FCR 20%, Adherence 20%
	overall := in.AvgQAScore*0.3 + in.CSATScore*0.3 + in.FirstCallResolv*0.2 + in.Adherence*0.2

	sc := &PerformanceScorecard{
		ID:              snowflake.NextID(),
		TenantID:        in.TenantID,
		AgentID:         in.AgentID,
		Period:          in.Period,
		TotalCalls:      in.TotalCalls,
		AvgHandleTime:   in.AvgHandleTime,
		AvgQAScore:      in.AvgQAScore,
		CSATScore:       in.CSATScore,
		FirstCallResolv: in.FirstCallResolv,
		Adherence:       in.Adherence,
		OverallScore:    overall,
		CreatedAt:       time.Now(),
	}
	if err := s.repo.Create(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *PerformanceScorecardService) List(ctx context.Context, tenantID int64, period string) ([]*PerformanceScorecard, error) {
	return s.repo.List(ctx, tenantID, period)
}
