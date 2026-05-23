package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/pkg/response"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/go-chi/chi/v5"
)

type SmsConfigHandler struct {
	repo integration.SmsConfigRepository
}

func NewSmsConfigHandler(repo integration.SmsConfigRepository) *SmsConfigHandler {
	return &SmsConfigHandler{repo: repo}
}

func (h *SmsConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Provider    string `json:"provider"`
		AccessKeyID string `json:"access_key_id"`
		SignName    string `json:"sign_name"`
		TemplateMap string `json:"template_map"`
		IsActive    bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	cfg := &integration.SmsConfig{
		ID: snowflake.NextID(), TenantID: 1,
		Provider: in.Provider, AccessKeyID: in.AccessKeyID, SignName: in.SignName,
		TemplateMap: in.TemplateMap, IsActive: in.IsActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), cfg); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, cfg)
}

func (h *SmsConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.repo.List(r.Context(), 1, 0, 50)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *SmsConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	cfg, err := h.repo.GetByID(r.Context(), id)
	if err != nil || cfg == nil {
		response.Error(w, http.StatusNotFound, "sms config not found")
		return
	}
	var in struct {
		Provider    string `json:"provider"`
		AccessKeyID string `json:"access_key_id"`
		SignName    string `json:"sign_name"`
		TemplateMap string `json:"template_map"`
		IsActive    bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg.Provider = in.Provider
	cfg.AccessKeyID = in.AccessKeyID
	cfg.SignName = in.SignName
	cfg.TemplateMap = in.TemplateMap
	cfg.IsActive = in.IsActive
	cfg.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), cfg); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *SmsConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
