package handler

import (
	"encoding/json"
	"net/http"

	"github.com/divord97/ccc/internal/application/email"
	"github.com/divord97/ccc/pkg/response"
)

type EmailInboundHandler struct {
	svc *email.Service
}

func NewEmailInboundHandler(svc *email.Service) *EmailInboundHandler {
	return &EmailInboundHandler{svc: svc}
}

func (h *EmailInboundHandler) Inbound(w http.ResponseWriter, r *http.Request) {
	var in email.InboundInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sess, err := h.svc.ProcessInbound(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, sess)
}
