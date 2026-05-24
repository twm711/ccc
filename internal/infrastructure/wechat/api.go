package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// API is a WeChat Official Account API client.
type API struct {
	appID     string
	appSecret string
	client    *http.Client

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time
}

func NewAPI(appID, appSecret string) *API {
	return &API{
		appID:     appID,
		appSecret: appSecret,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

type accessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// GetAccessToken retrieves or refreshes the access token.
func (a *API) GetAccessToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	if a.accessToken != "" && time.Now().Before(a.expiresAt) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock.
	if a.accessToken != "" && time.Now().Before(a.expiresAt) {
		return a.accessToken, nil
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		a.appID, a.appSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("wechat: create token request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("wechat: token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("wechat: read token response: %w", err)
	}

	var result accessTokenResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("wechat: parse token: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat: token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	a.accessToken = result.AccessToken
	a.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)
	return a.accessToken, nil
}

// SendTextMessage sends a text message to a WeChat user via customer service API.
func (a *API) SendTextMessage(ctx context.Context, openID, content string) error {
	token, err := a.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	msg := map[string]interface{}{
		"touser":  openID,
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
	body, _ := json.Marshal(msg)

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("wechat: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("wechat: send message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("wechat: read send response: %w", err)
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("wechat: parse send response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat: send error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}

// SendImageMessage sends an image message to a WeChat user.
func (a *API) SendImageMessage(ctx context.Context, openID, mediaID string) error {
	token, err := a.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	msg := map[string]interface{}{
		"touser":  openID,
		"msgtype": "image",
		"image": map[string]string{
			"media_id": mediaID,
		},
	}
	body, _ := json.Marshal(msg)

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("wechat: create image send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("wechat: send image: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("wechat: parse image send response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat: send image error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}
