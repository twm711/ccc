package wsutil

import (
	"net/http"
	"os"
	"strings"
)

// CheckOrigin returns a websocket.Upgrader-compatible CheckOrigin function
// that validates the request Origin against CORS_ALLOW_ORIGIN. When the env
// var is empty, only same-origin requests are allowed.
func CheckOrigin() func(r *http.Request) bool {
	allowed := os.Getenv("CORS_ALLOW_ORIGIN")
	if allowed == "*" {
		return func(r *http.Request) bool { return true }
	}

	allowedSet := make(map[string]bool)
	if allowed != "" {
		for _, o := range strings.Split(allowed, ",") {
			allowedSet[strings.TrimSpace(o)] = true
		}
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // same-origin or non-browser client
		}
		if len(allowedSet) > 0 {
			return allowedSet[origin]
		}
		// No allowlist configured: enforce same-origin
		host := r.Host
		return strings.HasSuffix(origin, "://"+host)
	}
}
