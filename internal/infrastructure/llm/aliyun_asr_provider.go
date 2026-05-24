package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	nlsFileTransURL = "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/FlashRecognizer"
)

// AliyunASRProvider implements ASRProvider using Aliyun NLS (Intelligent Speech Interaction).
type AliyunASRProvider struct {
	accessKeyID     string
	accessKeySecret string
	appKey          string
	client          *http.Client
}

func NewAliyunASRProvider(accessKeyID, accessKeySecret, appKey string) *AliyunASRProvider {
	return &AliyunASRProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		appKey:          appKey,
		client:          &http.Client{Timeout: 120 * time.Second},
	}
}

// Transcribe sends audio to Aliyun NLS for speech-to-text via the file transcription API.
func (p *AliyunASRProvider) Transcribe(ctx context.Context, audioURL string) (string, error) {
	token, err := p.getToken(ctx)
	if err != nil {
		return "", fmt.Errorf("asr: get token: %w", err)
	}

	taskReq := map[string]interface{}{
		"appkey":          p.appKey,
		"file_link":      audioURL,
		"version":        "4.0",
		"enable_words":   true,
	}
	body, _ := json.Marshal(taskReq)

	submitURL := "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/asr"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("asr: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NLS-Token", token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asr: http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("asr: read response: %w", err)
	}

	var result struct {
		TaskID  string `json:"TaskId"`
		Status  int    `json:"StatusCode"`
		Message string `json:"StatusText"`
		Result  struct {
			Sentences []struct {
				Text string `json:"Text"`
			} `json:"Sentences"`
		} `json:"Result"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("asr: unmarshal: %w", err)
	}

	if result.Status != 200 && result.Status != 21050000 {
		return "", fmt.Errorf("asr: API error %d: %s", result.Status, result.Message)
	}

	var texts []string
	for _, s := range result.Result.Sentences {
		if s.Text != "" {
			texts = append(texts, s.Text)
		}
	}
	return joinTexts(texts), nil
}

func joinTexts(texts []string) string {
	if len(texts) == 0 {
		return ""
	}
	result := texts[0]
	for i := 1; i < len(texts); i++ {
		result += " " + texts[i]
	}
	return result
}

type nlsTokenResponse struct {
	NlsRequestID string `json:"NlsRequestId"`
	RequestID    string `json:"RequestId"`
	Token        struct {
		ID         string `json:"Id"`
		ExpireTime int64  `json:"ExpireTime"`
	} `json:"Token"`
	ErrMsg string `json:"ErrMsg"`
}

func (p *AliyunASRProvider) getToken(ctx context.Context) (string, error) {
	tokenURL := "https://nls-meta.cn-shanghai.aliyuncs.com/pop/2018-05-18/tokens"

	reqBody := map[string]string{
		"AccessKeyId":     p.accessKeyID,
		"AccessKeySecret": p.accessKeySecret,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp nlsTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("asr: parse token response: %w", err)
	}
	if tokenResp.Token.ID == "" {
		return "", fmt.Errorf("asr: empty token: %s", tokenResp.ErrMsg)
	}
	return tokenResp.Token.ID, nil
}
