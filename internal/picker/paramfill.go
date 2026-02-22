package picker

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/template"
)

// initParamFill prepares the parameter fill state from the selected workflow.
func initParamFill(m *Model) {
	m.params = template.ExtractParams(m.selected.Command)
	m.paramInputs = make([]textinput.Model, len(m.params))
	for i, p := range m.params {
		ti := textinput.New()
		ti.Placeholder = p.Name
		ti.CharLimit = 256
		if p.Default != "" {
			ti.SetValue(p.Default)
		}
		if i == 0 {
			ti.Focus()
		}
		m.paramInputs[i] = ti
	}
	m.focusedParam = 0
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
		var cmd tea.Cmd
		m.paramInputs[m.focusedParam], cmd = m.paramInputs[m.focusedParam].Update(msg)
		return m, cmd
	}
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
		label := p.Name
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

		inputView := m.paramInputs[i].View()

		var descStr string
		if p.Default != "" && m.paramInputs[i].Value() == p.Default {
			descStr = dimStyle.Render(" (default)")
		}

		row := prefix + style.Render(label+": ") + inputView + descStr
		sections = append(sections, row)
	}

	sections = append(sections, "")

	// Footer hints
	sections = append(sections, hintStyle.Render("  tab next  shift+tab prev  enter paste to shell  esc cancel"))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
