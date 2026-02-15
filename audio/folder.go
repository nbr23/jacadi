package audio

import (
	"bufio"
	"fmt"
	"io"
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
		"--save-position-on-quit",
		"--watch-later-directory=/tmp/.watchlater",
		"--loop-playlist=inf",
	}

	if dev := os.Getenv("AUDIODEV"); dev != "" {
		args = append(args, "--audio-device=alsa/"+dev)
	}

	args = append(args, dirPath)

	cmd := exec.Command("mpv", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	p.logPipe(stdout, "stdout")
	p.logPipe(stderr, "stderr")

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

func (p *FolderPlayer) logPipe(pipe io.ReadCloser, stream string) {
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			p.logger.Info("mpv", "stream", stream, "msg", scanner.Text())
		}
	}()
}

func (p *FolderPlayer) killLocked() {
	if p.cmd != nil && p.cmd.Process != nil {
		p.logger.Info("killing mpv", "pid", p.cmd.Process.Pid)
		p.cmd.Process.Signal(os.Interrupt)
		p.cmd.Process.Wait()
		p.cmd = nil
	}
}
