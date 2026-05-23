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

	// Infrastructure
	RateLimiter  *redis.RateLimiter
	AuditLogRepo platform.AuditLogRepository
	JWTSecret    string
	Logger       zerolog.Logger
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
			r.Patch("/{id}/read", deps.VoicemailHandler.MarkRead)
			r.Delete("/{id}", deps.VoicemailHandler.Delete)
		})
	})

	return r
}
