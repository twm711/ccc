package aliyunsms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const smsEndpoint = "https://dysmsapi.aliyuncs.com"

// Client is an Aliyun SMS (dysmsapi) client.
type Client struct {
	accessKeyID     string
	accessKeySecret string
	httpClient      *http.Client
}

func NewClient(accessKeyID, accessKeySecret string) *Client {
	return &Client{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
}

// SendSms sends an SMS via Aliyun dysmsapi.
func (c *Client) SendSms(ctx context.Context, signName, templateID, phone string, params map[string]string) error {
	paramsJSON, _ := json.Marshal(params)

	queryParams := map[string]string{
		"AccessKeyId":      c.accessKeyID,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"RegionId":         "cn-hangzhou",
		"SignName":         signName,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   uuid.New().String(),
		"SignatureVersion": "1.0",
		"TemplateCode":     templateID,
		"TemplateParam":    string(paramsJSON),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2017-05-25",
	}

	signature := c.sign(queryParams)
	queryParams["Signature"] = signature

	vals := url.Values{}
	for k, v := range queryParams {
		vals.Set(k, v)
	}

	reqURL := smsEndpoint + "/?" + vals.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("sms: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sms: http call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("sms: read response: %w", err)
	}

	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("sms: parse response: %w", err)
	}
	if result.Code != "OK" {
		return fmt.Errorf("sms: API error %s: %s", result.Code, result.Message)
	}
	return nil
}

func (c *Client) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, specialURLEncode(k)+"="+specialURLEncode(params[k]))
	}
	canonicalized := strings.Join(pairs, "&")
	stringToSign := "GET&" + specialURLEncode("/") + "&" + specialURLEncode(canonicalized)

	mac := hmac.New(sha1.New, []byte(c.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func specialURLEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
