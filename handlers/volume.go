package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"jacadi/config"
)

func getALSACard() string {
	audiodev := config.GetEnv("AUDIODEV", "")
	if audiodev == "" {
		return "0"
	}
	re := regexp.MustCompile(`^(?:plug)?hw:(\d+)`)
	if matches := re.FindStringSubmatch(audiodev); len(matches) > 1 {
		return matches[1]
	}
	return "0"
}

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

	card := getALSACard()
	control := config.GetEnv("ALSA_CONTROL", "PCM")

	cmd := exec.Command("amixer", "-c", card, "sset", control, fmt.Sprintf("%d%%", volume))
	if output, err := cmd.CombinedOutput(); err != nil {
		h.logger.Error("amixer failed",
			"error", err,
			"output", string(output),
			"card", card,
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
		"card", card,
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
	card := getALSACard()
	control := config.GetEnv("ALSA_CONTROL", "PCM")

	cmd := exec.Command("amixer", "-c", card, "sget", control)
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.logger.Error("amixer get failed",
			"error", err,
			"output", string(output),
			"card", card,
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

	re := regexp.MustCompile(`\[(\d+)%\]`)
	matches := re.FindStringSubmatch(string(output))
	volume := 0
	if len(matches) > 1 {
		volume, _ = strconv.Atoi(matches[1])
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VolumeResponse{
		Status:    "ok",
		Volume:    volume,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
