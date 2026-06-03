package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name    string `yaml:"name"`
	Pattern string `yaml:"pattern"`
}

type Settings struct {
	FailOnMatch bool   `yaml:"fail_on_match"`
	WebhookURL  string `yaml:"webhook_url"`
}

type Config struct {
	Version  string   `yaml:"version"`
	Settings Settings `yaml:"settings"`
	Exclude  []string `yaml:"exclude"`
	Rules    []Rule   `yaml:"rules"`
}

const DefaultConfigFilename = "git-good.yaml"

func DefaultConfigPath() (string, error) {
	root, err := GitRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, DefaultConfigFilename), nil
}

func GenerateDefaultConfig() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	defaultConfig := Config{
		Version: "1",
		Settings: Settings{
			FailOnMatch: true,
			WebhookURL:  "",
		},
		Exclude: []string{"*.png", "node_modules/*", "go.sum"},
		Rules: []Rule{
			{Name: "AWS Access Key", Pattern: "AKIA[0-9A-Z]{16}"},
			{Name: "Generic Secret String", Pattern: `(?i)(password|secret|api_key|passwd)\s*=\s*['"][^'"]+['"]`},
		},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func parseConfig(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func LoadConfig() (*Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseConfig(data)
}

func GitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not determine git root (are you inside a git repo?): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
