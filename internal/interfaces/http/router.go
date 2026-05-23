package http

import (
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
	TenantHandler     *handler.TenantHandler
	UserHandler       *handler.UserHandler
	AgentHandler      *handler.AgentHandler
	SkillGroupHandler *handler.SkillGroupHandler
	RateLimiter       *redis.RateLimiter
	AuditLogRepo      platform.AuditLogRepository
	JWTSecret         string
	Logger            zerolog.Logger
}

func NewRouter(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.RequestLogger(deps.Logger))

	r.Get("/health", handler.Health)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(deps.JWTSecret))
		r.Use(middleware.RateLimit(deps.RateLimiter, 100))
		r.Use(middleware.AuditLog(deps.AuditLogRepo))

		// Tenants
		r.Route("/tenants", func(r chi.Router) {
			r.Post("/", deps.TenantHandler.Create)
			r.Get("/", deps.TenantHandler.List)
			r.Get("/{id}", deps.TenantHandler.Get)
			r.Put("/{id}", deps.TenantHandler.Update)
		})

		// Users
		r.Route("/users", func(r chi.Router) {
			r.Post("/", deps.UserHandler.Create)
			r.Get("/", deps.UserHandler.List)
			r.Get("/{id}", deps.UserHandler.Get)
			r.Put("/{id}", deps.UserHandler.Update)
		})

		// Agents
		r.Route("/agents", func(r chi.Router) {
			r.Post("/", deps.AgentHandler.Create)
			r.Get("/", deps.AgentHandler.List)
			r.Get("/{id}", deps.AgentHandler.Get)
		})

		// Skill Groups
		r.Route("/skill-groups", func(r chi.Router) {
			r.Post("/", deps.SkillGroupHandler.Create)
			r.Get("/", deps.SkillGroupHandler.List)
			r.Get("/{id}", deps.SkillGroupHandler.Get)
			r.Post("/{id}/members", deps.SkillGroupHandler.AddMember)
			r.Delete("/{id}/members/{agentId}", deps.SkillGroupHandler.RemoveMember)
			r.Get("/{id}/members", deps.SkillGroupHandler.GetMembers)
		})
	})

	return r
}
