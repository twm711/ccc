package handler

import (
	"encoding/json"
	"net/http"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
)

type TenantSettingsHandler struct {
	repo identity.TenantSettingsRepository
}

func NewTenantSettingsHandler(repo identity.TenantSettingsRepository) *TenantSettingsHandler {
	return &TenantSettingsHandler{repo: repo}
}

func (h *TenantSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	s, err := h.repo.GetByTenantID(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "settings not found")
		return
	}
	response.JSON(w, http.StatusOK, s)
}

func (h *TenantSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var req identity.TenantSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.TenantID = tenantID
	if err := h.repo.Upsert(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, req)
}
