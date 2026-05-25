package http

import (
	"net/http"

	"github.com/divord97/ccc/internal/application/advancedai"
	"github.com/divord97/ccc/internal/application/agenthub"
	"github.com/divord97/ccc/internal/application/dashboard"
	"github.com/divord97/ccc/internal/application/imhub"
	"github.com/divord97/ccc/internal/application/transcripthub"
	"github.com/divord97/ccc/internal/domain/ai"
	"github.com/divord97/ccc/internal/domain/platform"
	"github.com/divord97/ccc/internal/infrastructure/redis"
	"github.com/divord97/ccc/internal/interfaces/http/handler"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type RouterDeps struct {
	// Phase 0
	TenantHandler     *handler.TenantHandler
	UserHandler       *handler.UserHandler
	AgentHandler      *handler.AgentHandler
	SkillGroupHandler *handler.SkillGroupHandler

	// Phase 1
	IVRFlowHandler       *handler.IVRFlowHandler
	CarrierHandler       *handler.CarrierHandler
	SIPTrunkHandler      *handler.SIPTrunkHandler
	PhoneNumberHandler   *handler.PhoneNumberHandler
	RecordingHandler     *handler.RecordingHandler
	VoicemailHandler     *handler.VoicemailHandler
	CallNumberTagHandler *handler.CallNumberTagHandler
	AutoTagRuleHandler   *handler.AutoTagRuleHandler

	// Phase 2
	RoutingRuleHandler *handler.RoutingRuleHandler
	CLIPolicyHandler   *handler.CLIPolicyHandler
	DNCHandler         *handler.DNCHandler
	CallHandler        *handler.CallHandler

	// Phase 3
	CallControlHandler     *handler.CallControlHandler
	AgentPresenceHandler   *handler.AgentPresenceHandler
	WebhookConfigHandler   *handler.WebhookConfigHandler
	ScreenPopConfigHandler *handler.ScreenPopConfigHandler
	QuickReplyHandler      *handler.QuickReplyHandler
	SmsConfigHandler       *handler.SmsConfigHandler

	// Phase 4
	DashboardHandler *handler.DashboardHandler
	ReportHandler    *handler.ReportHandler
	CSATHandler      *handler.CSATHandler

	// Phase 5
	ProfileHandler *handler.ProfileHandler

	// Phase 6
	CampaignHandler   *handler.CampaignHandler
	B2BHandler        *handler.B2BHandler
	TrunkGroupHandler *handler.TrunkGroupHandler

	// Phase 7
	CustomerHandler    *handler.CustomerHandler
	TicketHandler      *handler.TicketHandler
	KnowledgeHandler   *handler.KnowledgeHandler
	AgentScriptHandler *handler.AgentScriptHandler
	SessionInfoHandler *handler.SessionInfoHandler

	// Phase 8
	IMChannelHandler    *handler.IMChannelHandler
	IMSessionHandler    *handler.IMSessionHandler
	WidgetHandler       *handler.WidgetHandler
	EmailInboundHandler *handler.EmailInboundHandler
	IMAssistHandler     *handler.IMAssistHandler

	// Phase 9
	DigitalEmployeeHandler *handler.DigitalEmployeeHandler
	QAHandler              *handler.QAHandler
	AIAnalysisHandler      *handler.AIAnalysisHandler
	ASRHotwordsHandler     *handler.ASRHotwordsHandler
	PerformanceHandler     *handler.PerformanceHandler

	// Phase 10
	AnnotationHandler    *handler.AnnotationHandler
	LLMGatewayHandler    *handler.LLMGatewayHandler
	WebRTCQualityHandler *handler.WebRTCQualityHandler

	// STT/TTS
	STTHandler *handler.STTHandler
	TTSHandler *handler.TTSHandler

	// Advanced AI
	CommAgentSvc  *ai.CommAgentService
	VoiceSvc      *ai.VoiceProfileService
	AnalysisSvc   *ai.ConversationAnalysisService
	TrainingSvc   *ai.TrainingService
	RingSvc       *ai.RingAnalysisService
	FullDuplexSvc *ai.FullDuplexService
	AdvancedAISvc *advancedai.Service

	// Config
	BreakReasonHandler     *handler.BreakReasonHandler
	DispositionCodeHandler *handler.DispositionCodeHandler
	AudioFileHandler       *handler.AudioFileHandler
	BusinessHoursHandler   *handler.BusinessHoursHandler
	CallTagDefHandler      *handler.CallTagDefHandler
	AuditLogHandler        *handler.AuditLogHandler

	// Social Channels
	SocialChannelHandler *handler.SocialChannelHandler

	// Tenant Settings
	TenantSettingsHandler *handler.TenantSettingsHandler

	// Phone Component Extras
	SupervisorHandler  *handler.SupervisorHandler
	ScreenPopHandler   *handler.ScreenPopHandler
	PreviewCaseHandler *handler.PreviewCaseHandler

	// Auth
	AuthHandler *handler.AuthHandler

	// WebSocket Hubs
	DashboardHub  *dashboard.Hub
	IMHub         *imhub.Hub
	AgentHub      *agenthub.Hub
	TranscriptHub *transcripthub.Hub

	// Infrastructure
	RateLimiter        *redis.RateLimiter
	TenantSettingsRepo middleware.TenantRateProvider
	AuditLogRepo       platform.AuditLogRepository
	JWTSecret          string
	ServiceAuthSecret  string
	Logger             zerolog.Logger
}

func NewRouter(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Tracing)
	r.Use(middleware.CORS)
	r.Use(middleware.Metrics)
	r.Use(middleware.RequestLogger(deps.Logger))

	r.Get("/health", handler.Health)
	r.Get("/readyz", handler.Readyz)
	r.Handle("/metrics", promhttp.Handler())

	// --- Public Auth Route (no JWT) ---
	r.Post("/api/v1/auth/login", deps.AuthHandler.Login)
	r.Post("/api/v1/auth/refresh", deps.AuthHandler.RefreshToken)

	// --- WebSocket Routes (auth via query param) ---
	if deps.DashboardHub != nil {
		r.Get("/api/v1/ws/dashboard", func(w http.ResponseWriter, r *http.Request) {
			deps.DashboardHub.ServeWS(w, r)
		})
	}
	if deps.IMHub != nil {
		r.Get("/api/v1/ws/im", func(w http.ResponseWriter, r *http.Request) {
			deps.IMHub.ServeWS(w, r)
		})
	}
	if deps.AgentHub != nil {
		r.Get("/api/v1/ws/agent-events", func(w http.ResponseWriter, r *http.Request) {
			deps.AgentHub.ServeWS(w, r)
		})
	}
	if deps.TranscriptHub != nil {
		r.Get("/api/v1/ws/transcript", func(w http.ResponseWriter, r *http.Request) {
			deps.TranscriptHub.ServeWS(w, r)
		})
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(deps.JWTSecret))
		r.Use(middleware.TenantGuard())
		r.Use(middleware.RateLimit(deps.RateLimiter, deps.TenantSettingsRepo, 100))
		r.Use(middleware.AuditLog(deps.AuditLogRepo))
		r.Use(middleware.PIIRedact())

		// --- Phase 0 Routes ---

		r.Route("/tenants", func(r chi.Router) {
			r.Post("/", deps.TenantHandler.Create)
			r.Get("/", deps.TenantHandler.List)
			r.Get("/{id}", deps.TenantHandler.Get)
			r.Put("/{id}", deps.TenantHandler.Update)
		})

		r.Route("/users", func(r chi.Router) {
			r.Post("/", deps.UserHandler.Create)
			r.Get("/", deps.UserHandler.List)
			r.Get("/{id}", deps.UserHandler.Get)
			r.Put("/{id}", deps.UserHandler.Update)
		})

		r.Route("/agents", func(r chi.Router) {
			r.Post("/", deps.AgentHandler.Create)
			r.Get("/", deps.AgentHandler.List)
			r.Get("/{id}", deps.AgentHandler.Get)
		})

		r.Route("/skill-groups", func(r chi.Router) {
			r.Post("/", deps.SkillGroupHandler.Create)
			r.Get("/", deps.SkillGroupHandler.List)
			r.Get("/{id}", deps.SkillGroupHandler.Get)
			r.Post("/{id}/members", deps.SkillGroupHandler.AddMember)
			r.Delete("/{id}/members/{agentId}", deps.SkillGroupHandler.RemoveMember)
			r.Get("/{id}/members", deps.SkillGroupHandler.GetMembers)
		})

		// --- Phase 1 Routes ---

		r.Route("/ivr-flows", func(r chi.Router) {
			r.Post("/", deps.IVRFlowHandler.Create)
			r.Get("/", deps.IVRFlowHandler.List)
			r.Get("/{id}", deps.IVRFlowHandler.Get)
			r.Put("/{id}", deps.IVRFlowHandler.Update)
			r.Post("/{id}/publish", deps.IVRFlowHandler.Publish)
			r.Post("/{id}/lock", deps.IVRFlowHandler.Lock)
			r.Post("/{id}/unlock", deps.IVRFlowHandler.Unlock)
			r.Post("/{id}/clone", deps.IVRFlowHandler.Clone)
			r.Get("/{id}/versions", deps.IVRFlowHandler.Versions)
			r.Post("/{id}/rollback/{version}", deps.IVRFlowHandler.Rollback)
		})

		r.Route("/carriers", func(r chi.Router) {
			r.Post("/", deps.CarrierHandler.Create)
			r.Get("/", deps.CarrierHandler.List)
		})

		r.Route("/sip-trunks", func(r chi.Router) {
			r.Post("/", deps.SIPTrunkHandler.Create)
			r.Get("/", deps.SIPTrunkHandler.List)
			r.Get("/{id}", deps.SIPTrunkHandler.Get)
			r.Put("/{id}", deps.SIPTrunkHandler.Update)
		})

		r.Route("/phone-numbers", func(r chi.Router) {
			r.Post("/", deps.PhoneNumberHandler.Create)
			r.Get("/", deps.PhoneNumberHandler.List)
			r.Get("/{id}", deps.PhoneNumberHandler.Get)
			r.Put("/{id}", deps.PhoneNumberHandler.Update)
		})

		r.Route("/call-number-tags", func(r chi.Router) {
			r.Post("/", deps.CallNumberTagHandler.Create)
			r.Get("/", deps.CallNumberTagHandler.List)
			r.Delete("/{id}", deps.CallNumberTagHandler.Delete)
		})

		r.Route("/auto-tag-rules", func(r chi.Router) {
			r.Post("/", deps.AutoTagRuleHandler.Create)
			r.Get("/", deps.AutoTagRuleHandler.List)
			r.Put("/{id}", deps.AutoTagRuleHandler.Update)
			r.Delete("/{id}", deps.AutoTagRuleHandler.Delete)
		})

		r.Route("/recordings", func(r chi.Router) {
			r.Get("/", deps.RecordingHandler.List)
			r.Get("/{id}", deps.RecordingHandler.Get)
			r.Get("/{id}/stream", deps.RecordingHandler.Stream)
			r.Get("/{id}/download", deps.RecordingHandler.Download)
		})

		r.Route("/voicemails", func(r chi.Router) {
			r.Get("/", deps.VoicemailHandler.List)
			r.Get("/{id}", deps.VoicemailHandler.Get)
			r.Put("/{id}/read", deps.VoicemailHandler.MarkRead)
			r.Delete("/{id}", deps.VoicemailHandler.Delete)
		})

		// --- Phase 2 Routes ---

		r.Route("/routing-rules", func(r chi.Router) {
			r.Post("/", deps.RoutingRuleHandler.Create)
			r.Get("/", deps.RoutingRuleHandler.List)
			r.Put("/{id}", deps.RoutingRuleHandler.Update)
			r.Delete("/{id}", deps.RoutingRuleHandler.Delete)
		})

		r.Route("/cli-policies", func(r chi.Router) {
			r.Post("/", deps.CLIPolicyHandler.Create)
			r.Get("/", deps.CLIPolicyHandler.List)
			r.Put("/{id}", deps.CLIPolicyHandler.Update)
		})

		r.Route("/dnc-list", func(r chi.Router) {
			r.Post("/", deps.DNCHandler.Create)
			r.Get("/", deps.DNCHandler.List)
			r.Delete("/{id}", deps.DNCHandler.Delete)
			r.Post("/check", deps.DNCHandler.Check)
		})

		r.Route("/calls", func(r chi.Router) {
			r.Get("/", deps.CallHandler.List)
			r.Get("/cursor", deps.CallHandler.ListCursor)
			r.Get("/export", deps.CallHandler.Export)
			r.Get("/{id}", deps.CallHandler.Get)
			r.Get("/{id}/events", deps.CallHandler.GetEvents)
			r.Get("/{id}/ivr-tracking", deps.CallHandler.GetIVRTracking)
			r.Post("/dial", deps.CallHandler.Dial)
			r.Post("/internal-dial", deps.CallHandler.InternalDial)
			r.Post("/{id}/tags", deps.CallHandler.AddTag)
			r.Delete("/{id}/tags/{tagId}", deps.CallHandler.RemoveTag)
			r.Post("/{id}/hold", deps.CallControlHandler.Hold)
			r.Post("/{id}/retrieve", deps.CallControlHandler.Retrieve)
			r.Post("/{id}/blind-transfer", deps.CallControlHandler.BlindTransfer)
			r.Post("/{id}/dtmf", deps.CallControlHandler.SendDTMF)
		})

		r.Post("/callbacks", deps.CallControlHandler.RequestCallback)

		// --- Phase 3 Routes ---

		r.Route("/agent-presence", func(r chi.Router) {
			r.Get("/", deps.AgentPresenceHandler.ListByTenant)
			r.Post("/check-in", deps.AgentPresenceHandler.CheckIn)
			r.Post("/status", deps.AgentPresenceHandler.ChangeStatus)
			r.Post("/{agentId}/check-out", deps.AgentPresenceHandler.CheckOut)
			r.Post("/{agentId}/transition", deps.AgentPresenceHandler.Transition)
			r.Post("/{agentId}/break", deps.AgentPresenceHandler.SetBreak)
			r.Post("/{agentId}/acw", deps.AgentPresenceHandler.SetACW)
			r.Post("/{agentId}/work-mode", deps.AgentPresenceHandler.SwitchWorkMode)
			r.Get("/{agentId}", deps.AgentPresenceHandler.GetPresence)
		})

		r.Route("/webhook-configs", func(r chi.Router) {
			r.Post("/", deps.WebhookConfigHandler.Create)
			r.Get("/", deps.WebhookConfigHandler.List)
			r.Put("/{id}", deps.WebhookConfigHandler.Update)
			r.Delete("/{id}", deps.WebhookConfigHandler.Delete)
		})

		r.Route("/screen-pop-configs", func(r chi.Router) {
			r.Post("/", deps.ScreenPopConfigHandler.Create)
			r.Get("/", deps.ScreenPopConfigHandler.List)
			r.Put("/{id}", deps.ScreenPopConfigHandler.Update)
			r.Delete("/{id}", deps.ScreenPopConfigHandler.Delete)
		})

		r.Route("/quick-replies", func(r chi.Router) {
			r.Post("/", deps.QuickReplyHandler.Create)
			r.Get("/", deps.QuickReplyHandler.List)
			r.Put("/{id}", deps.QuickReplyHandler.Update)
			r.Delete("/{id}", deps.QuickReplyHandler.Delete)
			r.Get("/available", deps.QuickReplyHandler.Available)
		})

		r.Route("/sms-configs", func(r chi.Router) {
			r.Post("/", deps.SmsConfigHandler.Create)
			r.Get("/", deps.SmsConfigHandler.List)
			r.Put("/{id}", deps.SmsConfigHandler.Update)
			r.Delete("/{id}", deps.SmsConfigHandler.Delete)
		})

		// --- Phase 4 Routes ---

		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/overview", deps.DashboardHandler.Overview)
			r.Get("/agents", deps.DashboardHandler.AgentStatus)
			r.Get("/skill-groups", deps.DashboardHandler.SkillGroupStatus)
			r.Get("/trend", deps.DashboardHandler.CallTrend)
			r.Get("/funnel", deps.DashboardHandler.CallFunnel)
		})

		r.Route("/reports", func(r chi.Router) {
			r.Get("/agents", deps.ReportHandler.AgentReport)
			r.Get("/agents/export", deps.ReportHandler.AgentReportExport)
			r.Get("/group-agents", deps.ReportHandler.GroupAgentReport)
			r.Get("/group-agents/export", deps.ReportHandler.GroupAgentReportExport)
			r.Get("/skill-groups", deps.ReportHandler.SkillGroupReport)
			r.Get("/skill-groups/export", deps.ReportHandler.SkillGroupReportExport)
			r.Get("/back2back", deps.ReportHandler.Back2BackReport)
			r.Get("/internal-calls", deps.ReportHandler.InternalCallReport)
			r.Get("/agent-status-log", deps.ReportHandler.AgentStatusLog)
			r.Get("/campaigns", deps.ReportHandler.CampaignReport)
		})

		r.Route("/csat", func(r chi.Router) {
			r.Post("/config", deps.CSATHandler.CreateConfig)
			r.Get("/config", deps.CSATHandler.ListConfigs)
			r.Put("/config", deps.CSATHandler.UpdateConfig)
			r.Get("/results", deps.CSATHandler.ListResults)
		})

		// --- Phase 5 Routes ---

		// Advanced call control (added to existing /calls route)
		r.Post("/calls/{id}/queue", deps.CallControlHandler.TransitionToQueue)
		r.Post("/calls/{id}/ring", deps.CallControlHandler.TransitionToRinging)
		r.Post("/calls/{id}/disposition", deps.CallControlHandler.SetDisposition)
		r.Post("/calls/{id}/attended-transfer", deps.CallControlHandler.AttendedTransfer)
		// Campaign resume
		r.Post("/campaigns/{id}/resume", deps.CampaignHandler.Start)
		r.Post("/calls/{id}/consult", deps.CallControlHandler.Consult)
		r.Post("/calls/{id}/consult-transfer", deps.CallControlHandler.ConsultTransfer)
		r.Post("/calls/{id}/consult-cancel", deps.CallControlHandler.ConsultCancel)
		r.Post("/calls/{id}/conference", deps.CallControlHandler.Conference)
		r.Post("/calls/{id}/monitor", deps.CallControlHandler.Monitor)
		r.Post("/calls/{id}/whisper", deps.CallControlHandler.Whisper)
		r.Post("/calls/{id}/barge", deps.CallControlHandler.Barge)
		r.Post("/calls/{id}/intercept", deps.CallControlHandler.InterceptCall)
		r.Post("/calls/{id}/coach", deps.CallControlHandler.Coach)
		r.Post("/calls/{id}/end", deps.CallControlHandler.EndCall)
		r.Post("/calls/{id}/answer", deps.CallControlHandler.AnswerCall)

		r.Route("/me", func(r chi.Router) {
			r.Get("/overview", deps.ProfileHandler.Overview)
			r.Get("/profile", deps.ProfileHandler.Overview)
			r.Put("/profile", deps.ProfileHandler.UpdateProfile)
			r.Put("/password", deps.ProfileHandler.ChangePassword)
			r.Post("/reset-state", deps.ProfileHandler.ResetState)
		})

		// --- Phase 6 Routes ---

		r.Route("/campaigns", func(r chi.Router) {
			r.Post("/", deps.CampaignHandler.Create)
			r.Get("/", deps.CampaignHandler.List)
			r.Get("/{id}", deps.CampaignHandler.GetByID)
			r.Put("/{id}", deps.CampaignHandler.Update)
			r.Post("/{id}/start", deps.CampaignHandler.Start)
			r.Post("/{id}/pause", deps.CampaignHandler.Pause)
			r.Post("/{id}/abort", deps.CampaignHandler.Abort)
			r.Post("/{id}/cases/import", deps.CampaignHandler.ImportCases)
			r.Get("/{id}/cases", deps.CampaignHandler.ListCases)
			r.Get("/{id}/stats", deps.CampaignHandler.Stats)
		})

		// Phone Component Extra Routes
		r.Get("/supervisor/active-calls", deps.SupervisorHandler.ActiveCalls)
		r.Get("/screen-pop/lookup", deps.ScreenPopHandler.Lookup)
		r.Get("/campaigns/preview/current", deps.PreviewCaseHandler.Current)
		r.Post("/campaigns/{campaignId}/cases/{caseId}/dial", deps.PreviewCaseHandler.DialCase)
		r.Post("/campaigns/{campaignId}/cases/{caseId}/skip", deps.PreviewCaseHandler.SkipCase)

		r.Post("/calls/back2back", deps.B2BHandler.Back2BackCall)
		r.Post("/calls/encrypted", deps.B2BHandler.EncryptedCall)
		r.Post("/flash-sms", deps.B2BHandler.FlashSMS)

		r.Route("/sip-trunk-groups", func(r chi.Router) {
			r.Post("/", deps.TrunkGroupHandler.Create)
			r.Get("/", deps.TrunkGroupHandler.List)
			r.Post("/{id}/members", deps.TrunkGroupHandler.AddMember)
			r.Get("/{id}/members", deps.TrunkGroupHandler.ListMembers)
		})

		// --- Phase 7 Routes ---

		r.Route("/customers", func(r chi.Router) {
			r.Post("/", deps.CustomerHandler.Create)
			r.Get("/", deps.CustomerHandler.List)
			r.Get("/{id}", deps.CustomerHandler.GetByID)
			r.Put("/{id}", deps.CustomerHandler.Update)
			r.Delete("/{id}", deps.CustomerHandler.Delete)
			r.Post("/import", deps.CustomerHandler.Import)
			r.Get("/by-phone/{phone}", deps.CustomerHandler.FindByPhone)
			r.Get("/{id}/interactions", deps.CustomerHandler.ListInteractions)
		})

		r.Route("/custom-fields", func(r chi.Router) {
			r.Post("/", deps.CustomerHandler.CreateFieldDefinition)
			r.Get("/", deps.CustomerHandler.ListFieldDefinitions)
		})

		r.Route("/ticket-categories", func(r chi.Router) {
			r.Post("/", deps.TicketHandler.CreateCategory)
			r.Get("/", deps.TicketHandler.ListCategories)
		})

		r.Route("/ticket-templates", func(r chi.Router) {
			r.Post("/", deps.TicketHandler.CreateTemplate)
			r.Get("/", deps.TicketHandler.ListTemplates)
			r.Get("/{id}", deps.TicketHandler.GetTemplate)
			r.Put("/{id}", deps.TicketHandler.UpdateTemplate)
			r.Post("/{id}/publish", deps.TicketHandler.PublishTemplate)
			r.Post("/{id}/offline", deps.TicketHandler.OfflineTemplate)
		})

		r.Route("/tickets", func(r chi.Router) {
			r.Post("/", deps.TicketHandler.Create)
			r.Get("/", deps.TicketHandler.ListTickets)
			r.Get("/{id}", deps.TicketHandler.GetTicket)
			r.Put("/{id}", deps.TicketHandler.UpdateTicket)
			r.Post("/{id}/assign", deps.TicketHandler.AssignTicket)
			r.Post("/{id}/comments", deps.TicketHandler.AddComment)
		})

		r.Route("/knowledge", func(r chi.Router) {
			r.Route("/categories", func(r chi.Router) {
				r.Post("/", deps.KnowledgeHandler.CreateCategory)
				r.Get("/", deps.KnowledgeHandler.ListCategories)
			})
			r.Route("/articles", func(r chi.Router) {
				r.Post("/", deps.KnowledgeHandler.CreateArticle)
				r.Get("/", deps.KnowledgeHandler.ListArticles)
				r.Get("/search", deps.KnowledgeHandler.Search)
				r.Get("/{id}", deps.KnowledgeHandler.GetArticle)
				r.Put("/{id}", deps.KnowledgeHandler.UpdateArticle)
			})
		})

		r.Route("/agent-scripts", func(r chi.Router) {
			r.Post("/", deps.AgentScriptHandler.Create)
			r.Get("/", deps.AgentScriptHandler.List)
			r.Put("/{id}", deps.AgentScriptHandler.Update)
		})

		r.Route("/session-info-templates", func(r chi.Router) {
			r.Post("/", deps.SessionInfoHandler.Create)
			r.Get("/", deps.SessionInfoHandler.List)
			r.Put("/{id}", deps.SessionInfoHandler.Update)
		})

		// --- Phase 8 Routes ---

		r.Route("/im", func(r chi.Router) {
			r.Route("/channels", func(r chi.Router) {
				r.Post("/", deps.IMChannelHandler.Create)
				r.Get("/", deps.IMChannelHandler.List)
				r.Put("/{id}", deps.IMChannelHandler.Update)
			})
			r.Route("/sessions", func(r chi.Router) {
				r.Get("/", deps.IMSessionHandler.List)
				r.Get("/{id}", deps.IMSessionHandler.Get)
				r.Post("/{id}/transfer", deps.IMSessionHandler.Transfer)
				r.Post("/{id}/close", deps.IMSessionHandler.Close)
				r.Get("/{id}/messages", deps.IMSessionHandler.ListMessages)
				r.Post("/{id}/messages", deps.IMSessionHandler.SendMessage)
			})
		})

		r.Route("/im/ai-assist", func(r chi.Router) {
			r.Post("/correct", deps.IMAssistHandler.Correct)
			r.Post("/expand", deps.IMAssistHandler.Expand)
			r.Post("/optimize", deps.IMAssistHandler.Optimize)
		})

		// --- Phase 9 Routes ---

		r.Route("/digital-employees", func(r chi.Router) {
			r.Post("/", deps.DigitalEmployeeHandler.Create)
			r.Get("/", deps.DigitalEmployeeHandler.List)
			r.Get("/{id}", deps.DigitalEmployeeHandler.Get)
			r.Put("/{id}", deps.DigitalEmployeeHandler.Update)
			r.Post("/{id}/scenes", deps.DigitalEmployeeHandler.CreateScene)
			r.Get("/{id}/scenes", deps.DigitalEmployeeHandler.ListScenes)
			r.Post("/{id}/scenes/{sceneId}/publish", deps.DigitalEmployeeHandler.PublishScene)
			r.Post("/{id}/scenes/{sceneId}/test", deps.DigitalEmployeeHandler.TestIntent)
		})

		r.Route("/qa/rules", func(r chi.Router) {
			r.Post("/", deps.QAHandler.CreateRule)
			r.Get("/", deps.QAHandler.ListRules)
			r.Get("/{id}", deps.QAHandler.GetRule)
			r.Put("/{id}", deps.QAHandler.UpdateRule)
			r.Delete("/{id}", deps.QAHandler.DeleteRule)
		})

		r.Route("/qa/schemes", func(r chi.Router) {
			r.Post("/", deps.QAHandler.CreateScheme)
			r.Get("/", deps.QAHandler.ListSchemes)
			r.Get("/{id}", deps.QAHandler.GetScheme)
			r.Put("/{id}", deps.QAHandler.UpdateScheme)
			r.Delete("/{id}", deps.QAHandler.DeleteScheme)
		})

		r.Route("/qa", func(r chi.Router) {
			r.Post("/run", deps.QAHandler.RunInspection)
			r.Get("/results", deps.QAHandler.ListResults)
			r.Get("/results/{id}", deps.QAHandler.GetResult)
			r.Post("/results/{id}/appeal", deps.QAHandler.Appeal)
			r.Post("/results/{id}/review", deps.QAHandler.Review)
		})

		// AI call analysis endpoints
		r.Post("/calls/{callId}/ai-summary", deps.AIAnalysisHandler.Summary)
		r.Post("/calls/{callId}/ai-sentiment", deps.AIAnalysisHandler.Sentiment)
		r.Post("/calls/{callId}/ai-tags", deps.AIAnalysisHandler.Tags)
		r.Post("/calls/{callId}/ai-satisfaction", deps.AIAnalysisHandler.Satisfaction)
		r.Post("/calls/{callId}/ai-ivr-analysis", deps.AIAnalysisHandler.IVRAnalysis)
		r.Post("/calls/{callId}/ai-completion", deps.AIAnalysisHandler.Completion)
		r.Post("/calls/{callId}/ai-post-actions", deps.AIAnalysisHandler.PostCallActions)
		r.Post("/calls/{callId}/auto-fill-ticket", deps.AIAnalysisHandler.AutoFill)
		r.Post("/calls/{callId}/script-recommendations", deps.AIAnalysisHandler.ScriptRecommend)
		r.Get("/ai/script-recommend/{callId}", deps.AIAnalysisHandler.ScriptRecommend)
		r.Post("/session-tag-analysis", deps.AIAnalysisHandler.BatchTags)
		r.Post("/hotword-analysis", deps.AIAnalysisHandler.HotwordAnalysis)

		// STT/TTS API endpoints
		r.Post("/ai/stt/transcribe", deps.STTHandler.Transcribe)
		r.Post("/ai/tts/synthesize", deps.TTSHandler.Synthesize)

		r.Route("/asr-hotwords", func(r chi.Router) {
			r.Post("/", deps.ASRHotwordsHandler.Create)
			r.Get("/", deps.ASRHotwordsHandler.List)
			r.Get("/{id}", deps.ASRHotwordsHandler.Get)
			r.Put("/{id}", deps.ASRHotwordsHandler.Update)
			r.Delete("/{id}", deps.ASRHotwordsHandler.Delete)
		})

		r.Route("/performance-scorecards", func(r chi.Router) {
			r.Get("/", deps.PerformanceHandler.List)
			r.Post("/generate", deps.PerformanceHandler.Generate)
		})

		// --- Phase 10 Routes ---

		r.Route("/annotation-tasks", func(r chi.Router) {
			r.Post("/", deps.AnnotationHandler.CreateTask)
			r.Get("/", deps.AnnotationHandler.ListTasks)
			r.Get("/{id}", deps.AnnotationHandler.GetTask)
			r.Post("/{id}/start", deps.AnnotationHandler.StartTask)
			r.Post("/{id}/complete", deps.AnnotationHandler.CompleteTask)
			r.Post("/{id}/cancel", deps.AnnotationHandler.CancelTask)
			r.Post("/{id}/annotations", deps.AnnotationHandler.SubmitAnnotation)
			r.Get("/{id}/annotations", deps.AnnotationHandler.ListResults)
		})

		r.Route("/llm-models", func(r chi.Router) {
			r.Post("/", deps.LLMGatewayHandler.CreateConfig)
			r.Get("/", deps.LLMGatewayHandler.ListConfigs)
			r.Get("/default", deps.LLMGatewayHandler.GetDefault)
			r.Get("/{id}", deps.LLMGatewayHandler.GetConfig)
			r.Put("/{id}", deps.LLMGatewayHandler.UpdateConfig)
			r.Delete("/{id}", deps.LLMGatewayHandler.DeleteConfig)
		})

		r.Post("/webrtc-quality", deps.WebRTCQualityHandler.Save)
		r.Get("/calls/{callId}/webrtc-quality", deps.WebRTCQualityHandler.ListByCall)
		r.Get("/calls/{callId}/tickets", deps.TicketHandler.ListByCall)
		r.Get("/agents/{agentId}/webrtc-quality", deps.WebRTCQualityHandler.ListByAgent)

		// --- Advanced AI Routes ---

		r.Route("/comm-agents", func(r chi.Router) {
			r.Get("/", handler.ListCommAgents(deps.CommAgentSvc))
			r.Post("/", handler.CreateCommAgent(deps.CommAgentSvc))
			r.Get("/{id}", handler.GetCommAgent(deps.CommAgentSvc))
			r.Delete("/{id}", handler.DeleteCommAgent(deps.CommAgentSvc))
		})

		r.Route("/voice-profiles", func(r chi.Router) {
			r.Get("/", handler.ListVoiceProfiles(deps.VoiceSvc))
			r.Post("/", handler.CreateVoiceProfile(deps.VoiceSvc))
			r.Get("/{id}", handler.GetVoiceProfile(deps.VoiceSvc))
			r.Post("/{id}/train", handler.StartVoiceTraining(deps.VoiceSvc))
			r.Delete("/{id}", handler.DeleteVoiceProfile(deps.VoiceSvc))
		})

		// Frontend AI aliases
		r.Route("/ai/voice-clone", func(r chi.Router) {
			r.Get("/tasks", handler.ListVoiceProfiles(deps.VoiceSvc))
			r.Post("/tasks", handler.CreateVoiceProfile(deps.VoiceSvc))
			r.Get("/tasks/{id}", handler.GetVoiceProfile(deps.VoiceSvc))
		})

		r.Route("/conversation-analysis", func(r chi.Router) {
			r.Get("/", handler.ListAnalysisTasks(deps.AnalysisSvc))
			r.Post("/", handler.CreateAnalysisTask(deps.AnalysisSvc))
			r.Get("/{id}", handler.GetAnalysisTask(deps.AnalysisSvc))
			r.Post("/{id}/run", handler.RunAnalysisTask(deps.AdvancedAISvc))
		})
		r.Post("/ai/conversation-analytics/analyze", handler.CreateAnalysisTask(deps.AnalysisSvc))

		r.Route("/training/courses", func(r chi.Router) {
			r.Get("/", handler.ListCourses(deps.TrainingSvc))
			r.Post("/", handler.CreateCourse(deps.TrainingSvc))
			r.Get("/{id}", handler.GetCourse(deps.TrainingSvc))
			r.Post("/{id}/publish", handler.PublishCourse(deps.TrainingSvc))
		})
		r.Post("/training/exams", handler.SubmitExam(deps.TrainingSvc))
		r.Get("/training/agents/{agentID}/exams", handler.ListExamsByAgent(deps.TrainingSvc))
		r.Post("/training/simulated-calls", handler.CreateSimulatedCall(deps.TrainingSvc))
		r.Get("/training/agents/{agentID}/simulated-calls", handler.ListSimulatedCalls(deps.TrainingSvc))
		// Frontend AI training aliases
		r.Post("/ai/training/generate-questions", handler.CreateCourse(deps.TrainingSvc))
		r.Post("/ai/training/evaluate", handler.SubmitExam(deps.TrainingSvc))

		r.Get("/ring-analysis/config", handler.GetRingAnalysisConfig(deps.RingSvc))
		r.Put("/ring-analysis/config", handler.UpsertRingAnalysisConfig(deps.RingSvc))
		r.Get("/calls/{callID}/ring-analysis", handler.GetRingAnalysisLogs(deps.RingSvc))

		r.Get("/full-duplex/config", handler.GetFullDuplexConfig(deps.FullDuplexSvc))
		r.Put("/full-duplex/config", handler.UpsertFullDuplexConfig(deps.FullDuplexSvc))

		// --- Config Routes ---
		r.Route("/break-reasons", func(r chi.Router) {
			r.Post("/", deps.BreakReasonHandler.Create)
			r.Get("/", deps.BreakReasonHandler.List)
			r.Get("/{id}", deps.BreakReasonHandler.Get)
			r.Put("/{id}", deps.BreakReasonHandler.Update)
			r.Delete("/{id}", deps.BreakReasonHandler.Delete)
		})

		r.Route("/disposition-codes", func(r chi.Router) {
			r.Post("/", deps.DispositionCodeHandler.Create)
			r.Get("/", deps.DispositionCodeHandler.List)
			r.Get("/{id}", deps.DispositionCodeHandler.Get)
			r.Put("/{id}", deps.DispositionCodeHandler.Update)
			r.Delete("/{id}", deps.DispositionCodeHandler.Delete)
		})

		r.Route("/audio-files", func(r chi.Router) {
			r.Post("/", deps.AudioFileHandler.Create)
			r.Get("/", deps.AudioFileHandler.List)
			r.Get("/{id}", deps.AudioFileHandler.Get)
			r.Delete("/{id}", deps.AudioFileHandler.Delete)
		})

		r.Route("/business-hours", func(r chi.Router) {
			r.Post("/", deps.BusinessHoursHandler.Create)
			r.Get("/", deps.BusinessHoursHandler.List)
			r.Get("/{id}", deps.BusinessHoursHandler.Get)
			r.Put("/{id}", deps.BusinessHoursHandler.Update)
			r.Delete("/{id}", deps.BusinessHoursHandler.Delete)
		})

		r.Route("/call-tags", func(r chi.Router) {
			r.Post("/", deps.CallTagDefHandler.Create)
			r.Get("/", deps.CallTagDefHandler.List)
			r.Get("/{id}", deps.CallTagDefHandler.Get)
			r.Put("/{id}", deps.CallTagDefHandler.Update)
			r.Delete("/{id}", deps.CallTagDefHandler.Delete)
		})

		r.Route("/audit-logs", func(r chi.Router) {
			r.Get("/", deps.AuditLogHandler.List)
		})

		// --- Tenant Settings ---
		r.Route("/tenant-settings", func(r chi.Router) {
			r.Get("/", deps.TenantSettingsHandler.Get)
			r.Put("/", deps.TenantSettingsHandler.Update)
		})

		// --- Social Channel Config ---
		r.Route("/social-channels", func(r chi.Router) {
			r.Post("/", deps.SocialChannelHandler.CreateConfig)
			r.Get("/", deps.SocialChannelHandler.ListConfigs)
			r.Get("/{channelID}", deps.SocialChannelHandler.GetConfig)
			r.Put("/{channelID}", deps.SocialChannelHandler.UpdateConfig)
			r.Delete("/{id}", deps.SocialChannelHandler.DeleteConfig)
		})
	})

	// --- Public Routes (no JWT auth) ---

	r.Route("/api/v1/widget", func(r chi.Router) {
		r.Post("/sessions", deps.WidgetHandler.CreateSession)
		r.Post("/sessions/{id}/messages", deps.WidgetHandler.SendMessage)
	})

	// Webchat alias (frontend uses /api/v1/webchat/sessions)
	r.Route("/api/v1/webchat", func(r chi.Router) {
		r.Post("/sessions", deps.WidgetHandler.CreateSession)
		r.Get("/sessions/{id}/messages", deps.WidgetHandler.ListMessages)
		r.Post("/sessions/{id}/messages", deps.WidgetHandler.SendMessage)
	})

	r.Post("/api/v1/email/inbound", deps.EmailInboundHandler.Inbound)

	// --- Internal service-to-service routes (HMAC, not JWT). ---
	// FreeSWITCH and other backend services authenticate via X-CCC-Service-Token
	// instead of carrying user JWTs they cannot obtain.
	r.Route("/internal/v1", func(r chi.Router) {
		r.Use(middleware.ServiceAuth(deps.ServiceAuthSecret))
		r.Post("/calls/inbound", deps.CallControlHandler.InboundCall)
	})

	// --- Social Channel Webhooks (public, no JWT) ---
	r.Route("/api/v1/social", func(r chi.Router) {
		r.Get("/wechat/{channelID}", deps.SocialChannelHandler.WeChatVerify)
		r.Post("/wechat/{channelID}", deps.SocialChannelHandler.WeChatReceive)
		r.Get("/weibo/{channelID}", deps.SocialChannelHandler.WeiboVerify)
		r.Post("/weibo/{channelID}", deps.SocialChannelHandler.WeiboReceive)
	})

	return r
}
