package history

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"time"
)

// parseFishHistory parses fish shell history data.
// Fish uses a pseudo-YAML format — parsed line-by-line, NOT with a YAML parser.
// Each entry starts with "- cmd: " followed by the command.
// Optional "when:" line provides a unix timestamp.
// "paths:" sections are ignored.
func parseFishHistory(data []byte) []HistoryEntry {
	if len(data) == 0 {
		return nil
	}

	var entries []HistoryEntry
	var current *HistoryEntry

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "- cmd: ") {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &HistoryEntry{
				Command: strings.TrimPrefix(line, "- cmd: "),
			}
		} else if current != nil && strings.HasPrefix(strings.TrimSpace(line), "when: ") {
			ts := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "when: "))
			if epoch, err := strconv.ParseInt(ts, 10, 64); err == nil {
				current.Timestamp = time.Unix(epoch, 0)
			}
		}
		// paths: and other lines are ignored
	}
	if current != nil {
		entries = append(entries, *current)
	}

	return entries
}

// fishReader implements HistoryReader for fish.
type fishReader struct {
	data []byte
	path string
}

func (r *fishReader) LastN(n int) ([]HistoryEntry, error) {
	data := r.data
	if data == nil {
		return nil, nil
	}

	entries := parseFishHistory(data)

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

func (r *fishReader) Last() (HistoryEntry, error) {
	entries, err := r.LastN(1)
	if err != nil {
		return HistoryEntry{}, err
	}
	if len(entries) == 0 {
		return HistoryEntry{}, errNoHistory
	}
	return entries[0], nil
}
