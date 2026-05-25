package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/application/outbound"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/integration"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

const maxExportRows = 10000

type CallHandler struct {
	callSvc    *call.CallService
	outboundSvc *outbound.Service
	tagSvc     *integration.CallTagService
}

func NewCallHandler(callSvc *call.CallService, outboundSvc *outbound.Service, tagSvc *integration.CallTagService) *CallHandler {
	return &CallHandler{callSvc: callSvc, outboundSvc: outboundSvc, tagSvc: tagSvc}
}

func (h *CallHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

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

// Export streams call records as CSV. Accepts the same filter query params as
// List. Capped at maxExportRows to avoid OOM — callers needing more should
// page using cursor APIs and stitch client-side.
func (h *CallHandler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
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

	calls, _, err := h.callSvc.ListCalls(r.Context(), tenantID, filter, 0, maxExportRows)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="calls_%s.csv"`, time.Now().Format("20060102_150405")))
	// UTF-8 BOM so Excel renders Chinese correctly.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"id", "direction", "call_type", "caller", "callee", "status",
		"agent_user_id", "skill_group_id", "started_at", "answered_at", "ended_at",
		"duration_sec", "ring_duration_sec", "queue_duration_sec", "hangup_reason",
	})
	for _, c := range calls {
		var agent, sg, answered, ended, hangup string
		if c.AgentUserID != nil {
			agent = strconv.FormatInt(*c.AgentUserID, 10)
		}
		if c.SkillGroupID != nil {
			sg = strconv.FormatInt(*c.SkillGroupID, 10)
		}
		if c.AnsweredAt != nil {
			answered = c.AnsweredAt.Format(time.RFC3339)
		}
		if c.EndedAt != nil {
			ended = c.EndedAt.Format(time.RFC3339)
		}
		if c.HangupReason != nil {
			hangup = string(*c.HangupReason)
		}
		_ = cw.Write([]string{
			strconv.FormatInt(c.ID, 10),
			string(c.Direction),
			string(c.CallType),
			c.Caller,
			c.Callee,
			string(c.Status),
			agent, sg,
			c.StartedAt.Format(time.RFC3339),
			answered, ended,
			strconv.Itoa(c.DurationSec),
			strconv.Itoa(c.RingDurationSec),
			strconv.Itoa(c.QueueDurationSec),
			hangup,
		})
	}
	cw.Flush()
}

func (h *CallHandler) ListCursor(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if limit <= 0 || limit > 100 {
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

	calls, err := h.callSvc.ListCallsWithCursor(r.Context(), tenantID, filter, cursor, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var nextCursor int64
	if len(calls) == limit {
		nextCursor = calls[len(calls)-1].ID
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": calls, "next_cursor": nextCursor})
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
	tenantID := middleware.TenantIDFromCtx(r.Context())
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
	tenantID := middleware.TenantIDFromCtx(r.Context())
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
	tenantID := middleware.TenantIDFromCtx(r.Context())
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
