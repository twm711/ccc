package handler

import (
	"encoding/json"
	"net/http"

	"github.com/divord97/ccc/internal/application/b2b"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
)

type B2BHandler struct {
	svc *b2b.Service
}

func NewB2BHandler(svc *b2b.Service) *B2BHandler {
	return &B2BHandler{svc: svc}
}

func (h *B2BHandler) Back2BackCall(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		CallerNumber string `json:"caller_number"`
		CalleeNumber string `json:"callee_number"`
		Gateway      string `json:"gateway"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	c, err := h.svc.Back2BackCall(r.Context(), tenantID, in.CallerNumber, in.CalleeNumber, in.Gateway)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *B2BHandler) FlashSMS(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		PhoneNumber string `json:"phone_number"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.FlashSMS(r.Context(), tenantID, in.PhoneNumber, in.Message); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *B2BHandler) EncryptedCall(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		CallerNumber       string `json:"caller_number"`
		CalleeNumber       string `json:"callee_number"`
		IntermediateNumber string `json:"intermediate_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	c, err := h.svc.EncryptedCall(r.Context(), tenantID, in.CallerNumber, in.CalleeNumber, in.IntermediateNumber)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}
