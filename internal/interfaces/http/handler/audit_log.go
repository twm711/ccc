package handler

import (
	"net/http"

	"github.com/divord97/ccc/internal/domain/platform"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
)

type AuditLogHandler struct {
	repo platform.AuditLogRepository
}

func NewAuditLogHandler(repo platform.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

func (h *AuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	logs, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"items":     logs,
		"total":     total,
		"page":      p.Page,
		"page_size": p.PageSize,
	})
}
