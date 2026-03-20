package audio

import (
	"log/slog"
	"sync"
)

type Coordinator struct {
	mu        sync.Mutex
	volumeMu  sync.Mutex
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

	c.volumeMu.Lock()
	defer c.volumeMu.Unlock()

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

	c.aplay.PlaySync(path)

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
	c.volumeMu.Lock()
	defer c.volumeMu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.folder.IsPlaying() && c.resumeDir == dirPath {
		c.logger.Info("folder already playing, skipping restart", "dir", dirPath)
		return nil
	}

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

