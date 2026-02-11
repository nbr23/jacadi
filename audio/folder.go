package audio

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

type FolderPlayer struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	logger  *slog.Logger
	closing bool
}

func NewFolderPlayer(logger *slog.Logger) *FolderPlayer {
	return &FolderPlayer{logger: logger}
}

func (p *FolderPlayer) Start(dirPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closing {
		return fmt.Errorf("folder player is closing")
	}

	p.killLocked()

	args := []string{
		"--no-video",
		"--no-terminal",
		"--save-position-on-quit",
		"--loop-playlist=inf",
	}

	if dev := os.Getenv("AUDIODEV"); dev != "" {
		args = append(args, "--audio-device=alsa/"+dev)
	}

	args = append(args, dirPath)

	cmd := exec.Command("mpv", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	p.cmd = cmd
	p.logger.Info("folder started", "dir", dirPath, "pid", cmd.Process.Pid)

	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.cmd == cmd {
			p.cmd = nil
		}
		if err != nil && !p.closing {
			p.logger.Warn("mpv exited", "error", err)
		}
	}()

	return nil
}

func (p *FolderPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.killLocked()
}

func (p *FolderPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil
}

func (p *FolderPlayer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closing = true
	p.killLocked()
}

func (p *FolderPlayer) killLocked() {
	if p.cmd != nil && p.cmd.Process != nil {
		p.logger.Info("killing mpv", "pid", p.cmd.Process.Pid)
		p.cmd.Process.Signal(os.Interrupt)
		p.cmd.Process.Wait()
		p.cmd = nil
	}
}
