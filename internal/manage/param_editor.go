package manage

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/store"
)

// paramEntry represents a single parameter in the editor.
type paramEntry struct {
	name        string
	paramType   string // "text", "enum", "dynamic"
	defaultVal  string
	options     []string
	dynamicCmd  string
	description string
	expanded    bool
	nameInput   textinput.Model
}

// ParamEditorModel is a custom Bubble Tea model for parameter CRUD with
// accordion expand/collapse. It replaces the flat huh form for parameters.
type ParamEditorModel struct {
	params        []paramEntry
	cursor        int  // which param row is focused
	editing       bool // whether expanded param's name field is in edit mode
	onAddButton   bool // whether cursor is on the "+ Add Parameter" row
	confirmDelete int  // index of param pending delete (-1 = none)

	theme Theme
	width int
}

// NewParamEditor creates a ParamEditorModel from existing workflow args.
func NewParamEditor(args []store.Arg, theme Theme, width int) ParamEditorModel {
	m := ParamEditorModel{
		confirmDelete: -1,
		theme:         theme,
		width:         width,
	}

	if len(args) == 0 {
		m.onAddButton = true
		return m
	}

	m.params = make([]paramEntry, len(args))
	for i, arg := range args {
		ti := textinput.New()
		ti.SetValue(arg.Name)
		ti.CharLimit = 64
		ti.Placeholder = "param_name"

		pt := arg.Type
		if pt == "" {
			pt = "text"
		}

		m.params[i] = paramEntry{
			name:        arg.Name,
			paramType:   pt,
			defaultVal:  arg.Default,
			options:     arg.Options,
			dynamicCmd:  arg.DynamicCmd,
			description: arg.Description,
			nameInput:   ti,
		}
	}

	return m
}

// Init returns the initial command for the param editor.
func (m ParamEditorModel) Init() tea.Cmd {
	return nil
}

// Update processes messages for the param editor.
func (m ParamEditorModel) Update(msg tea.Msg) (ParamEditorModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages to active name input if editing.
		if m.editing && m.cursor >= 0 && m.cursor < len(m.params) && m.params[m.cursor].expanded {
			var cmd tea.Cmd
			m.params[m.cursor].nameInput, cmd = m.params[m.cursor].nameInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Handle delete confirmation mode.
	if m.confirmDelete >= 0 {
		return m.updateConfirmDelete(keyMsg)
	}

	// Handle editing mode (name input focused).
	if m.editing {
		return m.updateEditing(keyMsg)
	}

	// Handle navigation and actions.
	return m.updateNavigation(keyMsg)
}

// updateConfirmDelete handles y/n input during delete confirmation.
func (m ParamEditorModel) updateConfirmDelete(msg tea.KeyMsg) (ParamEditorModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		idx := m.confirmDelete
		m.confirmDelete = -1
		// Remove the param.
		m.params = append(m.params[:idx], m.params[idx+1:]...)
		// Adjust cursor.
		if len(m.params) == 0 {
			m.cursor = 0
			m.onAddButton = true
		} else if m.cursor >= len(m.params) {
			m.cursor = len(m.params) - 1
		}
	case "n", "N", "esc":
		m.confirmDelete = -1
	}
	return m, nil
}

// updateEditing handles key events while editing the name field.
func (m ParamEditorModel) updateEditing(msg tea.KeyMsg) (ParamEditorModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Commit the name edit.
		m.params[m.cursor].name = m.params[m.cursor].nameInput.Value()
		m.params[m.cursor].nameInput.Blur()
		m.editing = false
		return m, nil
	case "esc":
		// Cancel the name edit — restore original name.
		m.params[m.cursor].nameInput.SetValue(m.params[m.cursor].name)
		m.params[m.cursor].nameInput.Blur()
		m.editing = false
		return m, nil
	}

	// Forward to the text input.
	var cmd tea.Cmd
	m.params[m.cursor].nameInput, cmd = m.params[m.cursor].nameInput.Update(msg)
	return m, cmd
}

// updateNavigation handles normal navigation and action keys.
func (m ParamEditorModel) updateNavigation(msg tea.KeyMsg) (ParamEditorModel, tea.Cmd) {
	totalItems := len(m.params) + 1 // params + add button

	switch msg.String() {
	case "up", "k":
		if m.onAddButton {
			if len(m.params) > 0 {
				m.onAddButton = false
				m.cursor = len(m.params) - 1
			}
		} else if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.onAddButton {
			return m, nil
		}
		if m.cursor < len(m.params)-1 {
			m.cursor++
		} else {
			m.onAddButton = true
		}
		return m, nil

	case "enter":
		if m.onAddButton {
			return m.addParam()
		}
		// Toggle expand/collapse on the current param.
		return m.toggleExpand(), nil

	case "ctrl+n":
		return m.addParam()

	case "ctrl+d":
		if !m.onAddButton && len(m.params) > 0 && m.cursor < len(m.params) {
			m.confirmDelete = m.cursor
		}
		return m, nil

	case "tab":
		// If a param is expanded, start editing its name field.
		if !m.onAddButton && m.cursor < len(m.params) && m.params[m.cursor].expanded {
			m.editing = true
			m.params[m.cursor].nameInput.Focus()
			return m, textinput.Blink
		}
		// Otherwise, move to next item.
		if m.onAddButton {
			// Tab past the add button — let the parent handle it.
			return m, nil
		}
		if m.cursor < len(m.params)-1 {
			m.cursor++
		} else {
			m.onAddButton = true
		}
		return m, nil
	}

	_ = totalItems
	return m, nil
}

// addParam appends a new parameter and focuses its name field.
func (m ParamEditorModel) addParam() (ParamEditorModel, tea.Cmd) {
	// Collapse any currently expanded param.
	for i := range m.params {
		m.params[i].expanded = false
	}

	ti := textinput.New()
	ti.SetValue("new_param")
	ti.CharLimit = 64
	ti.Placeholder = "param_name"
	ti.Focus()
	// Select all text so user can immediately type a new name.
	ti.CursorEnd()

	entry := paramEntry{
		name:      "new_param",
		paramType: "text",
		expanded:  true,
		nameInput: ti,
	}

	m.params = append(m.params, entry)
	m.cursor = len(m.params) - 1
	m.onAddButton = false
	m.editing = true

	return m, textinput.Blink
}

// toggleExpand expands/collapses the param at cursor (accordion style).
func (m ParamEditorModel) toggleExpand() ParamEditorModel {
	if m.cursor < 0 || m.cursor >= len(m.params) {
		return m
	}

	wasExpanded := m.params[m.cursor].expanded

	// Collapse all.
	for i := range m.params {
		m.params[i].expanded = false
		m.params[i].nameInput.Blur()
	}
	m.editing = false

	// If it was collapsed, expand it.
	if !wasExpanded {
		m.params[m.cursor].expanded = true
	}

	return m
}

// View renders the param list.
func (m ParamEditorModel) View() string {
	s := m.theme.Styles()

	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Colors.Primary))
	dimStyle := s.Dim
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	accentBg := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Primary)).
		Bold(true)

	var rows []string

	if len(m.params) == 0 && m.confirmDelete < 0 {
		rows = append(rows, dimStyle.Render("  No parameters"))
	}

	for i, p := range m.params {
		isFocused := !m.onAddButton && i == m.cursor

		// Delete confirmation row.
		if m.confirmDelete == i {
			prompt := errorStyle.Render(fmt.Sprintf("  Delete '%s'? ", p.name)) +
				dimStyle.Render("y/n")
			rows = append(rows, prompt)
			continue
		}

		if p.expanded {
			// Expanded row: ▾ name (type)
			arrow := "▾"
			nameDisplay := primaryStyle.Render(p.name)
			typeDisplay := dimStyle.Render("(" + p.paramType + ")")

			if isFocused {
				arrow = accentBg.Render("▾")
			}

			header := fmt.Sprintf("  %s %s  %s", arrow, nameDisplay, typeDisplay)
			rows = append(rows, header)

			// Indented fields.
			if m.editing && i == m.cursor {
				rows = append(rows, "    Name: "+m.params[i].nameInput.View())
			} else {
				rows = append(rows, "    "+dimStyle.Render("Name: ")+primaryStyle.Render(p.name))
			}
			rows = append(rows, "    "+dimStyle.Render("Type: ")+p.paramType)
			if p.defaultVal != "" {
				rows = append(rows, "    "+dimStyle.Render("Default: ")+p.defaultVal)
			}
			if p.description != "" {
				rows = append(rows, "    "+dimStyle.Render("Desc: ")+p.description)
			}
			if len(p.options) > 0 {
				rows = append(rows, "    "+dimStyle.Render("Options: ")+fmt.Sprintf("%v", p.options))
			}
			if p.dynamicCmd != "" {
				rows = append(rows, "    "+dimStyle.Render("Dynamic: ")+p.dynamicCmd)
			}
		} else {
			// Collapsed row: ▸ name (type)
			arrow := "▸"
			nameDisplay := p.name
			typeDisplay := dimStyle.Render("(" + p.paramType + ")")

			if isFocused {
				nameDisplay = accentBg.Render(p.name)
				arrow = accentBg.Render("▸")
			}

			row := fmt.Sprintf("  %s %s  %s", arrow, nameDisplay, typeDisplay)
			rows = append(rows, row)
		}
	}

	// Add Parameter button row.
	addLabel := dimStyle.Render("  + Add Parameter")
	if m.onAddButton {
		addLabel = accentBg.Render("  + Add Parameter")
	}
	rows = append(rows, addLabel)

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// ToArgs converts the current param entries back to store.Arg for persistence.
func (m ParamEditorModel) ToArgs() []store.Arg {
	if len(m.params) == 0 {
		return nil
	}

	args := make([]store.Arg, len(m.params))
	for i, p := range m.params {
		args[i] = store.Arg{
			Name:        p.name,
			Default:     p.defaultVal,
			Description: p.description,
			Type:        p.paramType,
			Options:     p.options,
			DynamicCmd:  p.dynamicCmd,
		}
	}
	return args
}

// SetWidth updates the layout width.
func (m *ParamEditorModel) SetWidth(w int) {
	m.width = w
}

// ParamCount returns the number of parameters.
func (m ParamEditorModel) ParamCount() int {
	return len(m.params)
}

// Focused returns whether the param editor should be considered focused.
// The parent form uses this to determine key routing.
func (m ParamEditorModel) Focused() bool {
	return m.editing
}
