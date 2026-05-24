package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/domain/configuration"
	"github.com/divord97/ccc/internal/domain/operation"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/go-chi/chi/v5"
)

// --- BreakReasonHandler ---

type BreakReasonHandler struct {
	repo configuration.BreakReasonRepository
}

func NewBreakReasonHandler(repo configuration.BreakReasonRepository) *BreakReasonHandler {
	return &BreakReasonHandler{repo: repo}
}

func (h *BreakReasonHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		IsSystem  bool   `json:"is_system"`
		SortOrder int    `json:"sort_order"`
		Enabled   bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "code and name are required")
		return
	}
	now := time.Now()
	br := &configuration.BreakReason{
		ID:        snowflake.NextID(),
		TenantID:  middleware.TenantIDFromCtx(r.Context()),
		Code:      req.Code,
		Name:      req.Name,
		IsSystem:  req.IsSystem,
		SortOrder: req.SortOrder,
		Enabled:   req.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), br); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, br)
}

func (h *BreakReasonHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.repo.List(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *BreakReasonHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	br, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	response.JSON(w, http.StatusOK, br)
}

func (h *BreakReasonHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	br, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	var req struct {
		Code      *string `json:"code"`
		Name      *string `json:"name"`
		IsSystem  *bool   `json:"is_system"`
		SortOrder *int    `json:"sort_order"`
		Enabled   *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Code != nil {
		br.Code = *req.Code
	}
	if req.Name != nil {
		br.Name = *req.Name
	}
	if req.IsSystem != nil {
		br.IsSystem = *req.IsSystem
	}
	if req.SortOrder != nil {
		br.SortOrder = *req.SortOrder
	}
	if req.Enabled != nil {
		br.Enabled = *req.Enabled
	}
	br.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), br); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, br)
}

func (h *BreakReasonHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- DispositionCodeHandler ---

type DispositionCodeHandler struct {
	repo configuration.DispositionCodeRepository
}

func NewDispositionCodeHandler(repo configuration.DispositionCodeRepository) *DispositionCodeHandler {
	return &DispositionCodeHandler{repo: repo}
}

func (h *DispositionCodeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Category  string `json:"category"`
		SortOrder int    `json:"sort_order"`
		Enabled   bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "code and name are required")
		return
	}
	now := time.Now()
	dc := &configuration.DispositionCode{
		ID:        snowflake.NextID(),
		TenantID:  middleware.TenantIDFromCtx(r.Context()),
		Code:      req.Code,
		Name:      req.Name,
		Category:  req.Category,
		SortOrder: req.SortOrder,
		Enabled:   req.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), dc); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, dc)
}

func (h *DispositionCodeHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.repo.List(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *DispositionCodeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	dc, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	response.JSON(w, http.StatusOK, dc)
}

func (h *DispositionCodeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	dc, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	var req struct {
		Code      *string `json:"code"`
		Name      *string `json:"name"`
		Category  *string `json:"category"`
		SortOrder *int    `json:"sort_order"`
		Enabled   *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Code != nil {
		dc.Code = *req.Code
	}
	if req.Name != nil {
		dc.Name = *req.Name
	}
	if req.Category != nil {
		dc.Category = *req.Category
	}
	if req.SortOrder != nil {
		dc.SortOrder = *req.SortOrder
	}
	if req.Enabled != nil {
		dc.Enabled = *req.Enabled
	}
	dc.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), dc); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, dc)
}

func (h *DispositionCodeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- AudioFileHandler ---

type AudioFileHandler struct {
	repo operation.AudioFileRepository
}

func NewAudioFileHandler(repo operation.AudioFileRepository) *AudioFileHandler {
	return &AudioFileHandler{repo: repo}
}

func (h *AudioFileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string                  `json:"name"`
		FileName string                  `json:"file_name"`
		Category operation.AudioCategory `json:"category"`
		FilePath string                  `json:"file_path"`
		FileSize int64                   `json:"file_size"`
		Duration int                     `json:"duration"`
		MimeType string                  `json:"mime_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	af := &operation.AudioFile{
		ID:        snowflake.NextID(),
		TenantID:  middleware.TenantIDFromCtx(r.Context()),
		Name:      req.Name,
		FileName:  req.FileName,
		Category:  req.Category,
		FilePath:  req.FilePath,
		FileSize:  req.FileSize,
		Duration:  req.Duration,
		MimeType:  req.MimeType,
		CreatedAt: time.Now(),
	}
	if err := h.repo.Create(r.Context(), af); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, af)
}

func (h *AudioFileHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	category := operation.AudioCategory(r.URL.Query().Get("category"))
	items, err := h.repo.List(r.Context(), tenantID, category)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *AudioFileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	af, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	response.JSON(w, http.StatusOK, af)
}

func (h *AudioFileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- BusinessHoursHandler ---

type BusinessHoursHandler struct {
	repo         operation.BusinessHoursRepository
	scheduleRepo operation.BusinessHoursScheduleRepository
}

func NewBusinessHoursHandler(repo operation.BusinessHoursRepository, scheduleRepo operation.BusinessHoursScheduleRepository) *BusinessHoursHandler {
	return &BusinessHoursHandler{repo: repo, scheduleRepo: scheduleRepo}
}

func (h *BusinessHoursHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		IsDefault bool   `json:"is_default"`
		Timezone  string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	now := time.Now()
	bh := &operation.BusinessHours{
		ID:        snowflake.NextID(),
		TenantID:  middleware.TenantIDFromCtx(r.Context()),
		Name:      req.Name,
		IsDefault: req.IsDefault,
		Timezone:  req.Timezone,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), bh); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, bh)
}

func (h *BusinessHoursHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.repo.List(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *BusinessHoursHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	bh, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	schedules, _ := h.scheduleRepo.GetByBusinessHoursID(r.Context(), id)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"business_hours": bh,
		"schedules":      schedules,
	})
}

func (h *BusinessHoursHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	bh, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	var req struct {
		Name      *string `json:"name"`
		IsDefault *bool   `json:"is_default"`
		Timezone  *string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		bh.Name = *req.Name
	}
	if req.IsDefault != nil {
		bh.IsDefault = *req.IsDefault
	}
	if req.Timezone != nil {
		bh.Timezone = *req.Timezone
	}
	bh.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), bh); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, bh)
}

func (h *BusinessHoursHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = h.scheduleRepo.DeleteByBusinessHoursID(r.Context(), id)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- CallTagHandler (for tag definitions, not assignments) ---

type CallTagDefHandler struct {
	repo configuration.CallTagRepository
}

func NewCallTagDefHandler(repo configuration.CallTagRepository) *CallTagDefHandler {
	return &CallTagDefHandler{repo: repo}
}

func (h *CallTagDefHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	ct := &configuration.CallTag{
		ID:        snowflake.NextID(),
		TenantID:  middleware.TenantIDFromCtx(r.Context()),
		Name:      req.Name,
		Color:     req.Color,
		CreatedAt: time.Now(),
	}
	if err := h.repo.Create(r.Context(), ct); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, ct)
}

func (h *CallTagDefHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.repo.List(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *CallTagDefHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	ct, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "not found")
		return
	}
	response.JSON(w, http.StatusOK, ct)
}

func (h *CallTagDefHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
