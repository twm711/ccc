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

type QuickReplyHandler struct {
	repo integration.QuickReplyRepository
}

func NewQuickReplyHandler(repo integration.QuickReplyRepository) *QuickReplyHandler {
	return &QuickReplyHandler{repo: repo}
}

func (h *QuickReplyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Scope     string `json:"scope"`
		ScopeID   *int64 `json:"scope_id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Shortcut  string `json:"shortcut"`
		SortOrder int    `json:"sort_order"`
		IsActive  bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	qr := &integration.QuickReply{
		ID: snowflake.NextID(), TenantID: 1,
		Scope: integration.QuickReplyScope(in.Scope), ScopeID: in.ScopeID,
		Title: in.Title, Content: in.Content, Shortcut: in.Shortcut,
		SortOrder: in.SortOrder, IsActive: in.IsActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.repo.Create(r.Context(), qr); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, qr)
}

func (h *QuickReplyHandler) List(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.repo.List(r.Context(), 1, 0, 50)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *QuickReplyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	qr, err := h.repo.GetByID(r.Context(), id)
	if err != nil || qr == nil {
		response.Error(w, http.StatusNotFound, "quick reply not found")
		return
	}
	var in struct {
		Title     string `json:"title"`
		Content   string `json:"content"`
		Shortcut  string `json:"shortcut"`
		SortOrder int    `json:"sort_order"`
		IsActive  bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	qr.Title = in.Title
	qr.Content = in.Content
	qr.Shortcut = in.Shortcut
	qr.SortOrder = in.SortOrder
	qr.IsActive = in.IsActive
	qr.UpdatedAt = time.Now()
	if err := h.repo.Update(r.Context(), qr); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, qr)
}

func (h *QuickReplyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

func (h *QuickReplyHandler) Available(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListAvailable(r.Context(), 1, nil, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, items)
}
