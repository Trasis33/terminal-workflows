package picker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	parammeta "github.com/fredriklanga/wf/internal/params"
	"github.com/fredriklanga/wf/internal/template"
)

// dynamicResultMsg is sent when a dynamic parameter's shell command completes.
type dynamicResultMsg struct {
	paramIndex int
	options    []string
	err        error
}

// initParamFill prepares the parameter fill state from the selected workflow.
func initParamFill(m *Model) {
	m.params = parammeta.OverlayMetadata(m.selected.Command, m.selected.Args)

	n := len(m.params)
	m.paramInputs = make([]textinput.Model, n)
	m.paramTypes = make([]template.ParamType, n)
	m.paramOptions = make([][]string, n)
	m.paramOptionCursor = make([]int, n)
	m.paramLoading = make([]bool, n)
	m.paramFailed = make([]bool, n)
	m.paramListStates = make([]listPickerState, n)

	for i, p := range m.params {
		ti := textinput.New()
		ti.Placeholder = p.Name
		ti.CharLimit = 256

		m.paramTypes[i] = p.Type

		switch p.Type {
		case template.ParamEnum:
			m.paramOptions[i] = p.Options
			// Find default in options and set cursor
			defaultIdx := 0
			if p.Default != "" {
				for j, opt := range p.Options {
					if opt == p.Default {
						defaultIdx = j
						break
					}
				}
			}
			m.paramOptionCursor[i] = defaultIdx
			if len(p.Options) > 0 {
				ti.SetValue(p.Options[defaultIdx])
			}
			// Make textinput read-only for enum by setting width to 0
			// The value is set programmatically from option selection
			ti.Placeholder = ""

		case template.ParamDynamic:
			m.paramLoading[i] = true
			ti.Placeholder = "Loading..."

		case template.ParamList:
			m.paramListStates[i] = newListPickerState(p)
			if p.Default != "" {
				ti.SetValue(p.Default)
				ti.TextStyle = defaultTextStyle
				ti.CursorEnd()
			}
			ti.Placeholder = "Choose from list"

		default: // ParamText
			if p.Default != "" {
				ti.SetValue(p.Default)
				ti.TextStyle = defaultTextStyle
				ti.CursorEnd()
			}
		}

		m.paramInputs[i] = ti
	}
	m.focusedParam = 0
	m.focusParam(0)
}

// initParamFillCmds returns tea.Cmds to execute dynamic parameter commands.
// Must be called after initParamFill, from an Update that returns Cmds.
func initParamFillCmds(m *Model) []tea.Cmd {
	var cmds []tea.Cmd
	for i, p := range m.params {
		if p.Type == template.ParamDynamic {
			idx := i
			dynCmd := p.DynamicCmd
			cmds = append(cmds, func() tea.Msg {
				return executeDynamic(idx, dynCmd)
			})
		}
	}
	return cmds
}

// executeDynamic runs a shell command with a 5-second timeout and returns options.
func executeDynamic(paramIndex int, command string) dynamicResultMsg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return dynamicResultMsg{
			paramIndex: paramIndex,
			err:        fmt.Errorf("dynamic command failed: %w", err),
		}
	}

	options := strings.FieldsFunc(string(output), func(r rune) bool { return r == '\n' || r == '\r' })
	for i := range options {
		options[i] = strings.TrimSpace(options[i])
	}

	if len(options) == 0 {
		return dynamicResultMsg{
			paramIndex: paramIndex,
			err:        fmt.Errorf("dynamic command returned no output"),
		}
	}

	return dynamicResultMsg{
		paramIndex: paramIndex,
		options:    options,
	}
}

func (m *Model) focusParam(index int) {
	if index < 0 || index >= len(m.paramInputs) {
		return
	}
	if m.isListPickerParam(index) {
		m.paramListStates[index].focus()
		m.paramInputs[index].Blur()
		return
	}
	m.paramInputs[index].Focus()
}

func (m *Model) blurParam(index int) {
	if index < 0 || index >= len(m.paramInputs) {
		return
	}
	m.paramInputs[index].Blur()
	if m.isListPickerParam(index) {
		m.paramListStates[index].blur()
	}
}

func (m *Model) moveFocus(next int) {
	if len(m.paramInputs) == 0 {
		return
	}
	m.blurParam(m.focusedParam)
	m.focusedParam = (next + len(m.paramInputs)) % len(m.paramInputs)
	m.focusParam(m.focusedParam)
}

// liveRender builds the command preview with current input values.
func liveRender(m *Model) string {
	values := make(map[string]string)
	for i, p := range m.params {
		v := m.paramInputs[i].Value()
		if v != "" {
			values[p.Name] = v
		}
	}
	return template.Render(m.selected.Command, values)
}

// updateParamFill handles key events in the parameter fill state.
func (m Model) updateParamFill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.Result = ""
		return m, tea.Quit

	case "tab":
		m.moveFocus(m.focusedParam + 1)
		return m, nil

	case "shift+tab":
		m.moveFocus(m.focusedParam - 1)
		return m, nil
	}

	if m.isListPickerParam(m.focusedParam) {
		return m.updateFocusedListParam(msg)
	}

	switch msg.String() {
	case "up":
		// For enum/dynamic params, cycle options upward
		if m.isListParam(m.focusedParam) {
			opts := m.paramOptions[m.focusedParam]
			if len(opts) > 0 {
				cur := m.paramOptionCursor[m.focusedParam]
				cur = (cur - 1 + len(opts)) % len(opts)
				m.paramOptionCursor[m.focusedParam] = cur
				m.paramInputs[m.focusedParam].SetValue(opts[cur])
			}
			return m, nil
		}
		// For text params, pass through to textinput
		var cmd tea.Cmd
		m.paramInputs[m.focusedParam], cmd = m.paramInputs[m.focusedParam].Update(msg)
		m.updateFocusedTextStyle()
		return m, cmd

	case "down":
		// For enum/dynamic params, cycle options downward
		if m.isListParam(m.focusedParam) {
			opts := m.paramOptions[m.focusedParam]
			if len(opts) > 0 {
				cur := m.paramOptionCursor[m.focusedParam]
				cur = (cur + 1) % len(opts)
				m.paramOptionCursor[m.focusedParam] = cur
				m.paramInputs[m.focusedParam].SetValue(opts[cur])
			}
			return m, nil
		}
		// For text params, pass through to textinput
		var cmd tea.Cmd
		m.paramInputs[m.focusedParam], cmd = m.paramInputs[m.focusedParam].Update(msg)
		m.updateFocusedTextStyle()
		return m, cmd

	case "enter":
		// If on last param or all filled, render and quit
		if m.focusedParam == len(m.paramInputs)-1 || allParamsFilled(m) {
			values := make(map[string]string)
			for i, p := range m.params {
				v := m.paramInputs[i].Value()
				if v != "" {
					values[p.Name] = v
				}
			}
			m.Result = template.Render(m.selected.Command, values)
			return m, tea.Quit
		}
		// Otherwise advance to next param (same as tab)
		m.moveFocus(m.focusedParam + 1)
		return m, nil

	default:
		// For enum/dynamic params with options, ignore text input
		if m.isListParam(m.focusedParam) {
			return m, nil
		}
		// For text params (and failed dynamic params), normal text input
		var cmd tea.Cmd
		m.paramInputs[m.focusedParam], cmd = m.paramInputs[m.focusedParam].Update(msg)
		m.updateFocusedTextStyle()
		return m, cmd
	}
}

func (m Model) updateFocusedListParam(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := &m.paramListStates[m.focusedParam]
	param := m.params[m.focusedParam]

	if state.hasConfirmation() && msg.String() != "enter" {
		state.clearConfirmation()
	}

	switch msg.String() {
	case "up":
		state.moveCursor(-1)
		return m, nil
	case "down":
		state.moveCursor(1)
		return m, nil
	case "enter":
		if state.hasConfirmation() {
			m.paramInputs[m.focusedParam].SetValue(state.acceptConfirmedValue())
			if m.focusedParam == len(m.paramInputs)-1 || allParamsFilled(m) {
				values := make(map[string]string)
				for i, p := range m.params {
					v := m.paramInputs[i].Value()
					if v != "" {
						values[p.Name] = v
					}
				}
				m.Result = template.Render(m.selected.Command, values)
				return m, tea.Quit
			}
			m.moveFocus(m.focusedParam + 1)
			return m, nil
		}
		state.confirmSelection(param)
		return m, nil
	case "backspace":
		if state.deleteNumberSelection() {
			return m, nil
		}
		cmd := state.updateFilter(msg)
		return m, cmd
	case "d":
		if state.hasLoadError() && state.loadErrDetail != "" {
			state.showErrorDetail = !state.showErrorDetail
		}
		return m, nil
	default:
		if listPickerUsesNumberSelection(msg, state.filterInput.Value()) {
			state.appendNumberSelection(msg.Runes[0])
			return m, nil
		}
		cmd := state.updateFilter(msg)
		return m, cmd
	}
}

func (m *Model) updateFocusedTextStyle() {
	if m.focusedParam < 0 || m.focusedParam >= len(m.paramInputs) {
		return
	}
	if m.paramTypes[m.focusedParam] != template.ParamText {
		return
	}
	def := m.params[m.focusedParam].Default
	if def != "" && m.paramInputs[m.focusedParam].Value() == def {
		m.paramInputs[m.focusedParam].TextStyle = defaultTextStyle
		return
	}
	m.paramInputs[m.focusedParam].TextStyle = normalStyle
}

func (m Model) isListPickerParam(i int) bool {
	return i >= 0 && i < len(m.paramTypes) && m.paramTypes[i] == template.ParamList
}

// isListParam returns true if the param at index i has a selectable option list.
// This is true for enum params and for dynamic params that loaded successfully.
func (m Model) isListParam(i int) bool {
	if i < 0 || i >= len(m.paramTypes) {
		return false
	}
	if m.paramTypes[i] == template.ParamEnum {
		return true
	}
	if m.paramTypes[i] == template.ParamDynamic && !m.paramFailed[i] && !m.paramLoading[i] && len(m.paramOptions[i]) > 0 {
		return true
	}
	return false
}

// allParamsFilled returns true if every parameter input has a non-empty value.
func allParamsFilled(m Model) bool {
	for i, input := range m.paramInputs {
		if m.isListPickerParam(i) && m.paramListStates[i].hasLoadError() {
			return false
		}
		if input.Value() == "" {
			return false
		}
	}
	return true
}

// viewParamFill renders the parameter fill state.
func (m Model) viewParamFill() string {
	var sections []string

	// Header: workflow name
	sections = append(sections, selectedStyle.Render("  "+m.selected.Name))
	sections = append(sections, "")

	// Live-rendered command
	rendered := liveRender(&m)
	sections = append(sections, previewBorderStyle.Render(rendered))
	sections = append(sections, "")

	// Parameter inputs
	for i, p := range m.params {
		isFocused := i == m.focusedParam

		var style lipgloss.Style
		if isFocused {
			style = paramActiveStyle
		} else {
			style = paramLabelStyle
		}

		prefix := "  "
		if isFocused {
			prefix = cursorStyle.Render("❯ ")
		}

		label := p.Name

		switch {
		case m.paramTypes[i] == template.ParamDynamic && m.paramLoading[i]:
			// Dynamic param still loading
			loadingText := dimStyle.Render("Loading... (" + p.DynamicCmd + ")")
			row := prefix + style.Render(label+": ") + loadingText
			sections = append(sections, row)

		case m.paramTypes[i] == template.ParamDynamic && m.paramFailed[i]:
			// Dynamic param failed — show error and text input
			inputView := m.paramInputs[i].View()
			errNote := dimStyle.Render(" (command failed, type manually)")
			row := prefix + style.Render(label+": ") + inputView + errNote
			sections = append(sections, row)

		case m.isListParam(i):
			// Enum or successful dynamic — show option list
			opts := m.paramOptions[i]
			cur := m.paramOptionCursor[i]

			// Show selected value on the label line
			selectedVal := ""
			if cur >= 0 && cur < len(opts) {
				selectedVal = opts[cur]
			}

			var descStr string
			if p.Default != "" && selectedVal == p.Default {
				descStr = dimStyle.Render(" (default)")
			}

			row := prefix + style.Render(label+": ") + highlightStyle.Render(selectedVal) + descStr
			sections = append(sections, row)

			// Show option list when focused
			if isFocused {
				// Show up to 5 options with scroll
				maxShow := 5
				start := 0
				if len(opts) > maxShow {
					// Center the cursor in the view
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
					if j == cur {
						optPrefix = "    " + cursorStyle.Render("❯ ")
					}
					optStyle := dimStyle
					if j == cur {
						optStyle = highlightStyle
					}
					sections = append(sections, optPrefix+optStyle.Render(opts[j]))
				}

				if len(opts) > maxShow {
					scrollInfo := fmt.Sprintf("    %s", dimStyle.Render(fmt.Sprintf("(%d/%d)", cur+1, len(opts))))
					sections = append(sections, scrollInfo)
				}
			}

		case m.isListPickerParam(i):
			state := m.paramListStates[i]
			selectedVal := m.paramInputs[i].Value()
			valueStyle := dimStyle
			if selectedVal == "" {
				switch {
				case state.hasLoadError():
					selectedVal = state.loadErrShort
				case state.hasConfirmation():
					selectedVal = state.confirmValue
				default:
					selectedVal = "Choose from list"
				}
			} else {
				valueStyle = highlightStyle
			}
			if state.hasConfirmation() {
				valueStyle = highlightStyle
			}

			var descStr string
			if p.Default != "" && selectedVal == p.Default {
				descStr = dimStyle.Render(" (default)")
			}

			row := prefix + style.Render(label+": ") + valueStyle.Render(selectedVal) + descStr
			sections = append(sections, row)
			if isFocused {
				sections = append(sections, state.renderLines()...)
			}

		default:
			// Text param — unchanged
			inputView := m.paramInputs[i].View()
			var descStr string
			if p.Default != "" && m.paramInputs[i].Value() == p.Default {
				descStr = dimStyle.Render(" (default)")
			}
			row := prefix + style.Render(label+": ") + inputView + descStr
			sections = append(sections, row)
		}
	}

	sections = append(sections, "")

	// Footer hints
	hasListParam := false
	for i := range m.paramTypes {
		if m.isListParam(i) || m.isListPickerParam(i) {
			hasListParam = true
			break
		}
	}
	if m.isListPickerParam(m.focusedParam) {
		sections = append(sections, hintStyle.Render("  type to filter  1-9 select row  enter confirm  tab next  esc cancel"))
	} else if hasListParam {
		sections = append(sections, hintStyle.Render("  ↑↓ select option  tab next  shift+tab prev  enter paste to shell  esc cancel"))
	} else {
		sections = append(sections, hintStyle.Render("  tab next  shift+tab prev  enter paste to shell  esc cancel"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
