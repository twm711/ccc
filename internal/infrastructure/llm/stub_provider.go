package llm

import "context"

// StubProvider is a placeholder LLM provider that echoes input.
// Replace with Aliyun Tongyi or other provider in Phase 9.
type StubProvider struct{}

func NewStubProvider() *StubProvider { return &StubProvider{} }

func (p *StubProvider) Correct(_ context.Context, text string) (string, error) {
	return "[corrected] " + text, nil
}

func (p *StubProvider) Expand(_ context.Context, text string) (string, error) {
	return "[expanded] " + text, nil
}

func (p *StubProvider) Optimize(_ context.Context, text string) (string, error) {
	return "[optimized] " + text, nil
}
