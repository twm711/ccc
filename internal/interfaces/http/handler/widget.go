package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

// IMSessionAutoRouter assigns an idle agent to a fresh IM session.
type IMSessionAutoRouter interface {
	AutoRouteSession(ctx context.Context, sess *im.IMSession) error
}

type WidgetHandler struct {
	svc         *im.IMService
	broadcaster IMBroadcaster
	autoRouter  IMSessionAutoRouter
}

func NewWidgetHandler(svc *im.IMService) *WidgetHandler {
	return &WidgetHandler{svc: svc}
}

func (h *WidgetHandler) SetBroadcaster(b IMBroadcaster) {
	h.broadcaster = b
}

func (h *WidgetHandler) SetAutoRouter(r IMSessionAutoRouter) {
	h.autoRouter = r
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
	if h.autoRouter != nil && sess.SkillGroupID != nil {
		go func(s *im.IMSession) {
			_ = h.autoRouter.AutoRouteSession(context.Background(), s)
		}(sess)
	}
	response.JSON(w, http.StatusCreated, sess)
}

func (h *WidgetHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	limit, offset := pagination.ParseLimitOffset(r, 50, 200)
	items, err := h.svc.ListMessages(r.Context(), id, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
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
	if h.broadcaster != nil {
		h.broadcaster.BroadcastEvent(id, "message.new", msg)
	}
	response.JSON(w, http.StatusCreated, msg)
}
