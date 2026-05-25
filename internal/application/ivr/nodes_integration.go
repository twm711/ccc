package ivr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/divord97/ccc/internal/domain/routing"
)

// ivrHTTPClient is a dedicated HTTP client for IVR external requests, with a
// global timeout to prevent resource leaks from misconfigured IVR flows.
var ivrHTTPClient = &http.Client{Timeout: 30 * time.Second}

// validateURL checks that a URL uses an allowed scheme (http/https) and does
// not target private/loopback addresses, mitigating SSRF risk.
func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("ivr: invalid URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("ivr: URL scheme %q not allowed, must be http or https", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "169.254.") ||
		strings.HasPrefix(host, "172.") && len(host) > 4 {
		return fmt.Errorf("ivr: URL host %q is a private/loopback address", host)
	}
	return nil
}

// FunctionHandler invokes an external Webhook/function.
type FunctionHandler struct{}

type functionConfig struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	TimeoutSec int               `json:"timeout_sec"`
	OutputVar  string            `json:"output_variable"`
}

func (h *FunctionHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg functionConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.URL == "" {
		return "error", nil
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.TimeoutSec == 0 {
		cfg.TimeoutSec = 10
	}

	body := interpolateVars(cfg.Body, sess.Variables)
	url := interpolateVars(cfg.URL, sess.Variables)

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, cfg.Method, url, strings.NewReader(body))
	if err != nil {
		if cfg.OutputVar != "" {
			sess.Variables[cfg.OutputVar] = ""
		}
		return "error", nil
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, interpolateVars(v, sess.Variables))
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if err := validateURL(url); err != nil {
		if cfg.OutputVar != "" {
			sess.Variables[cfg.OutputVar] = ""
		}
		return "error", nil
	}

	resp, err := ivrHTTPClient.Do(req)
	if err != nil {
		if cfg.OutputVar != "" {
			sess.Variables[cfg.OutputVar] = ""
		}
		return "error", nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if cfg.OutputVar != "" {
		sess.Variables[cfg.OutputVar] = string(respBody)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "success", nil
	}
	return "error", nil
}

// HTTPRequestHandler makes a direct HTTP request.
type HTTPRequestHandler struct{}

type httpRequestConfig struct {
	Method           string            `json:"method"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers"`
	Body             string            `json:"body"`
	TimeoutSec       int               `json:"timeout_seconds"`
	ResponseVariable string            `json:"response_variable"`
}

func (h *HTTPRequestHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg httpRequestConfig
	_ = parseConfig(node.Config, &cfg)

	if cfg.URL == "" {
		return "error", nil
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.TimeoutSec == 0 {
		cfg.TimeoutSec = 10
	}

	url := interpolateVars(cfg.URL, sess.Variables)
	body := interpolateVars(cfg.Body, sess.Variables)

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, cfg.Method, url, bodyReader)
	if err != nil {
		return "error", nil
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, interpolateVars(v, sess.Variables))
	}

	if err := validateURL(url); err != nil {
		return "error", nil
	}

	resp, err := ivrHTTPClient.Do(req)
	if err != nil {
		return "error", nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if cfg.ResponseVariable != "" {
		sess.Variables[cfg.ResponseVariable] = string(respBody)
	}
	sess.Variables["http_status_code"] = fmt.Sprintf("%d", resp.StatusCode)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "success", nil
	}
	return "error", nil
}

// JSONParserHandler parses JSON from a variable and extracts fields.
type JSONParserHandler struct{}

type jsonParserConfig struct {
	SourceVariable string        `json:"source_variable"`
	Mappings       []jsonMapping `json:"mappings"`
}

type jsonMapping struct {
	JSONPath       string `json:"json_path"`
	TargetVariable string `json:"target_variable"`
}

func (h *JSONParserHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg jsonParserConfig
	_ = parseConfig(node.Config, &cfg)

	src := sess.Variables[cfg.SourceVariable]
	if src == "" {
		return "error", nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(src), &data); err != nil {
		return "error", nil
	}

	for _, m := range cfg.Mappings {
		key := strings.TrimPrefix(m.JSONPath, "$.")
		if val, ok := data[key]; ok {
			sess.Variables[m.TargetVariable] = fmt.Sprintf("%v", val)
		} else {
			sess.Variables[m.TargetVariable] = ""
		}
	}
	return "success", nil
}

// SMSHandler sends an SMS message via HTTP API.
type SMSHandler struct{}

type smsConfig struct {
	To             string            `json:"to"`
	TemplateCode   string            `json:"template_code"`
	TemplateParams map[string]string `json:"template_params"`
	APIEndpoint    string            `json:"api_endpoint"`
}

func (h *SMSHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg smsConfig
	_ = parseConfig(node.Config, &cfg)

	to := cfg.To
	if to == "" {
		to = sess.Variables["caller_number"]
	}

	params := make(map[string]string)
	for k, v := range cfg.TemplateParams {
		params[k] = interpolateVars(v, sess.Variables)
	}

	endpoint := cfg.APIEndpoint
	if endpoint == "" {
		sess.Variables["sms_sent"] = "true"
		return "success", nil
	}

	payload := map[string]interface{}{
		"to":              to,
		"template_code":   cfg.TemplateCode,
		"template_params": params,
	}
	body, _ := json.Marshal(payload)

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "error", nil
	}
	req.Header.Set("Content-Type", "application/json")

	if err := validateURL(endpoint); err != nil {
		return "error", nil
	}

	resp, err := ivrHTTPClient.Do(req)
	if err != nil {
		return "error", nil
	}
	resp.Body.Close()

	sess.Variables["sms_sent"] = "true"
	return "success", nil
}

// SubFlowHandler invokes another IVR flow as a sub-flow.
type SubFlowHandler struct {
	engine     *Engine
	flowLoader func(ctx context.Context, flowID int64) (*routing.FlowGraph, error)
}

type subFlowConfig struct {
	FlowID     string            `json:"flow_id"`
	InputVars  map[string]string `json:"input_variables"`
	OutputVars []string          `json:"output_variables"`
}

func (h *SubFlowHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg subFlowConfig
	_ = parseConfig(node.Config, &cfg)

	for k, v := range cfg.InputVars {
		sess.Variables["subflow_"+k] = interpolateVars(v, sess.Variables)
	}
	sess.Variables["subflow_id"] = cfg.FlowID

	if h.engine == nil || h.flowLoader == nil {
		return "default", nil
	}

	flowID, err := strconv.ParseInt(cfg.FlowID, 10, 64)
	if err != nil {
		return "error", nil
	}
	graph, err := h.flowLoader(ctx, flowID)
	if err != nil {
		return "error", nil
	}

	subSess := &Session{
		CallID:      sess.CallID,
		TenantID:    sess.TenantID,
		FlowID:      flowID,
		CallUUID:    sess.CallUUID,
		ESL:         sess.ESL,
		ASRProvider: sess.ASRProvider,
		Variables:   make(map[string]string),
	}
	for k, v := range sess.Variables {
		subSess.Variables[k] = v
	}

	if err := h.engine.Execute(ctx, subSess, graph); err != nil {
		return "error", nil
	}

	for _, name := range cfg.OutputVars {
		if val, ok := subSess.Variables[name]; ok {
			sess.Variables[name] = val
		}
	}
	return "default", nil
}

// DigitalEmployeeHandler transfers to an AI bot.
type DigitalEmployeeHandler struct{}

type digitalEmployeeConfig struct {
	DigitalEmployeeID string `json:"digital_employee_id"`
	SceneID           string `json:"scene_id"`
	MaxTurns          int    `json:"max_turns"`
	TransferOnFailure bool   `json:"transfer_on_failure"`
}

func (h *DigitalEmployeeHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg digitalEmployeeConfig
	_ = parseConfig(node.Config, &cfg)

	sess.Variables["digital_employee_id"] = cfg.DigitalEmployeeID
	sess.Variables["digital_employee_scene"] = cfg.SceneID
	if cfg.MaxTurns > 0 {
		sess.Variables["digital_employee_max_turns"] = fmt.Sprintf("%d", cfg.MaxTurns)
	}
	if cfg.TransferOnFailure {
		sess.Variables["digital_employee_transfer_on_failure"] = "true"
	}
	return "success", nil
}

func interpolateVars(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}
