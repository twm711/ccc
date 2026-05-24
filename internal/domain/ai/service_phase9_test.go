package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDEService() *DigitalEmployeeService {
	return NewDigitalEmployeeService(
		NewMockDigitalEmployeeRepo(),
		NewMockDigitalEmployeeSceneRepo(),
	)
}

// mockQALLM is a test LLM provider that returns fixed scores.
type mockQALLM struct{}

func (m *mockQALLM) QAInspectLLM(_ context.Context, _, _ string) (float64, string, error) {
	return 80, "LLM inspection passed (mock)", nil
}

func newTestQIService() *QualityInspectionService {
	svc := NewQualityInspectionService(
		NewMockQARuleRepo(),
		NewMockQASchemeRepo(),
		NewMockQAResultRepo(),
	)
	svc.SetLLMProvider(&mockQALLM{})
	return svc
}

// --- Digital Employee Tests ---

func TestDigitalEmployeeService_Create_Success(t *testing.T) {
	svc := newTestDEService()
	ctx := context.Background()

	de, err := svc.Create(ctx, CreateDigitalEmployeeInput{
		TenantID:    1,
		Name:        "智能客服助手",
		Description: "处理常见问题的AI机器人",
	})
	require.NoError(t, err)
	assert.NotZero(t, de.ID)
	assert.Equal(t, "智能客服助手", de.Name)
	assert.True(t, de.IsActive)
}

func TestDigitalEmployeeService_IntentMatch(t *testing.T) {
	svc := newTestDEService()
	ctx := context.Background()

	de, _ := svc.Create(ctx, CreateDigitalEmployeeInput{TenantID: 1, Name: "Bot"})

	scene, err := svc.CreateScene(ctx, CreateSceneInput{
		DigitalEmployeeID: de.ID,
		TenantID:          1,
		Name:              "售后场景",
		Intents:           `[{"name":"退款","keywords":["退款","退钱","退货"],"response":"好的，我来帮您处理退款","transfer":false},{"name":"投诉","keywords":["投诉","举报"],"response":"","transfer":true}]`,
	})
	require.NoError(t, err)

	result, err := svc.MatchIntent(ctx, scene.ID, "我想要退款")
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.Equal(t, "退款", result.IntentName)
	assert.False(t, result.Transfer)
}

func TestDigitalEmployeeService_TransferToHuman_Trigger(t *testing.T) {
	svc := newTestDEService()
	ctx := context.Background()

	de, _ := svc.Create(ctx, CreateDigitalEmployeeInput{TenantID: 1, Name: "Bot"})
	scene, _ := svc.CreateScene(ctx, CreateSceneInput{
		DigitalEmployeeID: de.ID,
		TenantID:          1,
		Name:              "投诉场景",
		Intents:           `[{"name":"投诉","keywords":["投诉","举报"],"response":"","transfer":true}]`,
	})

	result, err := svc.MatchIntent(ctx, scene.ID, "我要投诉你们")
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.True(t, result.Transfer)
}

// --- Quality Inspection Tests ---

func TestQualityInspection_RuleMatch_Keyword(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule, err := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1,
		Name:     "禁止辱骂",
		Type:     QARuleTypeKeyword,
		Config:   `{"keywords":["笨蛋","傻瓜"],"require":"absent"}`,
		Severity: "critical",
	})
	require.NoError(t, err)

	scheme, err := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID:  1,
		Name:      "基础方案",
		RuleIDs:   []SchemeRuleWeight{{RuleID: rule.ID, Weight: 100}},
		IsDefault: true,
	})
	require.NoError(t, err)

	// Clean transcript — should pass
	result, err := svc.RunInspection(ctx, 1, 100, scheme.ID, "您好，请问有什么可以帮您？好的，马上为您处理。")
	require.NoError(t, err)
	assert.Equal(t, float64(100), result.Score)

	// Dirty transcript — should fail
	result2, err := svc.RunInspection(ctx, 1, 101, scheme.ID, "你个笨蛋，怎么这么慢")
	require.NoError(t, err)
	assert.Equal(t, float64(0), result2.Score)
}

func TestQualityInspection_RuleMatch_Silence(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "静音检测", Type: QARuleTypeSilence, Config: `{"max_silence_sec":5}`,
	})
	scheme, _ := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID: 1, Name: "静音方案", RuleIDs: []SchemeRuleWeight{{RuleID: rule.ID, Weight: 100}},
	})

	result, err := svc.RunInspection(ctx, 1, 200, scheme.ID, "some transcript")
	require.NoError(t, err)
	assert.Equal(t, float64(100), result.Score) // stub always passes
}

func TestQualityInspection_RuleMatch_Speed(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "语速检测", Type: QARuleTypeSpeed, Config: `{"max_words_per_min":200}`,
	})
	scheme, _ := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID: 1, Name: "语速方案", RuleIDs: []SchemeRuleWeight{{RuleID: rule.ID, Weight: 100}},
	})

	result, err := svc.RunInspection(ctx, 1, 300, scheme.ID, "transcript content")
	require.NoError(t, err)
	assert.Equal(t, float64(100), result.Score)
}

func TestQualityInspection_RuleMatch_LLM(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "LLM质检", Type: QARuleTypeLLM, Config: `{"prompt":"检查是否专业"}`,
	})
	scheme, _ := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID: 1, Name: "LLM方案", RuleIDs: []SchemeRuleWeight{{RuleID: rule.ID, Weight: 100}},
	})

	result, err := svc.RunInspection(ctx, 1, 400, scheme.ID, "专业通话内容")
	require.NoError(t, err)
	assert.Equal(t, float64(80), result.Score) // LLM stub returns 80
}

func TestQualityInspection_SchemeScore_Calculation(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule1, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "禁止辱骂", Type: QARuleTypeKeyword,
		Config: `{"keywords":["笨蛋"],"require":"absent"}`, Severity: "critical",
	})
	rule2, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "LLM检查", Type: QARuleTypeLLM,
		Config: `{"prompt":"check"}`,
	})

	// Weight: keyword 60%, LLM 40%
	scheme, _ := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID: 1, Name: "综合方案",
		RuleIDs: []SchemeRuleWeight{
			{RuleID: rule1.ID, Weight: 60},
			{RuleID: rule2.ID, Weight: 40},
		},
	})

	// Clean: keyword=100, LLM=80 → (100*60 + 80*40) / 100 = 92
	result, err := svc.RunInspection(ctx, 1, 500, scheme.ID, "正常通话内容")
	require.NoError(t, err)
	assert.Equal(t, float64(92), result.Score)
}

func TestQualityInspection_Appeal_Flow(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "rule", Type: QARuleTypeKeyword,
		Config: `{"keywords":["bad"],"require":"absent"}`,
	})
	scheme, _ := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID: 1, Name: "scheme", RuleIDs: []SchemeRuleWeight{{RuleID: rule.ID, Weight: 100}},
	})

	result, _ := svc.RunInspection(ctx, 1, 600, scheme.ID, "this is bad")
	assert.Equal(t, QAResultStatusCompleted, result.Status)

	// Appeal
	appealed, err := svc.Appeal(ctx, result.ID, "误判，语境不同")
	require.NoError(t, err)
	assert.Equal(t, QAResultStatusAppealed, appealed.Status)

	// Cannot appeal again
	_, err = svc.Appeal(ctx, result.ID, "again")
	assert.ErrorIs(t, err, ErrQAResultNotAppealable)

	// Review
	reviewed, err := svc.Review(ctx, result.ID, 999, "确认误判", 85)
	require.NoError(t, err)
	assert.Equal(t, QAResultStatusReviewed, reviewed.Status)
	assert.Equal(t, float64(85), reviewed.Score)
}

func TestQualityInspection_EmptyTranscript_Error(t *testing.T) {
	svc := newTestQIService()
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, CreateQARuleInput{
		TenantID: 1, Name: "r", Type: QARuleTypeKeyword, Config: `{"keywords":["x"],"require":"absent"}`,
	})
	scheme, _ := svc.CreateScheme(ctx, CreateQASchemeInput{
		TenantID: 1, Name: "s", RuleIDs: []SchemeRuleWeight{{RuleID: rule.ID, Weight: 100}},
	})

	_, err := svc.RunInspection(ctx, 1, 700, scheme.ID, "")
	assert.ErrorIs(t, err, ErrEmptyTranscript)
}

func TestQualityInspection_InvalidRuleType_Error(t *testing.T) {
	svc := newTestQIService()
	_, err := svc.CreateRule(context.Background(), CreateQARuleInput{
		TenantID: 1, Name: "bad", Type: "invalid_type", Config: "{}",
	})
	assert.ErrorIs(t, err, ErrInvalidQARuleType)
}

// --- ASR Hotwords ---

func TestASRHotwordsService_CRUD(t *testing.T) {
	svc := NewASRHotwordsService(NewMockASRHotwordsRepo())
	ctx := context.Background()

	h, err := svc.Create(ctx, CreateASRHotwordsInput{
		TenantID: 1, Name: "产品术语", Words: `["CCC","IVR","ACD","SIP"]`,
	})
	require.NoError(t, err)
	assert.NotZero(t, h.ID)

	got, err := svc.GetByID(ctx, h.ID)
	require.NoError(t, err)
	assert.Equal(t, "产品术语", got.Name)

	list, err := svc.List(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

// --- Performance Scorecard ---

func TestPerformanceScorecard_Generate(t *testing.T) {
	svc := NewPerformanceScorecardService(NewMockPerformanceScorecardRepo())
	ctx := context.Background()

	sc, err := svc.Generate(ctx, GenerateScorecardInput{
		TenantID:        1,
		AgentID:         100,
		Period:          "2026-05",
		TotalCalls:      150,
		AvgHandleTime:   180,
		AvgQAScore:      90,
		CSATScore:       85,
		FirstCallResolv: 75,
		Adherence:       95,
	})
	require.NoError(t, err)
	assert.NotZero(t, sc.ID)
	// Overall = 90*0.3 + 85*0.3 + 75*0.2 + 95*0.2 = 27 + 25.5 + 15 + 19 = 86.5
	assert.Equal(t, 86.5, sc.OverallScore)
}

// --- Scene publish ---

func TestDigitalEmployeeService_PublishScene(t *testing.T) {
	svc := newTestDEService()
	ctx := context.Background()

	de, _ := svc.Create(ctx, CreateDigitalEmployeeInput{TenantID: 1, Name: "Bot"})
	scene, _ := svc.CreateScene(ctx, CreateSceneInput{
		DigitalEmployeeID: de.ID, TenantID: 1, Name: "场景1",
		Intents: `[{"name":"test","keywords":["hi"],"response":"hello","transfer":false}]`,
	})
	assert.Equal(t, SceneStatusDraft, scene.Status)

	published, err := svc.PublishScene(ctx, scene.ID)
	require.NoError(t, err)
	assert.Equal(t, SceneStatusPublished, published.Status)

	// Cannot publish again
	_, err = svc.PublishScene(ctx, scene.ID)
	assert.ErrorIs(t, err, ErrSceneAlreadyPublished)
}

func TestDigitalEmployeeService_CreateScene_InvalidIntents(t *testing.T) {
	svc := newTestDEService()
	ctx := context.Background()

	de, _ := svc.Create(ctx, CreateDigitalEmployeeInput{TenantID: 1, Name: "Bot"})
	_, err := svc.CreateScene(ctx, CreateSceneInput{
		DigitalEmployeeID: de.ID, TenantID: 1, Name: "Bad",
		Intents: `{not valid json`,
	})
	assert.ErrorIs(t, err, ErrInvalidIntentConfig)
}

func TestDigitalEmployeeService_IntentMatch_NoMatch(t *testing.T) {
	svc := newTestDEService()
	ctx := context.Background()

	de, _ := svc.Create(ctx, CreateDigitalEmployeeInput{TenantID: 1, Name: "Bot"})
	scene, _ := svc.CreateScene(ctx, CreateSceneInput{
		DigitalEmployeeID: de.ID, TenantID: 1, Name: "场景",
		Intents: `[{"name":"退款","keywords":["退款"],"response":"ok","transfer":false}]`,
	})

	result, err := svc.MatchIntent(ctx, scene.ID, "今天天气真好")
	require.NoError(t, err)
	assert.False(t, result.Matched)
}
