package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/pkg/response"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserFinder interface {
	FindByUsernameGlobal(ctx context.Context, username string) (*identity.User, error)
}

type AuthHandler struct {
	finder    UserFinder
	jwtSecret string
}

func NewAuthHandler(finder UserFinder, jwtSecret string) *AuthHandler {
	return &AuthHandler{finder: finder, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.Username == "" || in.Password == "" {
		response.Error(w, http.StatusBadRequest, "username and password required")
		return
	}

	user, err := h.finder.FindByUsernameGlobal(r.Context(), in.Username)
	if err != nil || user == nil {
		response.Error(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	if user.PasswordHash == "" {
		response.Error(w, http.StatusUnauthorized, "password not set for this user")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       user.ID,
		"tenant_id": user.TenantID,
		"user_id":   user.ID,
		"role":      string(user.Role),
		"iat":       now.Unix(),
		"exp":       now.Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"token": tokenStr,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
			"tenantId": user.TenantID,
		},
	})
}
