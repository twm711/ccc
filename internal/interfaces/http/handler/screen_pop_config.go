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

type ScreenPopConfigHandler struct {
	repo integration.ScreenPopConfigRepository
}

func NewScreenPopConfigHandler(repo integration.ScreenPopConfigRepository) *ScreenPopConfigHandler {
	return &ScreenPopConfigHandler{repo: repo}
}

func (h *ScreenPopConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string `json:"name"`
		URLTemplate string `json:"url_template"`
		Position    int    `json:"position"`
		IsActive    bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	cfg := &integration.ScreenPopConfig{
		ID: snowflake.NextID(), TenantID: 1,
		Name: in.Name, URLTemplate: in.URLTemplate, Position: in.Position,
		IsActive: in.IsActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), cfg); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, cfg)
}

func (h *ScreenPopConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.repo.List(r.Context(), 1, 0, 50)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *ScreenPopConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	cfg, err := h.repo.GetByID(r.Context(), id)
	if err != nil || cfg == nil {
		response.Error(w, http.StatusNotFound, "screen pop config not found")
		return
	}
	var in struct {
		Name        string `json:"name"`
		URLTemplate string `json:"url_template"`
		Position    int    `json:"position"`
		IsActive    bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg.Name = in.Name
	cfg.URLTemplate = in.URLTemplate
	cfg.Position = in.Position
	cfg.IsActive = in.IsActive
	cfg.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), cfg); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *ScreenPopConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
