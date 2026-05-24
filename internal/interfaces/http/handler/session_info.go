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

type SessionInfoHandler struct {
	svc *ai.SessionInfoTemplateService
}

func NewSessionInfoHandler(svc *ai.SessionInfoTemplateService) *SessionInfoHandler {
	return &SessionInfoHandler{svc: svc}
}

func (h *SessionInfoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TenantID  int64  `json:"tenant_id"`
		Name      string `json:"name"`
		Fields    string `json:"fields"`
		IsDefault bool   `json:"is_default"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.svc.Create(r.Context(), in.TenantID, in.Name, in.Fields, in.IsDefault)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

func (h *SessionInfoHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	items, err := h.svc.List(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *SessionInfoHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	var in struct {
		Name      *string `json:"name"`
		Fields    *string `json:"fields"`
		IsDefault *bool   `json:"is_default"`
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
	if in.IsDefault != nil {
		t.IsDefault = *in.IsDefault
	}
	if err := h.svc.Update(r.Context(), t); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}
