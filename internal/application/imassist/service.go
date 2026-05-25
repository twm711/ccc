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

// KnowledgeSearcher searches the knowledge base for relevant articles.
type KnowledgeSearcher interface {
	Search(ctx context.Context, tenantID int64, query string, limit int) ([]KBArticle, error)
}

// KBArticle is a simplified knowledge base article for RAG context.
type KBArticle struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// AssistResult holds the AI-processed text.
type AssistResult struct {
	Original string `json:"original"`
	Result   string `json:"result"`
}

// Service provides IM AI assist capabilities (correct/expand/optimize/suggest).
type Service struct {
	llm    LLMProvider
	kb     KnowledgeSearcher
	logger zerolog.Logger
}

func NewService(llm LLMProvider, logger zerolog.Logger) *Service {
	return &Service{llm: llm, logger: logger}
}

// SetKnowledgeSearcher wires the knowledge base for RAG-powered suggestions.
func (s *Service) SetKnowledgeSearcher(kb KnowledgeSearcher) {
	s.kb = kb
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

// SuggestResult holds RAG-powered reply suggestions.
type SuggestResult struct {
	Query    string      `json:"query"`
	Articles []KBArticle `json:"articles"`
	Reply    string      `json:"reply"`
}

// Suggest searches the knowledge base for relevant articles and generates a reply suggestion.
func (s *Service) Suggest(ctx context.Context, tenantID int64, customerMessage string) (*SuggestResult, error) {
	if customerMessage == "" {
		return nil, ErrEmptyInput
	}
	result := &SuggestResult{Query: customerMessage}
	if s.kb != nil {
		articles, err := s.kb.Search(ctx, tenantID, customerMessage, 3)
		if err == nil {
			result.Articles = articles
		}
	}
	if len(result.Articles) > 0 {
		var context string
		for _, a := range result.Articles {
			context += a.Title + ": " + a.Content + "\n"
		}
		reply, err := s.llm.Expand(ctx, "Based on: "+context+"\nCustomer asks: "+customerMessage)
		if err == nil {
			result.Reply = reply
		}
	}
	return result, nil
}
