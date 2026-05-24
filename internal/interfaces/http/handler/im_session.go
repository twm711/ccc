package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/application/webhook"
	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type IMSessionHandler struct {
	svc         *im.IMService
	customerSvc *crm.CustomerService
	webhookSvc  *webhook.Service
}

func NewIMSessionHandler(svc *im.IMService, customerSvc *crm.CustomerService, webhookSvc *webhook.Service) *IMSessionHandler {
	return &IMSessionHandler{svc: svc, customerSvc: customerSvc, webhookSvc: webhookSvc}
}

func (h *IMSessionHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.URL.Query().Get("tenant_id"), 10, 64)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	items, err := h.svc.ListSessions(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *IMSessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	sess, err := h.svc.GetSession(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, sess)
}

func (h *IMSessionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		ToAgentUserID int64 `json:"to_agent_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.TransferSession(r.Context(), id, in.ToAgentUserID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "transferred"})
}

func (h *IMSessionHandler) Close(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	sess, err := h.svc.CloseSession(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// CRM interaction record
	if h.customerSvc != nil && sess.CustomerID != nil && sess.AgentUserID != nil {
		var durationSec int
		if sess.EndAt != nil {
			durationSec = int(sess.EndAt.Sub(sess.StartAt).Seconds())
		}
		_ = h.customerSvc.RecordInteraction(r.Context(), crm.RecordInteractionInput{
			CustomerID: *sess.CustomerID,
			TenantID:   sess.TenantID,
			Channel:    "im",
			Direction:  "inbound",
			Summary:    fmt.Sprintf("IM session, duration %ds", durationSec),
			AgentName:  fmt.Sprintf("agent_%d", *sess.AgentUserID),
		})
	}

	// Webhook notification
	if h.webhookSvc != nil {
		h.webhookSvc.Deliver(r.Context(), webhook.Event{
			TenantID:  sess.TenantID,
			Type:      "im.session.closed",
			Payload:   sess,
			Timestamp: time.Now(),
		})
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (h *IMSessionHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	items, err := h.svc.ListMessages(r.Context(), id, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *IMSessionHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		SenderType  im.SenderType  `json:"sender_type"`
		SenderID    string         `json:"sender_id"`
		ContentType im.ContentType `json:"content_type"`
		Content     string         `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	msg, err := h.svc.SendMessage(r.Context(), id, in.SenderType, in.SenderID, in.ContentType, in.Content)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, msg)
}
