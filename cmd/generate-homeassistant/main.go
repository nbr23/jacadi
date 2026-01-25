package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"jacadi/config"
)

//go:embed templates/*.tmpl
var templates embed.FS

var (
	baseURL      = flag.String("base-url", "http://jacadi.local:8080", "Base URL for jacadi API")
	prefix       = flag.String("prefix", "jacadi_", "Prefix for rest_command service names")
	output       = flag.String("output", "ha-config/homeassistant_rest.yml", "Output file path")
	tts          = flag.Bool("tts", false, "Generate TTS rest_command and script")
	defaultVoice = flag.String("default-voice", "en_US-amy-low", "Default voice for TTS template")
	scriptOutput = flag.String("script-output", "ha-config/homeassistant_scripts.yml", "Output file path for scripts")
)

type TemplateData struct {
	Devices      config.DeviceConfig
	BaseURL      string
	Prefix       string
	IncludeTTS   bool
	DefaultVoice string
}

func main() {
	flag.Parse()

	deviceConfig, err := config.LoadDeviceConfig("routes.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	data := TemplateData{
		Devices:      deviceConfig,
		BaseURL:      *baseURL,
		Prefix:       *prefix,
		IncludeTTS:   *tts,
		DefaultVoice: *defaultVoice,
	}

	funcMap := template.FuncMap{
		"replace": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
	}

	restTmpl, err := template.New("rest_command.yaml.tmpl").Funcs(funcMap).ParseFS(templates, "templates/rest_command.yaml.tmpl")
	if err != nil {
		log.Fatalf("Failed to parse rest_command template: %v", err)
	}

	dir := filepath.Dir(*output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	restFile, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer restFile.Close()

	if err := restTmpl.Execute(restFile, data); err != nil {
		log.Fatalf("Failed to execute rest_command template: %v", err)
	}

	commandCount := deviceConfig.TotalCommands()
	if *tts {
		commandCount++
	}

	fmt.Printf("Home Assistant config generated at %s\n", *output)
	fmt.Printf("Generated %d rest_command entries\n", commandCount)
	fmt.Printf("Base URL: %s\n", *baseURL)
	fmt.Printf("Service prefix: %s\n", *prefix)

	if *tts {
		scriptTmpl, err := template.New("script.yaml.tmpl").Funcs(funcMap).ParseFS(templates, "templates/script.yaml.tmpl")
		if err != nil {
			log.Fatalf("Failed to parse script template: %v", err)
		}

		scriptDir := filepath.Dir(*scriptOutput)
		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			log.Fatalf("Failed to create script directory: %v", err)
		}

		scriptFile, err := os.Create(*scriptOutput)
		if err != nil {
			log.Fatalf("Failed to create script file: %v", err)
		}
		defer scriptFile.Close()

		if err := scriptTmpl.Execute(scriptFile, data); err != nil {
			log.Fatalf("Failed to execute script template: %v", err)
		}

		fmt.Printf("Home Assistant script generated at %s\n", *scriptOutput)
		fmt.Printf("Default voice: %s\n", *defaultVoice)
	}
}
