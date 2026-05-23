package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/domain/telephony"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/go-chi/chi/v5"
)

// CallNumberTagHandler

type CallNumberTagHandler struct {
	repo telephony.CallNumberTagRepository
}

func NewCallNumberTagHandler(repo telephony.CallNumberTagRepository) *CallNumberTagHandler {
	return &CallNumberTagHandler{repo: repo}
}

func (h *CallNumberTagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Number string `json:"number"`
		Tag    string `json:"tag"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Number == "" || req.Tag == "" {
		response.Error(w, http.StatusBadRequest, "number and tag are required")
		return
	}

	t := &telephony.CallNumberTag{
		ID:        snowflake.NextID(),
		TenantID:  middleware.TenantIDFromCtx(r.Context()),
		Number:    req.Number,
		Tag:       req.Tag,
		Source:    req.Source,
		CreatedAt: time.Now(),
	}
	if err := h.repo.Create(r.Context(), t); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

func (h *CallNumberTagHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	tags, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": tags, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

func (h *CallNumberTagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// AutoTagRuleHandler

type AutoTagRuleHandler struct {
	repo telephony.AutoTagRuleRepository
}

func NewAutoTagRuleHandler(repo telephony.AutoTagRuleRepository) *AutoTagRuleHandler {
	return &AutoTagRuleHandler{repo: repo}
}

func (h *AutoTagRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		MatchType  string `json:"match_type"`
		MatchValue string `json:"match_value"`
		Tag        string `json:"tag"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now()
	rule := &telephony.AutoTagRule{
		ID:         snowflake.NextID(),
		TenantID:   middleware.TenantIDFromCtx(r.Context()),
		Name:       req.Name,
		MatchType:  req.MatchType,
		MatchValue: req.MatchValue,
		Tag:        req.Tag,
		Priority:   req.Priority,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.repo.Create(r.Context(), rule); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, rule)
}

func (h *AutoTagRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	rules, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": rules, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

func (h *AutoTagRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	rule, err := h.repo.GetByID(r.Context(), id)
	if err != nil || rule == nil {
		response.Error(w, http.StatusNotFound, "auto tag rule not found")
		return
	}

	var req struct {
		Name       *string `json:"name"`
		MatchType  *string `json:"match_type"`
		MatchValue *string `json:"match_value"`
		Tag        *string `json:"tag"`
		Priority   *int    `json:"priority"`
		IsActive   *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.MatchType != nil {
		rule.MatchType = *req.MatchType
	}
	if req.MatchValue != nil {
		rule.MatchValue = *req.MatchValue
	}
	if req.Tag != nil {
		rule.Tag = *req.Tag
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	rule.UpdatedAt = time.Now()

	if err := h.repo.Update(r.Context(), rule); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *AutoTagRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
