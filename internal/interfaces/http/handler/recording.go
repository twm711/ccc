package handler

import (
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type RecordingHandler struct {
	repo call.RecordingRepository
}

func NewRecordingHandler(repo call.RecordingRepository) *RecordingHandler {
	return &RecordingHandler{repo: repo}
}

func (h *RecordingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	rec, err := h.repo.GetByID(r.Context(), id)
	if err != nil || rec == nil {
		response.Error(w, http.StatusNotFound, "recording not found")
		return
	}
	response.JSON(w, http.StatusOK, rec)
}

func (h *RecordingHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	recs, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": recs, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

func (h *RecordingHandler) Stream(w http.ResponseWriter, r *http.Request) {
	// In production: stream audio file from MinIO/local storage
	response.Error(w, http.StatusNotImplemented, "streaming requires storage integration")
}

func (h *RecordingHandler) Download(w http.ResponseWriter, r *http.Request) {
	// In production: serve file download from MinIO/local storage
	response.Error(w, http.StatusNotImplemented, "download requires storage integration")
}

// VoicemailHandler

type VoicemailHandler struct {
	repo call.VoicemailRepository
}

func NewVoicemailHandler(repo call.VoicemailRepository) *VoicemailHandler {
	return &VoicemailHandler{repo: repo}
}

func (h *VoicemailHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	vm, err := h.repo.GetByID(r.Context(), id)
	if err != nil || vm == nil {
		response.Error(w, http.StatusNotFound, "voicemail not found")
		return
	}
	response.JSON(w, http.StatusOK, vm)
}

func (h *VoicemailHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	p := pagination.Parse(r)
	vms, total, err := h.repo.List(r.Context(), tenantID, p.Offset, p.PageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": vms, "total": total, "page": p.Page, "page_size": p.PageSize,
	})
}

func (h *VoicemailHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	vm, err := h.repo.GetByID(r.Context(), id)
	if err != nil || vm == nil {
		response.Error(w, http.StatusNotFound, "voicemail not found")
		return
	}

	vm.IsRead = true
	if err := h.repo.Update(r.Context(), vm); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, vm)
}

func (h *VoicemailHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
