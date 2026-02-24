package picker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	m.params = template.ExtractParams(m.selected.Command)
	n := len(m.params)
	m.paramInputs = make([]textinput.Model, n)
	m.paramTypes = make([]template.ParamType, n)
	m.paramOptions = make([][]string, n)
	m.paramOptionCursor = make([]int, n)
	m.paramLoading = make([]bool, n)
	m.paramFailed = make([]bool, n)

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

		default: // ParamText
			if p.Default != "" {
				ti.SetValue(p.Default)
			}
		}

		if i == 0 {
			ti.Focus()
		}
		m.paramInputs[i] = ti
	}
	m.focusedParam = 0
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

	var options []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			options = append(options, line)
		}
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
		m.paramInputs[m.focusedParam].Blur()
		m.focusedParam = (m.focusedParam + 1) % len(m.paramInputs)
		m.paramInputs[m.focusedParam].Focus()
		return m, nil

	case "shift+tab":
		m.paramInputs[m.focusedParam].Blur()
		m.focusedParam = (m.focusedParam - 1 + len(m.paramInputs)) % len(m.paramInputs)
		m.paramInputs[m.focusedParam].Focus()
		return m, nil

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
		m.paramInputs[m.focusedParam].Blur()
		m.focusedParam++
		m.paramInputs[m.focusedParam].Focus()
		return m, nil

	default:
		// For enum/dynamic params with options, ignore text input
		if m.isListParam(m.focusedParam) {
			return m, nil
		}
		// For text params (and failed dynamic params), normal text input
		var cmd tea.Cmd
		m.paramInputs[m.focusedParam], cmd = m.paramInputs[m.focusedParam].Update(msg)
		return m, cmd
	}
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
	for _, input := range m.paramInputs {
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
		if m.isListParam(i) {
			hasListParam = true
			break
		}
	}
	if hasListParam {
		sections = append(sections, hintStyle.Render("  ↑↓ select option  tab next  shift+tab prev  enter paste to shell  esc cancel"))
	} else {
		sections = append(sections, hintStyle.Render("  tab next  shift+tab prev  enter paste to shell  esc cancel"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
