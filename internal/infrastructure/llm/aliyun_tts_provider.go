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

// AliyunTTSProvider implements TTSProvider using Aliyun NLS REST API.
type AliyunTTSProvider struct {
	token      string
	appKey     string
	region     string
	voice      string
	sampleRate int
	client     *http.Client
}

func NewAliyunTTSProvider(token, appKey, region, voice string, sampleRate int) *AliyunTTSProvider {
	if region == "" {
		region = "cn-shanghai"
	}
	if voice == "" {
		voice = "zhixiaoxia"
	}
	if sampleRate == 0 {
		sampleRate = 16000
	}
	return &AliyunTTSProvider{
		token:      token,
		appKey:     appKey,
		region:     region,
		voice:      voice,
		sampleRate: sampleRate,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Synthesize converts text to speech audio bytes using Aliyun NLS REST API.
func (p *AliyunTTSProvider) Synthesize(ctx context.Context, text string, voice string) ([]byte, error) {
	if voice == "" {
		voice = p.voice
	}

	reqBody := map[string]interface{}{
		"appkey":      p.appKey,
		"text":        text,
		"format":      "wav",
		"sample_rate": p.sampleRate,
		"voice":       voice,
	}
	body, _ := json.Marshal(reqBody)

	ttsURL := fmt.Sprintf("https://nls-gateway-%s.aliyuncs.com/stream/v1/tts", p.region)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ttsURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tts: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NLS-Token", p.token)

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
