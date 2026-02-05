package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jacadi/audio"
	"jacadi/config"
	"jacadi/handlers"
	"jacadi/tts"
)

var startTime = time.Now()

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting audio playback server")

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "routes.json"
	}

	host := config.GetEnv("HOST", "0.0.0.0")
	port := config.GetEnvInt("PORT", 8080)

	deviceConfig, err := config.LoadDeviceConfig(configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err, "path", configPath)
		os.Exit(1)
	}

	extraPath := os.Getenv("EXTRA_ROUTES_PATH")
	if extraPath != "" {
		if _, err := os.Stat(extraPath); err == nil {
			extraConfig, err := config.LoadDeviceConfig(extraPath)
			if err != nil {
				logger.Warn("failed to load extra config", "error", err, "path", extraPath)
			} else {
				for deviceName, device := range extraConfig {
					for audioName, cmd := range device.Commands {
						cmd.IsExtra = true
						device.Commands[audioName] = cmd
					}
					extraConfig[deviceName] = device
				}
				deviceConfig = config.MergeConfigs(deviceConfig, extraConfig)
				logger.Info("loaded extra routes", "path", extraPath)
			}
		}
	}

	logger.Info("configuration loaded",
		"devices", len(deviceConfig),
		"total_commands", deviceConfig.TotalCommands(),
		"audio_base_path", config.GetEnv("AUDIO_BASE_PATH", "/audio"),
		"host", host,
		"port", port,
	)

	player, err := audio.NewAplayPlayer(logger)
	if err != nil {
		logger.Error("failed to initialize audio player", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthCheckHandler(deviceConfig, logger))

	for deviceName, device := range deviceConfig {
		for audioName, cmd := range device.Commands {
			audioPath := config.GetAudioFilePathForCommand(deviceName, audioName, cmd.IsExtra)
			handler := handlers.NewPlaybackHandler(player, audioPath, logger)
			pattern := fmt.Sprintf("POST /play/%s/%s", deviceName, audioName)

			mux.Handle(pattern, handler)

			logger.Info("registered route",
				"pattern", pattern,
				"device", deviceName,
				"audio_name", audioName,
				"text", cmd.Text,
			)
		}
	}

	volumeHandler := handlers.NewVolumeHandler(logger)
	mux.Handle("POST /volume", volumeHandler)
	logger.Info("registered route", "pattern", "POST /volume")

	volumeGetHandler := handlers.NewVolumeGetHandler(logger)
	mux.Handle("GET /volume", volumeGetHandler)
	logger.Info("registered route", "pattern", "GET /volume")

	var speaker *tts.PiperSpeaker
	if config.IsPiperEmbedded() {
		var err error
		speaker, err = tts.NewPiperSpeaker(logger)
		if err != nil {
			logger.Error("failed to initialize TTS speaker", "error", err)
			os.Exit(1)
		}

		ttsHandler := handlers.NewTTSHandler(speaker, logger)
		mux.Handle("POST /play/tts", ttsHandler)
		logger.Info("registered TTS route", "pattern", "POST /play/tts")
	} else {
		logger.Info("TTS endpoint disabled (set PIPER_EMBEDDED=true to enable)")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, starting graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	if err := player.Close(); err != nil {
		logger.Error("error closing audio player", "error", err)
	}

	if speaker != nil {
		if err := speaker.Close(); err != nil {
			logger.Error("error closing TTS speaker", "error", err)
		}
	}

	logger.Info("server shutdown complete")
}

func healthCheckHandler(deviceConfig config.DeviceConfig, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(startTime)

		devices := make(map[string]int)
		for name, device := range deviceConfig {
			devices[name] = len(device.Commands)
		}

		response := map[string]interface{}{
			"status":         "ok",
			"devices":        devices,
			"total_commands": deviceConfig.TotalCommands(),
			"uptime_seconds": int(uptime.Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
