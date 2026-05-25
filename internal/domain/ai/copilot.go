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

// NBAAction represents a Next Best Action recommendation for the agent.
type NBAAction struct {
	Action      string  `json:"action"`      // e.g. "transfer_vip", "offer_product", "create_ticket"
	Label       string  `json:"label"`       // human-readable label
	Confidence  float64 `json:"confidence"`  // 0-1
	Reason      string  `json:"reason"`      // why this action is recommended
}

// NBAProvider generates next-best-action recommendations from conversation context.
type NBAProvider interface {
	RecommendActions(ctx context.Context, tenantID int64, customerLevel string, conversationText string) ([]NBAAction, error)
}

// CopilotService provides real-time agent assistance during calls.
type CopilotService struct {
	retriever KnowledgeRetriever
	nba       NBAProvider
}

// NewCopilotService creates a new copilot service.
func NewCopilotService() *CopilotService { return &CopilotService{} }

// SetRetriever configures the knowledge base retriever (RAG backend).
func (s *CopilotService) SetRetriever(r KnowledgeRetriever) { s.retriever = r }

// SetNBAProvider configures the next-best-action recommendation engine.
func (s *CopilotService) SetNBAProvider(p NBAProvider) { s.nba = p }

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

// NextBestActions returns ranked action recommendations based on conversation
// context and customer profile. Falls back to rule-based defaults when no
// NBAProvider is configured.
func (s *CopilotService) NextBestActions(ctx context.Context, tenantID int64, customerLevel, conversationText string) ([]NBAAction, error) {
	if s.nba != nil {
		return s.nba.RecommendActions(ctx, tenantID, customerLevel, conversationText)
	}
	// Rule-based fallback when no AI provider is configured.
	return defaultNBA(customerLevel), nil
}

func defaultNBA(customerLevel string) []NBAAction {
	var actions []NBAAction
	switch customerLevel {
	case "svip":
		actions = append(actions, NBAAction{
			Action: "transfer_vip", Label: "转接VIP专线", Confidence: 0.9,
			Reason: "SVIP客户建议优先由VIP专线处理",
		})
	case "vip":
		actions = append(actions, NBAAction{
			Action: "prioritize", Label: "提升服务优先级", Confidence: 0.8,
			Reason: "VIP客户应获得优先服务",
		})
	}
	actions = append(actions, NBAAction{
		Action: "create_ticket", Label: "创建工单", Confidence: 0.5,
		Reason: "记录客户诉求以便跟进",
	})
	return actions
}
