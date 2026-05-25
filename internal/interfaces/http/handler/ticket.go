package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/ticket"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/bizlog"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type TicketHandler struct {
	svc     *ticket.TicketService
	tmplSvc *ticket.TicketTemplateService
	logger  zerolog.Logger
}

func NewTicketHandler(svc *ticket.TicketService, tmplSvc *ticket.TicketTemplateService, logger zerolog.Logger) *TicketHandler {
	return &TicketHandler{svc: svc, tmplSvc: tmplSvc, logger: logger}
}

func (h *TicketHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TenantID int64  `json:"tenant_id"`
		Name     string `json:"name"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := h.tmplSvc.CreateCategory(r.Context(), in.TenantID, in.Name, in.ParentID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *TicketHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.tmplSvc.ListCategories(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *TicketHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var in ticket.CreateTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.tmplSvc.Create(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

func (h *TicketHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.tmplSvc.ListTemplates(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *TicketHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, err := h.tmplSvc.GetTemplate(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "template not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *TicketHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, err := h.tmplSvc.GetTemplate(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	var in struct {
		Name      *string `json:"name"`
		Fields    *string `json:"fields"`
		FlowGraph *string `json:"flow_graph"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		t.Name = *in.Name
	}
	if in.Fields != nil {
		t.Fields = *in.Fields
	}
	if in.FlowGraph != nil {
		t.FlowGraph = *in.FlowGraph
	}
	if err := h.tmplSvc.UpdateTemplate(r.Context(), t); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *TicketHandler) PublishTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, err := h.tmplSvc.Publish(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *TicketHandler) OfflineTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, err := h.tmplSvc.Offline(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in ticket.CreateTicketInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.TenantID == 0 {
		in.TenantID = middleware.TenantIDFromCtx(r.Context())
	}
	tk, err := h.svc.Create(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	bizlog.TicketEvent(h.logger, tk.TenantID, tk.ID, "ticket.created").
		Str("priority", tk.Priority).Msg("ticket created")
	response.JSON(w, http.StatusCreated, tk)
}

func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)
	items, err := h.svc.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *TicketHandler) ListByCall(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	callID, _ := strconv.ParseInt(chi.URLParam(r, "callId"), 10, 64)
	items, err := h.svc.ListByCallID(r.Context(), callID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	scoped := items[:0]
	for _, t := range items {
		if t.TenantID == tenantID {
			scoped = append(scoped, t)
		}
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": scoped})
}

func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tk, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, tk)
}

func (h *TicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tk, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	var in struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Priority    *string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Title != nil {
		tk.Title = *in.Title
	}
	if in.Description != nil {
		tk.Description = *in.Description
	}
	if in.Priority != nil {
		tk.Priority = *in.Priority
	}
	if err := h.svc.Update(r.Context(), tk); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, tk)
}

func (h *TicketHandler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		AgentID int64 `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	tk, err := h.svc.Assign(r.Context(), id, in.AgentID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	bizlog.TicketEvent(h.logger, tk.TenantID, tk.ID, "ticket.assigned").
		Int64("assignee_id", in.AgentID).Msg("ticket assigned")
	response.JSON(w, http.StatusOK, tk)
}

func (h *TicketHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		AuthorID int64  `json:"author_id"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.AddComment(r.Context(), id, in.AuthorID, in.Content); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	bizlog.TicketEvent(h.logger, middleware.TenantIDFromCtx(r.Context()), id, "ticket.comment_added").
		Int64("author_id", in.AuthorID).Msg("ticket comment added")
	response.JSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
