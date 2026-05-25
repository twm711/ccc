package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/application/dialer"
	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type CampaignHandler struct {
	svc       *campaign.CampaignService
	dialerSvc *dialer.Service
}

func NewCampaignHandler(svc *campaign.CampaignService, dialerSvc *dialer.Service) *CampaignHandler {
	return &CampaignHandler{svc: svc, dialerSvc: dialerSvc}
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		Name         string               `json:"name"`
		DialingMode  campaign.DialingMode `json:"dialing_mode"`
		SkillGroupID int64                `json:"skill_group_id"`
		CLIPolicyID  *int64               `json:"cli_policy_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	c, err := h.svc.Create(r.Context(), campaign.CreateCampaignInput{
		TenantID:     tenantID,
		Name:         in.Name,
		DialingMode:  in.DialingMode,
		SkillGroupID: in.SkillGroupID,
		CLIPolicyID:  in.CLIPolicyID,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	items, total, err := h.svc.List(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	var in struct {
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		c.Name = *in.Name
	}
	if err := h.svc.Update(r.Context(), c); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *CampaignHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.Start(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = h.dialerSvc.StartDialing(r.Context(), id)
	response.JSON(w, http.StatusOK, c)
}

func (h *CampaignHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.Pause(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	h.dialerSvc.StopDialing(id)
	response.JSON(w, http.StatusOK, c)
}

func (h *CampaignHandler) Abort(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, err := h.svc.Abort(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	h.dialerSvc.StopDialing(id)
	response.JSON(w, http.StatusOK, c)
}

func (h *CampaignHandler) ImportCases(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var in struct {
		Cases []campaign.CaseInput `json:"cases"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.ImportCases(r.Context(), id, in.Cases); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *CampaignHandler) ListCases(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	limit, offset := pagination.ParseLimitOffset(r, 20, 200)

	items, total, err := h.svc.ListCases(r.Context(), id, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *CampaignHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	d := h.dialerSvc.GetStats(id)

	out := map[string]interface{}{
		"campaign_id":     d.CampaignID,
		"mode":            d.Mode,
		"active_calls":    d.ActiveCalls,
		"total_dialed":    d.TotalDialed,
		"connected_calls": d.ConnectedCalls,
		"connected":       d.ConnectedCalls, // alias
		"abandon_count":   d.AbandonCount,
		"abandon_rate":    d.AbandonRate / 100.0, // expose as 0..1 fraction
		"connect_rate":    d.ConnectRate / 100.0, // expose as 0..1 fraction
		"is_running":      d.IsRunning,
		"started_at":      d.StartedAt,
		"uptime_seconds":  d.UptimeSeconds,
	}
	if c, err := h.svc.GetByID(r.Context(), id); err == nil && c != nil {
		out["total_cases"] = c.TotalCases
		out["completed"] = c.CompletedCases
		out["success"] = c.SuccessCases
		out["failed"] = c.FailedCases
		out["concurrent"] = c.ConcurrentLimit
		if d.UptimeSeconds > 0 {
			out["elapsed_min"] = d.UptimeSeconds / 60
		} else {
			out["elapsed_min"] = 0
		}
	}
	response.JSON(w, http.StatusOK, out)
}
