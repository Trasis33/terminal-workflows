package history

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectShell returns the current shell name based on $SHELL environment variable.
// Falls back to "bash" if $SHELL is empty or unrecognized.
func DetectShell() string {
	return detectShellFromPath(os.Getenv("SHELL"))
}

// detectShellFromPath extracts the shell name from a path like "/bin/zsh".
// Returns "bash" as fallback for empty or unrecognized paths.
func detectShellFromPath(shellPath string) string {
	if shellPath == "" {
		return "bash"
	}
	base := filepath.Base(shellPath)
	switch {
	case strings.Contains(base, "zsh"):
		return "zsh"
	case strings.Contains(base, "fish"):
		return "fish"
	case strings.Contains(base, "bash"):
		return "bash"
	default:
		return "bash"
	}
}

// defaultHistoryPath returns the default history file path for a given shell.
func defaultHistoryPath(shell string) string {
	home, _ := os.UserHomeDir()
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zsh_history")
	case "fish":
		// Fish uses XDG_DATA_HOME/fish/fish_history
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, "fish", "fish_history")
	case "bash":
		return filepath.Join(home, ".bash_history")
	default:
		return filepath.Join(home, ".bash_history")
	}
}

// NewReader creates a HistoryReader for the current shell.
// It auto-detects the shell from $SHELL and uses $HISTFILE if set.
func NewReader() (HistoryReader, error) {
	shell := DetectShell()
	histFile := os.Getenv("HISTFILE")
	if histFile == "" {
		histFile = defaultHistoryPath(shell)
	}

	data, err := os.ReadFile(histFile)
	if err != nil {
		return nil, err
	}

	switch shell {
	case "zsh":
		return &zshReader{data: data, path: histFile}, nil
	case "fish":
		return &fishReader{data: data, path: histFile}, nil
	default:
		return &bashReader{data: data, path: histFile}, nil
	}
}
