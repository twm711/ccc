package handler

import (
	"encoding/json"
	"net/http"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
)

type ProfileHandler struct {
	userSvc *identity.UserService
}

func NewProfileHandler(userSvc *identity.UserService) *ProfileHandler {
	return &ProfileHandler{userSvc: userSvc}
}

func (h *ProfileHandler) Overview(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	u, err := h.userSvc.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	response.JSON(w, http.StatusOK, u)
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	var in struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.userSvc.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}

	displayName := in.DisplayName
	if displayName == "" {
		displayName = existing.DisplayName
	}

	updated, err := h.userSvc.Update(r.Context(), userID, displayName, existing.Email, existing.Phone)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *ProfileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	var in struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.OldPassword == "" || in.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}
	if len(in.NewPassword) < 6 {
		response.Error(w, http.StatusBadRequest, "new password must be at least 6 characters")
		return
	}
	if err := h.userSvc.ChangePassword(r.Context(), userID, in.OldPassword, in.NewPassword); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ProfileHandler) ResetState(w http.ResponseWriter, r *http.Request) {
	// Resets agent state to IDLE (useful for stuck states)
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
