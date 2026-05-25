package ai

import "context"

// KnowledgeSuggestion is a search result from the knowledge retriever with relevance score.
type KnowledgeSuggestion struct {
	ArticleID int64   `json:"article_id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Score     float64 `json:"score"`
}

// KnowledgeRetriever searches the knowledge base using semantic similarity.
type KnowledgeRetriever interface {
	Search(ctx context.Context, tenantID int64, query string, topK int) ([]KnowledgeSuggestion, error)
}

// CopilotService provides real-time agent assistance during calls.
type CopilotService struct {
	retriever KnowledgeRetriever
}

// NewCopilotService creates a new copilot service.
func NewCopilotService() *CopilotService { return &CopilotService{} }

// SetRetriever configures the knowledge base retriever (RAG backend).
func (s *CopilotService) SetRetriever(r KnowledgeRetriever) { s.retriever = r }

// Suggest returns knowledge articles relevant to the current conversation context.
func (s *CopilotService) Suggest(ctx context.Context, tenantID int64, conversationText string, topK int) ([]KnowledgeSuggestion, error) {
	if s.retriever == nil {
		return nil, nil
	}
	if topK <= 0 {
		topK = 3
	}
	return s.retriever.Search(ctx, tenantID, conversationText, topK)
}
