package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"jacadi/audio"
)

type StopHandler struct {
	coordinator *audio.Coordinator
	logger      *slog.Logger
}

func NewStopHandler(coordinator *audio.Coordinator, logger *slog.Logger) *StopHandler {
	return &StopHandler{
		coordinator: coordinator,
		logger:      logger,
	}
}

func (h *StopHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.coordinator.StopFolder()

	h.logger.Info("folder stopped", "remote_addr", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PlaybackResponse{
		Status:    "stopped",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
