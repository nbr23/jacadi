package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	writeFile(t, path, string(b))
}

func intPtr(v int) *int { return &v }

func TestParseDeviceConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		data := []byte(`{"dreame":{"volume":80,"commands":{"ok":{"text":"Okay"}}}}`)
		cfg, err := ParseDeviceConfig(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		dev, ok := cfg["dreame"]
		if !ok {
			t.Fatal("dreame missing")
		}
		if dev.Volume == nil || *dev.Volume != 80 {
			t.Fatalf("volume: got %v, want 80", dev.Volume)
		}
		if dev.Commands["ok"].Text != "Okay" {
			t.Fatalf("text: got %q", dev.Commands["ok"].Text)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		if _, err := ParseDeviceConfig([]byte(`{not json`)); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no volume defaults to nil", func(t *testing.T) {
		cfg, err := ParseDeviceConfig([]byte(`{"d":{"commands":{"a":{"text":"A"}}}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg["d"].Volume != nil {
			t.Fatalf("volume: got %v, want nil", *cfg["d"].Volume)
		}
	})
}

func TestMergeConfigs(t *testing.T) {
	t.Run("extra adds new device", func(t *testing.T) {
		base := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "X"}}}}
		extra := DeviceConfig{"b": {Commands: map[string]Command{"y": {Text: "Y"}}}}
		merged := MergeConfigs(base, extra)
		if _, ok := merged["a"]; !ok {
			t.Fatal("device a missing")
		}
		if _, ok := merged["b"]; !ok {
			t.Fatal("device b missing")
		}
	})

	t.Run("extra adds command to existing device", func(t *testing.T) {
		base := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "X"}}}}
		extra := DeviceConfig{"a": {Commands: map[string]Command{"y": {Text: "Y"}}}}
		merged := MergeConfigs(base, extra)
		if len(merged["a"].Commands) != 2 {
			t.Fatalf("want 2 commands, got %d", len(merged["a"].Commands))
		}
	})

	t.Run("extra overrides command", func(t *testing.T) {
		base := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "old"}}}}
		extra := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "new"}}}}
		merged := MergeConfigs(base, extra)
		if merged["a"].Commands["x"].Text != "new" {
			t.Fatalf("want new, got %q", merged["a"].Commands["x"].Text)
		}
	})

	t.Run("extra volume overrides base volume", func(t *testing.T) {
		base := DeviceConfig{"a": {Volume: intPtr(50), Commands: map[string]Command{"x": {Text: "X"}}}}
		extra := DeviceConfig{"a": {Volume: intPtr(90), Commands: map[string]Command{}}}
		merged := MergeConfigs(base, extra)
		if merged["a"].Volume == nil || *merged["a"].Volume != 90 {
			t.Fatalf("want 90, got %v", merged["a"].Volume)
		}
	})

	t.Run("nil extra volume preserves base volume", func(t *testing.T) {
		base := DeviceConfig{"a": {Volume: intPtr(50), Commands: map[string]Command{"x": {Text: "X"}}}}
		extra := DeviceConfig{"a": {Commands: map[string]Command{"y": {Text: "Y"}}}}
		merged := MergeConfigs(base, extra)
		if merged["a"].Volume == nil || *merged["a"].Volume != 50 {
			t.Fatalf("want 50, got %v", merged["a"].Volume)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("empty config fails", func(t *testing.T) {
		empty := DeviceConfig{}
		if err := empty.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("device with no commands fails", func(t *testing.T) {
		cfg := DeviceConfig{"a": {Commands: map[string]Command{}}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty text fails", func(t *testing.T) {
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: ""}}}}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing audio file fails", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "X"}}}}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "audio file not found") {
			t.Fatalf("want 'audio file not found', got %v", err)
		}
	})

	t.Run("valid single file passes", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		writeFile(t, filepath.Join(base, "a", "x.wav"), "fake")
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "X"}}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("folder type with missing dir fails", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"amb": {Text: "A", Type: "folder"}}}}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "folder directory not found") {
			t.Fatalf("want 'folder directory not found', got %v", err)
		}
	})

	t.Run("folder type with empty dir fails", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		if err := os.MkdirAll(filepath.Join(base, "a", "amb"), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"amb": {Text: "A", Type: "folder"}}}}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "is empty") {
			t.Fatalf("want 'is empty', got %v", err)
		}
	})

	t.Run("folder type with file at dir path fails", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		writeFile(t, filepath.Join(base, "a", "amb"), "not a dir")
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"amb": {Text: "A", Type: "folder"}}}}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("want 'not a directory', got %v", err)
		}
	})

	t.Run("folder with custom Path override", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		customDir := filepath.Join(base, "custom")
		writeFile(t, filepath.Join(customDir, "loop.wav"), "fake")
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"amb": {Text: "A", Type: "folder", Path: customDir}}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("folder with files passes", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		writeFile(t, filepath.Join(base, "a", "amb", "01.wav"), "fake")
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"amb": {Text: "A", Type: "folder"}}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("extra routes use extra audio path", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", base)
		writeFile(t, filepath.Join(base, "extra", "a", "x.wav"), "fake")
		cfg := DeviceConfig{"a": {Commands: map[string]Command{"x": {Text: "X", IsExtra: true}}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
}

func TestTotalCommands(t *testing.T) {
	cfg := DeviceConfig{
		"a": {Commands: map[string]Command{"x": {Text: "X"}, "y": {Text: "Y"}}},
		"b": {Commands: map[string]Command{"z": {Text: "Z"}}},
	}
	if got := cfg.TotalCommands(); got != 3 {
		t.Fatalf("want 3, got %d", got)
	}
}

func TestLoadAllRoutes(t *testing.T) {
	t.Run("merges routes and marks extras", func(t *testing.T) {
		base := t.TempDir()
		audioBase := filepath.Join(base, "audio")
		t.Setenv("AUDIO_BASE_PATH", audioBase)
		writeFile(t, filepath.Join(audioBase, "dreame", "hi.wav"), "fake")
		writeFile(t, filepath.Join(audioBase, "extra", "dreame", "bye.wav"), "fake")

		routesDir := filepath.Join(base, "routes")
		writeJSON(t, filepath.Join(routesDir, "dreame.json"), DeviceConfig{
			"dreame": {Commands: map[string]Command{"hi": {Text: "Hi"}}},
		})

		extraPath := filepath.Join(base, "extra_routes.json")
		writeJSON(t, extraPath, DeviceConfig{
			"dreame": {Commands: map[string]Command{"bye": {Text: "Bye"}}},
		})

		cfg, err := LoadAllRoutes(routesDir, extraPath)
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if !cfg["dreame"].Commands["bye"].IsExtra {
			t.Fatal("extra command should have IsExtra=true")
		}
		if cfg["dreame"].Commands["hi"].IsExtra {
			t.Fatal("base command should have IsExtra=false")
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("validate: %v", err)
		}
	})

	t.Run("missing extra routes file is fine", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("AUDIO_BASE_PATH", filepath.Join(base, "audio"))
		writeFile(t, filepath.Join(base, "audio", "a", "x.wav"), "fake")
		routesDir := filepath.Join(base, "routes")
		writeJSON(t, filepath.Join(routesDir, "a.json"), DeviceConfig{
			"a": {Commands: map[string]Command{"x": {Text: "X"}}},
		})
		if _, err := LoadAllRoutes(routesDir, filepath.Join(base, "does-not-exist.json")); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("empty routes dir fails", func(t *testing.T) {
		base := t.TempDir()
		if _, err := LoadAllRoutes(base, ""); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestApplyVolumeOverrides(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	t.Run("valid override applied", func(t *testing.T) {
		t.Setenv("DREAME_VOLUME_OVERRIDE", "20")
		cfg := DeviceConfig{"dreame": {Volume: intPtr(80), Commands: map[string]Command{"x": {Text: "X"}}}}
		ApplyVolumeOverrides(cfg, discard)
		if cfg["dreame"].Volume == nil || *cfg["dreame"].Volume != 20 {
			t.Fatalf("want 20, got %v", cfg["dreame"].Volume)
		}
	})

	t.Run("sets volume when previously nil", func(t *testing.T) {
		t.Setenv("DREAME_VOLUME_OVERRIDE", "55")
		cfg := DeviceConfig{"dreame": {Commands: map[string]Command{"x": {Text: "X"}}}}
		ApplyVolumeOverrides(cfg, discard)
		if cfg["dreame"].Volume == nil || *cfg["dreame"].Volume != 55 {
			t.Fatalf("want 55, got %v", cfg["dreame"].Volume)
		}
	})

	t.Run("invalid value ignored", func(t *testing.T) {
		t.Setenv("DREAME_VOLUME_OVERRIDE", "not-a-number")
		cfg := DeviceConfig{"dreame": {Volume: intPtr(80), Commands: map[string]Command{"x": {Text: "X"}}}}
		ApplyVolumeOverrides(cfg, discard)
		if cfg["dreame"].Volume == nil || *cfg["dreame"].Volume != 80 {
			t.Fatalf("want 80 preserved, got %v", cfg["dreame"].Volume)
		}
	})

	t.Run("out of range ignored", func(t *testing.T) {
		t.Setenv("DREAME_VOLUME_OVERRIDE", "150")
		cfg := DeviceConfig{"dreame": {Volume: intPtr(80), Commands: map[string]Command{"x": {Text: "X"}}}}
		ApplyVolumeOverrides(cfg, discard)
		if cfg["dreame"].Volume == nil || *cfg["dreame"].Volume != 80 {
			t.Fatalf("want 80 preserved, got %v", cfg["dreame"].Volume)
		}
	})

	t.Run("negative out of range ignored", func(t *testing.T) {
		t.Setenv("DREAME_VOLUME_OVERRIDE", "-5")
		cfg := DeviceConfig{"dreame": {Volume: intPtr(80), Commands: map[string]Command{"x": {Text: "X"}}}}
		ApplyVolumeOverrides(cfg, discard)
		if cfg["dreame"].Volume == nil || *cfg["dreame"].Volume != 80 {
			t.Fatalf("want 80 preserved, got %v", cfg["dreame"].Volume)
		}
	})

	t.Run("no override leaves volume untouched", func(t *testing.T) {
		cfg := DeviceConfig{"dreame": {Volume: intPtr(80), Commands: map[string]Command{"x": {Text: "X"}}}}
		ApplyVolumeOverrides(cfg, discard)
		if cfg["dreame"].Volume == nil || *cfg["dreame"].Volume != 80 {
			t.Fatalf("want 80, got %v", cfg["dreame"].Volume)
		}
	})
}

func TestGetAudioFilePathForCommand(t *testing.T) {
	t.Setenv("AUDIO_BASE_PATH", "/custom")
	if got := GetAudioFilePathForCommand("d", "x", false); got != "/custom/d/x.wav" {
		t.Fatalf("got %q", got)
	}
	if got := GetAudioFilePathForCommand("d", "x", true); got != "/custom/extra/d/x.wav" {
		t.Fatalf("got %q", got)
	}
}

func TestGetFolderDirPath(t *testing.T) {
	t.Setenv("AUDIO_BASE_PATH", "/custom")
	if got := GetFolderDirPath("d", "amb", false); got != "/custom/d/amb" {
		t.Fatalf("got %q", got)
	}
	if got := GetFolderDirPath("d", "amb", true); got != "/custom/extra/d/amb" {
		t.Fatalf("got %q", got)
	}
}

func TestCommandGetFolderPath(t *testing.T) {
	t.Setenv("AUDIO_BASE_PATH", "/custom")
	t.Run("custom Path wins", func(t *testing.T) {
		c := Command{Path: "/elsewhere/loop"}
		if got := c.GetFolderPath("d", "amb"); got != "/elsewhere/loop" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("default path when Path empty", func(t *testing.T) {
		c := Command{}
		if got := c.GetFolderPath("d", "amb"); got != "/custom/d/amb" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("extra routing", func(t *testing.T) {
		c := Command{IsExtra: true}
		if got := c.GetFolderPath("d", "amb"); got != "/custom/extra/d/amb" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestGetEnvHelpers(t *testing.T) {
	t.Run("GetEnv default", func(t *testing.T) {
		os.Unsetenv("JACADI_TEST_STR")
		if got := GetEnv("JACADI_TEST_STR", "fallback"); got != "fallback" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("GetEnv set", func(t *testing.T) {
		t.Setenv("JACADI_TEST_STR", "set")
		if got := GetEnv("JACADI_TEST_STR", "fallback"); got != "set" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("GetEnvInt default", func(t *testing.T) {
		os.Unsetenv("JACADI_TEST_INT")
		if got := GetEnvInt("JACADI_TEST_INT", 42); got != 42 {
			t.Fatalf("got %d", got)
		}
	})
	t.Run("GetEnvInt set", func(t *testing.T) {
		t.Setenv("JACADI_TEST_INT", "7")
		if got := GetEnvInt("JACADI_TEST_INT", 42); got != 7 {
			t.Fatalf("got %d", got)
		}
	})
	t.Run("GetEnvInt invalid falls back", func(t *testing.T) {
		t.Setenv("JACADI_TEST_INT", "oops")
		if got := GetEnvInt("JACADI_TEST_INT", 42); got != 42 {
			t.Fatalf("got %d", got)
		}
	})
	t.Run("GetEnvBool", func(t *testing.T) {
		t.Setenv("JACADI_TEST_BOOL", "true")
		if !GetEnvBool("JACADI_TEST_BOOL", false) {
			t.Fatal("want true")
		}
		t.Setenv("JACADI_TEST_BOOL", "garbage")
		if !GetEnvBool("JACADI_TEST_BOOL", true) {
			t.Fatal("invalid should fall back to default")
		}
	})
}
