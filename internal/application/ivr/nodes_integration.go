package ivr

import (
	"context"

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

func (h *FunctionHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg functionConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: HTTP call with variable substitution
	if cfg.OutputVar != "" {
		sess.Variables[cfg.OutputVar] = "{}"
	}
	return "success", nil
}

// HTTPRequestHandler makes a direct HTTP request.
type HTTPRequestHandler struct{}

type httpRequestConfig struct {
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	TimeoutSec      int               `json:"timeout_seconds"`
	ResponseVariable string           `json:"response_variable"`
}

func (h *HTTPRequestHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg httpRequestConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: execute HTTP request with variable interpolation
	if cfg.ResponseVariable != "" {
		sess.Variables[cfg.ResponseVariable] = "{}"
	}
	return "success", nil
}

// JSONParserHandler parses JSON from a variable and extracts fields.
type JSONParserHandler struct{}

type jsonParserConfig struct {
	SourceVariable string         `json:"source_variable"`
	Mappings       []jsonMapping  `json:"mappings"`
}

type jsonMapping struct {
	JSONPath       string `json:"json_path"`
	TargetVariable string `json:"target_variable"`
}

func (h *JSONParserHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg jsonParserConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: parse JSON from sess.Variables[cfg.SourceVariable]
	// and extract values via json_path into target variables
	for _, m := range cfg.Mappings {
		sess.Variables[m.TargetVariable] = ""
	}
	return "success", nil
}

// SMSHandler sends an SMS message.
type SMSHandler struct{}

type smsConfig struct {
	To             string            `json:"to"`
	TemplateCode   string            `json:"template_code"`
	TemplateParams map[string]string `json:"template_params"`
}

func (h *SMSHandler) Handle(_ context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg smsConfig
	_ = parseConfig(node.Config, &cfg)
	// In production: call Aliyun SMS API
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
	// In production: load sub-flow graph and execute recursively
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
	// In production: hand off to AI dialog engine
	return "success", nil
}
