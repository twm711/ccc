package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/application/csat"
	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/go-chi/chi/v5"
)

type CSATHandler struct {
	svc     *csat.Service
	configs integration.CSATConfigRepository
	results integration.CSATResultRepository
}

func NewCSATHandler(svc *csat.Service, configs integration.CSATConfigRepository, results integration.CSATResultRepository) *CSATHandler {
	return &CSATHandler{svc: svc, configs: configs, results: results}
}

func (h *CSATHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var c integration.CSATConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c.ID = snowflake.NextID()
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt

	if err := h.configs.Create(r.Context(), &c); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *CSATHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	items, total, err := h.configs.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *CSATHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	existing, err := h.configs.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	var in struct {
		TriggerType *string `json:"trigger_type"`
		ScaleMin    *int    `json:"scale_min"`
		ScaleMax    *int    `json:"scale_max"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.TriggerType != nil {
		existing.TriggerType = *in.TriggerType
	}
	if in.ScaleMin != nil {
		existing.ScaleMin = *in.ScaleMin
	}
	if in.ScaleMax != nil {
		existing.ScaleMax = *in.ScaleMax
	}
	if in.IsActive != nil {
		existing.IsActive = *in.IsActive
	}
	existing.UpdatedAt = time.Now()

	if err := h.configs.Update(r.Context(), existing); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, existing)
}

func (h *CSATHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 50, 200)

	items, total, err := h.results.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}
