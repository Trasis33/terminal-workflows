package manage

import (
	"fmt"

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

	// Placeholder — Plans 02-04 populate real child models.
	browseReady bool

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
		}
		return m, nil

	case switchToBrowseMsg:
		m.prevState = m.state
		m.state = viewBrowse
		return m, nil

	case switchToCreateMsg:
		m.prevState = m.state
		m.state = viewCreate
		return m, nil

	case switchToEditMsg:
		m.prevState = m.state
		m.state = viewEdit
		return m, nil

	case switchToSettingsMsg:
		m.prevState = m.state
		m.state = viewSettings
		return m, nil
	}

	// Route to active view.
	switch m.state {
	case viewBrowse:
		return m.updateBrowse(msg)
	}

	return m, nil
}

// updateBrowse handles messages in the browse state.
// Placeholder implementation — Plan 02 will add the full browse view.
func (m Model) updateBrowse(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
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
	default:
		// Placeholder for views not yet implemented.
		return m.viewBrowse()
	}
}

// viewBrowse renders the placeholder browse view.
func (m Model) viewBrowse() string {
	s := m.theme.Styles()

	title := s.Highlight.Render("wf manage")
	count := s.Dim.Render(fmt.Sprintf("%d workflows", len(m.workflows)))
	hint := s.Hint.Render("q quit  n new  e edit  d delete  ? help")

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		title,
		count,
		"",
		hint,
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
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
