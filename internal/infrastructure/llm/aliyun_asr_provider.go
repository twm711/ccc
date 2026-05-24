package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// AliyunASRProvider implements ASRProvider using Aliyun NLS one-sentence recognition REST API.
type AliyunASRProvider struct {
	token  string
	appKey string
	region string
	client *http.Client
}

func NewAliyunASRProvider(token, appKey, region string) *AliyunASRProvider {
	if region == "" {
		region = "cn-shanghai"
	}
	return &AliyunASRProvider{
		token:  token,
		appKey: appKey,
		region: region,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Transcribe sends an audio URL to the Aliyun one-sentence recognition REST API.
func (p *AliyunASRProvider) Transcribe(ctx context.Context, audioURL string) (string, error) {
	endpoint := fmt.Sprintf("https://nls-gateway-%s.aliyuncs.com/stream/v1/asr", p.region)

	params := url.Values{}
	params.Set("appkey", p.appKey)
	params.Set("format", "pcm")
	params.Set("sample_rate", "16000")
	params.Set("enable_punctuation_prediction", "true")
	params.Set("enable_inverse_text_normalization", "true")
	params.Set("audio_address", audioURL)

	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("asr: create request: %w", err)
	}
	req.Header.Set("X-NLS-Token", p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asr: http call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("asr: read response: %w", err)
	}

	var result asrResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("asr: unmarshal: %w (body: %s)", err, string(body))
	}

	if result.Status != 20000000 {
		return "", fmt.Errorf("asr: API error %d: %s", result.Status, result.Message)
	}

	return result.Result, nil
}

// TranscribeBytes sends raw audio bytes to the Aliyun one-sentence recognition REST API.
func (p *AliyunASRProvider) TranscribeBytes(ctx context.Context, audio []byte, format string, sampleRate int) (string, error) {
	endpoint := fmt.Sprintf("https://nls-gateway-%s.aliyuncs.com/stream/v1/asr", p.region)

	if format == "" {
		format = "pcm"
	}
	if sampleRate == 0 {
		sampleRate = 16000
	}

	params := url.Values{}
	params.Set("appkey", p.appKey)
	params.Set("format", format)
	params.Set("sample_rate", fmt.Sprintf("%d", sampleRate))
	params.Set("enable_punctuation_prediction", "true")
	params.Set("enable_inverse_text_normalization", "true")

	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(audio))
	if err != nil {
		return "", fmt.Errorf("asr: create request: %w", err)
	}
	req.Header.Set("X-NLS-Token", p.token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asr: http call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("asr: read response: %w", err)
	}

	var result asrResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("asr: unmarshal: %w (body: %s)", err, string(body))
	}

	if result.Status != 20000000 {
		return "", fmt.Errorf("asr: API error %d: %s", result.Status, result.Message)
	}

	return result.Result, nil
}

type asrResponse struct {
	TaskID  string `json:"task_id"`
	Result  string `json:"result"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}
