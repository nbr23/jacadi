package audio

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

type Player interface {
	PlayAsync(filepath string) error
	Close() error
}

type AplayPlayer struct {
	wg      sync.WaitGroup
	logger  *slog.Logger
	closing bool
}

func NewAplayPlayer(logger *slog.Logger) (*AplayPlayer, error) {
	if _, err := exec.LookPath("aplay"); err != nil {
		return nil, fmt.Errorf("aplay not found: %w", err)
	}

	logger.Info("audio player initialized using aplay")
	return &AplayPlayer{
		logger: logger,
	}, nil
}

func (p *AplayPlayer) PlayAsync(filepath string) error {
	if p.closing {
		return fmt.Errorf("player is closing")
	}

	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		p.logger.Info("audio playback started", "file", filepath)
		var cmd *exec.Cmd

		if os.Getenv("AUDIODEV") != "" {
			cmd = exec.Command("aplay", "-q", "-D", os.Getenv("AUDIODEV"), filepath)

		} else {
			cmd = exec.Command("aplay", "-q", filepath)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			p.logger.Error("audio playback failed",
				"file", filepath,
				"error", err,
				"output", string(output),
			)
			return
		}

		p.logger.Info("audio playback completed", "file", filepath)
	}()

	return nil
}

func (p *AplayPlayer) Close() error {
	p.closing = true
	p.logger.Info("closing audio player, waiting for active playback to finish...")
	p.wg.Wait()
	p.logger.Info("audio player closed")
	return nil
}
