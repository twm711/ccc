package handler

import (
	"net/http"
	"sync/atomic"

	"github.com/divord97/ccc/pkg/response"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var ready int32 = 1

func Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "version": Version})
}

// Readyz returns 200 when the instance is ready to receive traffic.
// During blue-green or canary rollouts the readiness can be toggled via SetReady.
func Readyz(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&ready) == 0 {
		response.Error(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// SetReady toggles readiness (0 = not ready, 1 = ready).
func SetReady(v bool) {
	if v {
		atomic.StoreInt32(&ready, 1)
	} else {
		atomic.StoreInt32(&ready, 0)
	}
}
