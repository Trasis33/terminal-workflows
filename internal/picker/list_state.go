package picker

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	parammeta "github.com/fredriklanga/wf/internal/params"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/sahilm/fuzzy"
)

const listPreviewMaxWidth = 72
const listVisibleMaxRows = 6

type listPickerState struct {
	filterInput     textinput.Model
	allRows         []parammeta.ListRow
	visibleRows     []parammeta.ListRow
	cursor          int
	numberBuffer    string
	parseError      string
	confirmValue    string
	loadErrShort    string
	loadErrDetail   string
	showErrorDetail bool
	emptyAfterSkip  bool
}

type listRowSource []parammeta.ListRow

func (rs listRowSource) String(i int) string { return rs[i].Raw }
func (rs listRowSource) Len() int            { return len(rs) }

func newListPickerState(p template.Param) listPickerState {
	filter := textinput.New()
	filter.Placeholder = "Filter rows..."
	filter.CharLimit = 256
	filter.Prompt = "filter> "

	state := listPickerState{filterInput: filter}
	state.load(p)
	return state
}

func (s *listPickerState) load(p template.Param) {
	s.loadErrShort = ""
	s.loadErrDetail = ""
	s.showErrorDetail = false
	s.parseError = ""
	s.confirmValue = ""
	s.numberBuffer = ""
	s.cursor = 0
	s.allRows = nil
	s.visibleRows = nil
	s.emptyAfterSkip = false

	source, err := parammeta.LoadListSource(p.ListCmd, p.ListSkipHeader)
	if err != nil {
		var sourceErr *parammeta.ListSourceError
		if errors.As(err, &sourceErr) {
			s.loadErrShort = sourceErr.Short
			s.loadErrDetail = sourceErr.Detail
		} else {
			s.loadErrShort = err.Error()
		}
		return
	}

	s.allRows = append([]parammeta.ListRow(nil), source.Rows...)
	s.emptyAfterSkip = source.EmptyAfterSkip
	s.applyFilter()
}

func (s *listPickerState) focus() {
	s.filterInput.Focus()
}

func (s *listPickerState) blur() {
	s.filterInput.Blur()
}

func (s listPickerState) hasLoadError() bool {
	return s.loadErrShort != ""
}

func (s listPickerState) hasConfirmation() bool {
	return s.confirmValue != ""
}

func (s *listPickerState) clearConfirmation() {
	s.confirmValue = ""
}

func (s *listPickerState) applyFilter() {
	s.visibleRows = s.visibleRows[:0]
	query := strings.TrimSpace(s.filterInput.Value())
	if query == "" {
		s.visibleRows = append(s.visibleRows, s.allRows...)
	} else {
		matches := fuzzy.FindFrom(query, listRowSource(s.allRows))
		for _, match := range matches {
			if match.Index >= 0 && match.Index < len(s.allRows) {
				s.visibleRows = append(s.visibleRows, s.allRows[match.Index])
			}
		}
	}
	if len(s.visibleRows) == 0 {
		s.cursor = 0
		return
	}
	if s.cursor >= len(s.visibleRows) {
		s.cursor = len(s.visibleRows) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
}

func (s *listPickerState) moveCursor(delta int) {
	if len(s.visibleRows) == 0 {
		return
	}
	s.cursor += delta
	if s.cursor < 0 {
		s.cursor = len(s.visibleRows) - 1
	}
	if s.cursor >= len(s.visibleRows) {
		s.cursor = 0
	}
	s.numberBuffer = ""
	s.parseError = ""
}

func (s *listPickerState) appendNumberSelection(digit rune) {
	if digit < '0' || digit > '9' {
		return
	}
	s.numberBuffer += string(digit)
	idx, err := strconv.Atoi(s.numberBuffer)
	if err == nil && idx >= 1 && idx <= len(s.visibleRows) {
		s.cursor = idx - 1
		s.parseError = ""
	}
}

func (s *listPickerState) deleteNumberSelection() bool {
	if s.numberBuffer == "" {
		return false
	}
	runes := []rune(s.numberBuffer)
	s.numberBuffer = string(runes[:len(runes)-1])
	if s.numberBuffer == "" {
		return true
	}
	idx, err := strconv.Atoi(s.numberBuffer)
	if err == nil && idx >= 1 && idx <= len(s.visibleRows) {
		s.cursor = idx - 1
	}
	return true
}

func (s *listPickerState) updateFilter(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	s.filterInput, cmd = s.filterInput.Update(msg)
	s.numberBuffer = ""
	s.parseError = ""
	s.applyFilter()
	return cmd
}

func (s *listPickerState) confirmSelection(p template.Param) bool {
	if s.hasLoadError() {
		return false
	}
	if len(s.visibleRows) == 0 {
		s.parseError = s.emptyMessage()
		return false
	}

	selectedIndex := s.cursor
	if s.numberBuffer != "" {
		idx, err := strconv.Atoi(s.numberBuffer)
		if err != nil || idx < 1 || idx > len(s.visibleRows) {
			s.parseError = fmt.Sprintf("row %q is not visible", s.numberBuffer)
			return false
		}
		selectedIndex = idx - 1
		s.cursor = selectedIndex
	}

	value, err := parammeta.ExtractListValue(s.visibleRows[selectedIndex].Raw, p.ListDelimiter, p.ListFieldIndex)
	if err != nil {
		s.parseError = err.Error()
		s.confirmValue = ""
		return false
	}

	s.parseError = ""
	s.numberBuffer = ""
	s.confirmValue = value
	return true
}

func (s *listPickerState) acceptConfirmedValue() string {
	value := s.confirmValue
	s.confirmValue = ""
	s.parseError = ""
	s.numberBuffer = ""
	return value
}

func (s listPickerState) emptyMessage() string {
	switch {
	case s.emptyAfterSkip && len(s.allRows) == 0:
		return "No selectable rows remain after skipping headers."
	case len(s.allRows) == 0:
		return "List command returned no selectable rows."
	case len(s.visibleRows) == 0 && strings.TrimSpace(s.filterInput.Value()) != "":
		return fmt.Sprintf("No rows match %q.", s.filterInput.Value())
	default:
		return "No selectable rows available."
	}
}

func listPickerUsesNumberSelection(msg tea.KeyMsg, currentFilter string) bool {
	return strings.TrimSpace(currentFilter) == "" && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && unicode.IsDigit(msg.Runes[0])
}

func (s listPickerState) renderLines() []string {
	lines := []string{"    " + s.filterInput.View()}

	if s.hasLoadError() {
		lines = append(lines, "    "+dimStyle.Render(s.loadErrShort))
		if s.showErrorDetail && s.loadErrDetail != "" {
			for _, line := range strings.Split(s.loadErrDetail, "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				lines = append(lines, "      "+dimStyle.Render(truncateStr(trimmed, listPreviewMaxWidth)))
			}
		}
		lines = append(lines, "    "+hintStyle.Render("[d] details  [esc] cancel"))
		return lines
	}

	if s.hasConfirmation() {
		lines = append(lines,
			"    "+highlightStyle.Render("Will insert: ")+highlightStyle.Render(s.confirmValue),
			"    "+hintStyle.Render("[enter] confirm  [any other key] choose again"),
		)
		return lines
	}

	if s.parseError != "" {
		lines = append(lines, "    "+dimStyle.Render(s.parseError))
	}

	if s.numberBuffer != "" {
		lines = append(lines, "    "+dimStyle.Render("row "+s.numberBuffer))
	}

	if len(s.visibleRows) == 0 {
		lines = append(lines, "    "+dimStyle.Render(s.emptyMessage()))
		lines = append(lines, "    "+hintStyle.Render("type to change the filter  [esc] cancel"))
		return lines
	}

	start := 0
	if s.cursor >= listVisibleMaxRows {
		start = s.cursor - listVisibleMaxRows + 1
	}
	end := start + listVisibleMaxRows
	if end > len(s.visibleRows) {
		end = len(s.visibleRows)
	}

	for i := start; i < end; i++ {
		prefix := "      "
		rowStyle := normalStyle
		if i == s.cursor {
			prefix = "    " + cursorStyle.Render("❯ ")
			rowStyle = highlightStyle
		}
		label := fmt.Sprintf("%d. %s", i+1, truncateStr(s.visibleRows[i].Raw, listPreviewMaxWidth))
		lines = append(lines, prefix+rowStyle.Render(label))
	}

	if len(s.visibleRows) > listVisibleMaxRows {
		lines = append(lines, "    "+dimStyle.Render(fmt.Sprintf("showing %d/%d visible rows", end-start, len(s.visibleRows))))
	}
	lines = append(lines, "    "+hintStyle.Render("type to filter  [1-9] jump  [↑/↓] move  [enter] select"))
	return lines
}
