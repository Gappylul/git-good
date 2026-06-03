package internal

import (
	"bytes"
	"fmt"
	"os"

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

const DefaultConfigPath = "git-good.yaml"

func GenerateDefaultConfig() error {
	defaultConfig := Config{
		Version: "1",
		Settings: Settings{
			FailOnMatch: true,
			WebhookURL:  "https://your-webhook-url-here.com",
		},
		Exclude: []string{"*.png, node_modules/*, go.sum"},
		Rules: []Rule{
			{Name: "AWS Access Key", Pattern: "AKIA[0-9A-Z]{16}"},
			{Name: "Generic Secret String", Pattern: `(?i)(password|secret|api_key|passwd)\s*=\s*['"][^'"]+['"]`},
		},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(DefaultConfigPath, data, 0644)
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(DefaultConfigPath)
	if err != nil {
		return nil, err
	}

	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err = decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration structure: %w", err)
	}

	for _, rule := range config.Rules {
		if rule.Name == "" || rule.Pattern == "" {
			return nil, fmt.Errorf("rule validation failed: both 'name' and 'pattern' are required fields")
		}
	}

	return &config, nil
}
