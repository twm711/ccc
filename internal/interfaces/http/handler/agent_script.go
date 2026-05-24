package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/ai"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type AgentScriptHandler struct {
	svc *ai.AgentScriptService
}

func NewAgentScriptHandler(svc *ai.AgentScriptService) *AgentScriptHandler {
	return &AgentScriptHandler{svc: svc}
}

func (h *AgentScriptHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in ai.CreateScriptInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s, err := h.svc.Create(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, s)
}

func (h *AgentScriptHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.svc.List(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *AgentScriptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	s, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	var in struct {
		Name     *string `json:"name"`
		Content  *string `json:"content"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		s.Name = *in.Name
	}
	if in.Content != nil {
		s.Content = *in.Content
	}
	if in.IsActive != nil {
		s.IsActive = *in.IsActive
	}
	if err := h.svc.Update(r.Context(), s); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, s)
}
