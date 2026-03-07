package params

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const listSourceTimeout = 5 * time.Second
const listSourceMaxTokenSize = 1024 * 1024

// ListRow preserves the raw command output row for later display and extraction.
type ListRow struct {
	Raw string
}

// ListSource contains the loaded selectable rows and whether header skipping
// exhausted otherwise-valid command output.
type ListSource struct {
	Rows           []ListRow
	EmptyAfterSkip bool
}

// ListSourceError surfaces a short readable message plus optional detailed
// diagnostics for command execution and output parsing failures.
type ListSourceError struct {
	Short  string
	Detail string
}

func (e *ListSourceError) Error() string {
	return e.Short
}

// LoadListSource runs a shell command, loads non-empty output rows, and applies
// header skipping before returning selectable rows.
func LoadListSource(command string, skipHeader int) (ListSource, error) {
	if skipHeader < 0 {
		skipHeader = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), listSourceTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return ListSource{}, buildListSourceError(err)
	}

	rows, err := scanListRows(output)
	if err != nil {
		return ListSource{}, err
	}

	if skipHeader >= len(rows) {
		return ListSource{EmptyAfterSkip: len(rows) > 0}, nil
	}

	return ListSource{Rows: rows[skipHeader:]}, nil
}

func scanListRows(output []byte) ([]ListRow, error) {
	var rows []ListRow

	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 0, 64*1024), listSourceMaxTokenSize)
	for scanner.Scan() {
		raw := scanner.Text()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		rows = append(rows, ListRow{Raw: raw})
	}

	if err := scanner.Err(); err != nil {
		return nil, &ListSourceError{
			Short:  "reading list output failed",
			Detail: err.Error(),
		}
	}

	return rows, nil
}

func buildListSourceError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &ListSourceError{Short: fmt.Sprintf("list command timed out after %s", listSourceTimeout)}
	}

	short := fmt.Sprintf("list command failed: %v", err)
	detail := ""

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		detail = strings.TrimSpace(string(exitErr.Stderr))
	}

	return &ListSourceError{Short: short, Detail: detail}
}
