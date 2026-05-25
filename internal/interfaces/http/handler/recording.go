package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/pagination"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

// RecordingStore is the subset of the object storage client RecordingHandler needs.
type RecordingStore interface {
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	GetPresignedURL(ctx context.Context, objectName string, expirySec int) (string, error)
}

type RecordingHandler struct {
	repo     call.RecordingRepository
	store    RecordingStore
	auditLog call.RecordingAccessLogger
}

func NewRecordingHandler(repo call.RecordingRepository) *RecordingHandler {
	return &RecordingHandler{repo: repo}
}

// SetAccessLogger wires an audit logger for recording stream/download access.
func (h *RecordingHandler) SetAccessLogger(l call.RecordingAccessLogger) {
	h.auditLog = l
}

// SetStore wires the object storage backend used by Stream/Download. When the
// store is nil, those endpoints return 501 (preserving previous behaviour).
func (h *RecordingHandler) SetStore(s RecordingStore) {
	h.store = s
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
	rec, ok := h.loadForServing(w, r)
	if !ok {
		return
	}
	h.logAccess(r, rec.ID, "stream")
	h.serveObject(w, r, rec, false)
}

func (h *RecordingHandler) Download(w http.ResponseWriter, r *http.Request) {
	rec, ok := h.loadForServing(w, r)
	if !ok {
		return
	}
	h.logAccess(r, rec.ID, "download")
	h.serveObject(w, r, rec, true)
}

func (h *RecordingHandler) loadForServing(w http.ResponseWriter, r *http.Request) (*call.Recording, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return nil, false
	}
	rec, err := h.repo.GetByID(r.Context(), id)
	if err != nil || rec == nil {
		response.Error(w, http.StatusNotFound, "recording not found")
		return nil, false
	}
	if h.store == nil {
		response.Error(w, http.StatusNotImplemented, "object storage not configured")
		return nil, false
	}
	return rec, true
}

func (h *RecordingHandler) serveObject(w http.ResponseWriter, r *http.Request, rec *call.Recording, asAttachment bool) {
	object := strings.TrimPrefix(rec.FilePath, "/")
	rc, err := h.store.Download(r.Context(), object)
	if err != nil {
		response.Error(w, http.StatusBadGateway, fmt.Sprintf("fetch recording: %v", err))
		return
	}
	defer rc.Close()

	contentType := rec.MimeType
	if contentType == "" {
		contentType = "audio/wav"
	}
	w.Header().Set("Content-Type", contentType)
	if asAttachment {
		name := rec.FileName
		if name == "" {
			name = filepath.Base(object)
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func (h *RecordingHandler) logAccess(r *http.Request, recordingID int64, action string) {
	if h.auditLog == nil {
		return
	}
	userID := middleware.UserIDFromCtx(r.Context())
	tenantID := middleware.TenantIDFromCtx(r.Context())
	go h.auditLog.LogAccess(context.Background(), tenantID, userID, recordingID, action, r.RemoteAddr)
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
