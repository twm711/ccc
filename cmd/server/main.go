package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/divord97/ccc/internal/application/b2b"
	"github.com/divord97/ccc/internal/application/csat"
	"github.com/divord97/ccc/internal/application/dialer"
	"github.com/divord97/ccc/internal/application/email"
	"github.com/divord97/ccc/internal/application/imassist"
	"github.com/divord97/ccc/internal/application/outbound"
	"github.com/divord97/ccc/internal/config"
	"github.com/divord97/ccc/internal/domain/ai"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/internal/domain/report"
	"github.com/divord97/ccc/internal/domain/routing"
	"github.com/divord97/ccc/internal/domain/telephony"
	"github.com/divord97/ccc/internal/domain/ticket"
	"github.com/divord97/ccc/internal/infrastructure/llm"
	infraMySQL "github.com/divord97/ccc/internal/infrastructure/mysql"
	infraRedis "github.com/divord97/ccc/internal/infrastructure/redis"
	httpRouter "github.com/divord97/ccc/internal/interfaces/http"
	"github.com/divord97/ccc/internal/interfaces/http/handler"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg := config.Load()

	if err := snowflake.Init(cfg.Snowflake.NodeID); err != nil {
		logger.Fatal().Err(err).Msg("failed to init snowflake")
	}

	db, err := infraMySQL.NewDB(cfg.Database.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect database")
	}
	defer db.Close()

	redisClient := infraRedis.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	defer redisClient.Close()

	// --- Phase 0 Repositories ---
	tenantRepo := infraMySQL.NewTenantRepo(db)
	tenantSettingsRepo := infraMySQL.NewTenantSettingsRepo(db)
	userRepo := infraMySQL.NewUserRepo(db)
	agentRepo := infraMySQL.NewAgentRepo(db)
	skillGroupRepo := infraMySQL.NewSkillGroupRepo(db)
	skillGroupMemberRepo := infraMySQL.NewSkillGroupMemberRepo(db)
	auditLogRepo := infraMySQL.NewAuditLogRepo(db)

	// --- Phase 1 Repositories ---
	ivrFlowRepo := infraMySQL.NewIVRFlowRepo(db)
	ivrFlowVersionRepo := infraMySQL.NewIVRFlowVersionRepo(db)
	carrierRepo := infraMySQL.NewCarrierRepo(db)
	sipTrunkRepo := infraMySQL.NewSIPTrunkRepo(db)
	phoneNumberRepo := infraMySQL.NewPhoneNumberRepo(db)
	recordingRepo := infraMySQL.NewRecordingRepo(db)
	voicemailRepo := infraMySQL.NewVoicemailRepo(db)
	callNumberTagRepo := infraMySQL.NewCallNumberTagRepo(db)
	autoTagRuleRepo := infraMySQL.NewAutoTagRuleRepo(db)

	// --- Phase 2 Repositories ---
	callRepo := infraMySQL.NewCallRepo(db)
	callEventRepo := infraMySQL.NewCallEventRepo(db)
	ivrTrackingRepo := infraMySQL.NewIVRTrackingRepo(db)
	routingRuleRepo := infraMySQL.NewRoutingRuleRepo(db)
	cliPolicyRepo := infraMySQL.NewCLIPolicyRepo(db)
	dncRepo := infraMySQL.NewDNCRepo(db)
	callTagRepo := infraMySQL.NewCallTagAssignmentRepo(db)

	// --- Phase 3 Repositories ---
	callbackRepo := infraMySQL.NewCallbackRequestRepo(db)
	agentPresenceRepo := infraMySQL.NewAgentPresenceRepo(db)
	agentPresenceLogRepo := infraMySQL.NewAgentPresenceLogRepo(db)
	webhookConfigRepo := infraMySQL.NewWebhookConfigRepo(db)
	screenPopConfigRepo := infraMySQL.NewScreenPopConfigRepo(db)
	quickReplyRepo := infraMySQL.NewQuickReplyRepo(db)
	smsConfigRepo := infraMySQL.NewSmsConfigRepo(db)

	// --- Phase 6 Repositories ---
	campaignRepo := infraMySQL.NewCampaignRepo(db)
	campaignCaseRepo := infraMySQL.NewCampaignCaseRepo(db)
	trunkGroupRepo := infraMySQL.NewSIPTrunkGroupRepo(db)

	// --- Phase 7 Repositories ---
	customerRepo := infraMySQL.NewCustomerRepo(db)
	customerPhoneRepo := infraMySQL.NewCustomerPhoneRepo(db)
	interactionRepo := infraMySQL.NewInteractionRepo(db)
	customFieldRepo := infraMySQL.NewCustomFieldRepo(db)
	ticketCategoryRepo := infraMySQL.NewTicketCategoryRepo(db)
	ticketTemplateRepo := infraMySQL.NewTicketTemplateRepo(db)
	ticketRepo := infraMySQL.NewTicketRepo(db)
	ticketCommentRepo := infraMySQL.NewTicketCommentRepo(db)
	knowledgeCategoryRepo := infraMySQL.NewKnowledgeCategoryRepo(db)
	knowledgeArticleRepo := infraMySQL.NewKnowledgeArticleRepo(db)
	agentScriptRepo := infraMySQL.NewAgentScriptRepo(db)
	sessionInfoTemplateRepo := infraMySQL.NewSessionInfoTemplateRepo(db)

	// --- Phase 4 Repositories ---
	agentReportRepo := infraMySQL.NewAgentReportRepo(db)
	groupAgentReportRepo := infraMySQL.NewGroupAgentReportRepo(db)
	skillGroupReportRepo := infraMySQL.NewSkillGroupReportRepo(db)
	b2bReportRepo := infraMySQL.NewBack2BackReportRepo(db)
	internalCallReportRepo := infraMySQL.NewInternalCallReportRepo(db)
	agentStatusLogRepo := infraMySQL.NewAgentStatusLogRepo(db)
	csatConfigRepo := infraMySQL.NewCSATConfigRepo(db)
	csatResultRepo := infraMySQL.NewCSATResultRepo(db)
	dashboardRepo := infraRedis.NewDashboardRepo(redisClient)

	// --- Domain Services ---
	tenantSvc := identity.NewTenantService(tenantRepo, tenantSettingsRepo)
	userSvc := identity.NewUserService(userRepo, agentRepo)
	agentSvc := identity.NewAgentService(agentRepo, userRepo, tenantSettingsRepo)
	skillGroupSvc := identity.NewSkillGroupService(skillGroupRepo, skillGroupMemberRepo)
	ivrFlowSvc := routing.NewIVRFlowService(ivrFlowRepo, ivrFlowVersionRepo)
	callSvc := call.NewCallService(callRepo, callEventRepo, ivrTrackingRepo, callbackRepo)
	agentPresenceSvc := identity.NewAgentPresenceService(agentPresenceRepo, agentPresenceLogRepo)
	routingSvc := telephony.NewRoutingService(routingRuleRepo)
	cliSvc := telephony.NewCLIPolicyService(cliPolicyRepo, phoneNumberRepo)
	dncSvc := integration.NewDNCService(dncRepo)
	callTagSvc := integration.NewCallTagService(callTagRepo)

	// --- Phase 4 Domain Services ---
	dashboardSvc := report.NewDashboardService(dashboardRepo)
	reportSvc := report.NewReportService(agentReportRepo, groupAgentReportRepo, skillGroupReportRepo, b2bReportRepo, internalCallReportRepo, agentStatusLogRepo)

	// --- Phase 6 Domain Services ---
	campaignSvc := campaign.NewCampaignService(campaignRepo, campaignCaseRepo)
	trunkHealthSvc := telephony.NewTrunkHealthService(sipTrunkRepo, trunkGroupRepo)
	_ = trunkHealthSvc

	// --- Phase 7 Domain Services ---
	customerSvc := crm.NewCustomerService(customerRepo, customerPhoneRepo, interactionRepo, customFieldRepo)
	ticketTemplateSvc := ticket.NewTicketTemplateService(ticketTemplateRepo, ticketCategoryRepo)
	ticketSvc := ticket.NewTicketService(ticketRepo, ticketTemplateRepo, ticketCommentRepo)
	knowledgeSvc := ai.NewKnowledgeService(knowledgeCategoryRepo, knowledgeArticleRepo)
	agentScriptSvc := ai.NewAgentScriptService(agentScriptRepo)
	sessionInfoSvc := ai.NewSessionInfoTemplateService(sessionInfoTemplateRepo)

	// --- Phase 8 Repositories ---
	imChannelRepo := infraMySQL.NewIMChannelRepo(db)
	imSessionRepo := infraMySQL.NewIMSessionRepo(db)
	imMessageRepo := infraMySQL.NewIMMessageRepo(db)

	// --- Phase 8 Domain Services ---
	imSvc := im.NewIMService(imChannelRepo, imSessionRepo, imMessageRepo, 5)

	// --- Application Services ---
	outboundSvc := outbound.NewService(callSvc, routingSvc, cliSvc, dncSvc, nil)
	csatSvc := csat.NewService(csatConfigRepo, csatResultRepo, logger)
	dialerSvc := dialer.NewService(campaignSvc, nil, logger)
	b2bSvc := b2b.NewService(callRepo, callEventRepo, nil, logger)
	_ = callbackRepo // used via callSvc
	emailSvc := email.NewService(imSvc, logger)
	llmProvider := llm.NewStubProvider()
	imAssistSvc := imassist.NewService(llmProvider, logger)

	// --- Infrastructure ---
	rateLimiter := infraRedis.NewRateLimiter(redisClient)

	// --- HTTP Handlers ---
	tenantHandler := handler.NewTenantHandler(tenantSvc)
	userHandler := handler.NewUserHandler(userSvc)
	agentHandler := handler.NewAgentHandler(agentSvc)
	skillGroupHandler := handler.NewSkillGroupHandler(skillGroupSvc)
	ivrFlowHandler := handler.NewIVRFlowHandler(ivrFlowSvc)
	carrierHandler := handler.NewCarrierHandler(carrierRepo)
	sipTrunkHandler := handler.NewSIPTrunkHandler(sipTrunkRepo)
	phoneNumberHandler := handler.NewPhoneNumberHandler(phoneNumberRepo)
	recordingHandler := handler.NewRecordingHandler(recordingRepo)
	voicemailHandler := handler.NewVoicemailHandler(voicemailRepo)
	callNumberTagHandler := handler.NewCallNumberTagHandler(callNumberTagRepo)
	autoTagRuleHandler := handler.NewAutoTagRuleHandler(autoTagRuleRepo)
	routingRuleHandler := handler.NewRoutingRuleHandler(routingRuleRepo)
	cliPolicyHandler := handler.NewCLIPolicyHandler(cliPolicyRepo)
	dncHandler := handler.NewDNCHandler(dncSvc, dncRepo)
	callHandler := handler.NewCallHandler(callSvc, outboundSvc, callTagSvc)
	callControlHandler := handler.NewCallControlHandler(callSvc)
	agentPresenceHandler := handler.NewAgentPresenceHandler(agentPresenceSvc)
	webhookConfigHandler := handler.NewWebhookConfigHandler(webhookConfigRepo)
	screenPopConfigHandler := handler.NewScreenPopConfigHandler(screenPopConfigRepo)
	quickReplyHandler := handler.NewQuickReplyHandler(quickReplyRepo)
	smsConfigHandler := handler.NewSmsConfigHandler(smsConfigRepo)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	reportHandler := handler.NewReportHandler(reportSvc)
	csatHandler := handler.NewCSATHandler(csatSvc, csatConfigRepo, csatResultRepo)
	profileHandler := handler.NewProfileHandler(userSvc)
	campaignHandler := handler.NewCampaignHandler(campaignSvc, dialerSvc)
	b2bHandler := handler.NewB2BHandler(b2bSvc)
	trunkGroupHandler := handler.NewTrunkGroupHandler(trunkGroupRepo)
	customerHandler := handler.NewCustomerHandler(customerSvc)
	ticketHandler := handler.NewTicketHandler(ticketSvc, ticketTemplateSvc)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeSvc)
	agentScriptHandler := handler.NewAgentScriptHandler(agentScriptSvc)
	sessionInfoHandler := handler.NewSessionInfoHandler(sessionInfoSvc)
	imChannelHandler := handler.NewIMChannelHandler(imSvc)
	imSessionHandler := handler.NewIMSessionHandler(imSvc)
	widgetHandler := handler.NewWidgetHandler(imSvc)
	emailInboundHandler := handler.NewEmailInboundHandler(emailSvc)
	imAssistHandler := handler.NewIMAssistHandler(imAssistSvc)

	// --- Router ---
	router := httpRouter.NewRouter(httpRouter.RouterDeps{
		TenantHandler:        tenantHandler,
		UserHandler:          userHandler,
		AgentHandler:         agentHandler,
		SkillGroupHandler:    skillGroupHandler,
		IVRFlowHandler:       ivrFlowHandler,
		CarrierHandler:       carrierHandler,
		SIPTrunkHandler:      sipTrunkHandler,
		PhoneNumberHandler:   phoneNumberHandler,
		RecordingHandler:     recordingHandler,
		VoicemailHandler:     voicemailHandler,
		CallNumberTagHandler: callNumberTagHandler,
		AutoTagRuleHandler:   autoTagRuleHandler,
		RoutingRuleHandler:   routingRuleHandler,
		CLIPolicyHandler:     cliPolicyHandler,
		DNCHandler:           dncHandler,
		CallHandler:          callHandler,
		CallControlHandler:    callControlHandler,
		AgentPresenceHandler:  agentPresenceHandler,
		WebhookConfigHandler:  webhookConfigHandler,
		ScreenPopConfigHandler: screenPopConfigHandler,
		QuickReplyHandler:     quickReplyHandler,
		SmsConfigHandler:      smsConfigHandler,
		DashboardHandler:     dashboardHandler,
		ReportHandler:        reportHandler,
		CSATHandler:          csatHandler,
		ProfileHandler:       profileHandler,
		CampaignHandler:      campaignHandler,
		B2BHandler:           b2bHandler,
		TrunkGroupHandler:    trunkGroupHandler,
		CustomerHandler:      customerHandler,
		TicketHandler:        ticketHandler,
		KnowledgeHandler:     knowledgeHandler,
		AgentScriptHandler:   agentScriptHandler,
		SessionInfoHandler:   sessionInfoHandler,
		IMChannelHandler:     imChannelHandler,
		IMSessionHandler:     imSessionHandler,
		WidgetHandler:        widgetHandler,
		EmailInboundHandler:  emailInboundHandler,
		IMAssistHandler:      imAssistHandler,
		RateLimiter:          rateLimiter,
		AuditLogRepo:         auditLogRepo,
		JWTSecret:            cfg.JWT.Secret,
		Logger:               logger,
	})

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info().Str("addr", addr).Msg("starting CCC server")
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Fatal().Err(err).Msg("server error")
	}
}
