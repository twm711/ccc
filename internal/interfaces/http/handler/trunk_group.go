package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/domain/telephony"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/go-chi/chi/v5"
)

type TrunkGroupHandler struct {
	groups telephony.SIPTrunkGroupRepository
}

func NewTrunkGroupHandler(groups telephony.SIPTrunkGroupRepository) *TrunkGroupHandler {
	return &TrunkGroupHandler{groups: groups}
}

func (h *TrunkGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var g telephony.SIPTrunkGroup
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	g.ID = snowflake.NextID()
	g.CreatedAt = time.Now()
	g.UpdatedAt = g.CreatedAt

	if err := h.groups.Create(r.Context(), &g); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, g)
}

func (h *TrunkGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	items, total, err := h.groups.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *TrunkGroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var in struct {
		SIPTrunkID int64 `json:"sip_trunk_id"`
		Priority   int   `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	m := &telephony.SIPTrunkGroupMember{
		ID:         snowflake.NextID(),
		GroupID:    groupID,
		SIPTrunkID: in.SIPTrunkID,
		Priority:   in.Priority,
	}
	if err := h.groups.AddMember(r.Context(), m); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, m)
}

func (h *TrunkGroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	members, err := h.groups.ListMembers(r.Context(), groupID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, members)
}
