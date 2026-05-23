package middleware

import (
	"net/http"
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
				IP:        r.RemoteAddr,
				UserAgent: r.UserAgent(),
				CreatedAt: time.Now(),
			}

			_ = repo.Create(r.Context(), log)
		})
	}
}
