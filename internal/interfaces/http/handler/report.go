package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/divord97/ccc/internal/application/export"
	"github.com/divord97/ccc/internal/domain/report"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
)

type ReportHandler struct {
	svc *report.ReportService
}

func NewReportHandler(svc *report.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func parseReportFilter(r *http.Request) report.ReportFilter {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	start, _ := time.Parse("2006-01-02", r.URL.Query().Get("start"))
	end, _ := time.Parse("2006-01-02", r.URL.Query().Get("end"))
	if start.IsZero() {
		start = time.Now().Truncate(24 * time.Hour)
	}
	if end.IsZero() {
		end = start.Add(24 * time.Hour)
	}

	f := report.ReportFilter{
		TenantID:  tenantID,
		StartTime: start,
		EndTime:   end,
		Offset:    offset,
		Limit:     limit,
	}

	if agentIDStr := r.URL.Query().Get("agent_id"); agentIDStr != "" {
		id, _ := strconv.ParseInt(agentIDStr, 10, 64)
		f.AgentID = &id
	}
	if sgIDStr := r.URL.Query().Get("skill_group_id"); sgIDStr != "" {
		id, _ := strconv.ParseInt(sgIDStr, 10, 64)
		f.SkillGroupID = &id
	}
	return f
}

func (h *ReportHandler) AgentReport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	items, total, err := h.svc.AgentReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *ReportHandler) AgentReportExport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	f.Limit = 10000
	items, _, err := h.svc.AgentReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=agent_report.csv")
	_ = export.WriteAgentReportCSV(w, items)
}

func (h *ReportHandler) GroupAgentReport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	items, total, err := h.svc.GroupAgentReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *ReportHandler) GroupAgentReportExport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	f.Limit = 10000
	items, _, err := h.svc.GroupAgentReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=group_agent_report.csv")
	_ = export.WriteGroupAgentReportCSV(w, items)
}

func (h *ReportHandler) SkillGroupReport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	items, total, err := h.svc.SkillGroupReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *ReportHandler) SkillGroupReportExport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	f.Limit = 10000
	items, _, err := h.svc.SkillGroupReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=skill_group_report.csv")
	_ = export.WriteSkillGroupReportCSV(w, items)
}

func (h *ReportHandler) Back2BackReport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	result, err := h.svc.Back2BackReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *ReportHandler) InternalCallReport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	result, err := h.svc.InternalCallReport(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *ReportHandler) AgentStatusLog(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	breakReason := r.URL.Query().Get("break_reason_code")
	items, total, err := h.svc.AgentStatusLogQuery(r.Context(), f, breakReason)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total})
}

func (h *ReportHandler) AgentStatusLogExport(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	f.Limit = 10000
	breakReason := r.URL.Query().Get("break_reason_code")
	items, _, err := h.svc.AgentStatusLogQuery(r.Context(), f, breakReason)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=agent_status_log.csv")
	_ = export.WriteAgentStatusLogCSV(w, items)
}
