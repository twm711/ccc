package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// DashScopeCommAgentProvider implements CommAgentProvider using DashScope.
type DashScopeCommAgentProvider struct {
	base *DashScopeProvider
}

func NewDashScopeCommAgentProvider(apiKey, model string) *DashScopeCommAgentProvider {
	return &DashScopeCommAgentProvider{base: NewDashScopeProvider(apiKey, model)}
}

func (p *DashScopeCommAgentProvider) GenerateReply(ctx context.Context, systemPrompt, conversationHistory, userMessage string) (string, error) {
	combined := systemPrompt + "\n\n历史对话：\n" + conversationHistory
	return p.base.call(ctx, combined, userMessage)
}

func (p *DashScopeCommAgentProvider) ShouldTransfer(ctx context.Context, systemPrompt, conversationHistory string) (bool, string, error) {
	resp, err := p.base.call(ctx,
		`你是一个转人工判断助手。根据对话历史判断是否需要转接人工坐席。返回 JSON 格式：{"transfer":true/false,"reason":"原因"}。只返回 JSON。`+"\n\n系统提示："+systemPrompt,
		conversationHistory)
	if err != nil {
		return false, "", err
	}

	resp = extractJSON(resp)
	var result struct {
		Transfer bool   `json:"transfer"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return false, "", nil
	}
	return result.Transfer, result.Reason, nil
}

// DashScopeVoiceCloningProvider implements VoiceCloningProvider using DashScope CosyVoice API.
type DashScopeVoiceCloningProvider struct {
	base *DashScopeProvider
}

func NewDashScopeVoiceCloningProvider(apiKey, model string) *DashScopeVoiceCloningProvider {
	return &DashScopeVoiceCloningProvider{base: NewDashScopeProvider(apiKey, model)}
}

func (p *DashScopeVoiceCloningProvider) StartCloneTraining(ctx context.Context, sampleAudioURL string) (string, error) {
	reqBody := map[string]interface{}{
		"model": "cosyvoice-clone-v1",
		"input": map[string]string{
			"audio_url": sampleAudioURL,
			"action":    "create_voice",
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := postDashScopeRaw(ctx, p.base.apiKey, "https://dashscope.aliyuncs.com/api/v1/services/audio/voice-clone/create", body, p.base.client)
	if err != nil {
		return "", fmt.Errorf("voice clone: start training: %w", err)
	}

	var result struct {
		Output struct {
			TaskID string `json:"task_id"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("voice clone: parse response: %w", err)
	}
	if result.Code != "" {
		return "", fmt.Errorf("voice clone: API error %s: %s", result.Code, result.Message)
	}
	return result.Output.TaskID, nil
}

func (p *DashScopeVoiceCloningProvider) CheckTrainingStatus(ctx context.Context, providerJobID string) (bool, string, error) {
	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/tasks/%s", providerJobID)

	resp, err := getDashScopeRaw(ctx, p.base.apiKey, url, p.base.client)
	if err != nil {
		return false, "", fmt.Errorf("voice clone: check status: %w", err)
	}

	var result struct {
		Output struct {
			TaskStatus string `json:"task_status"`
			VoiceID    string `json:"voice_id"`
		} `json:"output"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, "", fmt.Errorf("voice clone: parse status: %w", err)
	}

	if result.Output.TaskStatus == "SUCCEEDED" {
		return true, result.Output.VoiceID, nil
	}
	return false, "", nil
}

func (p *DashScopeVoiceCloningProvider) SynthesizeWithClone(ctx context.Context, text, providerVoiceID string) ([]byte, error) {
	reqBody := map[string]interface{}{
		"model": "cosyvoice-clone-v1",
		"input": map[string]string{
			"text":     text,
			"voice_id": providerVoiceID,
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := postDashScopeRaw(ctx, p.base.apiKey, "https://dashscope.aliyuncs.com/api/v1/services/audio/tts/synthesize", body, p.base.client)
	if err != nil {
		return nil, fmt.Errorf("voice clone synthesize: %w", err)
	}
	return resp, nil
}

// DashScopeConversationAnalyticsProvider implements ConversationAnalyticsProvider.
type DashScopeConversationAnalyticsProvider struct {
	base *DashScopeProvider
}

func NewDashScopeConversationAnalyticsProvider(apiKey, model string) *DashScopeConversationAnalyticsProvider {
	return &DashScopeConversationAnalyticsProvider{base: NewDashScopeProvider(apiKey, model)}
}

func (p *DashScopeConversationAnalyticsProvider) MineIntents(ctx context.Context, transcripts []string) (string, error) {
	combined := strings.Join(transcripts, "\n---\n")
	return p.base.call(ctx,
		`你是一个客户意图挖掘助手。分析以下多通通话记录，提取常见客户意图并统计频次。返回 JSON 数组格式：[{"intent":"意图名","count":数量,"examples":["示例"]}]。只返回 JSON。`,
		combined)
}

func (p *DashScopeConversationAnalyticsProvider) DiscoverSOPs(ctx context.Context, transcripts []string) (string, error) {
	combined := strings.Join(transcripts, "\n---\n")
	return p.base.call(ctx,
		`你是一个 SOP 发现助手。从坐席通话记录中提取标准操作流程。返回 JSON 数组格式：[{"sop":"SOP名称","steps":["步骤1","步骤2"],"frequency":出现次数}]。只返回 JSON。`,
		combined)
}

func (p *DashScopeConversationAnalyticsProvider) ExtractSalesScripts(ctx context.Context, transcripts []string) (string, error) {
	combined := strings.Join(transcripts, "\n---\n")
	return p.base.call(ctx,
		`你是一个销售话术分析助手。从通话记录中提取高效销售话术。返回 JSON 数组格式：[{"script":"话术名称","content":"话术内容","effectiveness":0.0-1.0}]。只返回 JSON。`,
		combined)
}

func (p *DashScopeConversationAnalyticsProvider) ClusterTopics(ctx context.Context, transcripts []string) (string, error) {
	combined := strings.Join(transcripts, "\n---\n")
	return p.base.call(ctx,
		`你是一个话题聚类助手。将以下通话记录按话题分类。返回 JSON 数组格式：[{"topic":"话题名","count":数量,"keywords":["关键词"]}]。只返回 JSON。`,
		combined)
}

// DashScopeRingAnalysisProvider implements RingAnalysisProvider using audio classification.
type DashScopeRingAnalysisProvider struct {
	base *DashScopeProvider
}

func NewDashScopeRingAnalysisProvider(apiKey, model string) *DashScopeRingAnalysisProvider {
	return &DashScopeRingAnalysisProvider{base: NewDashScopeProvider(apiKey, model)}
}

func (p *DashScopeRingAnalysisProvider) AnalyzeRingAudio(ctx context.Context, audioData []byte) (string, float64, error) {
	reqBody := map[string]interface{}{
		"model": "paraformer-realtime-v2",
		"input": map[string]interface{}{
			"audio_data": audioData,
			"task":       "audio-classification",
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := postDashScopeRaw(ctx, p.base.apiKey, "https://dashscope.aliyuncs.com/api/v1/services/audio/asr/recognition", body, p.base.client)
	if err != nil {
		return "unknown", 0, fmt.Errorf("ring analysis: %w", err)
	}

	var result struct {
		Output struct {
			Classification string  `json:"classification"`
			Confidence     float64 `json:"confidence"`
		} `json:"output"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "unknown", 0, nil
	}

	label := result.Output.Classification
	if label == "" {
		label = "human"
	}
	return label, result.Output.Confidence, nil
}

// DashScopeFullDuplexProvider implements FullDuplexProvider.
type DashScopeFullDuplexProvider struct {
	base *DashScopeProvider
}

func NewDashScopeFullDuplexProvider(apiKey, model string) *DashScopeFullDuplexProvider {
	return &DashScopeFullDuplexProvider{base: NewDashScopeProvider(apiKey, model)}
}

func (p *DashScopeFullDuplexProvider) DetectInterruption(ctx context.Context, audioChunk []byte, sensitivity float64) (bool, error) {
	reqBody := map[string]interface{}{
		"model": "paraformer-realtime-v2",
		"input": map[string]interface{}{
			"audio_data":  audioChunk,
			"task":        "vad",
			"sensitivity": sensitivity,
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := postDashScopeRaw(ctx, p.base.apiKey, "https://dashscope.aliyuncs.com/api/v1/services/audio/asr/recognition", body, p.base.client)
	if err != nil {
		return false, fmt.Errorf("full duplex detect: %w", err)
	}

	var result struct {
		Output struct {
			SpeechDetected bool `json:"speech_detected"`
		} `json:"output"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, nil
	}
	return result.Output.SpeechDetected, nil
}

func (p *DashScopeFullDuplexProvider) ContinueVoice(ctx context.Context, previousText, interruptionPoint string) (string, error) {
	return p.base.call(ctx,
		"你是一个智能语音助手。用户在你说到以下位置时打断了你，请从打断点自然地继续说下去，保持连贯性。之前说的内容："+previousText,
		"打断位置："+interruptionPoint)
}

// DashScopeTrainingProvider implements TrainingProvider.
type DashScopeTrainingProvider struct {
	base *DashScopeProvider
}

func NewDashScopeTrainingProvider(apiKey, model string) *DashScopeTrainingProvider {
	return &DashScopeTrainingProvider{base: NewDashScopeProvider(apiKey, model)}
}

func (p *DashScopeTrainingProvider) EvaluateSimulatedCall(ctx context.Context, scenario, transcript string) (string, int, error) {
	resp, err := p.base.call(ctx,
		`你是一个坐席培训评估助手。根据训练场景和模拟通话内容进行评分。返回 JSON 格式：{"feedback":"详细反馈","score":0-100,"strengths":["优点"],"improvements":["改进建议"]}。只返回 JSON。`+"\n\n场景："+scenario,
		transcript)
	if err != nil {
		return "", 0, err
	}

	resp = extractJSON(resp)
	var result struct {
		Feedback string `json:"feedback"`
		Score    int    `json:"score"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return resp, 50, nil
	}
	return result.Feedback, result.Score, nil
}

func (p *DashScopeTrainingProvider) GenerateExamQuestions(ctx context.Context, courseContent string, count int) (string, error) {
	prompt := fmt.Sprintf(
		`你是一个培训考试出题助手。根据以下课程内容生成 %d 道选择题。返回 JSON 数组格式：[{"question":"题目","options":["A.选项","B.选项","C.选项","D.选项"],"answer":"A","explanation":"解析"}]。只返回 JSON。`,
		count)
	return p.base.call(ctx, prompt, courseContent)
}
