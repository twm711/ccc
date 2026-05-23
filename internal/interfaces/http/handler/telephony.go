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

// CarrierHandler

type CarrierHandler struct {
	repo telephony.CarrierRepository
}

func NewCarrierHandler(repo telephony.CarrierRepository) *CarrierHandler {
	return &CarrierHandler{repo: repo}
}

func (h *CarrierHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Protocol    string `json:"protocol"`
		Host        string `json:"host"`
		Port        int    `json:"port"`
		MaxChannels int    `json:"max_channels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now()
	c := &telephony.Carrier{
		ID:          snowflake.NextID(),
		TenantID:    middleware.TenantIDFromCtx(r.Context()),
		Name:        req.Name,
		Protocol:    req.Protocol,
		Host:        req.Host,
		Port:        req.Port,
		Status:      "active",
		MaxChannels: req.MaxChannels,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.repo.Create(r.Context(), c); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *CarrierHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	carriers, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": carriers, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

// SIPTrunkHandler

type SIPTrunkHandler struct {
	repo telephony.SIPTrunkRepository
}

func NewSIPTrunkHandler(repo telephony.SIPTrunkRepository) *SIPTrunkHandler {
	return &SIPTrunkHandler{repo: repo}
}

func (h *SIPTrunkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CarrierID   int64  `json:"carrier_id"`
		Name        string `json:"name"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		Domain      string `json:"domain"`
		Transport   string `json:"transport"`
		Codecs      string `json:"codecs"`
		MaxChannels int    `json:"max_channels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now()
	t := &telephony.SIPTrunk{
		ID:          snowflake.NextID(),
		TenantID:    middleware.TenantIDFromCtx(r.Context()),
		CarrierID:   req.CarrierID,
		Name:        req.Name,
		Username:    req.Username,
		Password:    req.Password,
		Domain:      req.Domain,
		Transport:   req.Transport,
		Codecs:      req.Codecs,
		MaxChannels: req.MaxChannels,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.repo.Create(r.Context(), t); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

func (h *SIPTrunkHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.repo.GetByID(r.Context(), id)
	if err != nil || t == nil {
		response.Error(w, http.StatusNotFound, "sip trunk not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *SIPTrunkHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.repo.GetByID(r.Context(), id)
	if err != nil || t == nil {
		response.Error(w, http.StatusNotFound, "sip trunk not found")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Status      *string `json:"status"`
		MaxChannels *int    `json:"max_channels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if req.MaxChannels != nil {
		t.MaxChannels = *req.MaxChannels
	}
	t.UpdatedAt = time.Now()

	if err := h.repo.Update(r.Context(), t); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *SIPTrunkHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	trunks, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": trunks, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

// PhoneNumberHandler

type PhoneNumberHandler struct {
	repo telephony.PhoneNumberRepository
}

func NewPhoneNumberHandler(repo telephony.PhoneNumberRepository) *PhoneNumberHandler {
	return &PhoneNumberHandler{repo: repo}
}

func (h *PhoneNumberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Number       string `json:"number"`
		DisplayName  string `json:"display_name"`
		Usage        string `json:"usage"`
		SIPTrunkID   *int64 `json:"sip_trunk_id"`
		IVRFlowID    *int64 `json:"ivr_flow_id"`
		SkillGroupID *int64 `json:"skill_group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Number == "" {
		response.Error(w, http.StatusBadRequest, "number is required")
		return
	}

	now := time.Now()
	p := &telephony.PhoneNumber{
		ID:           snowflake.NextID(),
		TenantID:     middleware.TenantIDFromCtx(r.Context()),
		Number:       req.Number,
		DisplayName:  req.DisplayName,
		Usage:        req.Usage,
		SIPTrunkID:   req.SIPTrunkID,
		IVRFlowID:    req.IVRFlowID,
		SkillGroupID: req.SkillGroupID,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.repo.Create(r.Context(), p); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, p)
}

func (h *PhoneNumberHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil || p == nil {
		response.Error(w, http.StatusNotFound, "phone number not found")
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *PhoneNumberHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil || p == nil {
		response.Error(w, http.StatusNotFound, "phone number not found")
		return
	}

	var req struct {
		DisplayName  *string `json:"display_name"`
		Usage        *string `json:"usage"`
		IVRFlowID    *int64  `json:"ivr_flow_id"`
		SkillGroupID *int64  `json:"skill_group_id"`
		Status       *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DisplayName != nil {
		p.DisplayName = *req.DisplayName
	}
	if req.Usage != nil {
		p.Usage = *req.Usage
	}
	if req.IVRFlowID != nil {
		p.IVRFlowID = req.IVRFlowID
	}
	if req.SkillGroupID != nil {
		p.SkillGroupID = req.SkillGroupID
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	p.UpdatedAt = time.Now()

	if err := h.repo.Update(r.Context(), p); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *PhoneNumberHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	numbers, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": numbers, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}
