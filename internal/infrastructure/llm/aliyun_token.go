package llm

import (
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

type nlsTokenResponse struct {
	Token struct {
		ID         string `json:"Id"`
		ExpireTime int64  `json:"ExpireTime"`
	} `json:"Token"`
	ErrMsg    string `json:"ErrMsg"`
	RequestID string `json:"RequestId"`
}

// FetchNLSToken obtains a fresh NLS token using Alibaba Cloud POP API (CreateToken).
func FetchNLSToken(accessKeyID, accessKeySecret string) (token string, expireTime int64, err error) {
	params := map[string]string{
		"AccessKeyId":      accessKeyID,
		"Action":           "CreateToken",
		"Format":           "JSON",
		"RegionId":         "cn-shanghai",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   uuid.New().String(),
		"SignatureVersion": "1.0",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2019-02-28",
	}

	signature := signPOP(params, accessKeySecret, "GET")
	params["Signature"] = signature

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}

	reqURL := "https://nls-meta.cn-shanghai.aliyuncs.com/?" + q.Encode()
	resp, err := http.Get(reqURL)
	if err != nil {
		return "", 0, fmt.Errorf("token: http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("token: read body: %w", err)
	}

	var tokenResp nlsTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("token: unmarshal: %w (body: %s)", err, string(body))
	}

	if tokenResp.Token.ID == "" {
		return "", 0, fmt.Errorf("token: empty token: %s (body: %s)", tokenResp.ErrMsg, string(body))
	}

	return tokenResp.Token.ID, tokenResp.Token.ExpireTime, nil
}

// signPOP computes the Alibaba Cloud POP API signature (HMAC-SHA1).
func signPOP(params map[string]string, accessKeySecret, httpMethod string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalParts []string
	for _, k := range keys {
		canonicalParts = append(canonicalParts,
			percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalQueryString := strings.Join(canonicalParts, "&")

	stringToSign := httpMethod + "&" + percentEncode("/") + "&" + percentEncode(canonicalQueryString)

	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
