package imassist

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
)

var (
	ErrEmptyInput = errors.New("input text cannot be empty")
)

// LLMProvider is the pluggable interface for AI text operations.
type LLMProvider interface {
	Correct(ctx context.Context, text string) (string, error)
	Expand(ctx context.Context, text string) (string, error)
	Optimize(ctx context.Context, text string) (string, error)
}

// AssistResult holds the AI-processed text.
type AssistResult struct {
	Original string `json:"original"`
	Result   string `json:"result"`
}

// Service provides IM AI assist capabilities (correct/expand/optimize).
type Service struct {
	llm    LLMProvider
	logger zerolog.Logger
}

func NewService(llm LLMProvider, logger zerolog.Logger) *Service {
	return &Service{llm: llm, logger: logger}
}

func (s *Service) Correct(ctx context.Context, text string) (*AssistResult, error) {
	if text == "" {
		return nil, ErrEmptyInput
	}
	result, err := s.llm.Correct(ctx, text)
	if err != nil {
		s.logger.Error().Err(err).Msg("ai assist: correct failed")
		return nil, err
	}
	return &AssistResult{Original: text, Result: result}, nil
}

func (s *Service) Expand(ctx context.Context, text string) (*AssistResult, error) {
	if text == "" {
		return nil, ErrEmptyInput
	}
	result, err := s.llm.Expand(ctx, text)
	if err != nil {
		s.logger.Error().Err(err).Msg("ai assist: expand failed")
		return nil, err
	}
	return &AssistResult{Original: text, Result: result}, nil
}

func (s *Service) Optimize(ctx context.Context, text string) (*AssistResult, error) {
	if text == "" {
		return nil, ErrEmptyInput
	}
	result, err := s.llm.Optimize(ctx, text)
	if err != nil {
		s.logger.Error().Err(err).Msg("ai assist: optimize failed")
		return nil, err
	}
	return &AssistResult{Original: text, Result: result}, nil
}
