package manage

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/store"
)

// viewState tracks which view is active in the management TUI.
type viewState int

const (
	viewBrowse   viewState = iota // Workflow browsing/listing
	viewCreate                    // New workflow form
	viewEdit                      // Edit workflow form
	viewSettings                  // Theme/settings view
)

// --- Custom message types for view transitions ---

type switchToEditMsg struct{ workflow store.Workflow }
type switchToCreateMsg struct{}
type switchToBrowseMsg struct{}
type switchToSettingsMsg struct{}
type showDeleteDialogMsg struct{ workflow store.Workflow }
type deleteConfirmedMsg struct{ workflow store.Workflow }
type showFolderDialogMsg struct{ action string } // "create", "rename", "delete"
type refreshWorkflowsMsg struct{}
type workflowSavedMsg struct{ workflow store.Workflow }
type saveErrorMsg struct{ err error }
type themeSavedMsg struct{}
type moveWorkflowMsg struct{ workflow store.Workflow }

// workflowsLoadedMsg carries reloaded workflows from the store.
type workflowsLoadedMsg struct {
	workflows []store.Workflow
	err       error
}

// dialogState holds the state for an active confirmation dialog.
type dialogState struct {
	title     string
	message   string
	onConfirm tea.Cmd
}

// Model is the root Bubble Tea model for the management TUI.
type Model struct {
	state     viewState
	prevState viewState

	store     store.Store
	workflows []store.Workflow
	theme     Theme
	keys      keyMap

	width  int
	height int

	configDir string // for theme persistence

	// Child view models.
	browse BrowseModel
	form   FormModel

	// Dialog overlay (nil = no dialog active).
	dialog *dialogState
}

// Init returns the initial command for the management TUI.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update processes messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browse.SetDimensions(msg.Width, msg.Height)
		if m.state == viewCreate || m.state == viewEdit {
			m.form.SetDimensions(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyMsg:
		// Dialog gets priority if active.
		if m.dialog != nil {
			return m.updateDialog(msg)
		}

		// Global quit handling.
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case refreshWorkflowsMsg:
		return m, m.loadWorkflows()

	case workflowsLoadedMsg:
		if msg.err == nil {
			m.workflows = msg.workflows
			m.browse.UpdateData(msg.workflows, extractFolders(msg.workflows), extractTags(msg.workflows))
		}
		return m, nil

	case switchToBrowseMsg:
		m.prevState = m.state
		m.state = viewBrowse
		return m, nil

	case workflowSavedMsg:
		m.prevState = m.state
		m.state = viewBrowse
		return m, m.loadWorkflows()

	case saveErrorMsg:
		m.form.err = msg.err
		return m, nil

	case switchToCreateMsg:
		m.prevState = m.state
		m.state = viewCreate
		folders := extractFolders(m.workflows)
		tags := extractTags(m.workflows)
		m.form = NewFormModel("create", nil, m.store, tags, folders, m.theme)
		m.form.SetDimensions(m.width, m.height)
		return m, m.form.Init()

	case switchToEditMsg:
		m.prevState = m.state
		m.state = viewEdit
		wf := msg.workflow
		folders := extractFolders(m.workflows)
		tags := extractTags(m.workflows)
		m.form = NewFormModel("edit", &wf, m.store, tags, folders, m.theme)
		m.form.SetDimensions(m.width, m.height)
		return m, m.form.Init()

	case switchToSettingsMsg:
		m.prevState = m.state
		m.state = viewSettings
		return m, nil
	}

	// Route to active view.
	switch m.state {
	case viewBrowse:
		return m.updateBrowse(msg)
	case viewCreate, viewEdit:
		return m.updateForm(msg)
	}

	return m, nil
}

// updateBrowse routes messages to the BrowseModel.
func (m Model) updateBrowse(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.browse, cmd = m.browse.Update(msg)
	return m, cmd
}

// updateForm routes messages to the FormModel.
func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

// updateDialog handles key events when a dialog is active.
func (m Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		cmd := m.dialog.onConfirm
		m.dialog = nil
		return m, cmd
	case "n", "esc":
		m.dialog = nil
		return m, nil
	}
	return m, nil
}

// loadWorkflows returns a command that reloads workflows from the store.
func (m Model) loadWorkflows() tea.Cmd {
	return func() tea.Msg {
		wfs, err := m.store.List()
		return workflowsLoadedMsg{workflows: wfs, err: err}
	}
}

// View renders the management TUI.
func (m Model) View() string {
	switch m.state {
	case viewBrowse:
		base := m.viewBrowse()
		if m.dialog != nil {
			return m.renderOverlay(base, m.viewDialog())
		}
		return base
	case viewCreate, viewEdit:
		return m.form.View()
	default:
		return m.viewBrowse()
	}
}

// viewBrowse renders the browse view via the BrowseModel.
func (m Model) viewBrowse() string {
	return m.browse.View()
}

// viewDialog renders the active dialog overlay content.
func (m Model) viewDialog() string {
	if m.dialog == nil {
		return ""
	}
	s := m.theme.Styles()
	title := s.DialogTitle.Render(m.dialog.title)
	body := m.dialog.message + "\n\n" + s.Dim.Render("[y]es  [n]o")
	return s.DialogBox.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", body))
}

// renderOverlay centers dialog content over the full terminal area.
func (m Model) renderOverlay(_ string, dialog string) string {
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

// extractFolders derives unique folder paths from workflow names.
// A workflow named "infra/deploy/app" produces folders ["infra", "infra/deploy"].
func extractFolders(workflows []store.Workflow) []string {
	seen := make(map[string]struct{})
	for _, wf := range workflows {
		parts := strings.Split(wf.Name, "/")
		// Each prefix up to (but not including) the last segment is a folder.
		for i := 1; i < len(parts); i++ {
			folder := strings.Join(parts[:i], "/")
			seen[folder] = struct{}{}
		}
	}
	folders := make([]string, 0, len(seen))
	for f := range seen {
		folders = append(folders, f)
	}
	sort.Strings(folders)
	return folders
}

// extractTags derives unique, sorted tags from all workflows.
func extractTags(workflows []store.Workflow) []string {
	seen := make(map[string]struct{})
	for _, wf := range workflows {
		for _, t := range wf.Tags {
			seen[t] = struct{}{}
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}
