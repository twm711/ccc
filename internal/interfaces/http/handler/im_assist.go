package handler

import (
	"encoding/json"
	"net/http"

	"github.com/divord97/ccc/internal/application/imassist"
	"github.com/divord97/ccc/pkg/response"
)

type IMAssistHandler struct {
	svc *imassist.Service
}

func NewIMAssistHandler(svc *imassist.Service) *IMAssistHandler {
	return &IMAssistHandler{svc: svc}
}

func (h *IMAssistHandler) Correct(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Correct(r.Context(), in.Text)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *IMAssistHandler) Expand(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Expand(r.Context(), in.Text)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *IMAssistHandler) Optimize(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Optimize(r.Context(), in.Text)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}
