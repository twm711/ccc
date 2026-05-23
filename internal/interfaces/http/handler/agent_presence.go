package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type AgentPresenceHandler struct {
	svc *identity.AgentPresenceService
}

func NewAgentPresenceHandler(svc *identity.AgentPresenceService) *AgentPresenceHandler {
	return &AgentPresenceHandler{svc: svc}
}

func (h *AgentPresenceHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TenantID int64  `json:"tenant_id"`
		AgentID  int64  `json:"agent_id"`
		WorkMode string `json:"work_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.svc.CheckIn(r.Context(), in.TenantID, in.AgentID, identity.WorkMode(in.WorkMode))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *AgentPresenceHandler) CheckOut(w http.ResponseWriter, r *http.Request) {
	agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	if err := h.svc.CheckOut(r.Context(), agentID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AgentPresenceHandler) Transition(w http.ResponseWriter, r *http.Request) {
	agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	var in struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.svc.TransitionTo(r.Context(), agentID, identity.AgentPresenceStatus(in.Status))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *AgentPresenceHandler) SetBreak(w http.ResponseWriter, r *http.Request) {
	agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	var in struct {
		ReasonCode string `json:"reason_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.svc.SetBreak(r.Context(), agentID, in.ReasonCode)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *AgentPresenceHandler) SetACW(w http.ResponseWriter, r *http.Request) {
	agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	var in struct {
		DispositionCode string `json:"disposition_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.svc.SetACW(r.Context(), agentID, in.DispositionCode)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *AgentPresenceHandler) SwitchWorkMode(w http.ResponseWriter, r *http.Request) {
	agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	var in struct {
		WorkMode string `json:"work_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.svc.SwitchWorkMode(r.Context(), agentID, identity.WorkMode(in.WorkMode))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *AgentPresenceHandler) GetPresence(w http.ResponseWriter, r *http.Request) {
	agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentId"), 10, 64)
	p, err := h.svc.GetPresence(r.Context(), agentID)
	if err != nil || p == nil {
		response.Error(w, http.StatusNotFound, "presence not found")
		return
	}
	response.JSON(w, http.StatusOK, p)
}
