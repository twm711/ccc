package handler

import (
	"context"
	"encoding/json"
	"fmt"
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

func (h *AuthHandler) issueTokens(user *identity.User) (accessToken, refreshToken string, err error) {
	now := time.Now()
	accessClaims := jwt.MapClaims{
		"sub":       fmt.Sprintf("%d", user.ID),
		"tenant_id": user.TenantID,
		"user_id":   user.ID,
		"role":      string(user.Role),
		"type":      "access",
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = at.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return
	}

	refreshClaims := jwt.MapClaims{
		"sub":       fmt.Sprintf("%d", user.ID),
		"tenant_id": user.TenantID,
		"user_id":   user.ID,
		"type":      "refresh",
		"iat":       now.Unix(),
		"exp":       now.Add(7 * 24 * time.Hour).Unix(),
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = rt.SignedString([]byte(h.jwtSecret))
	return
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

	accessToken, refreshToken, err := h.issueTokens(user)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
			"tenantId": user.TenantID,
		},
	})
}

// RefreshToken validates a refresh token and issues new access + refresh tokens.
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	token, err := jwt.Parse(in.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		response.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "invalid token claims")
		return
	}
	if claims["type"] != "refresh" {
		response.Error(w, http.StatusUnauthorized, "not a refresh token")
		return
	}

	sub, _ := claims["sub"].(string)
	user, err := h.finder.FindByUsernameGlobal(r.Context(), sub)
	// sub is the user ID string; if FindByUsernameGlobal doesn't find by ID,
	// we construct a minimal user from claims for token re-issue.
	if err != nil || user == nil {
		tenantID, _ := claims["tenant_id"].(float64)
		userID, _ := claims["user_id"].(float64)
		user = &identity.User{
			ID:       int64(userID),
			TenantID: int64(tenantID),
		}
	}

	accessToken, refreshToken, err := h.issueTokens(user)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"token":         accessToken,
		"refresh_token": refreshToken,
	})
}
