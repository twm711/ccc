package middleware

import (
	"net/http"

	infraRedis "github.com/divord97/ccc/internal/infrastructure/redis"
	"github.com/divord97/ccc/pkg/response"
)

func RateLimit(limiter *infraRedis.RateLimiter, defaultRate int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromCtx(r.Context())
			if tenantID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			allowed, err := limiter.Allow(r.Context(), tenantID, defaultRate)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
