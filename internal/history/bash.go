package history

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"time"
)

// parseBashHistory parses bash history data.
// Supports both plain format (one command per line) and timestamped format
// (#epoch on one line, command on the next).
// Lines starting with # that are NOT pure digits are treated as regular commands.
func parseBashHistory(data []byte) []HistoryEntry {
	if len(data) == 0 {
		return nil
	}

	var entries []HistoryEntry
	var pendingTS time.Time

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			ts := strings.TrimPrefix(line, "#")
			if epoch, err := strconv.ParseInt(ts, 10, 64); err == nil {
				pendingTS = time.Unix(epoch, 0)
				continue
			}
			// Non-numeric # line is a regular command (e.g., comments in bash_history)
		}

		entry := HistoryEntry{Command: line, Timestamp: pendingTS}
		pendingTS = time.Time{}
		entries = append(entries, entry)
	}

	return entries
}

// bashReader implements HistoryReader for bash.
type bashReader struct {
	data []byte
	path string
}

func (r *bashReader) LastN(n int) ([]HistoryEntry, error) {
	data := r.data
	if data == nil {
		return nil, nil
	}

	entries := parseBashHistory(data)

	if n <= 0 {
		return nil, nil
	}
	if n > len(entries) {
		n = len(entries)
	}

	// Return last n entries, newest first
	result := make([]HistoryEntry, n)
	for i := 0; i < n; i++ {
		result[i] = entries[len(entries)-1-i]
	}
	return result, nil
}

func (r *bashReader) Last() (HistoryEntry, error) {
	entries, err := r.LastN(1)
	if err != nil {
		return HistoryEntry{}, err
	}
	if len(entries) == 0 {
		return HistoryEntry{}, errNoHistory
	}
	return entries[0], nil
}
