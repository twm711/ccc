package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/routing"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type IVRFlowHandler struct {
	svc *routing.IVRFlowService
}

func NewIVRFlowHandler(svc *routing.IVRFlowService) *IVRFlowHandler {
	return &IVRFlowHandler{svc: svc}
}

func (h *IVRFlowHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code     string          `json:"code"`
		Name     string          `json:"name"`
		FlowType string          `json:"flow_type"`
		Graph    json.RawMessage `json:"graph"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Code == "" || req.Name == "" || len(req.Graph) == 0 {
		response.Error(w, http.StatusBadRequest, "code, name, and graph are required")
		return
	}

	tenantID := middleware.TenantIDFromCtx(r.Context())
	flow, err := h.svc.Create(r.Context(), routing.CreateFlowInput{
		TenantID: tenantID,
		Code:     req.Code,
		Name:     req.Name,
		FlowType: routing.FlowType(req.FlowType),
		Graph:    req.Graph,
	})
	if err != nil {
		handleRoutingError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, flow)
}

func (h *IVRFlowHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	flow, err := h.svc.GetByID(r.Context(), id)
	if err != nil || flow == nil {
		response.Error(w, http.StatusNotFound, "flow not found")
		return
	}
	response.JSON(w, http.StatusOK, flow)
}

func (h *IVRFlowHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	flows, total, err := h.svc.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": flows, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

func (h *IVRFlowHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	flow, err := h.svc.GetByID(r.Context(), id)
	if err != nil || flow == nil {
		response.Error(w, http.StatusNotFound, "flow not found")
		return
	}

	var req struct {
		Name  string          `json:"name"`
		Graph json.RawMessage `json:"graph"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// In a full implementation, update fields on the flow and save
	response.JSON(w, http.StatusOK, flow)
}

func (h *IVRFlowHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	userID := middleware.UserIDFromCtx(r.Context())
	flow, err := h.svc.Publish(r.Context(), id, userID)
	if err != nil {
		handleRoutingError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flow)
}

func (h *IVRFlowHandler) Lock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	userID := middleware.UserIDFromCtx(r.Context())
	flow, err := h.svc.Lock(r.Context(), id, userID)
	if err != nil {
		handleRoutingError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flow)
}

func (h *IVRFlowHandler) Unlock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	userID := middleware.UserIDFromCtx(r.Context())
	flow, err := h.svc.Unlock(r.Context(), id, userID)
	if err != nil {
		handleRoutingError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flow)
}

func (h *IVRFlowHandler) Clone(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		response.Error(w, http.StatusBadRequest, "code and name required")
		return
	}

	flow, err := h.svc.Clone(r.Context(), id, req.Code, req.Name)
	if err != nil {
		handleRoutingError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, flow)
}

func (h *IVRFlowHandler) Versions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	versions, err := h.svc.GetVersions(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, versions)
}

func (h *IVRFlowHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid version")
		return
	}

	flow, err := h.svc.Rollback(r.Context(), id, version)
	if err != nil {
		handleRoutingError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flow)
}

func handleRoutingError(w http.ResponseWriter, err error) {
	switch err {
	case routing.ErrFlowNotFound, routing.ErrVersionNotFound:
		response.Error(w, http.StatusNotFound, err.Error())
	case routing.ErrFlowCodeExists:
		response.Error(w, http.StatusConflict, err.Error())
	case routing.ErrFlowLocked, routing.ErrFlowNotOwner, routing.ErrFlowNotLocked:
		response.Error(w, http.StatusForbidden, err.Error())
	case routing.ErrFlowNotDraft, routing.ErrFlowAlreadyPublished:
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
	case routing.ErrInvalidGraph, routing.ErrNoStartNode, routing.ErrNoEndNode,
		routing.ErrDisconnectedNode, routing.ErrInvalidNodeType:
		response.Error(w, http.StatusBadRequest, err.Error())
	default:
		response.Error(w, http.StatusInternalServerError, err.Error())
	}
}
