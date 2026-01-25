package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"jacadi/tts"
)

type TTSHandler struct {
	speaker tts.Speaker
	logger  *slog.Logger
}

type TTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice,omitempty"`
}

type TTSResponse struct {
	Status    string `json:"status"`
	Voice     string `json:"voice"`
	Timestamp string `json:"timestamp"`
}

func NewTTSHandler(speaker tts.Speaker, logger *slog.Logger) *TTSHandler {
	return &TTSHandler{
		speaker: speaker,
		logger:  logger,
	}
}

func (h *TTSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("invalid request body",
			"error", err,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
		})
		return
	}

	if req.Text == "" {
		h.logger.Error("empty text in request",
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "text is required",
			Message: "text field cannot be empty",
		})
		return
	}

	if err := h.speaker.SpeakAsync(req.Text, req.Voice); err != nil {
		h.logger.Error("TTS failed",
			"error", err,
			"voice", req.Voice,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "TTS failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("TTS started",
		"voice", req.Voice,
		"text_length", len(req.Text),
		"remote_addr", r.RemoteAddr,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TTSResponse{
		Status:    "speaking",
		Voice:     req.Voice,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
