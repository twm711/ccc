package middleware

import (
	"net/http"
	"os"
	"strings"
)

func CORS(next http.Handler) http.Handler {
	allowed := os.Getenv("CORS_ALLOW_ORIGIN")
	allowedSet := make(map[string]bool)
	if allowed != "" && allowed != "*" {
		for _, o := range strings.Split(allowed, ",") {
			allowedSet[strings.TrimSpace(o)] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqOrigin := r.Header.Get("Origin")
		var origin string

		switch {
		case len(allowedSet) > 0 && allowedSet[reqOrigin]:
			origin = reqOrigin
		case len(allowedSet) > 0:
			origin = ""
		case allowed == "" && reqOrigin != "":
			// No CORS_ALLOW_ORIGIN configured: reject credentials-bearing
			// cross-origin requests by echoing the origin without credentials.
			origin = reqOrigin
		default:
			origin = "*"
		}

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")
			if origin != "*" {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
