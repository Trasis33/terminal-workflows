package manage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/store"
)

// paramRenamedMsg is emitted when a parameter is renamed, so the parent form
// can update command template references in real-time.
type paramRenamedMsg struct {
	OldName string
	NewName string
}

// paramTypeChangedMsg is emitted when a parameter's type changes.
type paramTypeChangedMsg struct {
	Index   int
	OldType string
	NewType string
}

// Sub-field indices within an expanded parameter.
const (
	subFieldName       = iota // Name textinput
	subFieldType              // Type selector (left/right)
	subFieldDefault           // Default value textinput
	subFieldOptions           // Enum options textinput (only for enum type)
	subFieldDynamicCmd        // Dynamic command textinput (only for dynamic type)
)

// availableTypes lists the parameter types in selector order.
var availableTypes = []string{"text", "enum", "dynamic", "list"}

// paramEntry represents a single parameter in the editor.
type paramEntry struct {
	name        string
	paramType   string // "text", "enum", "dynamic", "list"
	defaultVal  string
	options     []string
	dynamicCmd  string
	description string
	expanded    bool

	// Input widgets for editable fields.
	nameInput       textinput.Model
	defaultInput    textinput.Model
	optionsInput    textinput.Model // comma-separated enum options
	dynamicCmdInput textinput.Model

	// Which sub-field has focus within the expanded view.
	focusedField int

	// Validation error for inline display.
	nameErr string
}

// ParamEditorModel is a custom Bubble Tea model for parameter CRUD with
// accordion expand/collapse. It replaces the flat huh form for parameters.
type ParamEditorModel struct {
	params        []paramEntry
	cursor        int  // which param row is focused
	editing       bool // whether an expanded param's sub-field is in edit mode
	onAddButton   bool // whether cursor is on the "+ Add Parameter" row
	confirmDelete int  // index of param pending delete (-1 = none)

	// Ghost text for AI suggestions on param sub-fields (field key → value).
	ghostText map[string]string

	// Ghost params — suggested by AI but not yet accepted.
	ghostParams []store.Arg

	theme Theme
	width int
}

// NewParamEditor creates a ParamEditorModel from existing workflow args.
func NewParamEditor(args []store.Arg, theme Theme, width int) ParamEditorModel {
	m := ParamEditorModel{
		confirmDelete: -1,
		theme:         theme,
		width:         width,
		ghostText:     make(map[string]string),
	}

	if len(args) == 0 {
		m.onAddButton = true
		return m
	}

	m.params = make([]paramEntry, len(args))
	for i, arg := range args {
		m.params[i] = newParamEntry(arg)
	}

	return m
}

// newParamEntry creates a paramEntry from a store.Arg with all input widgets initialized.
func newParamEntry(arg store.Arg) paramEntry {
	ti := textinput.New()
	ti.SetValue(arg.Name)
	ti.CharLimit = 64
	ti.Placeholder = "param_name"

	pt := arg.Type
	if pt == "" {
		pt = "text"
	}

	defInput := textinput.New()
	defInput.SetValue(arg.Default)
	defInput.CharLimit = 256
	defInput.Placeholder = "default value"

	optInput := textinput.New()
	optInput.SetValue(strings.Join(arg.Options, ", "))
	optInput.CharLimit = 512
	optInput.Placeholder = "opt1, opt2, *default_opt"

	dynInput := textinput.New()
	dynInput.SetValue(arg.DynamicCmd)
	dynInput.CharLimit = 512
	dynInput.Placeholder = "e.g., git branch --list"

	return paramEntry{
		name:            arg.Name,
		paramType:       pt,
		defaultVal:      arg.Default,
		options:         arg.Options,
		dynamicCmd:      arg.DynamicCmd,
		description:     arg.Description,
		nameInput:       ti,
		defaultInput:    defInput,
		optionsInput:    optInput,
		dynamicCmdInput: dynInput,
	}
}

// Init returns the initial command for the param editor.
func (m ParamEditorModel) Init() tea.Cmd {
	return nil
}

// Update processes messages for the param editor.
func (m ParamEditorModel) Update(msg tea.Msg) (ParamEditorModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages to the active input widget if editing.
		if m.editing && m.cursor >= 0 && m.cursor < len(m.params) && m.params[m.cursor].expanded {
			return m.forwardToActiveInput(msg)
		}
		return m, nil
	}

	// Handle delete confirmation mode.
	if m.confirmDelete >= 0 {
		return m.updateConfirmDelete(keyMsg)
	}

	// Handle editing mode (a sub-field is focused and editable).
	if m.editing {
		return m.updateEditing(keyMsg)
	}

	// Handle navigation and actions.
	return m.updateNavigation(keyMsg)
}

// forwardToActiveInput sends non-key messages to the currently focused input widget.
func (m ParamEditorModel) forwardToActiveInput(msg tea.Msg) (ParamEditorModel, tea.Cmd) {
	p := &m.params[m.cursor]
	var cmd tea.Cmd

	switch p.focusedField {
	case subFieldName:
		p.nameInput, cmd = p.nameInput.Update(msg)
	case subFieldDefault:
		p.defaultInput, cmd = p.defaultInput.Update(msg)
	case subFieldOptions:
		p.optionsInput, cmd = p.optionsInput.Update(msg)
	case subFieldDynamicCmd:
		p.dynamicCmdInput, cmd = p.dynamicCmdInput.Update(msg)
	}

	return m, cmd
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

// updateEditing handles key events while editing a sub-field.
func (m ParamEditorModel) updateEditing(msg tea.KeyMsg) (ParamEditorModel, tea.Cmd) {
	p := &m.params[m.cursor]
	key := msg.String()

	// Type selector uses left/right instead of text input.
	if p.focusedField == subFieldType {
		return m.updateTypeSelector(msg)
	}

	switch key {
	case "tab":
		// Commit current field, move to next sub-field.
		m.commitCurrentField()
		return m.advanceSubField(1)

	case "shift+tab":
		// Commit current field, move to previous sub-field.
		m.commitCurrentField()
		return m.advanceSubField(-1)

	case "enter":
		// Commit the current field edit.
		m.commitCurrentField()
		m.blurAllSubFields()
		m.editing = false
		return m, nil

	case "esc":
		// Cancel the current field edit — restore original values.
		m.cancelCurrentField()
		m.blurAllSubFields()
		m.editing = false
		return m, nil
	}

	// Forward to the active text input.
	var cmd tea.Cmd
	switch p.focusedField {
	case subFieldName:
		p.nameInput, cmd = p.nameInput.Update(msg)
		// Live rename: emit paramRenamedMsg on each keystroke.
		newName := p.nameInput.Value()
		if newName != p.name {
			// Check for duplicate names.
			p.nameErr = m.checkDuplicateName(m.cursor, newName)
			oldName := p.name
			p.name = newName
			return m, tea.Batch(cmd, func() tea.Msg {
				return paramRenamedMsg{OldName: oldName, NewName: newName}
			})
		}
	case subFieldDefault:
		p.defaultInput, cmd = p.defaultInput.Update(msg)
		p.defaultVal = p.defaultInput.Value()
	case subFieldOptions:
		p.optionsInput, cmd = p.optionsInput.Update(msg)
		p.options = parseEnumOptions(p.optionsInput.Value())
	case subFieldDynamicCmd:
		p.dynamicCmdInput, cmd = p.dynamicCmdInput.Update(msg)
		p.dynamicCmd = p.dynamicCmdInput.Value()
	}

	return m, cmd
}

// updateTypeSelector handles left/right keys for the type selector.
func (m ParamEditorModel) updateTypeSelector(msg tea.KeyMsg) (ParamEditorModel, tea.Cmd) {
	p := &m.params[m.cursor]
	key := msg.String()

	switch key {
	case "left", "h":
		idx := typeIndex(p.paramType)
		if idx > 0 {
			oldType := p.paramType
			p.paramType = availableTypes[idx-1]
			return m, func() tea.Msg {
				return paramTypeChangedMsg{Index: m.cursor, OldType: oldType, NewType: p.paramType}
			}
		}
		return m, nil

	case "right", "l":
		idx := typeIndex(p.paramType)
		if idx < len(availableTypes)-1 {
			oldType := p.paramType
			p.paramType = availableTypes[idx+1]
			return m, func() tea.Msg {
				return paramTypeChangedMsg{Index: m.cursor, OldType: oldType, NewType: p.paramType}
			}
		}
		return m, nil

	case "tab":
		return m.advanceSubField(1)

	case "shift+tab":
		return m.advanceSubField(-1)

	case "enter":
		m.blurAllSubFields()
		m.editing = false
		return m, nil

	case "esc":
		m.blurAllSubFields()
		m.editing = false
		return m, nil
	}

	return m, nil
}

// advanceSubField moves focus to the next/prev visible sub-field within the expanded param.
// direction: +1 for forward (tab), -1 for backward (shift+tab).
func (m ParamEditorModel) advanceSubField(direction int) (ParamEditorModel, tea.Cmd) {
	p := &m.params[m.cursor]
	m.blurAllSubFields()

	fields := m.visibleSubFields()
	currentIdx := -1
	for i, f := range fields {
		if f == p.focusedField {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		// Current field not found; start at beginning.
		if len(fields) > 0 {
			p.focusedField = fields[0]
			return m.focusSubField()
		}
		m.editing = false
		return m, nil
	}

	nextIdx := currentIdx + direction
	if nextIdx < 0 || nextIdx >= len(fields) {
		// Went past the boundary — stop editing, let parent handle tab.
		m.editing = false
		return m, nil
	}

	p.focusedField = fields[nextIdx]
	return m.focusSubField()
}

// visibleSubFields returns the list of sub-field indices visible for the current param's type.
func (m ParamEditorModel) visibleSubFields() []int {
	if m.cursor < 0 || m.cursor >= len(m.params) {
		return nil
	}
	p := m.params[m.cursor]
	fields := []int{subFieldName, subFieldType, subFieldDefault}

	switch p.paramType {
	case "enum":
		fields = append(fields, subFieldOptions)
	case "dynamic":
		fields = append(fields, subFieldDynamicCmd)
	}

	return fields
}

// focusSubField focuses the input widget for the currently selected sub-field.
func (m ParamEditorModel) focusSubField() (ParamEditorModel, tea.Cmd) {
	p := &m.params[m.cursor]
	m.editing = true

	switch p.focusedField {
	case subFieldName:
		p.nameInput.Focus()
		return m, textinput.Blink
	case subFieldType:
		// Type selector doesn't use a textinput — it's navigated with arrow keys.
		return m, nil
	case subFieldDefault:
		p.defaultInput.Focus()
		return m, textinput.Blink
	case subFieldOptions:
		p.optionsInput.Focus()
		return m, textinput.Blink
	case subFieldDynamicCmd:
		p.dynamicCmdInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

// blurAllSubFields removes focus from all input widgets in the current param.
func (m *ParamEditorModel) blurAllSubFields() {
	if m.cursor < 0 || m.cursor >= len(m.params) {
		return
	}
	p := &m.params[m.cursor]
	p.nameInput.Blur()
	p.defaultInput.Blur()
	p.optionsInput.Blur()
	p.dynamicCmdInput.Blur()
}

// commitCurrentField saves the value from the current sub-field's input widget.
func (m *ParamEditorModel) commitCurrentField() {
	if m.cursor < 0 || m.cursor >= len(m.params) {
		return
	}
	p := &m.params[m.cursor]

	switch p.focusedField {
	case subFieldName:
		newName := p.nameInput.Value()
		p.nameErr = m.checkDuplicateName(m.cursor, newName)
		p.name = newName
	case subFieldDefault:
		p.defaultVal = p.defaultInput.Value()
	case subFieldOptions:
		p.options = parseEnumOptions(p.optionsInput.Value())
	case subFieldDynamicCmd:
		p.dynamicCmd = p.dynamicCmdInput.Value()
	}
}

// cancelCurrentField restores the input widget to the stored value.
func (m *ParamEditorModel) cancelCurrentField() {
	if m.cursor < 0 || m.cursor >= len(m.params) {
		return
	}
	p := &m.params[m.cursor]

	switch p.focusedField {
	case subFieldName:
		p.nameInput.SetValue(p.name)
		p.nameErr = ""
	case subFieldDefault:
		p.defaultInput.SetValue(p.defaultVal)
	case subFieldOptions:
		p.optionsInput.SetValue(strings.Join(p.options, ", "))
	case subFieldDynamicCmd:
		p.dynamicCmdInput.SetValue(p.dynamicCmd)
	}
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
		// If a param is expanded, start editing its first sub-field.
		if !m.onAddButton && m.cursor < len(m.params) && m.params[m.cursor].expanded {
			m.params[m.cursor].focusedField = subFieldName
			return m.focusSubField()
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

	case "shift+tab":
		// If a param is expanded, start editing its last sub-field.
		if !m.onAddButton && m.cursor < len(m.params) && m.params[m.cursor].expanded {
			fields := m.visibleSubFields()
			if len(fields) > 0 {
				m.params[m.cursor].focusedField = fields[len(fields)-1]
				return m.focusSubField()
			}
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
		m.blurParamInputs(i)
	}

	entry := newParamEntry(store.Arg{Name: "new_param", Type: "text"})
	entry.expanded = true
	entry.focusedField = subFieldName
	entry.nameInput.Focus()
	entry.nameInput.CursorEnd()

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

	// Collapse all and blur their inputs.
	for i := range m.params {
		m.params[i].expanded = false
		m.blurParamInputs(i)
	}
	m.editing = false

	// If it was collapsed, expand it.
	if !wasExpanded {
		m.params[m.cursor].expanded = true
		m.params[m.cursor].focusedField = subFieldName
	}

	return m
}

// blurParamInputs blurs all input widgets for a specific param.
func (m *ParamEditorModel) blurParamInputs(idx int) {
	if idx < 0 || idx >= len(m.params) {
		return
	}
	m.params[idx].nameInput.Blur()
	m.params[idx].defaultInput.Blur()
	m.params[idx].optionsInput.Blur()
	m.params[idx].dynamicCmdInput.Blur()
}

// checkDuplicateName returns an error string if the name duplicates another param.
func (m ParamEditorModel) checkDuplicateName(selfIdx int, name string) string {
	if strings.TrimSpace(name) == "" {
		return "Parameter name is required"
	}
	for i, p := range m.params {
		if i != selfIdx && p.name == name {
			return "Parameter name already exists"
		}
	}
	return ""
}

// HasDuplicateNames returns true if any param has a duplicate name error.
func (m ParamEditorModel) HasDuplicateNames() bool {
	for i := range m.params {
		if m.checkDuplicateName(i, m.params[i].name) != "" {
			return true
		}
	}
	return false
}

// ValidateForSave checks for save-time validation issues.
// Returns a list of warning messages about incompatible metadata.
func (m ParamEditorModel) ValidateForSave() []string {
	var warnings []string
	for _, p := range m.params {
		switch p.paramType {
		case "text", "list":
			if len(p.options) > 0 {
				warnings = append(warnings, fmt.Sprintf("'%s' is type %s but has enum options — options will be cleared on save", p.name, p.paramType))
			}
			if p.dynamicCmd != "" {
				warnings = append(warnings, fmt.Sprintf("'%s' is type %s but has a dynamic command — command will be cleared on save", p.name, p.paramType))
			}
		case "enum":
			if len(p.options) == 0 {
				warnings = append(warnings, fmt.Sprintf("'%s' is type enum but has no options", p.name))
			}
		case "dynamic":
			if p.dynamicCmd == "" {
				warnings = append(warnings, fmt.Sprintf("'%s' is type dynamic but has no command", p.name))
			}
		}
	}
	return warnings
}

// View renders the param list.
func (m ParamEditorModel) View() string {
	s := m.theme.Styles()

	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Colors.Primary))
	dimStyle := s.Dim
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	accentBg := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Primary)).
		Bold(true)
	pillStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Secondary)).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

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
			rows = append(rows, m.renderExpandedParam(i, p, isFocused, primaryStyle, dimStyle, errorStyle, warnStyle, accentBg, pillStyle))
		} else {
			// Collapsed row: > name (type)
			arrow := ">"
			nameDisplay := p.name
			typeDisplay := dimStyle.Render("(" + p.paramType + ")")

			if isFocused {
				nameDisplay = accentBg.Render(p.name)
				arrow = accentBg.Render(">")
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

	// Ghost params (AI suggestions not yet accepted).
	if len(m.ghostParams) > 0 {
		ghostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Colors.Dim)).Italic(true)
		rows = append(rows, "")
		rows = append(rows, ghostStyle.Render("  AI Suggested Parameters (Enter to accept, Esc to dismiss):"))
		for i, gp := range m.ghostParams {
			typeStr := gp.Type
			if typeStr == "" {
				typeStr = "text"
			}
			row := ghostStyle.Render(fmt.Sprintf("    %d. %s (%s)", i+1, gp.Name, typeStr))
			if gp.Default != "" {
				row += ghostStyle.Render(fmt.Sprintf(" = %q", gp.Default))
			}
			rows = append(rows, row)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderExpandedParam renders the expanded view for a single parameter with all its sub-fields.
func (m ParamEditorModel) renderExpandedParam(
	idx int, p paramEntry, isFocused bool,
	primaryStyle, dimStyle, errorStyle, warnStyle, accentBg, pillStyle lipgloss.Style,
) string {
	var lines []string

	// Header: v name (type)
	arrow := "v"
	nameDisplay := primaryStyle.Render(p.name)
	typeDisplay := dimStyle.Render("(" + p.paramType + ")")
	if isFocused {
		arrow = accentBg.Render("v")
	}
	header := fmt.Sprintf("  %s %s  %s", arrow, nameDisplay, typeDisplay)
	lines = append(lines, header)

	isEditing := m.editing && idx == m.cursor
	ghostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Colors.Dim)).Italic(true)

	// Name field.
	if isEditing && p.focusedField == subFieldName {
		lines = append(lines, "    "+accentBg.Render("Name: ")+m.params[idx].nameInput.View())
		if p.nameErr != "" {
			lines = append(lines, "    "+errorStyle.Render("  "+p.nameErr))
		}
	} else {
		lines = append(lines, "    "+dimStyle.Render("Name: ")+primaryStyle.Render(p.name))
		if p.nameErr != "" {
			lines = append(lines, "    "+errorStyle.Render("  "+p.nameErr))
		}
	}

	// Type selector.
	if isEditing && p.focusedField == subFieldType {
		lines = append(lines, "    "+accentBg.Render("Type: ")+renderTypeSelector(p.paramType, accentBg, dimStyle))
		lines = append(lines, "    "+dimStyle.Render("  <-/-> to change type"))
	} else {
		lines = append(lines, "    "+dimStyle.Render("Type: ")+renderTypeSelector(p.paramType, primaryStyle, dimStyle))
	}

	// Default value field.
	defaultGhost := m.getGhostText(idx, "default")
	if isEditing && p.focusedField == subFieldDefault {
		ghostHint := ""
		if defaultGhost != "" {
			ghostHint = "  " + ghostStyle.Render(defaultGhost)
		}
		lines = append(lines, "    "+accentBg.Render("Default: ")+m.params[idx].defaultInput.View()+ghostHint)
	} else {
		if p.defaultVal != "" {
			lines = append(lines, "    "+dimStyle.Render("Default: ")+p.defaultVal)
		} else if defaultGhost != "" {
			lines = append(lines, "    "+dimStyle.Render("Default: ")+ghostStyle.Render(defaultGhost))
		} else {
			lines = append(lines, "    "+dimStyle.Render("Default: ")+dimStyle.Render("(none)"))
		}
	}

	// Enum options field (only for enum type).
	if p.paramType == "enum" {
		optionsGhost := m.getGhostText(idx, "options")
		if isEditing && p.focusedField == subFieldOptions {
			ghostHint := ""
			if optionsGhost != "" {
				ghostHint = "  " + ghostStyle.Render(optionsGhost)
			}
			lines = append(lines, "    "+accentBg.Render("Options: ")+m.params[idx].optionsInput.View()+ghostHint)
		} else {
			if len(p.options) > 0 {
				lines = append(lines, "    "+dimStyle.Render("Options: ")+formatOptions(p.options, pillStyle))
			} else if optionsGhost != "" {
				lines = append(lines, "    "+dimStyle.Render("Options: ")+ghostStyle.Render(optionsGhost))
			} else {
				lines = append(lines, "    "+dimStyle.Render("Options: ")+warnStyle.Render("(none — add comma-separated)"))
			}
		}
		// Show options as pills for visual feedback.
		if len(p.options) > 0 && isEditing && p.focusedField == subFieldOptions {
			lines = append(lines, "    "+dimStyle.Render("  ")+formatOptions(p.options, pillStyle))
		}
		// Warn if default is not one of the options.
		if p.defaultVal != "" && len(p.options) > 0 && !containsOption(p.options, p.defaultVal) {
			lines = append(lines, "    "+warnStyle.Render("  default not in options"))
		}
	}

	// Dynamic command field (only for dynamic type).
	if p.paramType == "dynamic" {
		dynGhost := m.getGhostText(idx, "dynamic_cmd")
		if isEditing && p.focusedField == subFieldDynamicCmd {
			ghostHint := ""
			if dynGhost != "" {
				ghostHint = "  " + ghostStyle.Render(dynGhost)
			}
			lines = append(lines, "    "+accentBg.Render("Command: ")+m.params[idx].dynamicCmdInput.View()+ghostHint)
		} else {
			if p.dynamicCmd != "" {
				lines = append(lines, "    "+dimStyle.Render("Command: ")+p.dynamicCmd)
			} else if dynGhost != "" {
				lines = append(lines, "    "+dimStyle.Render("Command: ")+ghostStyle.Render(dynGhost))
			} else {
				lines = append(lines, "    "+dimStyle.Render("Command: ")+warnStyle.Render("(none)"))
			}
		}
	}

	// Show description if present.
	if p.description != "" {
		lines = append(lines, "    "+dimStyle.Render("Desc: ")+p.description)
	}

	// Soft staging indicator: show preserved metadata for non-matching types.
	if p.paramType != "enum" && len(p.options) > 0 {
		lines = append(lines, "    "+dimStyle.Render("(enum options preserved: ")+dimStyle.Render(strings.Join(p.options, ", "))+dimStyle.Render(")"))
	}
	if p.paramType != "dynamic" && p.dynamicCmd != "" {
		lines = append(lines, "    "+dimStyle.Render("(dynamic cmd preserved: ")+dimStyle.Render(p.dynamicCmd)+dimStyle.Render(")"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderTypeSelector renders the inline type selector: [current] other other
func renderTypeSelector(current string, activeStyle, inactiveStyle lipgloss.Style) string {
	var parts []string
	for _, t := range availableTypes {
		if t == current {
			parts = append(parts, activeStyle.Render("["+t+"]"))
		} else {
			parts = append(parts, inactiveStyle.Render(" "+t+" "))
		}
	}
	return strings.Join(parts, "")
}

// formatOptions renders options as styled pills.
func formatOptions(opts []string, pillStyle lipgloss.Style) string {
	var pills []string
	for _, o := range opts {
		pills = append(pills, pillStyle.Render(o))
	}
	return strings.Join(pills, " ")
}

// containsOption checks if a value is in the options list.
func containsOption(opts []string, val string) bool {
	for _, o := range opts {
		if o == val {
			return true
		}
	}
	return false
}

// parseEnumOptions parses a comma-separated string into an options slice.
// Prefix an option with * to mark it as default (the * is stripped from the option name).
func parseEnumOptions(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var opts []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			// Strip leading * for default marker (handled at save time).
			opts = append(opts, strings.TrimPrefix(t, "*"))
		}
	}
	if len(opts) == 0 {
		return nil
	}
	return opts
}

// parseEnumOptionsWithDefault parses options and returns the default option if marked with *.
func parseEnumOptionsWithDefault(s string) ([]string, string) {
	if strings.TrimSpace(s) == "" {
		return nil, ""
	}
	parts := strings.Split(s, ",")
	var opts []string
	var defaultVal string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "*") {
			cleaned := t[1:]
			opts = append(opts, cleaned)
			defaultVal = cleaned
		} else {
			opts = append(opts, t)
		}
	}
	if len(opts) == 0 {
		return nil, ""
	}
	return opts, defaultVal
}

// ToArgs converts the current param entries back to store.Arg for persistence.
// It strips metadata incompatible with the current type.
func (m ParamEditorModel) ToArgs() []store.Arg {
	if len(m.params) == 0 {
		return nil
	}

	args := make([]store.Arg, len(m.params))
	for i, p := range m.params {
		arg := store.Arg{
			Name:        p.name,
			Default:     p.defaultVal,
			Description: p.description,
			Type:        p.paramType,
		}

		// Only include metadata compatible with the type.
		switch p.paramType {
		case "enum":
			arg.Options = p.options
		case "dynamic":
			arg.DynamicCmd = p.dynamicCmd
		}
		// text and list types: no options or dynamic cmd persisted.

		args[i] = arg
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

// hasExpandedParam returns true if any param is currently expanded.
func (m ParamEditorModel) hasExpandedParam() bool {
	for _, p := range m.params {
		if p.expanded {
			return true
		}
	}
	return false
}

// typeIndex returns the index of a type in availableTypes, or 0 if not found.
func typeIndex(t string) int {
	for i, at := range availableTypes {
		if at == t {
			return i
		}
	}
	return 0
}

// setGhostText sets AI suggestion ghost text for a param sub-field.
func (m *ParamEditorModel) setGhostText(fieldKey, value string) {
	if m.ghostText == nil {
		m.ghostText = make(map[string]string)
	}
	m.ghostText[fieldKey] = value
}

// getGhostText returns the ghost text for a param sub-field, if any.
func (m ParamEditorModel) getGhostText(paramIdx int, subField string) string {
	key := fmt.Sprintf("param:%d:%s", paramIdx, subField)
	return m.ghostText[key]
}

// clearGhostText removes ghost text for a specific param sub-field.
func (m *ParamEditorModel) clearGhostText(paramIdx int, subField string) {
	key := fmt.Sprintf("param:%d:%s", paramIdx, subField)
	delete(m.ghostText, key)
}

// addGhostParam adds an AI-suggested parameter that hasn't been accepted yet.
func (m *ParamEditorModel) addGhostParam(arg store.Arg) {
	// Check if a param with this name already exists.
	for _, p := range m.params {
		if p.name == arg.Name {
			return // skip duplicate
		}
	}
	// Check if a ghost param with this name already exists.
	for _, g := range m.ghostParams {
		if g.Name == arg.Name {
			return // skip duplicate
		}
	}
	m.ghostParams = append(m.ghostParams, arg)
}

// acceptGhostParam converts a ghost param into a real param entry.
func (m *ParamEditorModel) acceptGhostParam(idx int) {
	if idx < 0 || idx >= len(m.ghostParams) {
		return
	}
	arg := m.ghostParams[idx]
	entry := newParamEntry(arg)
	m.params = append(m.params, entry)
	// Remove from ghost params.
	m.ghostParams = append(m.ghostParams[:idx], m.ghostParams[idx+1:]...)
}

// dismissGhostParam removes a ghost param suggestion.
func (m *ParamEditorModel) dismissGhostParam(idx int) {
	if idx < 0 || idx >= len(m.ghostParams) {
		return
	}
	m.ghostParams = append(m.ghostParams[:idx], m.ghostParams[idx+1:]...)
}
