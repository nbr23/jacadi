package tts

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"jacadi/config"
)

type Speaker interface {
	SpeakAsync(text, voice string) error
	Close() error
}

type PiperSpeaker struct {
	wg         sync.WaitGroup
	logger     *slog.Logger
	closing    bool
	sampleRate int
	audiodev   string
}

func NewPiperSpeaker(logger *slog.Logger) (*PiperSpeaker, error) {
	if _, err := exec.LookPath("python"); err != nil {
		return nil, fmt.Errorf("python not found: %w", err)
	}
	if _, err := exec.LookPath("aplay"); err != nil {
		return nil, fmt.Errorf("aplay not found: %w", err)
	}

	sampleRate := config.GetPiperSampleRate()
	audiodev := os.Getenv("AUDIODEV")

	logger.Info("piper TTS speaker initialized",
		"sample_rate", sampleRate,
		"audiodev", audiodev,
	)

	return &PiperSpeaker{
		logger:     logger,
		sampleRate: sampleRate,
		audiodev:   audiodev,
	}, nil
}

func (s *PiperSpeaker) SpeakAsync(text, voice string) error {
	if s.closing {
		return fmt.Errorf("speaker is closing")
	}

	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("text cannot be empty")
	}

	if voice == "" {
		voice = config.GetDefaultVoice()
	}

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		s.logger.Info("TTS started", "voice", voice, "text_length", len(text))

		if err := s.speak(text, voice); err != nil {
			s.logger.Error("TTS failed",
				"voice", voice,
				"error", err,
			)
			return
		}

		s.logger.Info("TTS completed", "voice", voice)
	}()

	return nil
}

func (s *PiperSpeaker) speak(text, voice string) error {
	piperCmd := exec.Command("python", "-m", "piper", "--model", voice, "--output-raw", "--data-dir", os.Getenv("VOICES_DIR"))
	piperCmd.Stdin = strings.NewReader(text)

	aplayArgs := []string{
		"-r", fmt.Sprintf("%d", s.sampleRate),
		"-f", "S16_LE",
		"-t", "raw",
		"-q",
	}
	if s.audiodev != "" {
		aplayArgs = append(aplayArgs, "-D", s.audiodev)
	}
	aplayArgs = append(aplayArgs, "-")

	aplayCmd := exec.Command("aplay", aplayArgs...)

	piperStdout, err := piperCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create piper stdout pipe: %w", err)
	}
	aplayCmd.Stdin = piperStdout

	var piperStderr, aplayStderr strings.Builder
	piperCmd.Stderr = &piperStderr
	aplayCmd.Stderr = &aplayStderr

	if err := piperCmd.Start(); err != nil {
		return fmt.Errorf("failed to start piper: %w", err)
	}
	if err := aplayCmd.Start(); err != nil {
		piperCmd.Process.Kill()
		return fmt.Errorf("failed to start aplay: %w", err)
	}

	go func() {
		piperCmd.Wait()
		piperStdout.(io.Closer).Close()
	}()

	if err := aplayCmd.Wait(); err != nil {
		return fmt.Errorf("aplay failed: %w, stderr: %s", err, aplayStderr.String())
	}

	if err := piperCmd.Wait(); err != nil {
		if !strings.Contains(err.Error(), "already finished") {
			return fmt.Errorf("piper failed: %w, stderr: %s", err, piperStderr.String())
		}
	}

	return nil
}

func (s *PiperSpeaker) Close() error {
	s.closing = true
	s.logger.Info("closing TTS speaker, waiting for active speech to finish...")
	s.wg.Wait()
	s.logger.Info("TTS speaker closed")
	return nil
}
