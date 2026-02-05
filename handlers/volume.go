package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"jacadi/audio"
)

type VolumeHandler struct {
	logger *slog.Logger
}

type VolumeRequest struct {
	Volume int `json:"volume"`
}

type VolumeResponse struct {
	Status    string `json:"status"`
	Volume    int    `json:"volume"`
	Timestamp string `json:"timestamp"`
}

func NewVolumeHandler(logger *slog.Logger) *VolumeHandler {
	return &VolumeHandler{
		logger: logger,
	}
}

func (h *VolumeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req VolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("invalid request body", "error", err, "remote_addr", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid request",
			Message: "expected JSON with 'volume' field",
		})
		return
	}

	volume := req.Volume
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}

	if err := audio.SetVolume(volume); err != nil {
		h.logger.Error("volume set failed",
			"error", err,
			"volume", volume,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "volume set failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("volume set",
		"volume", volume,
		"remote_addr", r.RemoteAddr,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VolumeResponse{
		Status:    "ok",
		Volume:    volume,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

type VolumeGetHandler struct {
	logger *slog.Logger
}

func NewVolumeGetHandler(logger *slog.Logger) *VolumeGetHandler {
	return &VolumeGetHandler{
		logger: logger,
	}
}

func (h *VolumeGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	volume, err := audio.GetVolume()
	if err != nil {
		h.logger.Error("volume get failed",
			"error", err,
			"remote_addr", r.RemoteAddr,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "volume get failed",
			Message: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VolumeResponse{
		Status:    "ok",
		Volume:    volume,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
