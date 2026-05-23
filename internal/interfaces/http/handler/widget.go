package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type WidgetHandler struct {
	svc *im.IMService
}

func NewWidgetHandler(svc *im.IMService) *WidgetHandler {
	return &WidgetHandler{svc: svc}
}

func (h *WidgetHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var in im.CreateSessionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sess, err := h.svc.CreateSession(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, sess)
}

func (h *WidgetHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		Content     string         `json:"content"`
		ContentType im.ContentType `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.ContentType == "" {
		in.ContentType = im.ContentTypeText
	}
	visitorID := r.URL.Query().Get("visitor_id")
	msg, err := h.svc.SendMessage(r.Context(), id, im.SenderTypeVisitor, visitorID, in.ContentType, in.Content)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, msg)
}
