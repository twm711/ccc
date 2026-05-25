package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/divord97/ccc/internal/domain/platform"
	"github.com/divord97/ccc/pkg/snowflake"
)

func AuditLog(repo platform.AuditLogRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				return
			}

			tenantID := TenantIDFromCtx(r.Context())
			userID := UserIDFromCtx(r.Context())
			if tenantID == 0 {
				return
			}

			log := &platform.AuditLog{
				ID:        snowflake.NextID(),
				TenantID:  tenantID,
				UserID:    userID,
				Action:    r.Method,
				Resource:  r.URL.Path,
				IP:        clientIP(r),
				UserAgent: r.UserAgent(),
				CreatedAt: time.Now(),
			}

			// Async write to avoid blocking the response path.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = repo.Create(ctx, log)
			}()
		})
	}
}

// clientIP returns the client's real IP, preferring X-Forwarded-For and
// X-Real-IP headers set by reverse proxies.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
