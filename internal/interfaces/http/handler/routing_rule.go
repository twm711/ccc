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

type RoutingRuleHandler struct {
	repo telephony.RoutingRuleRepository
}

func NewRoutingRuleHandler(repo telephony.RoutingRuleRepository) *RoutingRuleHandler {
	return &RoutingRuleHandler{repo: repo}
}

func (h *RoutingRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var input struct {
		Name       string `json:"name"`
		MatchType  string `json:"match_type"`
		MatchValue string `json:"match_value"`
		SIPTrunkID int64  `json:"sip_trunk_id"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	rule := &telephony.RoutingRule{
		ID:         snowflake.NextID(),
		TenantID:   tenantID,
		Name:       input.Name,
		MatchType:  input.MatchType,
		MatchValue: input.MatchValue,
		SIPTrunkID: input.SIPTrunkID,
		Priority:   input.Priority,
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

func (h *RoutingRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	rules, total, err := h.repo.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": rules, "total": total})
}

func (h *RoutingRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rule, err := h.repo.GetByID(r.Context(), id)
	if err != nil || rule == nil {
		response.Error(w, http.StatusNotFound, "routing rule not found")
		return
	}

	var input struct {
		Name       string `json:"name"`
		MatchType  string `json:"match_type"`
		MatchValue string `json:"match_value"`
		SIPTrunkID int64  `json:"sip_trunk_id"`
		Priority   int    `json:"priority"`
		IsActive   *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name != "" {
		rule.Name = input.Name
	}
	if input.MatchType != "" {
		rule.MatchType = input.MatchType
	}
	if input.MatchValue != "" {
		rule.MatchValue = input.MatchValue
	}
	if input.SIPTrunkID > 0 {
		rule.SIPTrunkID = input.SIPTrunkID
	}
	if input.Priority > 0 {
		rule.Priority = input.Priority
	}
	if input.IsActive != nil {
		rule.IsActive = *input.IsActive
	}
	rule.UpdatedAt = time.Now()

	if err := h.repo.Update(r.Context(), rule); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *RoutingRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
