package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/divord97/ccc/internal/application/csat"
	"github.com/divord97/ccc/internal/application/outbound"
	"github.com/divord97/ccc/internal/config"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/internal/domain/report"
	"github.com/divord97/ccc/internal/domain/routing"
	"github.com/divord97/ccc/internal/domain/telephony"
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

	// --- Application Services ---
	outboundSvc := outbound.NewService(callSvc, routingSvc, cliSvc, dncSvc, nil)
	csatSvc := csat.NewService(csatConfigRepo, csatResultRepo, logger)
	_ = callbackRepo // used via callSvc

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
