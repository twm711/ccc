package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/divord97/ccc/internal/domain/platform"
	"github.com/divord97/ccc/pkg/snowflake"
)

// sensitiveGETPaths lists URL path prefixes where GET requests access
// privacy-sensitive data and must be audit-logged for compliance.
var sensitiveGETPaths = []string{
	"/api/v1/recordings",
	"/api/v1/audit-logs",
	"/api/v1/customers",
	"/api/v1/calls",
	"/api/v1/agents",
	"/api/v1/reports",
	"/api/v1/voicemails",
	"/api/v1/tickets",
}

func isSensitiveGET(path string) bool {
	for _, prefix := range sensitiveGETPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func AuditLog(repo platform.AuditLogRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			if r.Method == http.MethodHead || r.Method == http.MethodOptions {
				return
			}
			// Log all mutating requests; for GET, only log sensitive read paths.
			if r.Method == http.MethodGet && !isSensitiveGET(r.URL.Path) {
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
