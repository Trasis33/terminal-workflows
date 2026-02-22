package manage

import (
	"fmt"
	"os"
	"path/filepath"
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
	browse   BrowseModel
	form     FormModel
	settings SettingsModel

	// Dialog overlay (nil = no dialog active).
	dialog *DialogModel
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
		if m.state == viewSettings {
			m.settings.width = msg.Width
			m.settings.height = msg.Height
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

	case dialogResultMsg:
		return m.handleDialogResult(msg)

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

	case showDeleteDialogMsg:
		dlg := NewDeleteDialog(msg.workflow.Name, m.theme)
		m.dialog = &dlg
		return m, dlg.Init()

	case showFolderDialogMsg:
		return m.handleFolderDialogMsg(msg)

	case moveWorkflowMsg:
		folders := extractFolders(m.workflows)
		dlg := NewMoveDialog(msg.workflow.Name, folders, m.theme)
		m.dialog = &dlg
		return m, nil

	case switchToSettingsMsg:
		m.prevState = m.state
		m.state = viewSettings
		m.settings = NewSettingsModel(m.theme, m.configDir)
		m.settings.width = m.width
		m.settings.height = m.height
		return m, nil

	case themeSavedMsg:
		m.prevState = m.state
		m.state = viewBrowse
		// Reload saved theme.
		saved, err := LoadTheme(m.configDir)
		if err == nil {
			m.theme = saved
			// Rebuild browse model with new theme.
			folders := extractFolders(m.workflows)
			tags := extractTags(m.workflows)
			m.browse = NewBrowseModel(m.workflows, folders, tags, m.theme, m.keys)
			m.browse.SetDimensions(m.width, m.height)
		}
		return m, nil
	}

	// Route to active view.
	switch m.state {
	case viewBrowse:
		return m.updateBrowse(msg)
	case viewCreate, viewEdit:
		return m.updateForm(msg)
	case viewSettings:
		return m.updateSettings(msg)
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

// updateDialog routes key events to the active dialog model.
func (m Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	dlg := *m.dialog
	dlg, cmd = dlg.Update(msg)
	m.dialog = &dlg
	return m, cmd
}

// handleDialogResult processes results from dialog interactions.
func (m Model) handleDialogResult(msg dialogResultMsg) (tea.Model, tea.Cmd) {
	m.dialog = nil // dismiss dialog

	if !msg.confirmed {
		return m, nil
	}

	switch msg.dtype {
	case dialogDeleteConfirm:
		name := msg.data["name"]
		return m, func() tea.Msg {
			if err := m.store.Delete(name); err != nil {
				return saveErrorMsg{err: err}
			}
			return refreshWorkflowsMsg{}
		}

	case dialogFolderCreate:
		folderName := msg.data["name"]
		return m, func() tea.Msg {
			dir := filepath.Join(m.configDir, "workflows", folderName)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return saveErrorMsg{err: fmt.Errorf("create folder: %w", err)}
			}
			return refreshWorkflowsMsg{}
		}

	case dialogFolderRename:
		oldPath := msg.data["oldPath"]
		newName := msg.data["name"]
		return m, func() tea.Msg {
			oldDir := filepath.Join(m.configDir, "workflows", oldPath)
			newDir := filepath.Join(m.configDir, "workflows", newName)
			if err := os.Rename(oldDir, newDir); err != nil {
				return saveErrorMsg{err: fmt.Errorf("rename folder: %w", err)}
			}
			return refreshWorkflowsMsg{}
		}

	case dialogFolderDelete:
		folderPath := msg.data["folder"]
		return m, func() tea.Msg {
			dir := filepath.Join(m.configDir, "workflows", folderPath)
			if err := os.Remove(dir); err != nil {
				return saveErrorMsg{err: fmt.Errorf("delete folder: %w", err)}
			}
			return refreshWorkflowsMsg{}
		}

	case dialogMoveWorkflow:
		oldName := msg.data["name"]
		newFolder := msg.data["folder"]
		return m, func() tea.Msg {
			wf, err := m.store.Get(oldName)
			if err != nil || wf == nil {
				return saveErrorMsg{err: fmt.Errorf("workflow not found: %s", oldName)}
			}

			// Extract base name (last segment).
			baseName := oldName
			if idx := strings.LastIndex(oldName, "/"); idx >= 0 {
				baseName = oldName[idx+1:]
			}

			// Build new name.
			newName := baseName
			if newFolder != "" {
				newName = newFolder + "/" + baseName
			}

			if newName == oldName {
				return refreshWorkflowsMsg{} // no change needed
			}

			// Save with new name, delete old.
			moved := *wf
			moved.Name = newName
			if err := m.store.Save(&moved); err != nil {
				return saveErrorMsg{err: fmt.Errorf("move workflow: %w", err)}
			}
			if err := m.store.Delete(oldName); err != nil {
				return saveErrorMsg{err: fmt.Errorf("cleanup old workflow: %w", err)}
			}

			return refreshWorkflowsMsg{}
		}
	}

	return m, nil
}

// handleFolderDialogMsg creates the appropriate folder dialog based on action.
func (m Model) handleFolderDialogMsg(msg showFolderDialogMsg) (tea.Model, tea.Cmd) {
	switch msg.action {
	case "create":
		dlg := NewFolderCreateDialog(m.theme)
		m.dialog = &dlg
		return m, dlg.Init()
	case "rename":
		// Rename requires a selected folder — use sidebar's current selection.
		dlg := NewFolderRenameDialog(msg.action, m.theme)
		m.dialog = &dlg
		return m, dlg.Init()
	case "delete":
		dlg := NewFolderDeleteDialog(msg.action, m.theme)
		m.dialog = &dlg
		return m, nil
	}
	return m, nil
}

// updateSettings routes messages to the SettingsModel.
func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.settings, cmd = m.settings.Update(msg)
	return m, cmd
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
	var base string
	switch m.state {
	case viewCreate, viewEdit:
		base = m.form.View()
	case viewSettings:
		base = m.settings.View()
	default:
		base = m.viewBrowse()
	}

	if m.dialog != nil {
		return m.renderOverlay(base, m.dialog.View())
	}

	return base
}

// viewBrowse renders the browse view via the BrowseModel.
func (m Model) viewBrowse() string {
	return m.browse.View()
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
