package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	svc *identity.UserService
}

func NewUserHandler(svc *identity.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		Username    string         `json:"username"`
		DisplayName string         `json:"display_name"`
		Email       string         `json:"email"`
		Phone       string         `json:"phone"`
		Role        identity.UserRole `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.Username == "" {
		response.Error(w, http.StatusBadRequest, "username is required")
		return
	}

	user, err := h.svc.Create(r.Context(), identity.CreateUserInput{
		TenantID:    tenantID,
		Username:    in.Username,
		DisplayName: in.DisplayName,
		Email:       in.Email,
		Phone:       in.Phone,
		Role:        in.Role,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.svc.Update(r.Context(), id, in.DisplayName, in.Email, in.Phone)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	users, total, err := h.svc.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	totalPages := int(total) / p.PageSize
	if int(total)%p.PageSize > 0 {
		totalPages++
	}
	response.JSON(w, http.StatusOK, response.PagedData{
		Items:      users,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages,
	})
}
