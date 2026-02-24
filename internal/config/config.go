package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/goccy/go-yaml"
)

// AIModelConfig holds per-task model names for AI operations.
type AIModelConfig struct {
	Generate string `yaml:"generate,omitempty"`
	Autofill string `yaml:"autofill,omitempty"`
	Fallback string `yaml:"fallback,omitempty"`
}

// AISettings holds all AI-related configuration.
type AISettings struct {
	Models  AIModelConfig `yaml:"models,omitempty"`
	Timeout int           `yaml:"timeout,omitempty"` // Seconds per AI request, default 30
}

// AppConfig is the top-level application configuration read from config.yaml.
type AppConfig struct {
	AI AISettings `yaml:"ai,omitempty"`
}

// ConfigPath returns the path to the config.yaml file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// LoadAppConfig reads the application config from ~/.config/wf/config.yaml.
// If the file doesn't exist, returns a default config (not an error).
// If the file exists but is malformed, returns an error.
func LoadAppConfig() (*AppConfig, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultAppConfig(), nil
		}
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Apply defaults for missing timeout
	if cfg.AI.Timeout <= 0 {
		cfg.AI.Timeout = 30
	}
	return &cfg, nil
}

// defaultAppConfig returns the default application configuration.
func defaultAppConfig() *AppConfig {
	return &AppConfig{
		AI: AISettings{
			Timeout: 30,
		},
	}
}

// WorkflowsDir returns the path to the workflows directory.
// Uses XDG config home (~/.config/wf/workflows/) for cross-platform compliance.
// On macOS this resolves to ~/.config/wf/workflows/ (NOT ~/Library/Application Support).
func WorkflowsDir() string {
	return filepath.Join(xdg.ConfigHome, "wf", "workflows")
}

// ConfigDir returns the root config directory for wf.
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, "wf")
}

// EnsureDir creates the workflows directory if it doesn't exist.
func EnsureDir() error {
	return os.MkdirAll(WorkflowsDir(), 0755)
}
