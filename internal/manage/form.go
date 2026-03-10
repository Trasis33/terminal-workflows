package manage

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/ai"
	"github.com/fredriklanga/wf/internal/store"
)

// Validation errors for form fields.
var (
	errNameRequired    = errors.New("name is required")
	errNameNoSlash     = errors.New("name must not contain slashes (use folder field)")
	errCommandRequired = errors.New("command is required")
)

// formValues holds form field values as a shared reference.
// Because bubbletea copies models on each Update cycle, pointer-based
// Value() bindings would point to stale copies of FormModel fields.
// By storing values in a heap-allocated struct, all copies share the same data.
type formValues struct {
	name        string
	description string
	command     string
	tagInput    string // comma-separated tags
	folder      string
}

// formFieldIndex enumerates the focusable fields in the form.
type formFieldIndex int

const (
	fieldName formFieldIndex = iota
	fieldDescription
	fieldCommand
	fieldTags
	fieldFolder
	fieldAutofill // the autofill button
	fieldParams   // the param editor section
	fieldCount    // sentinel — total number of focusable fields
)

// FormModel manages a full-screen custom form for creating or editing a workflow.
type FormModel struct {
	mode string // "create" or "edit"

	// Edit mode: tracks the original workflow name to handle renames.
	originalName string

	store store.Store
	theme Theme

	width  int
	height int

	// Shared form field values.
	vals *formValues

	// Input widgets for metadata fields.
	nameInput   textinput.Model
	descInput   textinput.Model
	cmdInput    textarea.Model
	tagsInput   textinput.Model
	folderInput textinput.Model

	// Parameter editor (below metadata fields).
	paramEditor ParamEditorModel

	// Which field is currently focused.
	focused formFieldIndex

	// Suggestions for autocomplete.
	existingTags    []string
	existingFolders []string

	// Per-field AI ghost text suggestions (field name → suggested value).
	ghostText map[string]string

	// Per-field loading state (field name → loading).
	fieldLoading map[string]bool

	// Autofill lock: when true, the form is locked during bulk autofill.
	autofillLock bool
	spinnerFrame int

	// Error state for display.
	err error
}

// NewFormModel creates a FormModel for creating or editing a workflow.
//
// mode: "create" for a new workflow, "edit" to modify an existing one.
// wf: the workflow to edit (ignored for create mode; may be nil).
// s: the store for persistence.
// existingTags/existingFolders: used for input suggestions.
// theme: the current TUI theme.
func NewFormModel(mode string, wf *store.Workflow, s store.Store, existingTags, existingFolders []string, theme Theme) FormModel {
	m := FormModel{
		mode:            mode,
		store:           s,
		theme:           theme,
		vals:            &formValues{},
		existingTags:    existingTags,
		existingFolders: existingFolders,
		ghostText:       make(map[string]string),
		fieldLoading:    make(map[string]bool),
	}

	var args []store.Arg

	if wf != nil {
		if mode == "edit" {
			m.originalName = wf.Name
		}

		// Pre-fill fields from the provided workflow.
		if idx := strings.LastIndex(wf.Name, "/"); idx >= 0 {
			m.vals.folder = wf.Name[:idx]
			m.vals.name = wf.Name[idx+1:]
		} else {
			m.vals.name = wf.Name
		}

		m.vals.description = wf.Description
		m.vals.command = wf.Command
		m.vals.tagInput = strings.Join(wf.Tags, ", ")

		args = wf.Args
	}

	m.buildInputs()
	m.paramEditor = NewParamEditor(args, theme, m.width)

	return m
}

// buildInputs creates the textinput and textarea widgets for metadata fields.
func (m *FormModel) buildInputs() {
	// Name field.
	m.nameInput = textinput.New()
	m.nameInput.Placeholder = "my-workflow"
	m.nameInput.CharLimit = 128
	m.nameInput.SetValue(m.vals.name)
	// NOTE: do NOT Focus() here — the terminal may emit OSC responses
	// (e.g. background color query from lipgloss renderer) that get
	// captured as typed text into a focused input. Instead, we focus
	// in Init() which runs after the program starts reading input.

	// Description field.
	m.descInput = textinput.New()
	m.descInput.Placeholder = "What does this workflow do?"
	m.descInput.CharLimit = 256
	m.descInput.SetValue(m.vals.description)

	// Command field (textarea).
	m.cmdInput = textarea.New()
	m.cmdInput.Placeholder = "Enter command (alt+enter for newline)"
	m.cmdInput.CharLimit = 0 // no limit
	m.cmdInput.SetValue(m.vals.command)
	m.cmdInput.SetHeight(6)
	m.cmdInput.ShowLineNumbers = false

	// Tags field.
	m.tagsInput = textinput.New()
	m.tagsInput.Placeholder = "e.g., docker, deploy, infra"
	m.tagsInput.CharLimit = 256
	m.tagsInput.SetValue(m.vals.tagInput)

	// Folder field.
	m.folderInput = textinput.New()
	m.folderInput.Placeholder = "e.g., infra/deploy (empty for root)"
	m.folderInput.CharLimit = 128
	m.folderInput.SetValue(m.vals.folder)
}

// formInitMsg is sent by Init() to trigger initial field focus.
// This avoids focusing inputs in buildInputs() where terminal OSC
// responses (e.g. background color) would be captured as typed text.
type formInitMsg struct{}

// Init returns the initial command for the form.
func (m FormModel) Init() tea.Cmd {
	return func() tea.Msg { return formInitMsg{} }
}

// Update processes messages for the form model.
func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case formInitMsg:
		// Focus the name field now that the program is running and
		// terminal OSC queries have been processed.
		m.focused = fieldName
		m.nameInput.Focus()
		return m, textinput.Blink
	case tea.KeyMsg:
		// During autofill lock, only Esc is allowed (to cancel).
		if m.autofillLock {
			if msg.String() == "esc" {
				m.autofillLock = false
				m.fieldLoading = make(map[string]bool)
				m.spinnerFrame = 0
				return m, nil
			}
			return m, nil
		}
		return m.handleKey(msg)
	case spinnerTickMsg:
		if msg.scope != spinnerScopeForm || !m.hasActiveSpinner() {
			return m, nil
		}
		m.spinnerFrame = nextSpinnerFrame(m.spinnerFrame)
		return m, spinnerTickCmd(spinnerScopeForm)
	case perFieldAIResultMsg:
		return m.handlePerFieldAIResult(msg)
	case suggestParamsResultMsg:
		return m.handleSuggestParamsResult(msg)
	case paramRenamedMsg:
		// Live command template preview: update {{oldName...}} to {{newName...}} in command textarea.
		m.updateCommandTemplateOnRename(msg.OldName, msg.NewName)
		return m, nil
	case paramTypeChangedMsg:
		// Type changes are soft-staged — no command template modification until save.
		return m, nil
	}

	// Forward non-key messages to the focused widget.
	return m.updateFocusedWidget(msg)
}

// handleKey processes keyboard input for navigation and actions.
func (m FormModel) handleKey(msg tea.KeyMsg) (FormModel, tea.Cmd) {
	key := msg.String()

	// Esc returns to browse — but only if the param editor isn't in editing mode.
	if key == "esc" {
		// Clear ghost text on the focused field if present.
		fieldKey := m.fieldKey(m.focused)
		if _, hasGhost := m.ghostText[fieldKey]; hasGhost {
			delete(m.ghostText, fieldKey)
			return m, nil
		}

		if m.focused == fieldParams && m.paramEditor.Focused() {
			// Let the param editor handle esc (cancel name edit).
			var cmd tea.Cmd
			m.paramEditor, cmd = m.paramEditor.Update(msg)
			return m, cmd
		}
		if m.focused == fieldParams && m.paramEditor.confirmDelete >= 0 {
			// Let the param editor handle esc (cancel delete).
			var cmd tea.Cmd
			m.paramEditor, cmd = m.paramEditor.Update(msg)
			return m, cmd
		}
		return m, func() tea.Msg { return switchToBrowseMsg{} }
	}

	// Enter on a field with ghost text accepts the suggestion.
	if key == "enter" && m.focused != fieldAutofill && m.focused != fieldParams {
		fieldKey := m.fieldKey(m.focused)
		if suggestion, hasGhost := m.ghostText[fieldKey]; hasGhost {
			m.applyGhostText(m.focused, suggestion)
			delete(m.ghostText, fieldKey)
			return m, nil
		}
	}

	// Ctrl+G triggers per-field AI generation.
	if key == "ctrl+g" {
		return m.triggerPerFieldAI()
	}

	// Ctrl+S saves the workflow.
	if key == "ctrl+s" {
		if err := m.validate(); err != nil {
			m.err = err
			return m, nil
		}
		return m, m.saveWorkflow()
	}

	// Ctrl+N adds a param (regardless of focused field).
	if key == "ctrl+n" {
		var cmd tea.Cmd
		m.paramEditor, cmd = m.paramEditor.Update(msg)
		m.focused = fieldParams
		m.blurAllInputs()
		return m, cmd
	}

	// Ctrl+D deletes a param (only when param editor is focused).
	if key == "ctrl+d" && m.focused == fieldParams {
		var cmd tea.Cmd
		m.paramEditor, cmd = m.paramEditor.Update(msg)
		return m, cmd
	}

	// Any typing clears ghost text for the focused field.
	if len(key) == 1 || key == "backspace" || key == "delete" {
		fieldKey := m.fieldKey(m.focused)
		delete(m.ghostText, fieldKey)
	}

	// Tab / Shift+Tab navigation between fields.
	if key == "tab" && m.focused != fieldCommand {
		// In param editor, tab might be handled internally.
		if m.focused == fieldParams {
			if m.paramEditor.Focused() || m.paramEditor.hasExpandedParam() {
				// Let param editor handle tab internally.
				var cmd tea.Cmd
				m.paramEditor, cmd = m.paramEditor.Update(msg)
				return m, cmd
			}
			// Tab past param editor wraps to first field.
			m.focused = fieldName
			m.focusCurrentField()
			return m, textinput.Blink
		}
		// Autofill button is just a focus target — tab past it.
		m.focused++
		m.focusCurrentField()
		if m.focused == fieldCommand {
			return m, m.cmdInput.Focus()
		}
		return m, textinput.Blink
	}

	if key == "shift+tab" && m.focused != fieldCommand {
		if m.focused == fieldParams && (m.paramEditor.Focused() || m.paramEditor.hasExpandedParam()) {
			var cmd tea.Cmd
			m.paramEditor, cmd = m.paramEditor.Update(msg)
			return m, cmd
		}
		if m.focused > 0 {
			m.focused--
		} else {
			m.focused = fieldParams
		}
		m.focusCurrentField()
		if m.focused == fieldCommand {
			return m, m.cmdInput.Focus()
		}
		return m, textinput.Blink
	}

	// For the command textarea, tab should advance to next field (not insert tab).
	if key == "tab" && m.focused == fieldCommand {
		m.focused = fieldTags
		m.focusCurrentField()
		return m, textinput.Blink
	}
	if key == "shift+tab" && m.focused == fieldCommand {
		m.focused = fieldDescription
		m.focusCurrentField()
		return m, textinput.Blink
	}

	// Autofill button: Enter triggers autofill.
	if m.focused == fieldAutofill {
		if key == "enter" {
			return m.triggerAutofill()
		}
		// Autofill button doesn't consume other keys.
		return m, nil
	}

	// Route to param editor if focused.
	if m.focused == fieldParams {
		var cmd tea.Cmd
		m.paramEditor, cmd = m.paramEditor.Update(msg)
		return m, cmd
	}

	// Forward key to the focused widget.
	return m.updateFocusedWidget(msg)
}

// updateFocusedWidget forwards a message to whichever widget is focused.
func (m FormModel) updateFocusedWidget(msg tea.Msg) (FormModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focused {
	case fieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
		m.vals.name = m.nameInput.Value()
	case fieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
		m.vals.description = m.descInput.Value()
	case fieldCommand:
		m.cmdInput, cmd = m.cmdInput.Update(msg)
		m.vals.command = m.cmdInput.Value()
	case fieldTags:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
		m.vals.tagInput = m.tagsInput.Value()
	case fieldFolder:
		m.folderInput, cmd = m.folderInput.Update(msg)
		m.vals.folder = m.folderInput.Value()
	case fieldParams:
		m.paramEditor, cmd = m.paramEditor.Update(msg)
	}

	return m, cmd
}

// focusCurrentField blurs all inputs then focuses the one at m.focused.
func (m *FormModel) focusCurrentField() {
	m.blurAllInputs()

	switch m.focused {
	case fieldName:
		m.nameInput.Focus()
	case fieldDescription:
		m.descInput.Focus()
	case fieldCommand:
		// textarea focus is handled by caller (returns a cmd)
	case fieldTags:
		m.tagsInput.Focus()
	case fieldFolder:
		m.folderInput.Focus()
	case fieldAutofill:
		// autofill button doesn't need explicit focus call
	case fieldParams:
		// param editor doesn't need explicit focus call
	}
}

// blurAllInputs removes focus from all input widgets.
func (m *FormModel) blurAllInputs() {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.cmdInput.Blur()
	m.tagsInput.Blur()
	m.folderInput.Blur()
}

// validate checks required fields before saving.
func (m FormModel) validate() error {
	if strings.TrimSpace(m.vals.name) == "" {
		return errNameRequired
	}
	if strings.ContainsAny(m.vals.name, "/\\") {
		return errNameNoSlash
	}
	if strings.TrimSpace(m.vals.command) == "" {
		return errCommandRequired
	}
	return nil
}

// saveWorkflow builds a Workflow from form fields and persists it via Store.
func (m FormModel) saveWorkflow() tea.Cmd {
	v := m.vals
	st := m.store
	mode := m.mode
	originalName := m.originalName
	args := m.paramEditor.ToArgs()

	return func() tea.Msg {
		// Parse tags from comma-separated input.
		tags := parseTags(v.tagInput)

		// Build full name with folder prefix.
		fullName := strings.TrimSpace(v.name)
		folder := strings.TrimSpace(v.folder)
		folder = strings.Trim(folder, "/")
		if folder != "" {
			fullName = folder + "/" + fullName
		}

		wf := store.Workflow{
			Name:        fullName,
			Command:     strings.TrimSpace(v.command),
			Description: strings.TrimSpace(v.description),
			Tags:        tags,
			Args:        args,
		}

		// If editing and name changed, delete the old workflow first.
		if mode == "edit" && originalName != "" && originalName != fullName {
			if err := st.Delete(originalName); err != nil {
				return saveErrorMsg{err: err}
			}
		}

		if err := st.Save(&wf); err != nil {
			return saveErrorMsg{err: err}
		}

		return workflowSavedMsg{workflow: wf}
	}
}

// View renders the form with title, fields, param editor, and hints.
func (m FormModel) View() string {
	s := m.theme.Styles()

	// Form title.
	var title string
	if m.mode == "edit" {
		title = s.FormTitle.Render("Edit Workflow")
	} else {
		title = s.FormTitle.Render("Create Workflow")
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Colors.Dim))
	focusLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Primary)).
		Bold(true)
	ghostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Colors.Dim)).Italic(true)
	loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("49"))
	sectionDivider := labelStyle.Render("── Parameters ──")

	// Build field rows.
	var rows []string
	rows = append(rows, "", title)

	// Helper to render field AI indicator.
	aiIndicator := func(fieldKey string) string {
		if m.fieldLoading[fieldKey] {
			return loadingStyle.Render(" " + spinnerFrame(m.spinnerFrame))
		}
		if ai.IsAvailable() && m.focused == m.fieldIndexForKey(fieldKey) {
			return labelStyle.Render(" ⚡AI")
		}
		return ""
	}

	// Helper to render ghost text suffix.
	// Rendered on a separate line below the field to avoid right-alignment
	// caused by the fixed-width textinput view.
	ghostLine := func(fieldKey string) string {
		if ghost, ok := m.ghostText[fieldKey]; ok && ghost != "" {
			return "  " + ghostStyle.Render("  → "+ghost+" (Enter accept, Esc dismiss)")
		}
		return ""
	}

	// Name.
	lbl := labelStyle
	if m.focused == fieldName {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Name")+aiIndicator("name"))
	rows = append(rows, "  "+m.nameInput.View())
	if gl := ghostLine("name"); gl != "" {
		rows = append(rows, gl)
	}

	// Description.
	lbl = labelStyle
	if m.focused == fieldDescription {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Description")+aiIndicator("description"))
	rows = append(rows, "  "+m.descInput.View())
	if gl := ghostLine("description"); gl != "" {
		rows = append(rows, gl)
	}

	// Command.
	lbl = labelStyle
	if m.focused == fieldCommand {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Command")+aiIndicator("command"))
	// Use PaddingLeft to indent all lines of the multi-line textarea consistently.
	cmdView := lipgloss.NewStyle().PaddingLeft(2).Render(m.cmdInput.View())
	rows = append(rows, cmdView)

	// Tags.
	lbl = labelStyle
	if m.focused == fieldTags {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Tags (comma-separated)")+aiIndicator("tags"))
	rows = append(rows, "  "+m.tagsInput.View())
	if gl := ghostLine("tags"); gl != "" {
		rows = append(rows, gl)
	}

	// Folder.
	lbl = labelStyle
	if m.focused == fieldFolder {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Folder"))
	rows = append(rows, "  "+m.folderInput.View())

	// Autofill button.
	rows = append(rows, "")
	if m.autofillLock {
		rows = append(rows, loadingStyle.Render("  "+spinnerFrame(m.spinnerFrame)+" Autofilling...  ")+labelStyle.Render("esc to cancel"))
	} else if m.focused == fieldAutofill {
		autofillBtn := focusLabelStyle.Render("  [ ⚡ Autofill Empty Fields ]")
		rows = append(rows, autofillBtn)
	} else {
		autofillBtn := labelStyle.Render("  [ ⚡ Autofill Empty Fields ]")
		rows = append(rows, autofillBtn)
	}

	// Parameter section.
	rows = append(rows, "")
	if m.focused == fieldParams {
		rows = append(rows, focusLabelStyle.Render("  "+sectionDivider))
	} else {
		rows = append(rows, "  "+sectionDivider)
	}
	rows = append(rows, m.paramEditor.View())

	// Error display.
	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		rows = append(rows, "", errStyle.Render("  Error: "+m.err.Error()))
	}

	// Hints.
	hintText := "  esc cancel  tab next  ctrl+s save  ctrl+n new param  ctrl+d delete param"
	if ai.IsAvailable() {
		hintText += "  ctrl+g AI suggest"
	}
	hints := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Dim)).
		Render(hintText)

	rows = append(rows, "", hints)

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// SetDimensions updates the form's available space.
func (m *FormModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height

	// Set widths for inputs — leave some padding.
	inputWidth := width - 8
	if inputWidth < 30 {
		inputWidth = 30
	}
	if inputWidth > 80 {
		inputWidth = 80
	}

	m.nameInput.Width = inputWidth
	m.descInput.Width = inputWidth
	m.cmdInput.SetWidth(inputWidth)
	m.tagsInput.Width = inputWidth
	m.folderInput.Width = inputWidth
	m.paramEditor.SetWidth(inputWidth)
}

// updateCommandTemplateOnRename replaces all {{oldName...}} patterns with {{newName...}}
// in the command textarea, supporting all template syntax variants:
// {{name}}, {{name:default}}, {{name|opt1|opt2}}, {{name!cmd}}
func (m *FormModel) updateCommandTemplateOnRename(oldName, newName string) {
	if oldName == "" || newName == "" || oldName == newName {
		return
	}

	cmd := m.cmdInput.Value()
	if cmd == "" {
		return
	}

	// Match {{oldName}} or {{oldName:...}} or {{oldName|...}} or {{oldName!...}}
	// The old name is escaped for regex safety.
	pattern := `\{\{` + regexp.QuoteMeta(oldName) + `([}:!|][^}]*)?\}\}`
	re := regexp.MustCompile(pattern)

	updated := re.ReplaceAllStringFunc(cmd, func(match string) string {
		// Extract the suffix after the old name (e.g., ":default", "|opt1|opt2", "!cmd").
		inner := match[2 : len(match)-2] // strip {{ and }}
		suffix := inner[len(oldName):]   // everything after the old name
		return "{{" + newName + suffix + "}}"
	})

	if updated != cmd {
		m.cmdInput.SetValue(updated)
		m.vals.command = updated
	}
}

// parseTags splits a comma-separated string into a clean tag slice.
func parseTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

// --- Per-field AI helpers ---

// fieldKey returns a string key for the given field index (used for ghostText and fieldLoading maps).
func (m FormModel) fieldKey(f formFieldIndex) string {
	switch f {
	case fieldName:
		return "name"
	case fieldDescription:
		return "description"
	case fieldCommand:
		return "command"
	case fieldTags:
		return "tags"
	case fieldFolder:
		return "folder"
	default:
		return ""
	}
}

// fieldIndexForKey returns the formFieldIndex for a given field key string.
func (m FormModel) fieldIndexForKey(key string) formFieldIndex {
	switch key {
	case "name":
		return fieldName
	case "description":
		return fieldDescription
	case "command":
		return fieldCommand
	case "tags":
		return fieldTags
	case "folder":
		return fieldFolder
	default:
		return fieldCount
	}
}

// formSnapshot captures the current values of all form fields as a context map for AI.
func (m FormModel) formSnapshot() map[string]string {
	snap := map[string]string{
		"name":        m.vals.name,
		"description": m.vals.description,
		"command":     m.vals.command,
		"tags":        m.vals.tagInput,
		"folder":      m.vals.folder,
	}
	return snap
}

// hasSeedField returns true if at least one seed field (name, description, command) has content.
func (m FormModel) hasSeedField() bool {
	return strings.TrimSpace(m.vals.name) != "" ||
		strings.TrimSpace(m.vals.description) != "" ||
		strings.TrimSpace(m.vals.command) != ""
}

// triggerPerFieldAI initiates per-field AI generation for the currently focused field.
func (m FormModel) triggerPerFieldAI() (FormModel, tea.Cmd) {
	if !ai.IsAvailable() {
		m.err = fmt.Errorf("AI unavailable: %s", ai.ErrUnavailable.Error())
		return m, nil
	}

	if !m.hasSeedField() {
		m.err = fmt.Errorf("add a title, description, or command first")
		return m, nil
	}

	// Determine the field to generate.
	var targetField string
	switch m.focused {
	case fieldName:
		targetField = "name"
	case fieldDescription:
		targetField = "description"
	case fieldCommand:
		targetField = "command"
	case fieldTags:
		targetField = "tags"
	case fieldParams:
		// Delegate to param editor Ctrl+G handling.
		return m.triggerParamFieldAI()
	default:
		return m, nil
	}

	m.err = nil
	m.fieldLoading[targetField] = true
	snap := m.formSnapshot()
	startSpinner := tea.Cmd(nil)
	if !m.hasActiveSpinner() || (len(m.fieldLoading) == 1 && m.spinnerFrame == 0) {
		startSpinner = spinnerTickCmd(spinnerScopeForm)
	}
	return m, tea.Batch(perFieldAICmd(targetField, snap), startSpinner)
}

// triggerParamFieldAI initiates per-field AI generation for the focused param sub-field.
func (m FormModel) triggerParamFieldAI() (FormModel, tea.Cmd) {
	if !m.paramEditor.editing || m.paramEditor.cursor < 0 || m.paramEditor.cursor >= len(m.paramEditor.params) {
		return m, nil
	}

	p := m.paramEditor.params[m.paramEditor.cursor]
	idx := m.paramEditor.cursor

	var subField string
	switch p.focusedField {
	case subFieldDefault:
		subField = "default"
	case subFieldOptions:
		subField = "options"
	case subFieldDynamicCmd:
		subField = "dynamic_cmd"
	default:
		return m, nil
	}

	fieldKey := fmt.Sprintf("param:%d:%s", idx, subField)
	m.fieldLoading[fieldKey] = true

	snap := m.formSnapshot()
	snap["param_name"] = p.name
	snap["param_type"] = p.paramType

	startSpinner := tea.Cmd(nil)
	if !m.hasActiveSpinner() || (len(m.fieldLoading) == 1 && m.spinnerFrame == 0) {
		startSpinner = spinnerTickCmd(spinnerScopeForm)
	}
	return m, tea.Batch(perFieldAICmd(fieldKey, snap), startSpinner)
}

// triggerAutofill initiates bulk AI autofill for all empty fields.
func (m FormModel) triggerAutofill() (FormModel, tea.Cmd) {
	if !ai.IsAvailable() {
		m.err = fmt.Errorf("AI unavailable: %s", ai.ErrUnavailable.Error())
		return m, nil
	}

	if !m.hasSeedField() {
		m.err = fmt.Errorf("add a title, description, or command first")
		return m, nil
	}

	m.err = nil
	m.autofillLock = true
	m.spinnerFrame = 0

	// Determine which fields are empty and need filling.
	var fields []string
	if strings.TrimSpace(m.vals.name) == "" {
		fields = append(fields, "name")
		m.fieldLoading["name"] = true
	}
	if strings.TrimSpace(m.vals.description) == "" {
		fields = append(fields, "description")
		m.fieldLoading["description"] = true
	}
	if strings.TrimSpace(m.vals.tagInput) == "" {
		fields = append(fields, "tags")
		m.fieldLoading["tags"] = true
	}
	// Always include args for param suggestions.
	fields = append(fields, "args")

	if len(fields) == 0 {
		m.autofillLock = false
		return m, nil
	}

	// Build the workflow from current state for autofill.
	wf := store.Workflow{
		Name:        m.vals.name,
		Command:     m.vals.command,
		Description: m.vals.description,
		Tags:        parseTags(m.vals.tagInput),
		Args:        m.paramEditor.ToArgs(),
	}
	if folder := strings.TrimSpace(m.vals.folder); folder != "" {
		wf.Name = folder + "/" + wf.Name
	}

	return m, tea.Batch(autofillWorkflowCmd(wf, fields), spinnerTickCmd(spinnerScopeForm))
}

// applyGhostText sets a field value from a ghost text suggestion.
func (m *FormModel) applyGhostText(field formFieldIndex, value string) {
	switch field {
	case fieldName:
		m.vals.name = value
		m.nameInput.SetValue(value)
	case fieldDescription:
		m.vals.description = value
		m.descInput.SetValue(value)
	case fieldCommand:
		m.vals.command = value
		m.cmdInput.SetValue(value)
	case fieldTags:
		m.vals.tagInput = value
		m.tagsInput.SetValue(value)
	}
}

// handlePerFieldAIResult processes the result of a per-field AI generation.
func (m FormModel) handlePerFieldAIResult(msg perFieldAIResultMsg) (FormModel, tea.Cmd) {
	// Clear loading state for this field.
	delete(m.fieldLoading, msg.fieldName)
	if !m.hasActiveSpinner() {
		m.spinnerFrame = 0
	}

	if msg.err != nil {
		errMsg := msg.err.Error()
		if strings.Contains(errMsg, "file already closed") || strings.Contains(errMsg, "broken pipe") || strings.Contains(errMsg, "EOF") {
			m.err = fmt.Errorf("AI unavailable — Copilot connection failed. Try again")
		} else {
			m.err = fmt.Errorf("AI: %s", errMsg)
		}
		return m, nil
	}

	if msg.value == "" {
		return m, nil
	}

	// Check if this is a param-level field.
	if strings.HasPrefix(msg.fieldName, "param:") {
		// Route to param editor ghost text.
		m.paramEditor.setGhostText(msg.fieldName, msg.value)
		return m, nil
	}

	// Store as ghost text for the user to accept/dismiss.
	m.ghostText[msg.fieldName] = msg.value
	return m, nil
}

// handleSuggestParamsResult processes command-to-params AI analysis results.
func (m FormModel) handleSuggestParamsResult(msg suggestParamsResultMsg) (FormModel, tea.Cmd) {
	if msg.err != nil {
		m.err = fmt.Errorf("AI param suggestion: %s", msg.err.Error())
		return m, nil
	}

	if len(msg.args) == 0 {
		return m, nil
	}

	// Add suggested params as ghost entries in the param editor.
	for _, arg := range msg.args {
		m.paramEditor.addGhostParam(arg)
	}
	return m, nil
}

// HandleAutofillResult processes autofill results in the form context.
// Called from model.go when an aiAutofillResultMsg arrives while in form view.
func (m FormModel) HandleAutofillResult(result *ai.AutofillResult) FormModel {
	m.autofillLock = false
	m.fieldLoading = make(map[string]bool)
	m.spinnerFrame = 0

	if result == nil {
		return m
	}

	// Apply results as ghost text for each field.
	if result.Name != nil && strings.TrimSpace(*result.Name) != "" && strings.TrimSpace(m.vals.name) == "" {
		m.ghostText["name"] = *result.Name
	}
	if result.Description != nil && strings.TrimSpace(*result.Description) != "" && strings.TrimSpace(m.vals.description) == "" {
		m.ghostText["description"] = *result.Description
	}
	if len(result.Tags) > 0 && strings.TrimSpace(m.vals.tagInput) == "" {
		m.ghostText["tags"] = strings.Join(result.Tags, ", ")
	}
	if len(result.Args) > 0 {
		for _, arg := range result.Args {
			m.paramEditor.addGhostParam(arg)
		}
	}

	return m
}

func (m FormModel) hasActiveSpinner() bool {
	return m.autofillLock || len(m.fieldLoading) > 0
}
