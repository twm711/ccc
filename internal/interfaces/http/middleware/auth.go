package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/divord97/ccc/pkg/response"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ContextKeyTenantID contextKey = "tenant_id"
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyRole     contextKey = "role"
)

type Claims struct {
	jwt.RegisteredClaims
	TenantID int64  `json:"tenant_id"`
	UserID   int64  `json:"user_id"`
	Role     string `json:"role"`
}

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				response.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyTenantID, claims.TenantID)
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TenantIDFromCtx(ctx context.Context) int64 {
	v, _ := ctx.Value(ContextKeyTenantID).(int64)
	return v
}

func UserIDFromCtx(ctx context.Context) int64 {
	v, _ := ctx.Value(ContextKeyUserID).(int64)
	return v
}

func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRole).(string)
	return v
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromCtx(r.Context())
			if !allowed[role] {
				response.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}
