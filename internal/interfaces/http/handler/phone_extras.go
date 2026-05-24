package handler

import (
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/campaign"
	"github.com/divord97/ccc/internal/domain/crm"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type SupervisorHandler struct {
	callSvc *call.CallService
}

func NewSupervisorHandler(callSvc *call.CallService) *SupervisorHandler {
	return &SupervisorHandler{callSvc: callSvc}
}

func (h *SupervisorHandler) ActiveCalls(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	calls, _, err := h.callSvc.ListCalls(r.Context(), tenantID, call.CallListFilter{}, 0, 100)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"items": calls})
}

type ScreenPopHandler struct {
	customerSvc *crm.CustomerService
}

func NewScreenPopHandler(customerSvc *crm.CustomerService) *ScreenPopHandler {
	return &ScreenPopHandler{customerSvc: customerSvc}
}

func (h *ScreenPopHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		response.JSON(w, http.StatusOK, map[string]interface{}{"customer": nil})
		return
	}
	customer, err := h.customerSvc.FindByPhone(r.Context(), tenantID, phone)
	if err != nil || customer == nil {
		response.JSON(w, http.StatusOK, map[string]interface{}{"customer": nil})
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"customer": customer})
}

type PreviewCaseHandler struct {
	campaignSvc *campaign.CampaignService
}

func NewPreviewCaseHandler(campaignSvc *campaign.CampaignService) *PreviewCaseHandler {
	return &PreviewCaseHandler{campaignSvc: campaignSvc}
}

func (h *PreviewCaseHandler) Current(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	campaigns, _, err := h.campaignSvc.List(r.Context(), tenantID, 0, 50)
	if err != nil || len(campaigns) == 0 {
		response.JSON(w, http.StatusOK, map[string]interface{}{"case": nil})
		return
	}
	for _, c := range campaigns {
		if c.Status != campaign.CampaignStatusRunning || c.DialingMode != campaign.DialingModePreview {
			continue
		}
		next, err := h.campaignSvc.GetNextCase(r.Context(), c.ID)
		if err != nil || next == nil {
			continue
		}
		response.JSON(w, http.StatusOK, map[string]interface{}{"case": next, "campaign": c})
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"case": nil})
}

func (h *PreviewCaseHandler) DialCase(w http.ResponseWriter, r *http.Request) {
	campaignID, _ := strconv.ParseInt(chi.URLParam(r, "campaignId"), 10, 64)
	caseID, _ := strconv.ParseInt(chi.URLParam(r, "caseId"), 10, 64)
	if err := h.campaignSvc.DialCase(r.Context(), campaignID, caseID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "dialing"})
}

func (h *PreviewCaseHandler) SkipCase(w http.ResponseWriter, r *http.Request) {
	campaignID, _ := strconv.ParseInt(chi.URLParam(r, "campaignId"), 10, 64)
	caseID, _ := strconv.ParseInt(chi.URLParam(r, "caseId"), 10, 64)
	if err := h.campaignSvc.SkipCase(r.Context(), campaignID, caseID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "skipped"})
}
