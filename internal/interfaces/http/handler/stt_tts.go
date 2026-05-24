package handler

import (
	"encoding/json"
	"net/http"

	"github.com/divord97/ccc/internal/infrastructure/llm"
	"github.com/divord97/ccc/pkg/response"
)

// STTHandler provides a REST endpoint for speech-to-text transcription.
type STTHandler struct {
	provider llm.ASRProvider
}

func NewSTTHandler(provider llm.ASRProvider) *STTHandler {
	return &STTHandler{provider: provider}
}

func (h *STTHandler) Transcribe(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		response.Error(w, http.StatusServiceUnavailable, "STT provider not configured")
		return
	}
	var in struct {
		AudioURL string `json:"audio_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.AudioURL == "" {
		response.Error(w, http.StatusBadRequest, "audio_url is required")
		return
	}
	text, err := h.provider.Transcribe(r.Context(), in.AudioURL)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"text": text})
}

// TTSHandler provides a REST endpoint for text-to-speech synthesis.
type TTSHandler struct {
	provider llm.TTSProvider
}

func NewTTSHandler(provider llm.TTSProvider) *TTSHandler {
	return &TTSHandler{provider: provider}
}

func (h *TTSHandler) Synthesize(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		response.Error(w, http.StatusServiceUnavailable, "TTS provider not configured")
		return
	}
	var in struct {
		Text  string `json:"text"`
		Voice string `json:"voice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Text == "" {
		response.Error(w, http.StatusBadRequest, "text is required")
		return
	}
	audio, err := h.provider.Synthesize(r.Context(), in.Text, in.Voice)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.WriteHeader(http.StatusOK)
	w.Write(audio)
}
