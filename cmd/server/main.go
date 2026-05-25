package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/divord97/ccc/internal/application/acd"
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
	"github.com/divord97/ccc/internal/application/postcall"
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
	infraNATS "github.com/divord97/ccc/internal/infrastructure/nats"
	infraRedis "github.com/divord97/ccc/internal/infrastructure/redis"
	"github.com/divord97/ccc/internal/infrastructure/storage"
	"github.com/divord97/ccc/internal/infrastructure/tracing"
	httpRouter "github.com/divord97/ccc/internal/interfaces/http"
	"github.com/divord97/ccc/internal/interfaces/http/handler"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg := config.Load()

	// Apply log level from config/env
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
		logger = logger.Level(lvl)
	}

	if cfg.JWT.Secret == "change-me-in-production" {
		logger.Fatal().Msg("JWT_SECRET must be changed from its default value before running in production")
	}

	if err := snowflake.Init(cfg.Snowflake.NodeID); err != nil {
		logger.Fatal().Err(err).Msg("failed to init snowflake")
	}

	// OpenTelemetry distributed tracing (optional).
	if cfg.OTEL.Endpoint != "" {
		shutdownTracer, err := tracing.Init(context.Background(), tracing.Config{
			Endpoint:    cfg.OTEL.Endpoint,
			ServiceName: "ccc-server",
			Insecure:    cfg.OTEL.Insecure,
		})
		if err != nil {
			logger.Error().Err(err).Msg("otel: init failed, tracing disabled")
		} else {
			defer shutdownTracer(context.Background())
			logger.Info().Str("endpoint", cfg.OTEL.Endpoint).Msg("otel: tracing enabled")
		}
	} else {
		logger.Warn().Msg("otel: OTEL_EXPORTER_OTLP_ENDPOINT not set, tracing disabled")
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
	agentShiftLogRepo := infraMySQL.NewAgentShiftLogRepo(db)
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
	agentPresenceSvc.SetShiftLogRepo(agentShiftLogRepo)
	// Resolve effective ACW seconds for an agent: prefer agent.acw_seconds,
	// then tenant_settings.default_acw_seconds. This wiring fixes a P1 defect
	// where agent_presence rows never carried an ACWSeconds value, so the
	// scheduled SetACW timeout never fired.
	agentPresenceSvc.SetACWResolver(func(ctx context.Context, tenantID, agentID int64) int {
		if a, err := agentRepo.GetByUserID(ctx, agentID); err == nil && a != nil && a.ACWSeconds > 0 {
			return a.ACWSeconds
		}
		if ts, err := tenantSettingsRepo.GetByTenantID(ctx, tenantID); err == nil && ts != nil && ts.DefaultACWSeconds > 0 {
			return ts.DefaultACWSeconds
		}
		return 0
	})
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
	lifecycleSvc := lifecycle.NewService(callSvc, agentPresenceSvc, csatSvc, webhookSvc, customerSvc, screenPopSvc, recordingRepo, eslClient, logger)
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
	acdSvc := acd.NewService(acd.Config{
		Redis:      redisClient,
		Lifecycle:  lifecycleSvc,
		Presence:   agentPresenceRepo,
		Members:    skillGroupMemberRepo,
		SkillGroup: skillGroupRepo,
		Calls:      callRepo,
		Logger:     logger,
	})
	// Familiar-customer affinity: lifecycle.EndCall records who served each
	// inbound caller; ACD reads this cache on dispatch for routing_policy=familiar.
	lifecycleSvc.SetFamiliarRecorder(acdSvc, func(tenantID int64) int {
		if ts, err := tenantSettingsRepo.GetByTenantID(context.Background(), tenantID); err == nil && ts != nil && ts.FamiliarAgentDays > 0 {
			return ts.FamiliarAgentDays
		}
		return 30
	})
	ivrEngine := ivr.DefaultEngine(eslClient, ivrFlowLoader, acdSvc)
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
	var aliyunASR *llm.AliyunASRProvider
	var aliyunTTS *llm.AliyunTTSProvider
	var nlsExpireSec int64
	if cfg.Aliyun.NLSAppKey != "" {
		nlsToken := cfg.Aliyun.NLSToken
		if nlsToken == "" && cfg.Aliyun.AccessKeyID != "" {
			t, exp, err := llm.FetchNLSToken(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
			if err != nil {
				logger.Error().Err(err).Msg("ASR/TTS: failed to fetch NLS token")
			} else {
				nlsToken = t
				nlsExpireSec = exp
				logger.Info().Int64("expire_at", exp).Msg("ASR/TTS: NLS token obtained via AccessKey")
			}
		}
		if nlsToken != "" {
			aliyunASR = llm.NewAliyunASRProvider(nlsToken, cfg.Aliyun.NLSAppKey, cfg.Aliyun.STTRegion)
			aliyunTTS = llm.NewAliyunTTSProvider(nlsToken, cfg.Aliyun.NLSAppKey, cfg.Aliyun.STTRegion, cfg.Aliyun.TTSVoice, cfg.Aliyun.TTSSampleRate)
			asrProvider = aliyunASR
			ttsProvider = aliyunTTS
			logger.Info().Str("appkey", cfg.Aliyun.NLSAppKey).Str("region", cfg.Aliyun.STTRegion).Msg("ASR/TTS: using Aliyun NLS providers")
		} else {
			logger.Warn().Msg("ASR/TTS: no NLS token available, ASR/TTS disabled")
		}
	} else {
		logger.Warn().Msg("ASR/TTS: NLS_APP_KEY not set, ASR/TTS disabled")
	}
	// Wire ASR provider into IVR engine for speech recognition nodes.
	ivrEngine.SetASRProvider(asrProvider)

	// NLS token refresher: Aliyun tokens expire (typically 24h). Refresh on a
	// schedule derived from expireTime, falling back to 12h on parse failure.
	if aliyunASR != nil && cfg.Aliyun.AccessKeyID != "" {
		go runNLSRefresher(cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret, nlsExpireSec, aliyunASR, aliyunTTS, logger)
	}

	// NATS event publisher (best-effort): publishes ccc.call.* and ccc.agent.*
	// to JetStream so downstream consumers (analytics, BI, third-party CRM
	// hooks) can subscribe without polling the DB.
	if cfg.NATS.URL != "" {
		natsClient, err := infraNATS.NewClient(infraNATS.Config{URL: cfg.NATS.URL, Logger: logger})
		if err != nil {
			logger.Error().Err(err).Msg("nats: connect failed, event publishing disabled")
		} else {
			if err := natsClient.EnsureStream(context.Background(), cfg.NATS.Stream, []string{"ccc.>"}); err != nil {
				logger.Warn().Err(err).Str("stream", cfg.NATS.Stream).Msg("nats: ensure stream")
			}
			lifecycleSvc.SetEventPublisher(natsClient)
			defer natsClient.Close()
			logger.Info().Str("url", cfg.NATS.URL).Str("stream", cfg.NATS.Stream).Msg("nats: event publisher wired")

			// Start NATS consumer for post-call processing (CDR, analytics).
			postCallWorker := postcall.NewWorker(callRepo, logger)
			natsConsumer := infraNATS.NewConsumer(natsClient, postCallWorker.HandleMessage)
			go func() {
				if err := natsConsumer.Subscribe(context.Background(), cfg.NATS.Stream, "postcall-worker", "ccc.call.>"); err != nil {
					logger.Error().Err(err).Msg("nats: postcall consumer stopped")
				}
			}()
			logger.Info().Msg("nats: postcall consumer started")
		}
	} else {
		logger.Warn().Msg("nats: NATS_URL not set, event publishing disabled")
	}

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
	recordingHandler.SetAccessLogger(infraMySQL.NewRecordingAccessLogger(auditLogRepo))
	if cfg.Storage.Endpoint != "" {
		store, err := storage.NewMinIOClient(storage.Config{
			Endpoint:  cfg.Storage.Endpoint,
			AccessKey: cfg.Storage.AccessKey,
			SecretKey: cfg.Storage.SecretKey,
			Bucket:    cfg.Storage.Bucket,
			UseSSL:    cfg.Storage.UseSSL,
			Logger:    logger,
		})
		if err != nil {
			logger.Error().Err(err).Msg("storage: init failed; recording stream/download disabled")
		} else {
			if err := store.EnsureBucket(context.Background()); err != nil {
				logger.Warn().Err(err).Str("bucket", cfg.Storage.Bucket).Msg("storage: ensure bucket")
			}
			recordingHandler.SetStore(store)
			logger.Info().Str("endpoint", cfg.Storage.Endpoint).Str("bucket", cfg.Storage.Bucket).Msg("storage: configured")
		}
	} else {
		logger.Warn().Msg("storage: STORAGE_ENDPOINT not set, recording stream/download return 501")
	}
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
		commAgentProv     llm.CommAgentProvider
		voiceCloneProv    llm.VoiceCloningProvider
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
		TenantHandler:          tenantHandler,
		UserHandler:            userHandler,
		AgentHandler:           agentHandler,
		SkillGroupHandler:      skillGroupHandler,
		IVRFlowHandler:         ivrFlowHandler,
		CarrierHandler:         carrierHandler,
		SIPTrunkHandler:        sipTrunkHandler,
		PhoneNumberHandler:     phoneNumberHandler,
		RecordingHandler:       recordingHandler,
		VoicemailHandler:       voicemailHandler,
		CallNumberTagHandler:   callNumberTagHandler,
		AutoTagRuleHandler:     autoTagRuleHandler,
		RoutingRuleHandler:     routingRuleHandler,
		CLIPolicyHandler:       cliPolicyHandler,
		DNCHandler:             dncHandler,
		CallHandler:            callHandler,
		CallControlHandler:     callControlHandler,
		AgentPresenceHandler:   agentPresenceHandler,
		WebhookConfigHandler:   webhookConfigHandler,
		ScreenPopConfigHandler: screenPopConfigHandler,
		QuickReplyHandler:      quickReplyHandler,
		SmsConfigHandler:       smsConfigHandler,
		DashboardHandler:       dashboardHandler,
		ReportHandler:          reportHandler,
		CSATHandler:            csatHandler,
		ProfileHandler:         profileHandler,
		CampaignHandler:        campaignHandler,
		B2BHandler:             b2bHandler,
		TrunkGroupHandler:      trunkGroupHandler,
		CustomerHandler:        customerHandler,
		TicketHandler:          ticketHandler,
		KnowledgeHandler:       knowledgeHandler,
		AgentScriptHandler:     agentScriptHandler,
		SessionInfoHandler:     sessionInfoHandler,
		IMChannelHandler:       imChannelHandler,
		IMSessionHandler:       imSessionHandler,
		WidgetHandler:          widgetHandler,
		EmailInboundHandler:    emailInboundHandler,
		IMAssistHandler:        imAssistHandler,
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
		SocialChannelHandler:   socialChannelHandler,
		TenantSettingsHandler:  tenantSettingsHandler,
		SupervisorHandler:      supervisorHandler,
		ScreenPopHandler:       screenPopHandler,
		PreviewCaseHandler:     previewCaseHandler,
		AuthHandler:            authHandler,
		DashboardHub:           dashboardHub,
		IMHub:                  imHub,
		AgentHub:               agentHub,
		TranscriptHub:          transcriptHub,
		RateLimiter:            rateLimiter,
		TenantSettingsRepo:     tenantSettingsRepo,
		AuditLogRepo:           auditLogRepo,
		JWTSecret:              cfg.JWT.Secret,
		ServiceAuthSecret:      cfg.ServiceAuth.Secret,
		Logger:                 logger,
	})

	// Wire JWT secret for WebSocket JWT auth.
	agentHub.SetJWTSecret(cfg.JWT.Secret)

	// Wire concurrency guard for per-tenant call limits.
	concurrencyGuard := infraRedis.NewConcurrencyGuard(redisClient)
	lifecycleSvc.SetConcurrencyGuard(concurrencyGuard, &tenantSettingsAdapter{repo: tenantSettingsRepo})

	// Wire recording announce lookup.
	lifecycleSvc.SetRecordingAnnounceLookup(func(ctx context.Context, tenantID int64) bool {
		if ts, err := tenantSettingsRepo.GetByTenantID(ctx, tenantID); err == nil && ts != nil {
			return ts.RecordingAnnounce
		}
		return false
	})

	// Wire lifecycle → agentHub for real-time agent notifications
	lifecycleSvc.SetAgentNotifier(agentHub)
	lifecycleSvc.SetCampaignService(campaignSvc)
	lifecycleSvc.SetQueueSnapshotRepo(queueSnapshotRepo)

	// Start dashboard refresher (aggregates DB data into Redis every 10s)
	dashRefresher := dashboard.NewRefresher(callRepo, agentPresenceRepo, tenantRepo, dashboardRepo, logger)

	// Start WebSocket hub goroutines
	hubCtx, hubCancel := context.WithCancel(context.Background())
	go dashRefresher.Start(hubCtx)
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

	// Ghost agent auto-reset: scan agents stuck in talking/dialing for over 4 hours.
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-hubCtx.Done():
				return
			case <-ticker.C:
				tenants, _, err := tenantRepo.List(hubCtx, 0, 1000)
				if err != nil {
					continue
				}
				for _, t := range tenants {
					if n, err := agentPresenceSvc.ResetGhostAgents(hubCtx, t.ID, 4*time.Hour); err == nil && n > 0 {
						logger.Warn().Int("reset", n).Int64("tenant_id", t.ID).Msg("ghost agent auto-reset")
					}
				}
			}
		}
	}()

	// Start trunk health monitor
	trunkMonitor.Start(hubCtx)

	// Wire IM router into session handler for auto-assignment
	imSessionHandler.SetRouter(imRouterSvc)
	// Wire IM hub for real-time broadcast of REST-posted messages.
	imSessionHandler.SetBroadcaster(imHub)
	widgetHandler.SetBroadcaster(imHub)

	// ACD dispatcher: pull queued calls and assign them to idle agents.
	go acdSvc.Run(hubCtx)
	logger.Info().Msg("ACD dispatcher started")

	// ESL event listener: drive call state machine from FreeSWITCH channel events.
	if cfg.FreeSWITCH.Host != "" {
		listener := esl.NewEventListener(esl.Config{
			Host:     cfg.FreeSWITCH.Host,
			Port:     cfg.FreeSWITCH.Port,
			Password: cfg.FreeSWITCH.Password,
			Logger:   logger,
		}, lifecycleSvc)
		go listener.Run(hubCtx)
		go eslClient.StartHealthCheck(hubCtx)
		logger.Info().Msg("ESL event listener started")
	}

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

	// Phase 1: stop accepting new traffic
	handler.SetReady(false)
	logger.Info().Msg("readiness probe disabled, draining connections...")

	// Phase 2: stop background tasks
	hubCancel()

	// Phase 3: drain period for load balancers to detect unavailability
	time.Sleep(5 * time.Second)

	// Phase 4: wait for in-flight requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("server shutdown error")
	}
	logger.Info().Msg("server stopped")
}

// tenantSettingsAdapter wraps the TenantSettingsRepository to satisfy the
// lifecycle.TenantSettingsLookup interface.
type tenantSettingsAdapter struct {
	repo identity.TenantSettingsRepository
}

func (a *tenantSettingsAdapter) GetByTenantID(ctx context.Context, tenantID int64) (maxConcurrentCalls int) {
	ts, err := a.repo.GetByTenantID(ctx, tenantID)
	if err != nil || ts == nil {
		return 0
	}
	return ts.MaxConcurrentCalls
}

// runNLSRefresher polls Aliyun NLS for a fresh token before the current one
// expires, then atomically swaps it onto the ASR/TTS providers. Aliyun's REST
// returns expireTime in seconds; we refresh ~5 min before. On failure we
// retry every 5 minutes with the previous (still valid) token until success.
func runNLSRefresher(accessKeyID, accessKeySecret string, expireSec int64, asr *llm.AliyunASRProvider, tts *llm.AliyunTTSProvider, logger zerolog.Logger) {
	nextDelay := func(expSec int64) time.Duration {
		if expSec <= 0 {
			return 12 * time.Hour
		}
		// expireTime from Aliyun is "seconds since epoch" when token expires.
		remain := time.Until(time.Unix(expSec, 0)) - 5*time.Minute
		if remain < time.Minute {
			return time.Minute
		}
		if remain > 23*time.Hour {
			return 23 * time.Hour
		}
		return remain
	}
	exp := expireSec
	for {
		d := nextDelay(exp)
		logger.Info().Dur("delay", d).Msg("nls: scheduling token refresh")
		time.Sleep(d)
		token, newExp, err := llm.FetchNLSToken(accessKeyID, accessKeySecret)
		if err != nil {
			logger.Error().Err(err).Msg("nls: token refresh failed; retrying in 5m")
			exp = time.Now().Add(5 * time.Minute).Unix()
			continue
		}
		if asr != nil {
			asr.SetToken(token)
		}
		if tts != nil {
			tts.SetToken(token)
		}
		exp = newExp
		logger.Info().Int64("expire_at", newExp).Msg("nls: token refreshed")
	}
}
