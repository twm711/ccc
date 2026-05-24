package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/divord97/ccc/internal/application/advancedai"
	"github.com/divord97/ccc/internal/application/agenthub"
	"github.com/divord97/ccc/internal/application/aianalysis"
	"github.com/divord97/ccc/internal/application/b2b"
	"github.com/divord97/ccc/internal/application/callback"
	"github.com/divord97/ccc/internal/application/csat"
	"github.com/divord97/ccc/internal/application/dashboard"
	"github.com/divord97/ccc/internal/application/dialer"
	"github.com/divord97/ccc/internal/application/email"
	"github.com/divord97/ccc/internal/application/imassist"
	"github.com/divord97/ccc/internal/application/imhub"
	"github.com/divord97/ccc/internal/application/imrouter"
	"github.com/divord97/ccc/internal/application/ivr"
	"github.com/divord97/ccc/internal/application/lifecycle"
	"github.com/divord97/ccc/internal/application/outbound"
	"github.com/divord97/ccc/internal/application/screenpop"
	"github.com/divord97/ccc/internal/application/transcripthub"
	"github.com/divord97/ccc/internal/application/trunk"
	"github.com/divord97/ccc/internal/application/webhook"
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
	"github.com/divord97/ccc/internal/infrastructure/esl"
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

	// --- Config Repositories ---
	breakReasonRepo := infraMySQL.NewBreakReasonRepo(db)
	dispositionCodeRepo := infraMySQL.NewDispositionCodeRepo(db)
	callTagDefRepo := infraMySQL.NewCallTagDefRepo(db)
	audioFileRepo := infraMySQL.NewAudioFileRepo(db)
	businessHoursRepo := infraMySQL.NewBusinessHoursRepo(db)
	businessHoursScheduleRepo := infraMySQL.NewBusinessHoursScheduleRepo(db)
	queueSnapshotRepo := infraMySQL.NewQueueSnapshotRepo(db)

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
	webhookLogRepo := infraMySQL.NewWebhookDeliveryLogRepo(db)
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
	campaignReportRepo := infraMySQL.NewCampaignReportRepo(db)
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

	// ESL Client: connect to FreeSWITCH when host is configured.
	var eslClient *esl.Client
	if cfg.FreeSWITCH.Host != "" {
		eslClient = esl.NewClient(esl.Config{
			Host:     cfg.FreeSWITCH.Host,
			Port:     cfg.FreeSWITCH.Port,
			Password: cfg.FreeSWITCH.Password,
			PoolSize: cfg.FreeSWITCH.PoolSize,
			Logger:   logger,
		})
		callSvc.SetTelephonyProvider(esl.NewTelephonyAdapter(eslClient))
		logger.Info().Str("host", cfg.FreeSWITCH.Host).Msg("ESL: FreeSWITCH telephony provider configured")
	} else {
		logger.Warn().Msg("ESL: FREESWITCH_HOST not set, telephony commands disabled")
	}
	agentPresenceSvc := identity.NewAgentPresenceService(agentPresenceRepo, agentPresenceLogRepo)
	routingSvc := telephony.NewRoutingService(routingRuleRepo)
	cliSvc := telephony.NewCLIPolicyService(cliPolicyRepo, phoneNumberRepo)
	dncSvc := integration.NewDNCService(dncRepo)
	callTagSvc := integration.NewCallTagService(callTagRepo)

	// --- Phase 4 Domain Services ---
	dashboardSvc := report.NewDashboardService(dashboardRepo)
	reportSvc := report.NewReportService(agentReportRepo, groupAgentReportRepo, skillGroupReportRepo, b2bReportRepo, internalCallReportRepo, agentStatusLogRepo, campaignReportRepo)

	// --- Phase 6 Domain Services ---
	campaignSvc := campaign.NewCampaignService(campaignRepo, campaignCaseRepo, dncSvc)
	trunkHealthSvc := telephony.NewTrunkHealthService(sipTrunkRepo, trunkGroupRepo)

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

	socialConfigRepo := infraMySQL.NewSocialChannelConfigRepo(db)

	// --- Phase 8 Domain Services ---
	imSvc := im.NewIMService(imChannelRepo, imSessionRepo, imMessageRepo, 5)
	socialChannelSvc := im.NewSocialChannelService(socialConfigRepo, imChannelRepo, imSessionRepo, imMessageRepo)

	// --- Application Services ---
	outboundSvc := outbound.NewService(callSvc, routingSvc, cliSvc, dncSvc, eslClient)
	csatSvc := csat.NewService(csatConfigRepo, csatResultRepo, logger)
	dialerSvc := dialer.NewService(campaignSvc, eslClient, logger)
	// Wire dialer to use outbound service for DNC/routing/CLI compliance
	dialerSvc.SetDialFunc(func(ctx context.Context, tenantID int64, callee string, campaignID, caseID int64) error {
		_, err := outboundSvc.Dial(ctx, outbound.DialRequest{
			TenantID:       tenantID,
			Callee:         callee,
			MediaType:      call.MediaTypeAudio,
			CampaignCaseID: &caseID,
		})
		return err
	})
	b2bSvc := b2b.NewService(callRepo, callEventRepo, eslClient, logger)
	callbackSch := callback.NewScheduler(callbackRepo, callSvc, outboundSvc, logger)
	screenPopSvc := screenpop.NewService(screenPopConfigRepo, customerSvc)
	webhookSvc := webhook.NewService(webhookConfigRepo, webhookLogRepo, logger)
	lifecycleSvc := lifecycle.NewService(callSvc, agentPresenceSvc, csatSvc, webhookSvc, customerSvc, screenPopSvc, recordingRepo, eslClient)
	imRouterSvc := imrouter.NewService(imSvc, agentPresenceSvc, skillGroupSvc, logger)
	trunkMonitor := trunk.NewHealthMonitor(sipTrunkRepo, trunkHealthSvc, logger, eslClient)
	emailSvc := email.NewService(imSvc, logger)

	// IVR Engine: create engine with all node handlers for flow execution.
	ivrFlowLoader := func(ctx context.Context, flowID int64) (*routing.FlowGraph, error) {
		f, err := ivrFlowSvc.GetByID(ctx, flowID)
		if err != nil || f == nil {
			return nil, fmt.Errorf("flow %d not found", flowID)
		}
		g, err := routing.ValidateGraph(f.Graph)
		if err != nil {
			return nil, err
		}
		return g, nil
	}
	ivrEngine := ivr.DefaultEngine(eslClient, ivrFlowLoader)
	lifecycleSvc.SetIVREngine(ivrEngine)
	// LLM Provider: use DashScope when API key configured, otherwise fallback to stub.
	var llmProvider llm.Provider
	if cfg.Aliyun.DashScopeAPIKey != "" {
		llmProvider = llm.NewDashScopeProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		logger.Info().Msg("LLM: using DashScope provider")
	} else {
		llmProvider = llm.NewStubProvider()
		logger.Warn().Msg("LLM: DASHSCOPE_API_KEY not set, using stub provider")
	}
	imAssistSvc := imassist.NewService(llmProvider, logger)

	// ASR/TTS Providers: use Aliyun NLS when appkey configured.
	var asrProvider llm.ASRProvider
	var ttsProvider llm.TTSProvider
	if cfg.Aliyun.NLSAppKey != "" {
		nlsToken := cfg.Aliyun.NLSToken
		if nlsToken == "" && cfg.Aliyun.AccessKeyID != "" {
			t, _, err := llm.FetchNLSToken(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
			if err != nil {
				logger.Error().Err(err).Msg("ASR/TTS: failed to fetch NLS token")
			} else {
				nlsToken = t
				logger.Info().Msg("ASR/TTS: NLS token obtained via AccessKey")
			}
		}
		if nlsToken != "" {
			asrProvider = llm.NewAliyunASRProvider(nlsToken, cfg.Aliyun.NLSAppKey, cfg.Aliyun.STTRegion)
			ttsProvider = llm.NewAliyunTTSProvider(nlsToken, cfg.Aliyun.NLSAppKey, cfg.Aliyun.STTRegion, cfg.Aliyun.TTSVoice, cfg.Aliyun.TTSSampleRate)
			logger.Info().Str("appkey", cfg.Aliyun.NLSAppKey).Str("region", cfg.Aliyun.STTRegion).Msg("ASR/TTS: using Aliyun NLS providers")
		} else {
			logger.Warn().Msg("ASR/TTS: no NLS token available, ASR/TTS disabled")
		}
	} else {
		logger.Warn().Msg("ASR/TTS: NLS_APP_KEY not set, ASR/TTS disabled")
	}
	// Wire ASR provider into IVR engine for speech recognition nodes.
	ivrEngine.SetASRProvider(asrProvider)

	// --- Phase 9 Repositories ---
	digitalEmployeeRepo := infraMySQL.NewDigitalEmployeeRepo(db)
	digitalEmployeeSceneRepo := infraMySQL.NewDigitalEmployeeSceneRepo(db)
	qaRuleRepo := infraMySQL.NewQARuleRepo(db)
	qaSchemeRepo := infraMySQL.NewQASchemeRepo(db)
	qaResultRepo := infraMySQL.NewQAResultRepo(db)
	asrHotwordsRepo := infraMySQL.NewASRHotwordsRepo(db)
	performanceScorecardRepo := infraMySQL.NewPerformanceScorecardRepo(db)

	// --- Phase 9 Domain Services ---
	digitalEmployeeSvc := ai.NewDigitalEmployeeService(digitalEmployeeRepo, digitalEmployeeSceneRepo)
	qiSvc := ai.NewQualityInspectionService(qaRuleRepo, qaSchemeRepo, qaResultRepo)
	asrHotwordsSvc := ai.NewASRHotwordsService(asrHotwordsRepo)
	performanceSvc := ai.NewPerformanceScorecardService(performanceScorecardRepo)
	aiAnalysisSvc := aianalysis.NewService(llmProvider, logger)

	// Set LLM provider on QI service for LLM-type QA rules.
	qiSvc.SetLLMProvider(llmProvider)
	// Set LLM provider on digital employee service for fallback intent matching.
	digitalEmployeeSvc.SetLLMProvider(llmProvider)

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
	callControlHandler := handler.NewCallControlHandler(callSvc, lifecycleSvc)
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
	imSessionHandler := handler.NewIMSessionHandler(imSvc, customerSvc, webhookSvc)
	widgetHandler := handler.NewWidgetHandler(imSvc)
	emailInboundHandler := handler.NewEmailInboundHandler(emailSvc)
	imAssistHandler := handler.NewIMAssistHandler(imAssistSvc)

	// Phase 9
	digitalEmployeeHandler := handler.NewDigitalEmployeeHandler(digitalEmployeeSvc)
	qaHandler := handler.NewQAHandler(qiSvc)
	aiAnalysisHandler := handler.NewAIAnalysisHandler(aiAnalysisSvc)
	asrHotwordsHandler := handler.NewASRHotwordsHandler(asrHotwordsSvc)
	performanceHandler := handler.NewPerformanceHandler(performanceSvc)

	// Config Handlers
	breakReasonHandler := handler.NewBreakReasonHandler(breakReasonRepo)
	dispositionCodeHandler := handler.NewDispositionCodeHandler(dispositionCodeRepo)
	audioFileHandler := handler.NewAudioFileHandler(audioFileRepo)
	businessHoursHandler := handler.NewBusinessHoursHandler(businessHoursRepo, businessHoursScheduleRepo)
	callTagDefHandler := handler.NewCallTagDefHandler(callTagDefRepo)
	auditLogHandler := handler.NewAuditLogHandler(auditLogRepo)

	// Social Channels
	socialChannelHandler := handler.NewSocialChannelHandler(socialChannelSvc)

	// Tenant Settings Handler
	tenantSettingsHandler := handler.NewTenantSettingsHandler(tenantSettingsRepo)

	// Phone Component Extra Handlers
	supervisorHandler := handler.NewSupervisorHandler(callSvc)
	screenPopHandler := handler.NewScreenPopHandler(customerSvc)
	previewCaseHandler := handler.NewPreviewCaseHandler(campaignSvc, dialerSvc)

	// Auth Handler
	authHandler := handler.NewAuthHandler(userRepo, cfg.JWT.Secret)

	// WebSocket Hubs
	dashboardHub := dashboard.NewHub(dashboardSvc, logger)
	imHub := imhub.NewHub(imSvc, logger)
	agentHub := agenthub.NewHub(logger)
	transcriptHub := transcripthub.NewHub(logger)

	// Phase 10 Repositories
	annotationTaskRepo := infraMySQL.NewAnnotationTaskRepo(db)
	annotationResultRepo := infraMySQL.NewAnnotationResultRepo(db)
	llmModelConfigRepo := infraMySQL.NewLLMModelConfigRepo(db)
	webrtcQualityRepo := infraMySQL.NewWebRTCQualityRepo(db)

	// Phase 10 Domain Services
	annotationSvc := ai.NewAnnotationService(annotationTaskRepo, annotationResultRepo)
	llmGatewaySvc := ai.NewLLMGatewayService(llmModelConfigRepo)

	// Phase 10 Handlers
	annotationHandler := handler.NewAnnotationHandler(annotationSvc)
	llmGatewayHandler := handler.NewLLMGatewayHandler(llmGatewaySvc)
	webrtcQualityHandler := handler.NewWebRTCQualityHandler(webrtcQualityRepo)

	// STT/TTS Handlers
	sttHandler := handler.NewSTTHandler(asrProvider)
	ttsHandler := handler.NewTTSHandler(ttsProvider)

	// Advanced AI Repositories
	commAgentRepo := infraMySQL.NewCommAgentRepo(db)
	commAgentSessionRepo := infraMySQL.NewCommAgentSessionRepo(db)
	voiceProfileRepo := infraMySQL.NewVoiceProfileRepo(db)
	analysisTaskRepo := infraMySQL.NewConversationAnalysisTaskRepo(db)
	trainingCourseRepo := infraMySQL.NewTrainingCourseRepo(db)
	trainingExamRepo := infraMySQL.NewTrainingExamRepo(db)
	simulatedCallRepo := infraMySQL.NewSimulatedCallRepo(db)
	ringAnalysisConfigRepo := infraMySQL.NewRingAnalysisConfigRepo(db)
	ringAnalysisLogRepo := infraMySQL.NewRingAnalysisLogRepo(db)
	fullDuplexConfigRepo := infraMySQL.NewFullDuplexConfigRepo(db)

	// Advanced AI Providers: use DashScope when API key configured, otherwise stubs.
	var (
		commAgentProv    llm.CommAgentProvider
		voiceCloneProv   llm.VoiceCloningProvider
		convAnalyticsProv llm.ConversationAnalyticsProvider
		ringAnalysisProv  llm.RingAnalysisProvider
		fullDuplexProv    llm.FullDuplexProvider
		trainingProv      llm.TrainingProvider
	)
	if cfg.Aliyun.DashScopeAPIKey != "" {
		commAgentProv = llm.NewDashScopeCommAgentProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		voiceCloneProv = llm.NewDashScopeVoiceCloningProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		convAnalyticsProv = llm.NewDashScopeConversationAnalyticsProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		ringAnalysisProv = llm.NewDashScopeRingAnalysisProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		fullDuplexProv = llm.NewDashScopeFullDuplexProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		trainingProv = llm.NewDashScopeTrainingProvider(cfg.Aliyun.DashScopeAPIKey, cfg.Aliyun.DashScopeModel)
		logger.Info().Msg("Advanced AI: using DashScope providers")
	} else {
		commAgentProv = llm.NewStubCommAgentProvider()
		voiceCloneProv = llm.NewStubVoiceCloningProvider()
		convAnalyticsProv = llm.NewStubConversationAnalyticsProvider()
		ringAnalysisProv = llm.NewStubRingAnalysisProvider()
		fullDuplexProv = llm.NewStubFullDuplexProvider()
		trainingProv = llm.NewStubTrainingProvider()
		logger.Warn().Msg("Advanced AI: DASHSCOPE_API_KEY not set, using stub providers")
	}
	// Advanced AI Domain Services
	commAgentSvc := ai.NewCommAgentService(commAgentRepo, commAgentSessionRepo)
	commAgentSvc.SetProvider(commAgentProv)
	voiceProfileSvc := ai.NewVoiceProfileService(voiceProfileRepo)
	voiceProfileSvc.SetProvider(voiceCloneProv)
	conversationAnalysisSvc := ai.NewConversationAnalysisService(analysisTaskRepo)
	conversationAnalysisSvc.SetProvider(convAnalyticsProv)
	trainingSvc := ai.NewTrainingService(trainingCourseRepo, trainingExamRepo, simulatedCallRepo)
	trainingSvc.SetProvider(trainingProv)
	ringAnalysisSvc := ai.NewRingAnalysisService(ringAnalysisConfigRepo, ringAnalysisLogRepo)
	ringAnalysisSvc.SetProvider(ringAnalysisProv)
	fullDuplexSvc := ai.NewFullDuplexService(fullDuplexConfigRepo)
	fullDuplexSvc.SetProvider(fullDuplexProv)

	// Advanced AI orchestration service
	advancedAISvc := advancedai.NewService(advancedai.Deps{
		CommAgentSvc:  commAgentSvc,
		VoiceSvc:      voiceProfileSvc,
		AnalysisSvc:   conversationAnalysisSvc,
		TrainingSvc:   trainingSvc,
		RingSvc:       ringAnalysisSvc,
		FullDuplexSvc: fullDuplexSvc,
		CommAgentLLM:  commAgentProv,
		VoiceCloneLLM: voiceCloneProv,
		AnalyticsLLM:  convAnalyticsProv,
		RingLLM:       ringAnalysisProv,
		FullDuplexLLM: fullDuplexProv,
		TrainingLLM:   trainingProv,
		Logger:        logger,
	})
	// advancedAISvc is passed to router for advanced AI endpoints

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
		DigitalEmployeeHandler: digitalEmployeeHandler,
		QAHandler:              qaHandler,
		AIAnalysisHandler:      aiAnalysisHandler,
		ASRHotwordsHandler:     asrHotwordsHandler,
		PerformanceHandler:     performanceHandler,
		AnnotationHandler:      annotationHandler,
		LLMGatewayHandler:      llmGatewayHandler,
		WebRTCQualityHandler:   webrtcQualityHandler,
		STTHandler:             sttHandler,
		TTSHandler:             ttsHandler,
		CommAgentSvc:           commAgentSvc,
		VoiceSvc:               voiceProfileSvc,
		AnalysisSvc:            conversationAnalysisSvc,
		TrainingSvc:            trainingSvc,
		RingSvc:                ringAnalysisSvc,
		FullDuplexSvc:          fullDuplexSvc,
		AdvancedAISvc:          advancedAISvc,
		BreakReasonHandler:     breakReasonHandler,
		DispositionCodeHandler: dispositionCodeHandler,
		AudioFileHandler:       audioFileHandler,
		BusinessHoursHandler:   businessHoursHandler,
		CallTagDefHandler:      callTagDefHandler,
		AuditLogHandler:        auditLogHandler,
		SocialChannelHandler:    socialChannelHandler,
		TenantSettingsHandler:   tenantSettingsHandler,
		SupervisorHandler:       supervisorHandler,
		ScreenPopHandler:        screenPopHandler,
		PreviewCaseHandler:      previewCaseHandler,
		AuthHandler:          authHandler,
		DashboardHub:         dashboardHub,
		IMHub:                imHub,
		AgentHub:             agentHub,
		TranscriptHub:        transcriptHub,
		RateLimiter:          rateLimiter,
		AuditLogRepo:         auditLogRepo,
		JWTSecret:            cfg.JWT.Secret,
		Logger:               logger,
	})

	// Wire lifecycle → agentHub for real-time agent notifications
	lifecycleSvc.SetAgentNotifier(agentHub)
	lifecycleSvc.SetCampaignService(campaignSvc)
	lifecycleSvc.SetQueueSnapshotRepo(queueSnapshotRepo)

	// Start WebSocket hub goroutines
	hubCtx, hubCancel := context.WithCancel(context.Background())
	go dashboardHub.StartBroadcast(hubCtx)
	go imHub.StartBroadcast(hubCtx)
	go agentHub.StartBroadcast(hubCtx)
	go transcriptHub.StartBroadcast(hubCtx)

	// Start callback scheduler (processes pending callbacks every 30s)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-hubCtx.Done():
				return
			case <-ticker.C:
				if n, err := callbackSch.ProcessAllPending(hubCtx); err == nil && n > 0 {
					logger.Info().Int("processed", n).Msg("callback scheduler: processed pending callbacks")
				}
			}
		}
	}()

	// Start trunk health monitor
	trunkMonitor.Start(hubCtx)

	// Wire IM router into session handler for auto-assignment
	imSessionHandler.SetRouter(imRouterSvc)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		logger.Info().Str("addr", addr).Msg("starting CCC server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down server...")
	hubCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("server shutdown error")
	}
	logger.Info().Msg("server stopped")
}
