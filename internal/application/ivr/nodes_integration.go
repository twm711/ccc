package ivr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/divord97/ccc/internal/domain/routing"
)

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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "error", nil
	}
	resp.Body.Close()

	sess.Variables["sms_sent"] = "true"
	return "success", nil
}

// SubFlowHandler invokes another IVR flow as a sub-flow.
type SubFlowHandler struct{}

type subFlowConfig struct {
	FlowID     string            `json:"flow_id"`
	InputVars  map[string]string `json:"input_variables"`
	OutputVars []string          `json:"output_variables"`
}

func (h *SubFlowHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg subFlowConfig
	_ = parseConfig(node.Config, &cfg)

	for k, v := range cfg.InputVars {
		sess.Variables["subflow_"+k] = interpolateVars(v, sess.Variables)
	}
	sess.Variables["subflow_id"] = cfg.FlowID
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
	return "success", nil
}

func interpolateVars(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}
