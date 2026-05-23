package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type CallControlHandler struct {
	svc *call.CallService
}

func NewCallControlHandler(svc *call.CallService) *CallControlHandler {
	return &CallControlHandler{svc: svc}
}

func (h *CallControlHandler) Hold(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.HoldCall(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CallControlHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.RetrieveCall(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CallControlHandler) BlindTransfer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		Type         string `json:"type"`
		SkillGroupID *int64 `json:"skill_group_id"`
		AgentUserID  *int64 `json:"agent_user_id"`
		ExternalNum  string `json:"external_num"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := h.svc.BlindTransfer(r.Context(), id, call.TransferTarget{
		Type: in.Type, SkillGroupID: in.SkillGroupID, AgentUserID: in.AgentUserID, ExternalNum: in.ExternalNum,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CallControlHandler) SendDTMF(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		Digits string `json:"digits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SendDTMF(r.Context(), id, in.Digits); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *CallControlHandler) RequestCallback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TenantID     int64  `json:"tenant_id"`
		CallID       int64  `json:"call_id"`
		SkillGroupID int64  `json:"skill_group_id"`
		Caller       string `json:"caller"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cb := &call.CallbackRequest{
		TenantID: in.TenantID, CallID: in.CallID, SkillGroupID: in.SkillGroupID, Caller: in.Caller,
	}
	if err := h.svc.RequestCallback(r.Context(), cb); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, cb)
}
