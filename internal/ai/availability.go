package ai

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	copilot "github.com/github/copilot-sdk/go"
)

var (
	mu       sync.Mutex
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
// Uses a mutex for lazy initialization with automatic recovery if the Copilot
// CLI process dies (pipe closed errors).
// Returns ErrUnavailable if the Copilot CLI is not found.
//
// IMPORTANT: The Copilot subprocess is started with context.Background() so it
// outlives any individual request timeout. Passing a request-scoped context
// to client.Start() would kill the subprocess when the request completes.
func GetGenerator(_ context.Context) (Generator, error) {
	mu.Lock()
	defer mu.Unlock()

	// Return cached instance if available.
	if instance != nil && initErr == nil {
		return instance, nil
	}

	// If we previously determined the CLI is unavailable, don't retry.
	if initErr == ErrUnavailable {
		return nil, initErr
	}

	// (Re)initialize — either first call or recovery from broken pipe.
	initErr = nil

	// Fast path: check CLI availability
	if _, err := exec.LookPath("copilot"); err != nil {
		initErr = ErrUnavailable
		return nil, initErr
	}

	// Load AI config for model names
	cfg := LoadModelConfig()

	// Create and start Copilot client.
	// Use context.Background() so the subprocess outlives individual requests.
	client := copilot.NewClient(&copilot.ClientOptions{
		LogLevel: "error",
	})
	if err := client.Start(context.Background()); err != nil {
		initErr = fmt.Errorf("failed to start Copilot CLI: %w", err)
		return nil, initErr
	}

	instance = &CopilotGenerator{
		client:  client,
		models:  cfg,
		timeout: defaultTimeout,
	}
	return instance, nil
}

// ResetGenerator resets the singleton for testing purposes.
// This allows tests to re-initialize with different configurations.
func ResetGenerator() {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		_ = instance.Close()
	}
	instance = nil
	initErr = nil
}
