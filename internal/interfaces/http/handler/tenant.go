package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type TenantHandler struct {
	svc *identity.TenantService
}

func NewTenantHandler(svc *identity.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.Code == "" || in.Name == "" {
		response.Error(w, http.StatusBadRequest, "code and name are required")
		return
	}

	tenant, err := h.svc.Create(r.Context(), identity.CreateTenantInput{
		Code: in.Code,
		Name: in.Name,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, tenant)
}

func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	tenant, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "tenant not found")
		return
	}
	response.JSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var in struct {
		Name   string               `json:"name"`
		Status identity.TenantStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenant, err := h.svc.Update(r.Context(), id, in.Name, in.Status)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.Parse(r)
	tenants, total, err := h.svc.List(r.Context(), p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list tenants")
		return
	}
	totalPages := int(total) / p.PageSize
	if int(total)%p.PageSize > 0 {
		totalPages++
	}
	response.JSON(w, http.StatusOK, response.PagedData{
		Items:      tenants,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages,
	})
}

func handleDomainError(w http.ResponseWriter, err error) {
	switch err {
	case identity.ErrTenantNotFound, identity.ErrUserNotFound, identity.ErrAgentNotFound, identity.ErrSkillGroupNotFound:
		response.Error(w, http.StatusNotFound, err.Error())
	case identity.ErrTenantCodeExists, identity.ErrUsernameExists, identity.ErrAgentAlreadyExists,
		identity.ErrSkillGroupCodeExists, identity.ErrMemberAlreadyExists:
		response.Error(w, http.StatusConflict, err.Error())
	case identity.ErrMaxAgentsReached:
		response.Error(w, http.StatusForbidden, err.Error())
	case identity.ErrInvalidRoutingPolicy:
		response.Error(w, http.StatusBadRequest, err.Error())
	default:
		response.Error(w, http.StatusInternalServerError, "internal error")
	}
}
