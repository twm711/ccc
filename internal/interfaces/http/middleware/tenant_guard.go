package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// TenantGuard prevents cross-tenant access by validating that any tenant_id
// in the URL path or query matches the authenticated user's JWT tenant.
func TenantGuard() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jwtTenant := TenantIDFromCtx(r.Context())
			if jwtTenant == 0 {
				next.ServeHTTP(w, r)
				return
			}

			role := RoleFromCtx(r.Context())
			if role == "super_admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Check query parameter tenant_id.
			if qTenant := r.URL.Query().Get("tenant_id"); qTenant != "" {
				if tid, err := strconv.ParseInt(qTenant, 10, 64); err == nil && tid != jwtTenant {
					http.Error(w, `{"error":"tenant access denied"}`, http.StatusForbidden)
					return
				}
			}

			// Check path segment after /tenants/.
			if idx := strings.Index(r.URL.Path, "/tenants/"); idx >= 0 {
				rest := r.URL.Path[idx+len("/tenants/"):]
				if slash := strings.IndexByte(rest, '/'); slash > 0 {
					rest = rest[:slash]
				}
				if tid, err := strconv.ParseInt(rest, 10, 64); err == nil && tid != jwtTenant {
					http.Error(w, `{"error":"tenant access denied"}`, http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
