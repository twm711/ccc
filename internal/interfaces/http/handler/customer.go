package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	svc *crm.CustomerService
}

func NewCustomerHandler(svc *crm.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TenantID   int64            `json:"tenant_id"`
		Name       string           `json:"name"`
		Email      string           `json:"email"`
		Company    string           `json:"company"`
		Level      string           `json:"level"`
		CustomData string           `json:"custom_data"`
		Phones     []crm.PhoneInput `json:"phones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	c, err := h.svc.Create(r.Context(), crm.CreateCustomerInput{
		TenantID:   in.TenantID,
		Name:       in.Name,
		Email:      in.Email,
		Company:    in.Company,
		Level:      in.Level,
		CustomData: in.CustomData,
		Phones:     in.Phones,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	items, err := h.svc.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *CustomerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	var in struct {
		Name    *string `json:"name"`
		Email   *string `json:"email"`
		Company *string `json:"company"`
		Level   *string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.Email != nil {
		c.Email = *in.Email
	}
	if in.Company != nil {
		c.Company = *in.Company
	}
	if in.Level != nil {
		c.Level = *in.Level
	}
	if err := h.svc.Update(r.Context(), c); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

func (h *CustomerHandler) FindByPhone(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	phone := chi.URLParam(r, "phone")

	c, err := h.svc.FindByPhone(r.Context(), tenantID, phone)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if c == nil {
		response.Error(w, http.StatusNotFound, "customer not found")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CustomerHandler) Import(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Records []crm.CreateCustomerInput `json:"records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.svc.BatchImport(r.Context(), in.Records)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListInteractions(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	items, err := h.svc.ListInteractions(r.Context(), id, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *CustomerHandler) ListFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		entityType = "customer"
	}

	fields, err := h.svc.ListFieldDefinitions(r.Context(), tenantID, entityType)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": fields})
}

func (h *CustomerHandler) CreateFieldDefinition(w http.ResponseWriter, r *http.Request) {
	var in crm.CustomFieldDefinition
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.CreateFieldDefinition(r.Context(), in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
