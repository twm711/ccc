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

type WebhookConfigHandler struct {
	repo integration.WebhookConfigRepository
}

func NewWebhookConfigHandler(repo integration.WebhookConfigRepository) *WebhookConfigHandler {
	return &WebhookConfigHandler{repo: repo}
}

func (h *WebhookConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Secret   string `json:"secret"`
		Events   string `json:"events"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	cfg := &integration.WebhookConfig{
		ID: snowflake.NextID(), TenantID: 1,
		Name: in.Name, URL: in.URL, Secret: in.Secret, Events: in.Events,
		IsActive: in.IsActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), cfg); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, cfg)
}

func (h *WebhookConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.repo.List(r.Context(), 1, 0, 50)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *WebhookConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	cfg, err := h.repo.GetByID(r.Context(), id)
	if err != nil || cfg == nil {
		response.Error(w, http.StatusNotFound, "webhook config not found")
		return
	}
	var in struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Secret   string `json:"secret"`
		Events   string `json:"events"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg.Name = in.Name
	cfg.URL = in.URL
	cfg.Secret = in.Secret
	cfg.Events = in.Events
	cfg.IsActive = in.IsActive
	cfg.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), cfg); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *WebhookConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
