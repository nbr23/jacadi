package audio

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

type AplayPlayer struct {
	wg      sync.WaitGroup
	logger  *slog.Logger
	closing atomic.Bool
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

func (p *AplayPlayer) PlaySync(filepath string) {
	p.wg.Add(1)
	defer p.wg.Done()

	p.logger.Info("audio playback started", "file", filepath)

	var cmd *exec.Cmd
	if dev := os.Getenv("AUDIODEV"); dev != "" {
		cmd = exec.Command("aplay", "-q", "-D", dev, filepath)
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
}

func (p *AplayPlayer) Close() error {
	p.closing.Store(true)
	p.logger.Info("closing audio player, waiting for active playback to finish...")
	p.wg.Wait()
	p.logger.Info("audio player closed")
	return nil
}
