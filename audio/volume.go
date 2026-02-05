package audio

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"jacadi/config"
)

func getALSACard() string {
	audiodev := config.GetEnv("AUDIODEV", "")
	if audiodev == "" {
		return "0"
	}
	re := regexp.MustCompile(`^(?:plug)?hw:(\d+)`)
	if matches := re.FindStringSubmatch(audiodev); len(matches) > 1 {
		return matches[1]
	}
	return "0"
}

func GetVolume() (int, error) {
	card := getALSACard()
	control := config.GetEnv("ALSA_CONTROL", "PCM")

	cmd := exec.Command("amixer", "-c", card, "sget", control)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("amixer get failed: %w, output: %s", err, string(output))
	}

	re := regexp.MustCompile(`\[(\d+)%\]`)
	matches := re.FindStringSubmatch(string(output))
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

	card := getALSACard()
	control := config.GetEnv("ALSA_CONTROL", "PCM")

	cmd := exec.Command("amixer", "-c", card, "sset", control, fmt.Sprintf("%d%%", volume))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("amixer set failed: %w, output: %s", err, string(output))
	}
	return nil
}
