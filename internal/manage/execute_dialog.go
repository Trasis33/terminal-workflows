package manage

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	parammeta "github.com/fredriklanga/wf/internal/params"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/sahilm/fuzzy"
)

const dialogExecute dialogType = 11
const executePreviewMinRows = 4
const executePreviewMaxCommandRows = 3
const executeDialogListPreviewMaxWidth = 72
const executeDialogListVisibleMaxRows = 6

type executePhase int

const (
	phaseParamFill executePhase = iota
	phaseActionMenu
)

type executeAction int

const (
	actionCopy executeAction = iota
	actionPasteToPrompt
	actionCancel
)

type executeDialogDynamicMsg struct {
	paramIndex int
	options    []string
	err        error
}

type ExecuteDialogModel struct {
	workflow store.Workflow
	phase    executePhase

	params            []template.Param
	paramInputs       []textinput.Model
	paramTypes        []template.ParamType
	paramOptions      [][]string
	paramOptionCursor []int
	paramLoading      []bool
	paramFailed       []bool
	paramListStates   []executeDialogListState
	focusedParam      int

	renderedCommand string
	actionCursor    int
	actions         []string

	width int
	theme Theme
}

type executeDialogListState struct {
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

func NewExecuteDialog(wf store.Workflow, width int, theme Theme) ExecuteDialogModel {
	params := parammeta.OverlayMetadata(wf.Command, wf.Args)

	d := ExecuteDialogModel{
		workflow: wf,
		params:   params,
		actions: []string{
			"Copy to clipboard",
			"Paste to prompt",
			"Cancel",
		},
		width: width,
		theme: theme,
	}

	if len(params) == 0 {
		d.phase = phaseActionMenu
		d.renderedCommand = template.Render(wf.Command, nil)
		return d
	}

	d.phase = phaseParamFill
	d.paramInputs = make([]textinput.Model, len(params))
	d.paramTypes = make([]template.ParamType, len(params))
	d.paramOptions = make([][]string, len(params))
	d.paramOptionCursor = make([]int, len(params))
	d.paramLoading = make([]bool, len(params))
	d.paramFailed = make([]bool, len(params))
	d.paramListStates = make([]executeDialogListState, len(params))

	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	for i, p := range params {
		ti := textinput.New()
		ti.Placeholder = p.Name
		ti.CharLimit = 256
		d.paramTypes[i] = p.Type

		switch p.Type {
		case template.ParamEnum:
			d.paramOptions[i] = p.Options
			defIdx := 0
			if p.Default != "" {
				for j, opt := range p.Options {
					if opt == p.Default {
						defIdx = j
						break
					}
				}
			}
			d.paramOptionCursor[i] = defIdx
			if len(p.Options) > 0 {
				ti.SetValue(p.Options[defIdx])
			}
		case template.ParamDynamic:
			d.paramLoading[i] = true
			ti.Placeholder = "Loading..."
		case template.ParamList:
			d.paramListStates[i] = newExecuteDialogListState(p)
			if p.Default != "" {
				ti.SetValue(p.Default)
				ti.TextStyle = defaultStyle
				ti.CursorEnd()
			}
			ti.Placeholder = "Choose from list"
		default:
			if p.Default != "" {
				ti.SetValue(p.Default)
				ti.TextStyle = defaultStyle
				ti.CursorEnd()
			}
		}

		d.paramInputs[i] = ti
	}
	d.focusParam(0)

	return d
}

func (d ExecuteDialogModel) Init() tea.Cmd {
	if d.phase == phaseParamFill && len(d.paramInputs) > 0 {
		return textinput.Blink
	}
	return nil
}

func (d ExecuteDialogModel) InitCmds() []tea.Cmd {
	var cmds []tea.Cmd
	for i, p := range d.params {
		if p.Type != template.ParamDynamic {
			continue
		}
		idx := i
		dynCmd := p.DynamicCmd
		cmds = append(cmds, func() tea.Msg {
			return executeDialogDynamic(idx, dynCmd)
		})
	}
	return cmds
}

func executeDialogDynamic(paramIndex int, command string) executeDialogDynamicMsg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return executeDialogDynamicMsg{paramIndex: paramIndex, err: fmt.Errorf("dynamic command failed: %w", err)}
	}

	options := strings.FieldsFunc(string(output), func(r rune) bool { return r == '\n' || r == '\r' })
	filtered := make([]string, 0, len(options))
	for i := range options {
		trimmed := strings.TrimSpace(options[i])
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	if len(filtered) == 0 {
		return executeDialogDynamicMsg{paramIndex: paramIndex, err: fmt.Errorf("dynamic command returned no output")}
	}
	return executeDialogDynamicMsg{paramIndex: paramIndex, options: filtered}
}

func newExecuteDialogListState(p template.Param) executeDialogListState {
	filter := textinput.New()
	filter.Placeholder = "Filter rows..."
	filter.CharLimit = 256
	filter.Prompt = "filter> "

	state := executeDialogListState{filterInput: filter}
	state.load(p)
	return state
}

func (s *executeDialogListState) load(p template.Param) {
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

func (s *executeDialogListState) focus() {
	s.filterInput.Focus()
}

func (s *executeDialogListState) blur() {
	s.filterInput.Blur()
}

func (s executeDialogListState) hasLoadError() bool {
	return s.loadErrShort != ""
}

func (s executeDialogListState) hasConfirmation() bool {
	return s.confirmValue != ""
}

func (s *executeDialogListState) clearConfirmation() {
	s.confirmValue = ""
}

func (s *executeDialogListState) applyFilter() {
	s.visibleRows = s.visibleRows[:0]
	query := strings.TrimSpace(s.filterInput.Value())
	if query == "" {
		s.visibleRows = append(s.visibleRows, s.allRows...)
	} else {
		matches := fuzzy.FindFrom(query, executeDialogListSource(s.allRows))
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

type executeDialogListSource []parammeta.ListRow

func (rs executeDialogListSource) String(i int) string { return rs[i].Raw }
func (rs executeDialogListSource) Len() int            { return len(rs) }

func (s *executeDialogListState) moveCursor(delta int) {
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

func (s *executeDialogListState) appendNumberSelection(digit rune) {
	s.numberBuffer += string(digit)
	idx, err := strconv.Atoi(s.numberBuffer)
	if err == nil && idx >= 1 && idx <= len(s.visibleRows) {
		s.cursor = idx - 1
		s.parseError = ""
	}
}

func (s *executeDialogListState) deleteNumberSelection() bool {
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

func (s *executeDialogListState) updateFilter(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	s.filterInput, cmd = s.filterInput.Update(msg)
	s.numberBuffer = ""
	s.parseError = ""
	s.applyFilter()
	return cmd
}

func (s *executeDialogListState) confirmSelection(p template.Param) bool {
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

func (s *executeDialogListState) acceptConfirmedValue() string {
	value := s.confirmValue
	s.confirmValue = ""
	s.parseError = ""
	s.numberBuffer = ""
	return value
}

func (s executeDialogListState) emptyMessage() string {
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

func executeDialogUsesNumberSelection(msg tea.KeyMsg, currentFilter string) bool {
	return strings.TrimSpace(currentFilter) == "" && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && unicode.IsDigit(msg.Runes[0])
}

func (d *ExecuteDialogModel) focusParam(index int) {
	if index < 0 || index >= len(d.paramInputs) {
		return
	}
	if d.isListPickerParam(index) {
		d.paramListStates[index].focus()
		d.paramInputs[index].Blur()
		return
	}
	d.paramInputs[index].Focus()
}

func (d *ExecuteDialogModel) blurParam(index int) {
	if index < 0 || index >= len(d.paramInputs) {
		return
	}
	d.paramInputs[index].Blur()
	if d.isListPickerParam(index) {
		d.paramListStates[index].blur()
	}
}

func (d *ExecuteDialogModel) moveFocus(next int) {
	if len(d.paramInputs) == 0 {
		return
	}
	d.blurParam(d.focusedParam)
	d.focusedParam = (next + len(d.paramInputs)) % len(d.paramInputs)
	d.focusParam(d.focusedParam)
}

func (d ExecuteDialogModel) Update(msg tea.Msg) (ExecuteDialogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case executeDialogDynamicMsg:
		return d.handleDynamicResult(msg)
	case tea.KeyMsg:
		if d.phase == phaseActionMenu {
			return d.updateActionMenu(msg)
		}
		return d.updateParamFill(msg)
	}
	return d, nil
}

func (d ExecuteDialogModel) handleDynamicResult(msg executeDialogDynamicMsg) (ExecuteDialogModel, tea.Cmd) {
	i := msg.paramIndex
	if i < 0 || i >= len(d.params) {
		return d, nil
	}

	d.paramLoading[i] = false
	if msg.err != nil || len(msg.options) == 0 {
		d.paramFailed[i] = true
		d.paramInputs[i].Placeholder = d.params[i].Name
		return d, nil
	}

	d.paramOptions[i] = msg.options
	d.paramOptionCursor[i] = 0
	d.paramTypes[i] = template.ParamEnum
	d.paramInputs[i].SetValue(msg.options[0])
	return d, nil
}

func (d ExecuteDialogModel) updateParamFill(msg tea.KeyMsg) (ExecuteDialogModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: dialogExecute, confirmed: false}
		}
	case "tab":
		d.moveFocus(d.focusedParam + 1)
		return d, nil
	case "shift+tab":
		d.moveFocus(d.focusedParam - 1)
		return d, nil
	}

	if d.isListPickerParam(d.focusedParam) {
		return d.updateListParamFill(msg)
	}

	switch msg.String() {
	case "up":
		if d.isListParam(d.focusedParam) {
			opts := d.paramOptions[d.focusedParam]
			if len(opts) > 0 {
				cur := d.paramOptionCursor[d.focusedParam]
				cur = (cur - 1 + len(opts)) % len(opts)
				d.paramOptionCursor[d.focusedParam] = cur
				d.paramInputs[d.focusedParam].SetValue(opts[cur])
			}
			return d, nil
		}
	case "down":
		if d.isListParam(d.focusedParam) {
			opts := d.paramOptions[d.focusedParam]
			if len(opts) > 0 {
				cur := d.paramOptionCursor[d.focusedParam]
				cur = (cur + 1) % len(opts)
				d.paramOptionCursor[d.focusedParam] = cur
				d.paramInputs[d.focusedParam].SetValue(opts[cur])
			}
			return d, nil
		}
	case "enter":
		if d.focusedParam == len(d.paramInputs)-1 || d.allParamsFilled() {
			d.phase = phaseActionMenu
			d.actionCursor = 0
			d.renderedCommand = d.liveRender()
			return d, nil
		}
		d.moveFocus(d.focusedParam + 1)
		return d, nil
	}

	if d.isListParam(d.focusedParam) {
		return d, nil
	}

	var cmd tea.Cmd
	d.paramInputs[d.focusedParam], cmd = d.paramInputs[d.focusedParam].Update(msg)
	d.updateFocusedTextStyle()
	return d, cmd
}

func (d ExecuteDialogModel) updateListParamFill(msg tea.KeyMsg) (ExecuteDialogModel, tea.Cmd) {
	state := &d.paramListStates[d.focusedParam]
	param := d.params[d.focusedParam]

	if state.hasConfirmation() && msg.String() != "enter" {
		state.clearConfirmation()
	}

	switch msg.String() {
	case "up":
		state.moveCursor(-1)
		return d, nil
	case "down":
		state.moveCursor(1)
		return d, nil
	case "enter":
		if state.hasConfirmation() {
			d.paramInputs[d.focusedParam].SetValue(state.acceptConfirmedValue())
			if d.focusedParam == len(d.paramInputs)-1 || d.allParamsFilled() {
				d.phase = phaseActionMenu
				d.actionCursor = 0
				d.renderedCommand = d.liveRender()
				return d, nil
			}
			d.moveFocus(d.focusedParam + 1)
			return d, nil
		}
		state.confirmSelection(param)
		return d, nil
	case "backspace":
		if state.deleteNumberSelection() {
			return d, nil
		}
		cmd := state.updateFilter(msg)
		return d, cmd
	case "d":
		if state.hasLoadError() && state.loadErrDetail != "" {
			state.showErrorDetail = !state.showErrorDetail
		}
		return d, nil
	default:
		if executeDialogUsesNumberSelection(msg, state.filterInput.Value()) {
			state.appendNumberSelection(msg.Runes[0])
			return d, nil
		}
		cmd := state.updateFilter(msg)
		return d, cmd
	}
}

func (d ExecuteDialogModel) updateActionMenu(msg tea.KeyMsg) (ExecuteDialogModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: dialogExecute, confirmed: false}
		}
	case "up", "k":
		if d.actionCursor > 0 {
			d.actionCursor--
		}
		return d, nil
	case "down", "j":
		if d.actionCursor < len(d.actions)-1 {
			d.actionCursor++
		}
		return d, nil
	case "enter":
		switch executeAction(d.actionCursor) {
		case actionCopy:
			return d, func() tea.Msg {
				return dialogResultMsg{
					dtype:     dialogExecute,
					confirmed: true,
					data: map[string]string{
						"action":  "copy",
						"command": d.renderedCommand,
					},
				}
			}
		case actionPasteToPrompt:
			return d, func() tea.Msg {
				return dialogResultMsg{
					dtype:     dialogExecute,
					confirmed: true,
					data: map[string]string{
						"action":  "paste",
						"command": d.renderedCommand,
					},
				}
			}
		default:
			return d, func() tea.Msg {
				return dialogResultMsg{dtype: dialogExecute, confirmed: false}
			}
		}
	}
	return d, nil
}

func (d ExecuteDialogModel) isListParam(i int) bool {
	if i < 0 || i >= len(d.paramTypes) {
		return false
	}
	if d.paramTypes[i] == template.ParamEnum {
		return true
	}
	return d.paramTypes[i] == template.ParamDynamic && !d.paramLoading[i] && !d.paramFailed[i] && len(d.paramOptions[i]) > 0
}

func (d ExecuteDialogModel) isListPickerParam(i int) bool {
	return i >= 0 && i < len(d.paramTypes) && d.paramTypes[i] == template.ParamList
}

func (d ExecuteDialogModel) allParamsFilled() bool {
	for i, input := range d.paramInputs {
		if d.isListPickerParam(i) && d.paramListStates[i].hasLoadError() {
			return false
		}
		if input.Value() == "" {
			return false
		}
	}
	return true
}

func (d *ExecuteDialogModel) updateFocusedTextStyle() {
	if d.focusedParam < 0 || d.focusedParam >= len(d.paramInputs) {
		return
	}
	if d.paramTypes[d.focusedParam] != template.ParamText {
		return
	}
	def := d.params[d.focusedParam].Default
	if def != "" && d.paramInputs[d.focusedParam].Value() == def {
		d.paramInputs[d.focusedParam].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
		return
	}
	d.paramInputs[d.focusedParam].TextStyle = lipgloss.NewStyle()
}

func (d ExecuteDialogModel) liveRender() string {
	values := make(map[string]string)
	for i, p := range d.params {
		v := d.paramInputs[i].Value()
		if v != "" {
			values[p.Name] = v
		}
	}
	return template.Render(d.workflow.Command, values)
}

func (d ExecuteDialogModel) View() string {
	s := d.theme.Styles()
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(d.theme.Colors.Primary)).
		Padding(1, 2).
		Width(d.width)

	title := s.DialogTitle.Render("Execute: " + d.workflow.Name)
	var body string
	if d.phase == phaseActionMenu {
		body = d.viewActionMenu()
	} else {
		body = d.viewParamFill()
	}
	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", body))
}

func (d ExecuteDialogModel) viewParamFill() string {
	s := d.theme.Styles()
	preview := d.renderPreview()

	var rows []string
	rows = append(rows, preview, "")

	for i, p := range d.params {
		isFocused := i == d.focusedParam
		prefix := "  "
		labelStyle := s.Dim
		if isFocused {
			prefix = s.Highlight.Render("❯ ")
			labelStyle = s.Highlight
		}
		label := labelStyle.Render(p.Name + ": ")

		switch {
		case d.paramTypes[i] == template.ParamDynamic && d.paramLoading[i]:
			rows = append(rows, prefix+label+s.Dim.Render("Loading..."))
		case d.paramTypes[i] == template.ParamDynamic && d.paramFailed[i]:
			rows = append(rows, prefix+label+d.paramInputs[i].View()+s.Dim.Render(" (command failed, type manually)"))
		case d.isListParam(i):
			opts := d.paramOptions[i]
			cur := d.paramOptionCursor[i]
			value := ""
			if cur >= 0 && cur < len(opts) {
				value = opts[cur]
			}
			rows = append(rows, prefix+label+s.Highlight.Render(value))
			if isFocused {
				maxShow := 5
				start := 0
				if len(opts) > maxShow {
					start = cur - maxShow/2
					if start < 0 {
						start = 0
					}
					if start+maxShow > len(opts) {
						start = len(opts) - maxShow
					}
				}
				end := start + maxShow
				if end > len(opts) {
					end = len(opts)
				}
				for j := start; j < end; j++ {
					optPrefix := "      "
					optStyle := s.Dim
					if j == cur {
						optPrefix = "    " + s.Highlight.Render("❯ ")
						optStyle = s.Highlight
					}
					rows = append(rows, optPrefix+optStyle.Render(opts[j]))
				}
				if len(opts) > maxShow {
					rows = append(rows, "    "+s.Dim.Render(fmt.Sprintf("(%d/%d)", cur+1, len(opts))))
				}
			}
		case d.isListPickerParam(i):
			state := d.paramListStates[i]
			value := d.paramInputs[i].Value()
			valueStyle := s.Dim
			if value == "" {
				switch {
				case state.hasLoadError():
					value = state.loadErrShort
				case state.hasConfirmation():
					value = state.confirmValue
				default:
					value = "Choose from list"
				}
			} else {
				valueStyle = s.Highlight
			}
			if state.hasConfirmation() {
				valueStyle = s.Highlight
			}
			desc := ""
			if p.Default != "" && value == p.Default {
				desc = s.Dim.Render(" (default)")
			}
			rows = append(rows, prefix+label+valueStyle.Render(value)+desc)
			if isFocused {
				rows = append(rows, d.renderListParamLines(state)...)
			}
		default:
			desc := ""
			if p.Default != "" && d.paramInputs[i].Value() == p.Default {
				desc = s.Dim.Render(" (default)")
			}
			rows = append(rows, prefix+label+d.paramInputs[i].View()+desc)
		}
	}

	if d.isListPickerParam(d.focusedParam) {
		rows = append(rows, "", s.Dim.Render("type to filter  [1-9] jump  [enter] confirm  [tab] next  [esc] cancel"))
	} else {
		rows = append(rows, "", s.Dim.Render("[tab] next  [shift+tab] prev  [up/down] select  [enter] submit  [esc] cancel"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (d ExecuteDialogModel) renderListParamLines(state executeDialogListState) []string {
	s := d.theme.Styles()
	lines := []string{"    " + state.filterInput.View()}

	if state.hasLoadError() {
		lines = append(lines, "    "+s.Dim.Render(state.loadErrShort))
		if state.showErrorDetail && state.loadErrDetail != "" {
			for _, line := range strings.Split(state.loadErrDetail, "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				lines = append(lines, "      "+s.Dim.Render(truncateWithEllipsis(trimmed, executeDialogListPreviewMaxWidth)))
			}
		}
		lines = append(lines, "    "+s.Dim.Render("[d] details  [esc] cancel"))
		return lines
	}

	if state.hasConfirmation() {
		lines = append(lines,
			"    "+s.Highlight.Render("Will insert: ")+s.Highlight.Render(state.confirmValue),
			"    "+s.Dim.Render("[enter] confirm  [any other key] choose again"),
		)
		return lines
	}

	if state.parseError != "" {
		lines = append(lines, "    "+s.Dim.Render(state.parseError))
	}
	if state.numberBuffer != "" {
		lines = append(lines, "    "+s.Dim.Render("row "+state.numberBuffer))
	}
	if len(state.visibleRows) == 0 {
		lines = append(lines, "    "+s.Dim.Render(state.emptyMessage()))
		lines = append(lines, "    "+s.Dim.Render("type to change the filter  [esc] cancel"))
		return lines
	}

	start := 0
	if state.cursor >= executeDialogListVisibleMaxRows {
		start = state.cursor - executeDialogListVisibleMaxRows + 1
	}
	end := start + executeDialogListVisibleMaxRows
	if end > len(state.visibleRows) {
		end = len(state.visibleRows)
	}
	for i := start; i < end; i++ {
		prefix := "      "
		rowStyle := s.Dim
		if i == state.cursor {
			prefix = "    " + s.Highlight.Render("❯ ")
			rowStyle = s.Highlight
		}
		label := fmt.Sprintf("%d. %s", i+1, truncateWithEllipsis(state.visibleRows[i].Raw, executeDialogListPreviewMaxWidth))
		lines = append(lines, prefix+rowStyle.Render(label))
	}
	if len(state.visibleRows) > executeDialogListVisibleMaxRows {
		lines = append(lines, "    "+s.Dim.Render(fmt.Sprintf("showing %d/%d visible rows", end-start, len(state.visibleRows))))
	}
	return lines
}

func (d ExecuteDialogModel) renderPreview() string {
	s := d.theme.Styles()
	label := s.Dim.Render("Command preview")
	command := d.liveRender()
	command = strings.ReplaceAll(command, "\n", " ")
	commandWidth := d.width - 10
	if commandWidth < 20 {
		commandWidth = 20
	}
	lines := wrapPreviewCommand(command, commandWidth, executePreviewMaxCommandRows)

	rows := []string{label}
	for _, line := range lines {
		rows = append(rows, "  "+s.Dim.Render(line))
	}
	for len(rows) < executePreviewMinRows {
		rows = append(rows, "")
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func wrapPreviewCommand(command string, width, maxLines int) []string {
	if maxLines <= 0 {
		return []string{""}
	}

	runes := []rune(command)
	if len(runes) == 0 {
		return []string{""}
	}

	if width <= 0 {
		width = 1
	}

	lines := make([]string, 0, maxLines)
	for len(runes) > 0 && len(lines) < maxLines {
		take := width
		if len(runes) < take {
			take = len(runes)
		}
		line := string(runes[:take])
		runes = runes[take:]

		if len(runes) > 0 && len(lines) == maxLines-1 {
			if width <= 1 {
				line = "…"
			} else {
				line = string([]rune(line)[:width-1]) + "…"
			}
			runes = nil
		}
		lines = append(lines, line)
	}

	return lines
}

func truncateWithEllipsis(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(r[:maxLen-1]) + "…"
}

func (d ExecuteDialogModel) viewActionMenu() string {
	s := d.theme.Styles()
	rows := []string{
		s.Highlight.Render(d.renderedCommand),
		"",
	}

	for i, action := range d.actions {
		if i == d.actionCursor {
			rows = append(rows, s.Selected.Render("❯ "+action))
		} else {
			rows = append(rows, "  "+action)
		}
	}
	rows = append(rows, "", s.Dim.Render("[up/down] select  [enter] confirm  [esc] cancel"))
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
