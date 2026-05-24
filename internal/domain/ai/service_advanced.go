package ai

import (
	"context"
	"errors"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

var (
	ErrCommAgentNameEmpty     = errors.New("communication agent name is required")
	ErrCommAgentPromptEmpty   = errors.New("system prompt is required")
	ErrCommAgentMaxTurns      = errors.New("max_turns must be between 1 and 100")
	ErrSessionAlreadyEnded    = errors.New("session already ended")
	ErrSessionMaxTurns        = errors.New("session exceeded max turns")
	ErrVoiceProfileNameEmpty  = errors.New("voice profile name is required")
	ErrVoiceSampleURLEmpty    = errors.New("sample audio URL is required")
	ErrAnalysisNameEmpty      = errors.New("analysis task name is required")
	ErrAnalysisDateRange      = errors.New("date range is required")
	ErrCourseNameEmpty        = errors.New("course title is required")
	ErrCoursePassScore        = errors.New("pass score must be between 1 and 100")
	ErrExamAlreadySubmitted   = errors.New("exam already submitted")
	ErrFullDuplexSensitivity  = errors.New("interruption sensitivity must be between 0 and 1")
)

// ─── CommAgent Service ───

// CommAgentProvider handles multi-turn autonomous conversations.
type CommAgentProvider interface {
	GenerateReply(ctx context.Context, systemPrompt, conversationHistory, userMessage string) (string, error)
	ShouldTransfer(ctx context.Context, systemPrompt, conversationHistory string) (bool, string, error)
}

type CommAgentService struct {
	agentRepo   CommAgentRepository
	sessionRepo CommAgentSessionRepository
	provider    CommAgentProvider
}

func (s *CommAgentService) SetProvider(p CommAgentProvider) { s.provider = p }

func NewCommAgentService(ar CommAgentRepository, sr CommAgentSessionRepository) *CommAgentService {
	return &CommAgentService{agentRepo: ar, sessionRepo: sr}
}

type CreateCommAgentInput struct {
	TenantID            int64     `json:"tenant_id"`
	DigitalEmployeeID   int64     `json:"digital_employee_id"`
	Name                string    `json:"name"`
	Mode                AgentMode `json:"mode"`
	SystemPrompt        string    `json:"system_prompt"`
	MaxTurns            int       `json:"max_turns"`
	TransferSkillGroupID *int64   `json:"transfer_skill_group_id,omitempty"`
	LLMModelID          *int64    `json:"llm_model_id,omitempty"`
}

func (s *CommAgentService) Create(ctx context.Context, in CreateCommAgentInput) (*CommAgent, error) {
	if in.Name == "" {
		return nil, ErrCommAgentNameEmpty
	}
	if in.SystemPrompt == "" {
		return nil, ErrCommAgentPromptEmpty
	}
	if in.MaxTurns < 1 || in.MaxTurns > 100 {
		return nil, ErrCommAgentMaxTurns
	}
	if in.Mode == "" {
		in.Mode = AgentModeInbound
	}
	a := &CommAgent{
		ID:                   snowflake.NextID(),
		TenantID:             in.TenantID,
		DigitalEmployeeID:    in.DigitalEmployeeID,
		Name:                 in.Name,
		Mode:                 in.Mode,
		SystemPrompt:         in.SystemPrompt,
		MaxTurns:             in.MaxTurns,
		TransferSkillGroupID: in.TransferSkillGroupID,
		LLMModelID:           in.LLMModelID,
		IsActive:             true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	if err := s.agentRepo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *CommAgentService) Get(ctx context.Context, tenantID, id int64) (*CommAgent, error) {
	return s.agentRepo.GetByID(ctx, tenantID, id)
}

func (s *CommAgentService) List(ctx context.Context, tenantID int64) ([]CommAgent, error) {
	return s.agentRepo.List(ctx, tenantID)
}

func (s *CommAgentService) Delete(ctx context.Context, tenantID, id int64) error {
	return s.agentRepo.Delete(ctx, tenantID, id)
}

// StartSession creates a new conversation session for the comm agent.
func (s *CommAgentService) StartSession(ctx context.Context, tenantID, commAgentID, callID int64) (*CommAgentSession, error) {
	sess := &CommAgentSession{
		ID:          snowflake.NextID(),
		TenantID:    tenantID,
		CommAgentID: commAgentID,
		CallID:      callID,
		Status:      AgentSessionActive,
		TurnCount:   0,
		StartedAt:   time.Now(),
	}
	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// AddTurn records a dialogue turn and checks max turn limit.
func (s *CommAgentService) AddTurn(ctx context.Context, sess *CommAgentSession, userMsg, aiReply string, maxTurns int) error {
	if sess.Status != AgentSessionActive {
		return ErrSessionAlreadyEnded
	}
	sess.TurnCount++
	if sess.TurnCount > maxTurns {
		return ErrSessionMaxTurns
	}
	sess.Transcript += "User: " + userMsg + "\nAI: " + aiReply + "\n"
	return s.sessionRepo.Update(ctx, sess)
}

// EndSession marks the session as completed or transferred.
func (s *CommAgentService) EndSession(ctx context.Context, sess *CommAgentSession, status AgentSessionStatus, summary string, transferTo *int64) error {
	if sess.Status != AgentSessionActive {
		return ErrSessionAlreadyEnded
	}
	now := time.Now()
	sess.Status = status
	sess.Summary = summary
	sess.EndedAt = &now
	sess.TransferredTo = transferTo
	return s.sessionRepo.Update(ctx, sess)
}

// ─── VoiceProfile Service ───

// VoiceCloningProvider handles voice profile training and synthesis.
type VoiceCloningProvider interface {
	StartCloneTraining(ctx context.Context, sampleAudioURL string) (providerJobID string, err error)
	CheckTrainingStatus(ctx context.Context, providerJobID string) (ready bool, providerVoiceID string, err error)
}

type VoiceProfileService struct {
	repo     VoiceProfileRepository
	provider VoiceCloningProvider
}

func (s *VoiceProfileService) SetProvider(p VoiceCloningProvider) { s.provider = p }

func NewVoiceProfileService(r VoiceProfileRepository) *VoiceProfileService {
	return &VoiceProfileService{repo: r}
}

type CreateVoiceProfileInput struct {
	TenantID       int64  `json:"tenant_id"`
	Name           string `json:"name"`
	SampleAudioURL string `json:"sample_audio_url"`
	Language       string `json:"language"`
}

func (s *VoiceProfileService) Create(ctx context.Context, in CreateVoiceProfileInput) (*VoiceProfile, error) {
	if in.Name == "" {
		return nil, ErrVoiceProfileNameEmpty
	}
	if in.SampleAudioURL == "" {
		return nil, ErrVoiceSampleURLEmpty
	}
	if in.Language == "" {
		in.Language = "zh-CN"
	}
	v := &VoiceProfile{
		ID:             snowflake.NextID(),
		TenantID:       in.TenantID,
		Name:           in.Name,
		SampleAudioURL: in.SampleAudioURL,
		Status:         VoiceProfilePending,
		Language:       in.Language,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *VoiceProfileService) Get(ctx context.Context, tenantID, id int64) (*VoiceProfile, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *VoiceProfileService) List(ctx context.Context, tenantID int64) ([]VoiceProfile, error) {
	return s.repo.List(ctx, tenantID)
}

// StartTraining transitions the profile to training state.
func (s *VoiceProfileService) StartTraining(ctx context.Context, tenantID, id int64) (*VoiceProfile, error) {
	v, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	v.Status = VoiceProfileTraining
	v.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// MarkReady marks the voice profile as ready after training completes.
func (s *VoiceProfileService) MarkReady(ctx context.Context, tenantID, id int64, providerVoiceID string) (*VoiceProfile, error) {
	v, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	v.Status = VoiceProfileReady
	v.ProviderVoiceID = providerVoiceID
	v.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *VoiceProfileService) Delete(ctx context.Context, tenantID, id int64) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// ─── ConversationAnalysis Service ───

// ConversationAnalyticsProvider runs batch analysis over transcripts.
type ConversationAnalyticsProvider interface {
	MineIntents(ctx context.Context, transcripts []string) (resultJSON string, err error)
	DiscoverSOPs(ctx context.Context, transcripts []string) (resultJSON string, err error)
}

type ConversationAnalysisService struct {
	repo     ConversationAnalysisTaskRepository
	provider ConversationAnalyticsProvider
}

func (s *ConversationAnalysisService) SetProvider(p ConversationAnalyticsProvider) { s.provider = p }

func NewConversationAnalysisService(r ConversationAnalysisTaskRepository) *ConversationAnalysisService {
	return &ConversationAnalysisService{repo: r}
}

type CreateAnalysisTaskInput struct {
	TenantID   int64        `json:"tenant_id"`
	Name       string       `json:"name"`
	Type       AnalysisType `json:"type"`
	DateFrom   string       `json:"date_from"`
	DateTo     string       `json:"date_to"`
	TotalCalls int          `json:"total_calls"`
}

func (s *ConversationAnalysisService) Create(ctx context.Context, in CreateAnalysisTaskInput) (*ConversationAnalysisTask, error) {
	if in.Name == "" {
		return nil, ErrAnalysisNameEmpty
	}
	if in.DateFrom == "" || in.DateTo == "" {
		return nil, ErrAnalysisDateRange
	}
	t := &ConversationAnalysisTask{
		ID:         snowflake.NextID(),
		TenantID:   in.TenantID,
		Name:       in.Name,
		Type:       in.Type,
		DateFrom:   in.DateFrom,
		DateTo:     in.DateTo,
		TotalCalls: in.TotalCalls,
		Status:     AnalysisTaskPending,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *ConversationAnalysisService) Get(ctx context.Context, tenantID, id int64) (*ConversationAnalysisTask, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *ConversationAnalysisService) List(ctx context.Context, tenantID int64) ([]ConversationAnalysisTask, error) {
	return s.repo.List(ctx, tenantID)
}

// MarkRunning transitions the task to running.
func (s *ConversationAnalysisService) MarkRunning(ctx context.Context, task *ConversationAnalysisTask) error {
	task.Status = AnalysisTaskRunning
	return s.repo.Update(ctx, task)
}

// Complete marks the task as completed with results.
func (s *ConversationAnalysisService) Complete(ctx context.Context, task *ConversationAnalysisTask, resultJSON string) error {
	now := time.Now()
	task.Status = AnalysisTaskCompleted
	task.ResultJSON = resultJSON
	task.ProcessedCalls = task.TotalCalls
	task.CompletedAt = &now
	return s.repo.Update(ctx, task)
}

// ─── Training Service ───

// TrainingProvider generates AI feedback for simulated calls.
type TrainingProvider interface {
	EvaluateSimulatedCall(ctx context.Context, scenario, transcript string) (feedback string, score int, err error)
}

type TrainingService struct {
	courseRepo TrainingCourseRepository
	examRepo   TrainingExamRepository
	simRepo    SimulatedCallRepository
	provider   TrainingProvider
}

func (s *TrainingService) SetProvider(p TrainingProvider) { s.provider = p }

func NewTrainingService(cr TrainingCourseRepository, er TrainingExamRepository, sr SimulatedCallRepository) *TrainingService {
	return &TrainingService{courseRepo: cr, examRepo: er, simRepo: sr}
}

type CreateCourseInput struct {
	TenantID    int64  `json:"tenant_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ContentJSON string `json:"content_json"`
	PassScore   int    `json:"pass_score"`
}

func (s *TrainingService) CreateCourse(ctx context.Context, in CreateCourseInput) (*TrainingCourse, error) {
	if in.Title == "" {
		return nil, ErrCourseNameEmpty
	}
	if in.PassScore < 1 || in.PassScore > 100 {
		return nil, ErrCoursePassScore
	}
	c := &TrainingCourse{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		Title:       in.Title,
		Description: in.Description,
		ContentJSON: in.ContentJSON,
		PassScore:   in.PassScore,
		Status:      CourseStatusDraft,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.courseRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *TrainingService) GetCourse(ctx context.Context, tenantID, id int64) (*TrainingCourse, error) {
	return s.courseRepo.GetByID(ctx, tenantID, id)
}

func (s *TrainingService) ListCourses(ctx context.Context, tenantID int64) ([]TrainingCourse, error) {
	return s.courseRepo.List(ctx, tenantID)
}

func (s *TrainingService) PublishCourse(ctx context.Context, tenantID, id int64) (*TrainingCourse, error) {
	c, err := s.courseRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	c.Status = CourseStatusPublished
	c.UpdatedAt = time.Now()
	if err := s.courseRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

type SubmitExamInput struct {
	TenantID    int64  `json:"tenant_id"`
	CourseID    int64  `json:"course_id"`
	AgentID     int64  `json:"agent_id"`
	Score       int    `json:"score"`
	MaxScore    int    `json:"max_score"`
	AnswersJSON string `json:"answers_json"`
}

func (s *TrainingService) SubmitExam(ctx context.Context, in SubmitExamInput) (*TrainingExam, error) {
	course, err := s.courseRepo.GetByID(ctx, in.TenantID, in.CourseID)
	if err != nil {
		return nil, err
	}
	status := ExamStatusFailed
	pct := 0
	if in.MaxScore > 0 {
		pct = in.Score * 100 / in.MaxScore
	}
	if pct >= course.PassScore {
		status = ExamStatusPassed
	}
	e := &TrainingExam{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		CourseID:    in.CourseID,
		AgentID:     in.AgentID,
		Score:       in.Score,
		MaxScore:    in.MaxScore,
		Status:      status,
		AnswersJSON: in.AnswersJSON,
		CreatedAt:   time.Now(),
	}
	if err := s.examRepo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *TrainingService) ListExamsByAgent(ctx context.Context, tenantID, agentID int64) ([]TrainingExam, error) {
	return s.examRepo.ListByAgent(ctx, tenantID, agentID)
}

type CreateSimulatedCallInput struct {
	TenantID    int64  `json:"tenant_id"`
	AgentID     int64  `json:"agent_id"`
	ScenarioID  int64  `json:"scenario_id"`
	Transcript  string `json:"transcript"`
	AIFeedback  string `json:"ai_feedback"`
	Score       int    `json:"score"`
	DurationSec int    `json:"duration_sec"`
}

func (s *TrainingService) CreateSimulatedCall(ctx context.Context, in CreateSimulatedCallInput) (*SimulatedCall, error) {
	sc := &SimulatedCall{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		AgentID:     in.AgentID,
		ScenarioID:  in.ScenarioID,
		Transcript:  in.Transcript,
		AIFeedback:  in.AIFeedback,
		Score:       in.Score,
		DurationSec: in.DurationSec,
		CreatedAt:   time.Now(),
	}
	if err := s.simRepo.Create(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *TrainingService) ListSimulatedCalls(ctx context.Context, tenantID, agentID int64) ([]SimulatedCall, error) {
	return s.simRepo.ListByAgent(ctx, tenantID, agentID)
}

// ─── RingAnalysis Service ───

// RingAnalysisProvider detects call answering patterns.
type RingAnalysisProvider interface {
	AnalyzeRingAudio(ctx context.Context, audioData []byte) (result string, confidence float64, err error)
}

type RingAnalysisService struct {
	configRepo RingAnalysisConfigRepository
	logRepo    RingAnalysisLogRepository
	provider   RingAnalysisProvider
}

func (s *RingAnalysisService) SetProvider(p RingAnalysisProvider) { s.provider = p }

func NewRingAnalysisService(cr RingAnalysisConfigRepository, lr RingAnalysisLogRepository) *RingAnalysisService {
	return &RingAnalysisService{configRepo: cr, logRepo: lr}
}

func (s *RingAnalysisService) GetConfig(ctx context.Context, tenantID int64) (*RingAnalysisConfig, error) {
	return s.configRepo.Get(ctx, tenantID)
}

func (s *RingAnalysisService) UpsertConfig(ctx context.Context, c *RingAnalysisConfig) error {
	if c.ID == 0 {
		c.ID = snowflake.NextID()
	}
	c.UpdatedAt = time.Now()
	return s.configRepo.Upsert(ctx, c)
}

func (s *RingAnalysisService) LogResult(ctx context.Context, tenantID, callID int64, result RingDetectionResult, confidence float64, durationMs int) (*RingAnalysisLog, error) {
	l := &RingAnalysisLog{
		ID:         snowflake.NextID(),
		TenantID:   tenantID,
		CallID:     callID,
		Result:     result,
		Confidence: confidence,
		DurationMs: durationMs,
		CreatedAt:  time.Now(),
	}
	if err := s.logRepo.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *RingAnalysisService) GetCallLogs(ctx context.Context, tenantID, callID int64) ([]RingAnalysisLog, error) {
	return s.logRepo.ListByCall(ctx, tenantID, callID)
}

// ─── FullDuplex Service ───

// FullDuplexProvider handles real-time full-duplex interaction.
type FullDuplexProvider interface {
	DetectInterruption(ctx context.Context, audioChunk []byte, sensitivity float64) (interrupted bool, err error)
}

type FullDuplexService struct {
	repo     FullDuplexConfigRepository
	provider FullDuplexProvider
}

func (s *FullDuplexService) SetProvider(p FullDuplexProvider) { s.provider = p }

func NewFullDuplexService(r FullDuplexConfigRepository) *FullDuplexService {
	return &FullDuplexService{repo: r}
}

func (s *FullDuplexService) GetConfig(ctx context.Context, tenantID int64) (*FullDuplexConfig, error) {
	return s.repo.Get(ctx, tenantID)
}

func (s *FullDuplexService) UpsertConfig(ctx context.Context, c *FullDuplexConfig) error {
	if c.InterruptionSensitivity < 0 || c.InterruptionSensitivity > 1 {
		return ErrFullDuplexSensitivity
	}
	if c.ID == 0 {
		c.ID = snowflake.NextID()
	}
	c.UpdatedAt = time.Now()
	return s.repo.Upsert(ctx, c)
}
