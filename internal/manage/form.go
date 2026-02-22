package manage

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/store"
)

// Validation errors for form fields.
var (
	errNameRequired    = errors.New("name is required")
	errNameNoSlash     = errors.New("name must not contain slashes (use folder field)")
	errCommandRequired = errors.New("command is required")
)

// FormModel manages a full-screen huh form for creating or editing a workflow.
type FormModel struct {
	form *huh.Form
	mode string // "create" or "edit"

	// Edit mode: tracks the original workflow name to handle renames.
	originalName string

	store store.Store
	theme Theme

	width  int
	height int

	// Form field values — bound to huh inputs via pointers.
	name        string
	description string
	command     string
	tagInput    string // comma-separated tags
	folder      string

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
		existingTags:    existingTags,
		existingFolders: existingFolders,
	}

	if mode == "edit" && wf != nil {
		m.originalName = wf.Name

		// Extract folder from name (everything before last /).
		if idx := strings.LastIndex(wf.Name, "/"); idx >= 0 {
			m.folder = wf.Name[:idx]
			m.name = wf.Name[idx+1:]
		} else {
			m.name = wf.Name
		}

		m.description = wf.Description
		m.command = wf.Command
		m.tagInput = strings.Join(wf.Tags, ", ")
	}

	m.form = m.buildForm()
	return m
}

// buildForm constructs the huh.Form with all workflow fields.
func (m *FormModel) buildForm() *huh.Form {
	nameInput := huh.NewInput().
		Title("Name").
		Value(&m.name).
		Placeholder("my-workflow").
		Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errNameRequired
			}
			if strings.ContainsAny(s, "/\\") {
				return errNameNoSlash
			}
			return nil
		})

	descInput := huh.NewInput().
		Title("Description").
		Value(&m.description).
		Placeholder("What does this workflow do?")

	cmdInput := huh.NewText().
		Title("Command").
		Value(&m.command).
		Lines(8).
		Placeholder("Enter command (alt+enter for newline)").
		Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errCommandRequired
			}
			return nil
		})

	tagInputField := huh.NewInput().
		Title("Tags (comma-separated)").
		Value(&m.tagInput).
		Placeholder("e.g., docker, deploy, infra")
	if len(m.existingTags) > 0 {
		tagInputField = tagInputField.SuggestionsFunc(func() []string {
			return m.existingTags
		}, &m.tagInput)
	}

	folderInput := huh.NewInput().
		Title("Folder").
		Value(&m.folder).
		Placeholder("e.g., infra/deploy (empty for root)")
	if len(m.existingFolders) > 0 {
		folderInput = folderInput.SuggestionsFunc(func() []string {
			return m.existingFolders
		}, &m.folder)
	}

	f := huh.NewForm(
		huh.NewGroup(
			nameInput,
			descInput,
			cmdInput,
			tagInputField,
			folderInput,
		),
	).WithTheme(huh.ThemeCharm())

	return f
}

// Init returns the initial command for the form (delegates to huh).
func (m FormModel) Init() tea.Cmd {
	return m.form.Init()
}

// Update processes messages for the form model.
func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	// Check for esc key to abort form and return to browse.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
			return m, func() tea.Msg { return switchToBrowseMsg{} }
		}
	}

	// Delegate to huh form.
	model, cmd := m.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.form = f
	}

	// Check form state after update.
	switch m.form.State {
	case huh.StateCompleted:
		return m, m.saveWorkflow()
	case huh.StateAborted:
		return m, func() tea.Msg { return switchToBrowseMsg{} }
	}

	return m, cmd
}

// saveWorkflow builds a Workflow from form fields and persists it via Store.
func (m FormModel) saveWorkflow() tea.Cmd {
	return func() tea.Msg {
		// Parse tags from comma-separated input.
		tags := parseTags(m.tagInput)

		// Build full name with folder prefix.
		fullName := strings.TrimSpace(m.name)
		folder := strings.TrimSpace(m.folder)
		folder = strings.Trim(folder, "/")
		if folder != "" {
			fullName = folder + "/" + fullName
		}

		wf := store.Workflow{
			Name:        fullName,
			Command:     strings.TrimSpace(m.command),
			Description: strings.TrimSpace(m.description),
			Tags:        tags,
		}

		// If editing and name changed, delete the old workflow first.
		if m.mode == "edit" && m.originalName != "" && m.originalName != fullName {
			if err := m.store.Delete(m.originalName); err != nil {
				return saveErrorMsg{err: err}
			}
		}

		if err := m.store.Save(&wf); err != nil {
			return saveErrorMsg{err: err}
		}

		return workflowSavedMsg{workflow: wf}
	}
}

// View renders the form with title and hints.
func (m FormModel) View() string {
	s := m.theme.Styles()

	// Form title.
	var title string
	if m.mode == "edit" {
		title = s.FormTitle.Render("Edit Workflow")
	} else {
		title = s.FormTitle.Render("Create Workflow")
	}

	// Form body.
	formView := m.form.View()

	// Error display.
	var errLine string
	if m.err != nil {
		errLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("Error: " + m.err.Error())
	}

	// Hints.
	hints := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Colors.Dim)).
		Render("  esc cancel  tab next field  alt+enter newline in command")

	// Compose vertically.
	sections := []string{"", title, formView}
	if errLine != "" {
		sections = append(sections, errLine)
	}
	sections = append(sections, hints)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// SetDimensions updates the form's available space.
func (m *FormModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.form.WithWidth(width)
	m.form.WithHeight(height - 6) // leave room for title, hints, padding
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
