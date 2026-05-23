package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type IMChannelHandler struct {
	svc *im.IMService
}

func NewIMChannelHandler(svc *im.IMService) *IMChannelHandler {
	return &IMChannelHandler{svc: svc}
}

func (h *IMChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TenantID     int64          `json:"tenant_id"`
		ChannelType  im.ChannelType `json:"channel_type"`
		Name         string         `json:"name"`
		SkillGroupID *int64         `json:"skill_group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ch, err := h.svc.CreateChannel(r.Context(), in.TenantID, in.ChannelType, in.Name, in.SkillGroupID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, ch)
}

func (h *IMChannelHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := strconv.ParseInt(r.URL.Query().Get("tenant_id"), 10, 64)
	items, err := h.svc.ListChannels(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *IMChannelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	ch, err := h.svc.GetChannel(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	var in struct {
		Name         *string            `json:"name"`
		Config       *string            `json:"config"`
		SkillGroupID *int64             `json:"skill_group_id"`
		Status       *im.ChannelStatus  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		ch.Name = *in.Name
	}
	if in.Config != nil {
		ch.Config = *in.Config
	}
	if in.SkillGroupID != nil {
		ch.SkillGroupID = in.SkillGroupID
	}
	if in.Status != nil {
		ch.Status = *in.Status
	}
	if err := h.svc.UpdateChannel(r.Context(), ch); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, ch)
}
