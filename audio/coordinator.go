package audio

import (
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

type Coordinator struct {
	mu        sync.Mutex
	aplay     *AplayPlayer
	folder    *FolderPlayer
	resumeDir string
	logger    *slog.Logger
}

func NewCoordinator(aplay *AplayPlayer, folder *FolderPlayer, logger *slog.Logger) *Coordinator {
	return &Coordinator{
		aplay:  aplay,
		folder: folder,
		logger: logger,
	}
}

func (c *Coordinator) PlaySingleFile(path string, volume *int) {
	c.mu.Lock()
	resumeDir := ""
	if c.folder.IsPlaying() {
		c.logger.Info("interrupting folder for single file", "file", path)
		c.folder.Stop()
		resumeDir = c.resumeDir
	}
	c.mu.Unlock()

	var originalVolume int
	restoreVolume := false
	if volume != nil {
		var err error
		originalVolume, err = GetVolume()
		if err != nil {
			c.logger.Warn("failed to get current volume", "error", err)
		} else {
			restoreVolume = true
			if err := SetVolume(*volume); err != nil {
				c.logger.Warn("failed to set device volume", "error", err, "volume", *volume)
				restoreVolume = false
			} else {
				c.logger.Info("volume set for playback", "volume", *volume, "original", originalVolume)
			}
		}
	}

	c.playSync(path)

	if restoreVolume {
		if err := SetVolume(originalVolume); err != nil {
			c.logger.Warn("failed to restore original volume", "error", err, "volume", originalVolume)
		} else {
			c.logger.Info("volume restored", "volume", originalVolume)
		}
	}

	if resumeDir != "" {
		c.mu.Lock()
		if c.resumeDir == resumeDir {
			c.logger.Info("resuming folder", "dir", resumeDir)
			c.folder.Start(resumeDir)
		}
		c.mu.Unlock()
	}
}

func (c *Coordinator) PlayFolder(dirPath string, volume *int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if volume != nil {
		if err := SetVolume(*volume); err != nil {
			c.logger.Warn("failed to set device volume for folder", "error", err, "volume", *volume)
		} else {
			c.logger.Info("volume set for folder", "volume", *volume)
		}
	}

	c.folder.Stop()
	c.resumeDir = dirPath
	return c.folder.Start(dirPath)
}

func (c *Coordinator) StopFolder() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.folder.Stop()
	c.resumeDir = ""
}

func (c *Coordinator) PlayAsync(filepath string, volume *int) error {
	go c.PlaySingleFile(filepath, volume)
	return nil
}

func (c *Coordinator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.folder.Close()
	return c.aplay.Close()
}

func (c *Coordinator) playSync(filepath string) {
	c.logger.Info("aplay sync started", "file", filepath)

	var cmd *exec.Cmd
	if dev := os.Getenv("AUDIODEV"); dev != "" {
		cmd = exec.Command("aplay", "-q", "-D", dev, filepath)
	} else {
		cmd = exec.Command("aplay", "-q", filepath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("aplay sync failed", "file", filepath, "error", err, "output", string(output))
		return
	}
	c.logger.Info("aplay sync completed", "file", filepath)
}
