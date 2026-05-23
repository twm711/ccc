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

type AgentHandler struct {
	svc *identity.AgentService
}

func NewAgentHandler(svc *identity.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		UserID       int64             `json:"user_id"`
		EmployeeID   string            `json:"employee_id"`
		Extension    string            `json:"extension"`
		WorkMode     identity.WorkMode `json:"work_mode"`
		MaxChatSlots int               `json:"max_chat_slots"`
		ACWSeconds   int               `json:"acw_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.UserID == 0 {
		response.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}

	agent, err := h.svc.Create(r.Context(), identity.CreateAgentInput{
		TenantID:     tenantID,
		UserID:       in.UserID,
		EmployeeID:   in.EmployeeID,
		Extension:    in.Extension,
		WorkMode:     in.WorkMode,
		MaxChatSlots: in.MaxChatSlots,
		ACWSeconds:   in.ACWSeconds,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, agent)
}

func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	agent, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "agent not found")
		return
	}
	response.JSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	agents, total, err := h.svc.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	totalPages := int(total) / p.PageSize
	if int(total)%p.PageSize > 0 {
		totalPages++
	}
	response.JSON(w, http.StatusOK, response.PagedData{
		Items:      agents,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages,
	})
}
