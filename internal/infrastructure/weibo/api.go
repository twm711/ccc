package weibo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// API is a Weibo Open Platform API client for private messages.
type API struct {
	appKey    string
	appSecret string
	client    *http.Client
}

func NewAPI(appKey, appSecret string) *API {
	return &API{
		appKey:    appKey,
		appSecret: appSecret,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// VerifySignature verifies a Weibo webhook signature using HMAC-SHA256.
func (a *API) VerifySignature(body, signature string) bool {
	mac := hmac.New(sha256.New, []byte(a.appSecret))
	mac.Write([]byte(body))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// SendPrivateMessage sends a private message to a Weibo user.
func (a *API) SendPrivateMessage(ctx context.Context, accessToken, recipientID, text string) error {
	msg := map[string]interface{}{
		"type": 1,
		"receiver_id": recipientID,
		"data": map[string]string{
			"text": text,
		},
	}
	body, _ := json.Marshal(msg)

	url := fmt.Sprintf(
		"https://m.api.weibo.com/2/messages/reply.json?access_token=%s",
		accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("weibo: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("weibo: send message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("weibo: read response: %w", err)
	}

	var result struct {
		ErrorCode int    `json:"error_code"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("weibo: parse response: %w", err)
	}
	if result.ErrorCode != 0 {
		return fmt.Errorf("weibo: API error %d: %s", result.ErrorCode, result.Error)
	}
	return nil
}
