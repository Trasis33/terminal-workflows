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
	if r.data == nil {
		return nil, nil
	}
	return lastN(parseFishHistory(r.data), n), nil
}

func (r *fishReader) Last() (HistoryEntry, error) {
	if r.data == nil {
		return HistoryEntry{}, errNoHistory
	}
	return last(parseFishHistory(r.data))
}
