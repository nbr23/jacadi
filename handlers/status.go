package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"jacadi/audio"
)

type StatusHandler struct {
	coordinator *audio.Coordinator
	logger      *slog.Logger
}

type StatusResponse struct {
	Playing   bool   `json:"playing"`
	File      bool   `json:"file"`
	Folder    bool   `json:"folder"`
	Timestamp string `json:"timestamp"`
}

func NewStatusHandler(coordinator *audio.Coordinator, logger *slog.Logger) *StatusHandler {
	return &StatusHandler{
		coordinator: coordinator,
		logger:      logger,
	}
}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := h.coordinator.Status()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(StatusResponse{
		Playing:   status.Playing,
		File:      status.File,
		Folder:    status.Folder,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
