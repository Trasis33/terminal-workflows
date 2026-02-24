// Package history provides shell history file parsing for zsh, bash, and fish.
package history

import (
	"errors"
	"time"
)

var errNoHistory = errors.New("no history entries")

// HistoryEntry represents a single command from shell history.
type HistoryEntry struct {
	Command   string
	Timestamp time.Time // zero value if unavailable
}

// HistoryReader reads commands from a shell history file.
type HistoryReader interface {
	// LastN returns the last n commands, newest first.
	LastN(n int) ([]HistoryEntry, error)
	// Last returns the most recent command.
	Last() (HistoryEntry, error)
}
