package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/divord97/ccc/internal/config"
	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/domain/routing"
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

	// --- Domain Services ---
	tenantSvc := identity.NewTenantService(tenantRepo, tenantSettingsRepo)
	userSvc := identity.NewUserService(userRepo, agentRepo)
	agentSvc := identity.NewAgentService(agentRepo, userRepo, tenantSettingsRepo)
	skillGroupSvc := identity.NewSkillGroupService(skillGroupRepo, skillGroupMemberRepo)
	ivrFlowSvc := routing.NewIVRFlowService(ivrFlowRepo, ivrFlowVersionRepo)

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
