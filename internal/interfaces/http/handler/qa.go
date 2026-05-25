package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/ai"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/redact"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type QAHandler struct {
	svc *ai.QualityInspectionService
}

func NewQAHandler(svc *ai.QualityInspectionService) *QAHandler {
	return &QAHandler{svc: svc}
}

// --- Rules ---

func (h *QAHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in ai.CreateQARuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in.TenantID = tenantID
	rule, err := h.svc.CreateRule(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, rule)
}

func (h *QAHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rule, err := h.svc.GetRule(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *QAHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rule, err := h.svc.GetRule(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	var in struct {
		Name     *string `json:"name"`
		Config   *string `json:"config"`
		Severity *string `json:"severity"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		rule.Name = *in.Name
	}
	if in.Config != nil {
		rule.Config = *in.Config
	}
	if in.Severity != nil {
		rule.Severity = *in.Severity
	}
	if in.IsActive != nil {
		rule.IsActive = *in.IsActive
	}
	if err := h.svc.UpdateRule(r.Context(), rule); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *QAHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.DeleteRule(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

func (h *QAHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	list, err := h.svc.ListRules(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// --- Schemes ---

func (h *QAHandler) CreateScheme(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in ai.CreateQASchemeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in.TenantID = tenantID
	scheme, err := h.svc.CreateScheme(r.Context(), in)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, scheme)
}

func (h *QAHandler) GetScheme(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	scheme, err := h.svc.GetScheme(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, scheme)
}

func (h *QAHandler) UpdateScheme(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	scheme, err := h.svc.GetScheme(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	var in struct {
		Name      *string              `json:"name"`
		RuleIDs   []ai.SchemeRuleWeight `json:"rule_ids"`
		IsDefault *bool                `json:"is_default"`
		IsActive  *bool                `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != nil {
		scheme.Name = *in.Name
	}
	if in.RuleIDs != nil {
		ruleJSON, _ := json.Marshal(in.RuleIDs)
		scheme.RuleIDs = string(ruleJSON)
	}
	if in.IsDefault != nil {
		scheme.IsDefault = *in.IsDefault
	}
	if in.IsActive != nil {
		scheme.IsActive = *in.IsActive
	}
	if err := h.svc.UpdateScheme(r.Context(), scheme); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, scheme)
}

func (h *QAHandler) DeleteScheme(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.DeleteScheme(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

func (h *QAHandler) ListSchemes(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	list, err := h.svc.ListSchemes(r.Context(), tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// --- Execution / Results ---

func (h *QAHandler) RunInspection(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	var in struct {
		CallID     int64  `json:"call_id"`
		SchemeID   int64  `json:"scheme_id"`
		Transcript string `json:"transcript"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.RunInspection(r.Context(), tenantID, in.CallID, in.SchemeID, redact.Text(in.Transcript))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *QAHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	result, err := h.svc.GetResult(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *QAHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	list, err := h.svc.ListResults(r.Context(), tenantID, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list)
}

func (h *QAHandler) Appeal(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Appeal(r.Context(), id, in.Note)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *QAHandler) Review(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := middleware.UserIDFromCtx(r.Context())
	var in struct {
		Note     string  `json:"note"`
		NewScore float64 `json:"new_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Review(r.Context(), id, userID, in.Note, in.NewScore)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}
