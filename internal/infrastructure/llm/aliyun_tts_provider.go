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

// AliyunTTSProvider implements TTSProvider using Aliyun NLS text-to-speech.
type AliyunTTSProvider struct {
	accessKeyID     string
	accessKeySecret string
	appKey          string
	client          *http.Client
}

func NewAliyunTTSProvider(accessKeyID, accessKeySecret, appKey string) *AliyunTTSProvider {
	return &AliyunTTSProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		appKey:          appKey,
		client:          &http.Client{Timeout: 30 * time.Second},
	}
}

// Synthesize converts text to speech audio bytes using Aliyun NLS REST API.
func (p *AliyunTTSProvider) Synthesize(ctx context.Context, text string, voice string) ([]byte, error) {
	token, err := p.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("tts: get token: %w", err)
	}

	if voice == "" {
		voice = "xiaoyun"
	}

	reqBody := map[string]interface{}{
		"appkey":      p.appKey,
		"text":        text,
		"format":      "wav",
		"sample_rate": 16000,
		"voice":       voice,
	}
	body, _ := json.Marshal(reqBody)

	ttsURL := "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/tts"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ttsURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tts: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NLS-Token", token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: http call: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts: read response: %w", err)
	}

	if contentType == "application/json" {
		var errResp struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Status != 200 {
			return nil, fmt.Errorf("tts: API error %d: %s", errResp.Status, errResp.Message)
		}
	}

	return respBody, nil
}

func (p *AliyunTTSProvider) getToken(ctx context.Context) (string, error) {
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
		return "", fmt.Errorf("tts: parse token: %w", err)
	}
	if tokenResp.Token.ID == "" {
		return "", fmt.Errorf("tts: empty token: %s", tokenResp.ErrMsg)
	}
	return tokenResp.Token.ID, nil
}
