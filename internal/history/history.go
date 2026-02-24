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

// lastN returns the last n entries from a slice, newest first.
// Returns nil for n <= 0 or empty input.
func lastN(entries []HistoryEntry, n int) []HistoryEntry {
	if n <= 0 || len(entries) == 0 {
		return nil
	}
	if n > len(entries) {
		n = len(entries)
	}
	result := make([]HistoryEntry, n)
	for i := 0; i < n; i++ {
		result[i] = entries[len(entries)-1-i]
	}
	return result
}

// last returns the most recent entry, or errNoHistory if empty.
func last(entries []HistoryEntry) (HistoryEntry, error) {
	if len(entries) == 0 {
		return HistoryEntry{}, errNoHistory
	}
	return entries[len(entries)-1], nil
}
