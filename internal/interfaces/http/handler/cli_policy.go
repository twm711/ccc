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

type CLIPolicyHandler struct {
	repo telephony.CLIPolicyRepository
}

func NewCLIPolicyHandler(repo telephony.CLIPolicyRepository) *CLIPolicyHandler {
	return &CLIPolicyHandler{repo: repo}
}

func (h *CLIPolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var input struct {
		Name          string `json:"name"`
		Strategy      string `json:"strategy"`
		FixedNumberID *int64 `json:"fixed_number_id"`
		NumberPoolIDs string `json:"number_pool_ids"`
		IsDefault     bool   `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	policy := &telephony.CLIPolicy{
		ID:            snowflake.NextID(),
		TenantID:      tenantID,
		Name:          input.Name,
		Strategy:      telephony.CLIStrategy(input.Strategy),
		FixedNumberID: input.FixedNumberID,
		NumberPoolIDs: input.NumberPoolIDs,
		IsDefault:     input.IsDefault,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.repo.Create(r.Context(), policy); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, policy)
}

func (h *CLIPolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	policies, total, err := h.repo.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": policies, "total": total})
}

func (h *CLIPolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	policy, err := h.repo.GetByID(r.Context(), id)
	if err != nil || policy == nil {
		response.Error(w, http.StatusNotFound, "CLI policy not found")
		return
	}

	var input struct {
		Name          string `json:"name"`
		Strategy      string `json:"strategy"`
		FixedNumberID *int64 `json:"fixed_number_id"`
		NumberPoolIDs string `json:"number_pool_ids"`
		IsDefault     *bool  `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name != "" {
		policy.Name = input.Name
	}
	if input.Strategy != "" {
		policy.Strategy = telephony.CLIStrategy(input.Strategy)
	}
	if input.FixedNumberID != nil {
		policy.FixedNumberID = input.FixedNumberID
	}
	if input.NumberPoolIDs != "" {
		policy.NumberPoolIDs = input.NumberPoolIDs
	}
	if input.IsDefault != nil {
		policy.IsDefault = *input.IsDefault
	}
	policy.UpdatedAt = time.Now()

	if err := h.repo.Update(r.Context(), policy); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, policy)
}
