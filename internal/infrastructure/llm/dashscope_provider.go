package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	dashScopeBaseURL      = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	dashScopeDefaultModel = "qwen-plus"
)

// DashScopeProvider implements Provider using Aliyun DashScope (通义千问) API.
type DashScopeProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewDashScopeProvider(apiKey, model string) *DashScopeProvider {
	if model == "" {
		model = dashScopeDefaultModel
	}
	return &DashScopeProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

type dashScopeRequest struct {
	Model string              `json:"model"`
	Input dashScopeInput      `json:"input"`
	Param *dashScopeParameter `json:"parameters,omitempty"`
}

type dashScopeInput struct {
	Messages []dashScopeMessage `json:"messages"`
}

type dashScopeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type dashScopeParameter struct {
	ResultFormat string  `json:"result_format,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
}

type dashScopeResponse struct {
	Output struct {
		Text    string `json:"text"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func (p *DashScopeProvider) call(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []dashScopeMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	reqBody := dashScopeRequest{
		Model: p.model,
		Input: dashScopeInput{Messages: messages},
		Param: &dashScopeParameter{
			ResultFormat: "message",
			Temperature:  0.7,
			MaxTokens:    2048,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("dashscope: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dashScopeBaseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("dashscope: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("dashscope: http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("dashscope: read response: %w", err)
	}

	var dsResp dashScopeResponse
	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return "", fmt.Errorf("dashscope: unmarshal response: %w", err)
	}
	if dsResp.Code != "" {
		return "", fmt.Errorf("dashscope: API error %s: %s", dsResp.Code, dsResp.Message)
	}

	if len(dsResp.Output.Choices) > 0 {
		return dsResp.Output.Choices[0].Message.Content, nil
	}
	if dsResp.Output.Text != "" {
		return dsResp.Output.Text, nil
	}
	return "", fmt.Errorf("dashscope: empty response")
}

func (p *DashScopeProvider) Correct(ctx context.Context, text string) (string, error) {
	return p.call(ctx,
		"你是一个中文文本纠错助手。请纠正用户输入中的错别字、语法错误，只返回纠正后的文本，不要解释。",
		text)
}

func (p *DashScopeProvider) Expand(ctx context.Context, text string) (string, error) {
	return p.call(ctx,
		"你是一个客服话术扩写助手。请将用户输入的简短回复扩展为更专业、详细的客服回复，保持礼貌和专业性。只返回扩写后的文本。",
		text)
}

func (p *DashScopeProvider) Optimize(ctx context.Context, text string) (string, error) {
	return p.call(ctx,
		"你是一个客服话术优化助手。请优化用户输入的客服回复，使其更加专业、得体、有说服力。只返回优化后的文本。",
		text)
}

func (p *DashScopeProvider) Summarize(ctx context.Context, transcript string) (string, error) {
	return p.call(ctx,
		"你是一个通话摘要生成助手。请根据以下通话内容生成简洁的摘要，包含：1.客户诉求 2.处理过程 3.处理结果。以 JSON 格式返回：{\"subject\":\"主题\",\"summary\":\"摘要\",\"result\":\"结果\"}",
		transcript)
}

func (p *DashScopeProvider) AnalyzeSentiment(ctx context.Context, text string) (SentimentResult, error) {
	resp, err := p.call(ctx,
		`你是一个情绪分析助手。分析以下文本的情绪，返回 JSON 格式：{"label":"positive/negative/neutral","confidence":0.0-1.0}。只返回 JSON。`,
		text)
	if err != nil {
		return SentimentResult{}, err
	}

	resp = extractJSON(resp)
	var result SentimentResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return SentimentResult{Label: "neutral", Confidence: 0.5}, nil
	}
	return result, nil
}

func (p *DashScopeProvider) ExtractTags(ctx context.Context, transcript string) ([]string, error) {
	resp, err := p.call(ctx,
		`你是一个会话标签分析助手。根据通话内容提取3-5个语义标签，返回 JSON 数组格式：["标签1","标签2","标签3"]。只返回 JSON 数组。`,
		transcript)
	if err != nil {
		return nil, err
	}

	resp = extractJSON(resp)
	var tags []string
	if err := json.Unmarshal([]byte(resp), &tags); err != nil {
		return []string{"general"}, nil
	}
	return tags, nil
}

func (p *DashScopeProvider) PredictSatisfaction(ctx context.Context, transcript string) (float64, error) {
	resp, err := p.call(ctx,
		`你是一个客户满意度预测助手。根据通话内容预测客户满意度，返回 JSON 格式：{"score":1.0-5.0,"reason":"原因"}。只返回 JSON。`,
		transcript)
	if err != nil {
		return 0, err
	}

	resp = extractJSON(resp)
	var result struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return 3.0, nil
	}
	return result.Score, nil
}

func (p *DashScopeProvider) AnalyzeIVRPath(ctx context.Context, ivrPath string) (string, error) {
	return p.call(ctx,
		"你是一个 IVR 路径分析助手。根据用户的 IVR 导航路径，分析客户可能的意图和关键信息，生成简洁的分析报告供坐席参考。",
		ivrPath)
}

func (p *DashScopeProvider) JudgeCompletion(ctx context.Context, transcript string) (float64, error) {
	resp, err := p.call(ctx,
		`你是一个通话完成度判断助手。根据通话内容判断客户诉求是否已被解决，返回 JSON 格式：{"score":1-5,"reason":"原因"}。1=完全未解决，5=完全解决。只返回 JSON。`,
		transcript)
	if err != nil {
		return 0, err
	}

	resp = extractJSON(resp)
	var result struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return 3.0, nil
	}
	return result.Score, nil
}

func (p *DashScopeProvider) ExtractPostCallActions(ctx context.Context, transcript string) ([]string, error) {
	resp, err := p.call(ctx,
		`你是一个话后动作提取助手。从通话内容中提取坐席需要后续跟进的待办事项，返回 JSON 数组格式：["待办1","待办2"]。只返回 JSON 数组。`,
		transcript)
	if err != nil {
		return nil, err
	}

	resp = extractJSON(resp)
	var actions []string
	if err := json.Unmarshal([]byte(resp), &actions); err != nil {
		return []string{}, nil
	}
	return actions, nil
}

func (p *DashScopeProvider) AutoFillTicket(ctx context.Context, transcript string) (map[string]string, error) {
	resp, err := p.call(ctx,
		`你是一个工单自动填充助手。从通话内容中提取关键信息用于填充工单，返回 JSON 格式：{"subject":"主题","description":"描述","category":"分类","priority":"紧急程度","customer_name":"客户姓名","contact":"联系方式"}。缺失字段留空字符串。只返回 JSON。`,
		transcript)
	if err != nil {
		return nil, err
	}

	resp = extractJSON(resp)
	var fields map[string]string
	if err := json.Unmarshal([]byte(resp), &fields); err != nil {
		return map[string]string{"subject": "Auto-generated", "description": transcript}, nil
	}
	return fields, nil
}

func (p *DashScopeProvider) RecommendScript(ctx context.Context, transcript string, scripts []string) (string, error) {
	scriptList := strings.Join(scripts, "\n---\n")
	return p.call(ctx,
		"你是一个实时话术推荐助手。根据当前通话内容，从以下可用话术脚本中选择最适合的一个推荐给坐席。只返回推荐的话术内容，不要解释。\n\n可用话术：\n"+scriptList,
		transcript)
}

func (p *DashScopeProvider) QAInspectLLM(ctx context.Context, transcript, prompt string) (float64, string, error) {
	systemPrompt := `你是一个智能质检助手。根据质检提示词分析通话内容，返回 JSON 格式：{"score":0-100,"detail":"质检详情"}。只返回 JSON。`
	resp, err := p.call(ctx, systemPrompt, "质检规则："+prompt+"\n\n通话内容："+transcript)
	if err != nil {
		return 0, "", err
	}

	resp = extractJSON(resp)
	var result struct {
		Score  float64 `json:"score"`
		Detail string  `json:"detail"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return 60, "LLM analysis completed", nil
	}
	return result.Score, result.Detail, nil
}

// extractJSON tries to extract JSON content from LLM response that may contain markdown code blocks.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	}
	return strings.TrimSpace(s)
}
