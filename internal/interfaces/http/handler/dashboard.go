package handler

import (
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/report"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
)

type DashboardHandler struct {
	svc *report.DashboardService
}

func NewDashboardHandler(svc *report.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	overview, err := h.svc.GetOverview(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, overview)
}

func (h *DashboardHandler) AgentStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	list, err := h.svc.GetAgentStatusList(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list)
}

func (h *DashboardHandler) SkillGroupStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	list, err := h.svc.GetAgentStatusList(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list)
}

func (h *DashboardHandler) CallTrend(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	interval, _ := strconv.Atoi(r.URL.Query().Get("interval"))
	if interval == 0 {
		interval = 30
	}
	trends, err := h.svc.GetCallTrend(r.Context(), tenantID, interval)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, trends)
}

func (h *DashboardHandler) CallFunnel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	funnel, err := h.svc.GetCallFunnel(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, funnel)
}
