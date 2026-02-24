package ai

import (
	"github.com/fredriklanga/wf/internal/config"
)

// ModelConfig holds per-task model configuration.
type ModelConfig struct {
	Generate string `yaml:"generate,omitempty"` // Model for workflow generation
	Autofill string `yaml:"autofill,omitempty"` // Model for metadata fill
	Fallback string `yaml:"fallback,omitempty"` // Default fallback model
}

// DefaultModelConfig returns sensible defaults for model selection.
func DefaultModelConfig() ModelConfig {
	return ModelConfig{
		Generate: "gpt-4o",
		Autofill: "gpt-4o-mini",
		Fallback: "gpt-4o-mini",
	}
}

// ForTask returns the model name for a given task, falling back through
// the configured fallback and then the hard-coded default.
func (c ModelConfig) ForTask(task string) string {
	switch task {
	case "generate":
		if c.Generate != "" {
			return c.Generate
		}
	case "autofill":
		if c.Autofill != "" {
			return c.Autofill
		}
	}
	if c.Fallback != "" {
		return c.Fallback
	}
	return DefaultModelConfig().Fallback
}

// LoadModelConfig loads AI model configuration from the app config file.
// Returns default config if the file doesn't exist or has no AI section.
func LoadModelConfig() ModelConfig {
	appCfg, err := config.LoadAppConfig()
	if err != nil || appCfg == nil {
		return DefaultModelConfig()
	}
	result := DefaultModelConfig()
	if appCfg.AI.Models.Generate != "" {
		result.Generate = appCfg.AI.Models.Generate
	}
	if appCfg.AI.Models.Autofill != "" {
		result.Autofill = appCfg.AI.Models.Autofill
	}
	if appCfg.AI.Models.Fallback != "" {
		result.Fallback = appCfg.AI.Models.Fallback
	}
	return result
}
