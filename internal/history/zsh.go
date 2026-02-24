package history

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"time"
)

// unmetafy decodes zsh's metafied encoding.
// When byte 0x83 is encountered, the following byte is XORed with 0x20.
func unmetafy(data []byte) []byte {
	var out []byte
	for i := 0; i < len(data); i++ {
		if data[i] == 0x83 && i+1 < len(data) {
			i++
			out = append(out, data[i]^0x20)
		} else {
			out = append(out, data[i])
		}
	}
	return out
}

// parseZshExtendedLine parses a line in ": timestamp:duration;command" format.
// Returns the command, timestamp, and whether it was extended format.
func parseZshExtendedLine(line string) (cmd string, ts time.Time, ok bool) {
	if !strings.HasPrefix(line, ": ") {
		return "", time.Time{}, false
	}
	rest := line[2:]
	semiIdx := strings.IndexByte(rest, ';')
	if semiIdx < 0 {
		return "", time.Time{}, false
	}
	meta := rest[:semiIdx]
	cmd = rest[semiIdx+1:]

	colonIdx := strings.IndexByte(meta, ':')
	if colonIdx < 0 {
		return "", time.Time{}, false
	}
	epoch, err := strconv.ParseInt(meta[:colonIdx], 10, 64)
	if err != nil {
		return cmd, time.Time{}, true // command ok, timestamp bad
	}
	return cmd, time.Unix(epoch, 0), true
}

// parseZshHistory parses zsh history data, supporting both extended and plain formats.
// In mixed format, lines that don't start with ": " after an extended entry are treated
// as continuation lines (multiline commands).
// Data should already be unmetafied.
func parseZshHistory(data []byte) []HistoryEntry {
	if len(data) == 0 {
		return nil
	}

	// Collect all lines first
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		return nil
	}

	// Track which lines are extended format
	type lineInfo struct {
		text       string
		isExtended bool
		cmd        string
		ts         time.Time
	}

	infos := make([]lineInfo, len(lines))
	for i, line := range lines {
		cmd, ts, isExt := parseZshExtendedLine(line)
		infos[i] = lineInfo{text: line, isExtended: isExt, cmd: cmd, ts: ts}
	}

	var entries []HistoryEntry

	for i := 0; i < len(infos); i++ {
		info := infos[i]
		if info.text == "" {
			continue
		}

		if info.isExtended {
			// Extended format line — start new entry
			entry := HistoryEntry{Command: info.cmd, Timestamp: info.ts}
			// Collect continuation lines (non-extended, non-empty lines following this)
			for i+1 < len(infos) && !infos[i+1].isExtended && infos[i+1].text != "" {
				i++
				entry.Command += "\n" + infos[i].text
			}
			entries = append(entries, entry)
		} else {
			// Plain format line — either pure plain file or a pre-EXTENDED_HISTORY
			// entry in a mixed file. Both cases: treat as a standalone command.
			entries = append(entries, HistoryEntry{Command: info.text})
		}
	}

	return entries
}

// zshReader implements HistoryReader for zsh.
type zshReader struct {
	data []byte // raw file data (for testing) or nil for file-based
	path string // history file path (for file-based reading)
}

func (r *zshReader) LastN(n int) ([]HistoryEntry, error) {
	data := r.data
	if data == nil {
		return nil, nil
	}
	data = unmetafy(data)
	return lastN(parseZshHistory(data), n), nil
}

func (r *zshReader) Last() (HistoryEntry, error) {
	data := r.data
	if data == nil {
		return HistoryEntry{}, errNoHistory
	}
	data = unmetafy(data)
	return last(parseZshHistory(data))
}
