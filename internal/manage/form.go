package manage

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	fieldParams // the param editor section
	fieldCount  // sentinel — total number of focusable fields
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
	m.nameInput.Focus() // name is initially focused

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

// Init returns the initial command for the form.
func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update processes messages for the form model.
func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward non-key messages to the focused widget.
	return m.updateFocusedWidget(msg)
}

// handleKey processes keyboard input for navigation and actions.
func (m FormModel) handleKey(msg tea.KeyMsg) (FormModel, tea.Cmd) {
	key := msg.String()

	// Esc returns to browse — but only if the param editor isn't in editing mode.
	if key == "esc" {
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
	sectionDivider := labelStyle.Render("── Parameters ──")

	// Build field rows.
	var rows []string
	rows = append(rows, "", title)

	// Name.
	lbl := labelStyle
	if m.focused == fieldName {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Name"))
	rows = append(rows, "  "+m.nameInput.View())

	// Description.
	lbl = labelStyle
	if m.focused == fieldDescription {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Description"))
	rows = append(rows, "  "+m.descInput.View())

	// Command.
	lbl = labelStyle
	if m.focused == fieldCommand {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Command"))
	rows = append(rows, "  "+m.cmdInput.View())

	// Tags.
	lbl = labelStyle
	if m.focused == fieldTags {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Tags (comma-separated)"))
	rows = append(rows, "  "+m.tagsInput.View())

	// Folder.
	lbl = labelStyle
	if m.focused == fieldFolder {
		lbl = focusLabelStyle
	}
	rows = append(rows, lbl.Render("  Folder"))
	rows = append(rows, "  "+m.folderInput.View())

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
	hints := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Dim)).
		Render("  esc cancel  tab next  ctrl+s save  ctrl+n new param  ctrl+d delete param")

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
