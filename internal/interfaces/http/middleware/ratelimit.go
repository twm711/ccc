package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/domain/identity"
	infraRedis "github.com/divord97/ccc/internal/infrastructure/redis"
	"github.com/divord97/ccc/pkg/response"
)

// TenantRateProvider resolves the per-tenant API rate limit. Implementations
// should be safe for concurrent use; the middleware caches results in-process.
type TenantRateProvider interface {
	GetByTenantID(ctx context.Context, tenantID int64) (*identity.TenantSettings, error)
}

const rateCacheTTL = 60 * time.Second

type rateCacheEntry struct {
	limit     int
	expiresAt time.Time
}

// RateLimit applies a Redis-backed token bucket per tenant. The per-tenant
// limit is looked up from tenant_settings.api_rate_limit_per_sec with a
// 60 second in-process cache; defaultRate is used when the tenant has no
// settings row or the lookup fails.
func RateLimit(limiter *infraRedis.RateLimiter, settings TenantRateProvider, defaultRate int) func(http.Handler) http.Handler {
	cache := map[int64]rateCacheEntry{}
	var mu sync.RWMutex

	resolve := func(ctx context.Context, tenantID int64) int {
		mu.RLock()
		entry, ok := cache[tenantID]
		mu.RUnlock()
		if ok && time.Now().Before(entry.expiresAt) {
			return entry.limit
		}
		limit := defaultRate
		if settings != nil {
			if ts, err := settings.GetByTenantID(ctx, tenantID); err == nil && ts != nil && ts.APIRateLimitPerSec > 0 {
				limit = ts.APIRateLimitPerSec
			}
		}
		mu.Lock()
		cache[tenantID] = rateCacheEntry{limit: limit, expiresAt: time.Now().Add(rateCacheTTL)}
		mu.Unlock()
		return limit
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromCtx(r.Context())
			if tenantID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			limit := resolve(r.Context(), tenantID)
			allowed, err := limiter.Allow(r.Context(), tenantID, limit)
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
