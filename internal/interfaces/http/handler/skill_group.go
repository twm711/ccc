package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type SkillGroupHandler struct {
	svc *identity.SkillGroupService
}

func NewSkillGroupHandler(svc *identity.SkillGroupService) *SkillGroupHandler {
	return &SkillGroupHandler{svc: svc}
}

func (h *SkillGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		Code          string                `json:"code"`
		Name          string                `json:"name"`
		Description   string                `json:"description"`
		RoutingPolicy identity.RoutingPolicy `json:"routing_policy"`
		Priority      int                   `json:"priority"`
		MaxWaitSec    int                   `json:"max_wait_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.Code == "" || in.Name == "" {
		response.Error(w, http.StatusBadRequest, "code and name are required")
		return
	}

	sg, err := h.svc.Create(r.Context(), identity.CreateSkillGroupInput{
		TenantID:      tenantID,
		Code:          in.Code,
		Name:          in.Name,
		Description:   in.Description,
		RoutingPolicy: in.RoutingPolicy,
		Priority:      in.Priority,
		MaxWaitSec:    in.MaxWaitSec,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, sg)
}

func (h *SkillGroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	sg, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "skill group not found")
		return
	}
	response.JSON(w, http.StatusOK, sg)
}

func (h *SkillGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	groups, total, err := h.svc.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list skill groups")
		return
	}
	totalPages := int(total) / p.PageSize
	if int(total)%p.PageSize > 0 {
		totalPages++
	}
	response.JSON(w, http.StatusOK, response.PagedData{
		Items:      groups,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages,
	})
}

func (h *SkillGroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	sgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		AgentID int64 `json:"agent_id"`
		Level   int   `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	m, err := h.svc.AddMember(r.Context(), sgID, in.AgentID, in.Level)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, m)
}

func (h *SkillGroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	sgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	agentID, err := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.svc.RemoveMember(r.Context(), sgID, agentID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to remove member")
		return
	}
	response.JSON(w, http.StatusOK, nil)
}

func (h *SkillGroupHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	sgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	members, err := h.svc.GetMembers(r.Context(), sgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	response.JSON(w, http.StatusOK, members)
}
