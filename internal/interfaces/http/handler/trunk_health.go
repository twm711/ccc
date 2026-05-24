package handler

import (
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/telephony"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type TrunkHealthHandler struct {
	svc *telephony.TrunkHealthService
}

func NewTrunkHealthHandler(svc *telephony.TrunkHealthService) *TrunkHealthHandler {
	return &TrunkHealthHandler{svc: svc}
}

func (h *TrunkHealthHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	trunkID, _ := strconv.ParseInt(chi.URLParam(r, "trunkId"), 10, 64)
	status := h.svc.GetHealthStatus(trunkID)
	if status == nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "unknown"})
		return
	}
	response.JSON(w, http.StatusOK, status)
}
