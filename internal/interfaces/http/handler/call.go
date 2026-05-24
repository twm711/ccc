package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/application/outbound"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type CallHandler struct {
	callSvc    *call.CallService
	outboundSvc *outbound.Service
	tagSvc     *integration.CallTagService
}

func NewCallHandler(callSvc *call.CallService, outboundSvc *outbound.Service, tagSvc *integration.CallTagService) *CallHandler {
	return &CallHandler{callSvc: callSvc, outboundSvc: outboundSvc, tagSvc: tagSvc}
}

func (h *CallHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	filter := call.CallListFilter{
		Caller: r.URL.Query().Get("caller"),
		Callee: r.URL.Query().Get("callee"),
	}

	if d := r.URL.Query().Get("direction"); d != "" {
		dir := call.CallDirection(d)
		filter.Direction = &dir
	}
	if ct := r.URL.Query().Get("call_type"); ct != "" {
		callType := call.CallType(ct)
		filter.CallType = &callType
	}
	if mt := r.URL.Query().Get("media_type"); mt != "" {
		mediaType := call.MediaType(mt)
		filter.MediaType = &mediaType
	}
	if s := r.URL.Query().Get("status"); s != "" {
		status := call.CallStatus(s)
		filter.Status = &status
	}
	if sf := r.URL.Query().Get("start_from"); sf != "" {
		if t, err := time.Parse(time.RFC3339, sf); err == nil {
			filter.StartFrom = &t
		}
	}
	if st := r.URL.Query().Get("start_to"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			filter.StartTo = &t
		}
	}

	calls, total, err := h.callSvc.ListCalls(r.Context(), tenantID, filter, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": calls, "total": total})
}

func (h *CallHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.callSvc.GetByID(r.Context(), id)
	if err != nil || c == nil {
		response.Error(w, http.StatusNotFound, "call not found")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CallHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	callID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	events, err := h.callSvc.GetEvents(r.Context(), callID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, events)
}

func (h *CallHandler) GetIVRTracking(w http.ResponseWriter, r *http.Request) {
	callID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tracking, err := h.callSvc.GetIVRTracking(r.Context(), callID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, tracking)
}

func (h *CallHandler) Dial(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)
	var input struct {
		AgentUserID int64  `json:"agent_user_id"`
		Callee      string `json:"callee"`
		MediaType   string `json:"media_type"`
		CLIPolicyID *int64 `json:"cli_policy_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.outboundSvc.Dial(r.Context(), outbound.DialRequest{
		TenantID:    tenantID,
		AgentUserID: input.AgentUserID,
		Callee:      input.Callee,
		MediaType:   call.MediaType(input.MediaType),
		CLIPolicyID: input.CLIPolicyID,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *CallHandler) InternalDial(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)
	var input struct {
		CallerAgentID int64  `json:"caller_agent_id"`
		CalleeAgentID int64  `json:"callee_agent_id"`
		CallerExt     string `json:"caller_ext"`
		CalleeExt     string `json:"callee_ext"`
		MediaType     string `json:"media_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.outboundSvc.DialInternal(r.Context(), outbound.InternalDialRequest{
		TenantID:      tenantID,
		CallerAgentID: input.CallerAgentID,
		CalleeAgentID: input.CalleeAgentID,
		CallerExt:     input.CallerExt,
		CalleeExt:     input.CalleeExt,
		MediaType:     call.MediaType(input.MediaType),
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *CallHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)
	callID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var input struct {
		TagID   int64  `json:"tag_id"`
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.tagSvc.AssignTag(r.Context(), &integration.CallTagAssignment{
		TenantID: tenantID,
		CallID:   callID,
		TagID:    input.TagID,
		TagName:  input.TagName,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (h *CallHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagId"), 10, 64)
	if err := h.tagSvc.RemoveTag(r.Context(), tagID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
