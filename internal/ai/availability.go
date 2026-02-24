package ai

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	copilot "github.com/github/copilot-sdk/go"
)

var (
	once     sync.Once
	instance *CopilotGenerator
	initErr  error
)

// ErrUnavailable is returned when the Copilot CLI is not installed.
var ErrUnavailable = fmt.Errorf("AI features require GitHub Copilot CLI. Install with: gh extension install github/gh-copilot")

// IsAvailable performs a fast check for the Copilot CLI binary.
// Used by TUI to decide whether to show AI-related errors without
// incurring the cost of full initialization.
func IsAvailable() bool {
	_, err := exec.LookPath("copilot")
	return err == nil
}

// GetGenerator returns the shared Generator instance, initializing on first call.
// Uses sync.Once for lazy singleton initialization.
// Returns ErrUnavailable if the Copilot CLI is not found.
func GetGenerator(ctx context.Context) (Generator, error) {
	once.Do(func() {
		// Fast path: check CLI availability
		if _, err := exec.LookPath("copilot"); err != nil {
			initErr = ErrUnavailable
			return
		}

		// Load AI config for model names
		cfg := LoadModelConfig()

		// Create and start Copilot client
		client := copilot.NewClient(&copilot.ClientOptions{
			LogLevel: "error",
		})
		if err := client.Start(ctx); err != nil {
			initErr = fmt.Errorf("failed to start Copilot CLI: %w", err)
			return
		}

		instance = &CopilotGenerator{
			client:  client,
			models:  cfg,
			timeout: defaultTimeout,
		}
	})
	return instance, initErr
}

// ResetGenerator resets the singleton for testing purposes.
// This allows tests to re-initialize with different configurations.
func ResetGenerator() {
	if instance != nil {
		_ = instance.Close()
	}
	once = sync.Once{}
	instance = nil
	initErr = nil
}
