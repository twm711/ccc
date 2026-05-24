package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type DNCHandler struct {
	svc  *integration.DNCService
	repo integration.DNCRepository
}

func NewDNCHandler(svc *integration.DNCService, repo integration.DNCRepository) *DNCHandler {
	return &DNCHandler{svc: svc, repo: repo}
}

func (h *DNCHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var input struct {
		Number string `json:"number"`
		Reason string `json:"reason"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entry := &integration.DNCEntry{
		TenantID: tenantID,
		Number:   input.Number,
		Reason:   input.Reason,
		Source:   input.Source,
	}
	if entry.Source == "" {
		entry.Source = "manual"
	}

	if err := h.svc.AddEntry(r.Context(), entry); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, entry)
}

func (h *DNCHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	entries, total, err := h.repo.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": entries, "total": total})
}

func (h *DNCHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.RemoveEntry(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

func (h *DNCHandler) Check(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var input struct {
		Numbers []string `json:"numbers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	blocked, err := h.svc.CheckBatch(r.Context(), tenantID, input.Numbers)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"blocked": blocked})
}
