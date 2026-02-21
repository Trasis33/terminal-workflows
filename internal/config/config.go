package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

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
