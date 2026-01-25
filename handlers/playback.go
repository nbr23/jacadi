package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"jacadi/audio"
)

type PlaybackHandler struct {
	player    audio.Player
	audioPath string
	logger    *slog.Logger
}

type PlaybackResponse struct {
	Status    string `json:"status"`
	File      string `json:"file"`
	Timestamp string `json:"timestamp"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	File    string `json:"file,omitempty"`
}

func NewPlaybackHandler(player audio.Player, audioPath string, logger *slog.Logger) *PlaybackHandler {
	return &PlaybackHandler{
		player:    player,
		audioPath: audioPath,
		logger:    logger,
	}
}

func (h *PlaybackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(h.audioPath)

	if _, err := os.Stat(h.audioPath); err != nil {
		if os.IsNotExist(err) {
			h.logger.Error("audio file not found",
				"path", h.audioPath,
				"file", filename,
				"remote_addr", r.RemoteAddr,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "audio file not found",
				File:  filename,
			})
			return
		}

		h.logger.Error("error checking audio file",
			"error", err,
			"path", h.audioPath,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal server error",
			Message: "failed to access audio file",
		})
		return
	}

	if err := h.player.PlayAsync(h.audioPath); err != nil {
		h.logger.Error("playback failed",
			"error", err,
			"path", h.audioPath,
			"file", filename,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "playback failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("audio playback started",
		"path", r.URL.Path,
		"file", filename,
		"remote_addr", r.RemoteAddr,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PlaybackResponse{
		Status:    "playing",
		File:      filename,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
