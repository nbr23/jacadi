package audio

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"jacadi/config"
)

var (
	alsaCard    string
	alsaControl string
	volumeRe    = regexp.MustCompile(`\[(\d+)%\]`)
)

func init() {
	alsaControl = config.GetEnv("ALSA_CONTROL", "PCM")
	audiodev := config.GetEnv("AUDIODEV", "")
	if audiodev == "" {
		alsaCard = "0"
		return
	}
	re := regexp.MustCompile(`^(?:plug)?hw:(\d+)`)
	if matches := re.FindStringSubmatch(audiodev); len(matches) > 1 {
		alsaCard = matches[1]
		return
	}
	alsaCard = "0"
}

func GetVolume() (int, error) {
	cmd := exec.Command("amixer", "-c", alsaCard, "sget", alsaControl)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("amixer get failed: %w, output: %s", err, string(output))
	}

	matches := volumeRe.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		vol, _ := strconv.Atoi(matches[1])
		return vol, nil
	}
	return 0, nil
}

func SetVolume(volume int) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}

	cmd := exec.Command("amixer", "-c", alsaCard, "sset", alsaControl, fmt.Sprintf("%d%%", volume))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("amixer set failed: %w, output: %s", err, string(output))
	}
	return nil
}
