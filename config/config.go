package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type DeviceConfig map[string]Device

type Device struct {
	Volume   *int               `json:"volume,omitempty"`
	Commands map[string]Command `json:"commands"`
}

type Command struct {
	Text    string `json:"text"`
	IsExtra bool   `json:"-"`
}

func LoadDeviceConfig(configPath string) (DeviceConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config DeviceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return config, nil
}

func LoadAllRoutes(routesDir, extraRoutesPath string) (DeviceConfig, error) {
	pattern := filepath.Join(routesDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob routes directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no route files found in %s", routesDir)
	}

	combined := make(DeviceConfig)

	for _, file := range files {
		cfg, err := LoadDeviceConfig(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		combined = MergeConfigs(combined, cfg)
	}

	if extraRoutesPath != "" {
		if _, err := os.Stat(extraRoutesPath); err == nil {
			extraCfg, err := LoadDeviceConfig(extraRoutesPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load extra routes: %w", err)
			}
			for deviceName, device := range extraCfg {
				for cmdName, cmd := range device.Commands {
					cmd.IsExtra = true
					device.Commands[cmdName] = cmd
				}
				extraCfg[deviceName] = device
			}
			combined = MergeConfigs(combined, extraCfg)
		}
	}

	return combined, nil
}

func (c DeviceConfig) Validate() error {
	if len(c) == 0 {
		return fmt.Errorf("no devices configured")
	}

	for deviceName, device := range c {
		if deviceName == "" {
			return fmt.Errorf("device name cannot be empty")
		}

		if len(device.Commands) == 0 {
			return fmt.Errorf("device %s has no commands", deviceName)
		}

		for audioName, cmd := range device.Commands {
			if audioName == "" {
				return fmt.Errorf("device %s: audio_name cannot be empty", deviceName)
			}
			if cmd.Text == "" {
				return fmt.Errorf("device %s: text cannot be empty for command %s", deviceName, audioName)
			}

			audioPath := GetAudioFilePathForCommand(deviceName, audioName, cmd.IsExtra)
			if _, err := os.Stat(audioPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("device %s: audio file not found: %s", deviceName, audioPath)
				}
				return fmt.Errorf("device %s: error checking audio file %s: %w", deviceName, audioPath, err)
			}
		}
	}

	return nil
}

func (c DeviceConfig) TotalCommands() int {
	total := 0
	for _, device := range c {
		total += len(device.Commands)
	}
	return total
}

func MergeConfigs(base, extra DeviceConfig) DeviceConfig {
	for deviceName, device := range extra {
		if existing, ok := base[deviceName]; ok {
			if device.Volume != nil {
				existing.Volume = device.Volume
			}
			for audioName, cmd := range device.Commands {
				existing.Commands[audioName] = cmd
			}
			base[deviceName] = existing
		} else {
			base[deviceName] = device
		}
	}
	return base
}

func GetAudioFilePath(deviceName, audioName string) string {
	return filepath.Join(GetEnv("AUDIO_BASE_PATH", "/audio"), deviceName, audioName+".wav")
}

func GetAudioFilePathForCommand(deviceName, audioName string, isExtra bool) string {
	base := GetEnv("AUDIO_BASE_PATH", "/audio")
	if isExtra {
		return filepath.Join(base, "extra", deviceName, audioName+".wav")
	}
	return filepath.Join(base, deviceName, audioName+".wav")
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func IsPiperEmbedded() bool {
	return GetEnvBool("PIPER_EMBEDDED", false)
}

func GetPiperSampleRate() int {
	return GetEnvInt("PIPER_SAMPLE_RATE", 16000)
}

func GetDefaultVoice() string {
	return GetEnv("VOICE", "en_US-amy-low")
}
