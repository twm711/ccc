package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/divord97/ccc/internal/config"
	"github.com/divord97/ccc/internal/domain/identity"
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

	// Repositories
	tenantRepo := infraMySQL.NewTenantRepo(db)
	tenantSettingsRepo := infraMySQL.NewTenantSettingsRepo(db)
	userRepo := infraMySQL.NewUserRepo(db)
	agentRepo := infraMySQL.NewAgentRepo(db)
	skillGroupRepo := infraMySQL.NewSkillGroupRepo(db)
	skillGroupMemberRepo := infraMySQL.NewSkillGroupMemberRepo(db)
	auditLogRepo := infraMySQL.NewAuditLogRepo(db)

	// Domain services
	tenantSvc := identity.NewTenantService(tenantRepo, tenantSettingsRepo)
	userSvc := identity.NewUserService(userRepo, agentRepo)
	agentSvc := identity.NewAgentService(agentRepo, userRepo, tenantSettingsRepo)
	skillGroupSvc := identity.NewSkillGroupService(skillGroupRepo, skillGroupMemberRepo)

	// Rate limiter
	rateLimiter := infraRedis.NewRateLimiter(redisClient)

	// HTTP handlers
	tenantHandler := handler.NewTenantHandler(tenantSvc)
	userHandler := handler.NewUserHandler(userSvc)
	agentHandler := handler.NewAgentHandler(agentSvc)
	skillGroupHandler := handler.NewSkillGroupHandler(skillGroupSvc)

	// Router
	router := httpRouter.NewRouter(httpRouter.RouterDeps{
		TenantHandler:     tenantHandler,
		UserHandler:       userHandler,
		AgentHandler:      agentHandler,
		SkillGroupHandler: skillGroupHandler,
		RateLimiter:       rateLimiter,
		AuditLogRepo:      auditLogRepo,
		JWTSecret:         cfg.JWT.Secret,
		Logger:            logger,
	})

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info().Str("addr", addr).Msg("starting CCC server")
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Fatal().Err(err).Msg("server error")
	}
}
