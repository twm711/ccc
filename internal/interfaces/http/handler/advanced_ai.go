package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/application/advancedai"
	"github.com/divord97/ccc/internal/domain/ai"
	"github.com/divord97/ccc/internal/interfaces/http/middleware"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

// ─── CommAgent Handlers ───

func ListCommAgents(svc *ai.CommAgentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		items, err := svc.List(r.Context(), tenantID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, items)
	}
}

func CreateCommAgent(svc *ai.CommAgentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var in ai.CreateCommAgentInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		in.TenantID = tenantID
		a, err := svc.Create(r.Context(), in)
		if err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 201, a)
	}
}

func GetCommAgent(svc *ai.CommAgentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		a, err := svc.Get(r.Context(), tenantID, id)
		if err != nil {
			response.Error(w, 404, err.Error())
			return
		}
		response.JSON(w, 200, a)
	}
}

func DeleteCommAgent(svc *ai.CommAgentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err := svc.Delete(r.Context(), tenantID, id); err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		w.WriteHeader(204)
	}
}

// ─── VoiceProfile Handlers ───

func ListVoiceProfiles(svc *ai.VoiceProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		items, err := svc.List(r.Context(), tenantID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, items)
	}
}

func CreateVoiceProfile(svc *ai.VoiceProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var in ai.CreateVoiceProfileInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		in.TenantID = tenantID
		v, err := svc.Create(r.Context(), in)
		if err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 201, v)
	}
}

func GetVoiceProfile(svc *ai.VoiceProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		v, err := svc.Get(r.Context(), tenantID, id)
		if err != nil {
			response.Error(w, 404, err.Error())
			return
		}
		response.JSON(w, 200, v)
	}
}

func StartVoiceTraining(svc *ai.VoiceProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		v, err := svc.StartTraining(r.Context(), tenantID, id)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, v)
	}
}

func DeleteVoiceProfile(svc *ai.VoiceProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err := svc.Delete(r.Context(), tenantID, id); err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		w.WriteHeader(204)
	}
}

// ─── ConversationAnalysis Handlers ───

func ListAnalysisTasks(svc *ai.ConversationAnalysisService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		items, err := svc.List(r.Context(), tenantID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, items)
	}
}

func CreateAnalysisTask(svc *ai.ConversationAnalysisService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var in ai.CreateAnalysisTaskInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		in.TenantID = tenantID
		t, err := svc.Create(r.Context(), in)
		if err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 201, t)
	}
}

func GetAnalysisTask(svc *ai.ConversationAnalysisService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		t, err := svc.Get(r.Context(), tenantID, id)
		if err != nil {
			response.Error(w, 404, err.Error())
			return
		}
		response.JSON(w, 200, t)
	}
}

// ─── Training Handlers ───

func ListCourses(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		items, err := svc.ListCourses(r.Context(), tenantID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, items)
	}
}

func CreateCourse(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var in ai.CreateCourseInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		in.TenantID = tenantID
		c, err := svc.CreateCourse(r.Context(), in)
		if err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 201, c)
	}
}

func GetCourse(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		c, err := svc.GetCourse(r.Context(), tenantID, id)
		if err != nil {
			response.Error(w, 404, err.Error())
			return
		}
		response.JSON(w, 200, c)
	}
}

func PublishCourse(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		c, err := svc.PublishCourse(r.Context(), tenantID, id)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, c)
	}
}

func SubmitExam(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var in ai.SubmitExamInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		in.TenantID = tenantID
		e, err := svc.SubmitExam(r.Context(), in)
		if err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 201, e)
	}
}

func ListExamsByAgent(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentID"), 10, 64)
		items, err := svc.ListExamsByAgent(r.Context(), tenantID, agentID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, items)
	}
}

func CreateSimulatedCall(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var in ai.CreateSimulatedCallInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		in.TenantID = tenantID
		sc, err := svc.CreateSimulatedCall(r.Context(), in)
		if err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 201, sc)
	}
}

func ListSimulatedCalls(svc *ai.TrainingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		agentID, _ := strconv.ParseInt(chi.URLParam(r, "agentID"), 10, 64)
		items, err := svc.ListSimulatedCalls(r.Context(), tenantID, agentID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, items)
	}
}

// ─── ConversationAnalysis Run Handler ───

func RunAnalysisTask(svc *advancedai.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		taskID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		var in struct {
			Transcripts []string `json:"transcripts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		if err := svc.RunConversationAnalysis(r.Context(), tenantID, taskID, in.Transcripts); err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, map[string]string{"status": "completed"})
	}
}

// ─── RingAnalysis Handlers ───

func GetRingAnalysisConfig(svc *ai.RingAnalysisService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		c, err := svc.GetConfig(r.Context(), tenantID)
		if err != nil {
			response.Error(w, 404, err.Error())
			return
		}
		response.JSON(w, 200, c)
	}
}

func UpsertRingAnalysisConfig(svc *ai.RingAnalysisService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var c ai.RingAnalysisConfig
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		c.TenantID = tenantID
		if err := svc.UpsertConfig(r.Context(), &c); err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, c)
	}
}

func GetRingAnalysisLogs(svc *ai.RingAnalysisService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		callID, _ := strconv.ParseInt(chi.URLParam(r, "callID"), 10, 64)
		logs, err := svc.GetCallLogs(r.Context(), tenantID, callID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.JSON(w, 200, logs)
	}
}

// ─── FullDuplex Handlers ───

func GetFullDuplexConfig(svc *ai.FullDuplexService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		c, err := svc.GetConfig(r.Context(), tenantID)
		if err != nil {
			response.Error(w, 404, err.Error())
			return
		}
		response.JSON(w, 200, c)
	}
}

func UpsertFullDuplexConfig(svc *ai.FullDuplexService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.TenantIDFromCtx(r.Context())
		var c ai.FullDuplexConfig
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			response.Error(w, 400, err.Error())
			return
		}
		c.TenantID = tenantID
		if err := svc.UpsertConfig(r.Context(), &c); err != nil {
			response.Error(w, 422, err.Error())
			return
		}
		response.JSON(w, 200, c)
	}
}
