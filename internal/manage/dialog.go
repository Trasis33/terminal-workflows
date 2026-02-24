package manage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dialogType identifies the kind of dialog being displayed.
type dialogType int

const (
	dialogDeleteConfirm dialogType = iota
	dialogFolderCreate
	dialogFolderRename
	dialogFolderDelete
	dialogMoveWorkflow
)

// dialogResultMsg carries the result of a dialog interaction back to the root model.
type dialogResultMsg struct {
	dtype     dialogType
	confirmed bool
	data      map[string]string // key-value pairs (e.g., "name", "folder")
}

// DialogModel manages an overlay dialog with optional text input or list selection.
type DialogModel struct {
	dtype   dialogType
	title   string
	message string

	input    textinput.Model
	hasInput bool // whether this dialog has a text input field

	options      []string // for move dialog: list of available folders
	optionCursor int      // for move dialog: selected folder index

	workflowName string // the workflow being acted on (for delete/move)
	folderPath   string // for folder rename: the original path

	theme Theme
}

// NewDeleteDialog creates a delete confirmation dialog for a workflow.
func NewDeleteDialog(workflowName string, theme Theme) DialogModel {
	return DialogModel{
		dtype:        dialogDeleteConfirm,
		title:        "Delete Workflow",
		message:      fmt.Sprintf("Delete '%s'?\nThis cannot be undone.", workflowName),
		workflowName: workflowName,
		theme:        theme,
	}
}

// NewFolderCreateDialog creates a dialog with text input for a new folder name.
func NewFolderCreateDialog(theme Theme) DialogModel {
	ti := textinput.New()
	ti.Placeholder = "folder-name"
	ti.CharLimit = 64
	ti.Focus()

	return DialogModel{
		dtype:    dialogFolderCreate,
		title:    "Create Folder",
		message:  "Enter new folder name:",
		input:    ti,
		hasInput: true,
		theme:    theme,
	}
}

// NewFolderRenameDialog creates a dialog with text input pre-filled with the old folder name.
func NewFolderRenameDialog(oldPath string, theme Theme) DialogModel {
	ti := textinput.New()
	ti.SetValue(oldPath)
	ti.CharLimit = 64
	ti.Focus()

	return DialogModel{
		dtype:      dialogFolderRename,
		title:      "Rename Folder",
		message:    fmt.Sprintf("Rename '%s' to:", oldPath),
		input:      ti,
		hasInput:   true,
		folderPath: oldPath,
		theme:      theme,
	}
}

// NewFolderDeleteDialog creates a delete confirmation for a folder.
func NewFolderDeleteDialog(folderPath string, theme Theme) DialogModel {
	return DialogModel{
		dtype:      dialogFolderDelete,
		title:      "Delete Folder",
		message:    fmt.Sprintf("Delete folder '%s'?\nFolder must be empty.", folderPath),
		folderPath: folderPath,
		theme:      theme,
	}
}

// NewMoveDialog creates a dialog for moving a workflow to a different folder.
func NewMoveDialog(workflowName string, folders []string, theme Theme) DialogModel {
	// Add root ("") as an option for moving to top-level.
	options := make([]string, 0, len(folders)+1)
	options = append(options, "(root)")
	options = append(options, folders...)

	return DialogModel{
		dtype:        dialogMoveWorkflow,
		title:        "Move Workflow",
		message:      fmt.Sprintf("Move '%s' to:", workflowName),
		workflowName: workflowName,
		options:      options,
		theme:        theme,
	}
}

// Init returns the initial command for the dialog.
func (d DialogModel) Init() tea.Cmd {
	if d.hasInput {
		return textinput.Blink
	}
	return nil
}

// Update processes key messages for the dialog.
func (d DialogModel) Update(msg tea.Msg) (DialogModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Forward non-key messages to text input if present.
		if d.hasInput {
			var cmd tea.Cmd
			d.input, cmd = d.input.Update(msg)
			return d, cmd
		}
		return d, nil
	}

	switch d.dtype {
	case dialogDeleteConfirm, dialogFolderDelete:
		return d.updateConfirm(keyMsg)
	case dialogFolderCreate, dialogFolderRename, dialogAIGenerate:
		return d.updateInput(keyMsg)
	case dialogMoveWorkflow:
		return d.updateMove(keyMsg)
	}

	return d, nil
}

// updateConfirm handles y/n confirmation dialogs (delete workflow, delete folder).
func (d DialogModel) updateConfirm(msg tea.KeyMsg) (DialogModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		data := map[string]string{}
		if d.workflowName != "" {
			data["name"] = d.workflowName
		}
		if d.folderPath != "" {
			data["folder"] = d.folderPath
		}
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: d.dtype, confirmed: true, data: data}
		}
	case "n", "N", "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: d.dtype, confirmed: false}
		}
	}
	return d, nil
}

// updateInput handles text input dialogs (folder create, folder rename).
func (d DialogModel) updateInput(msg tea.KeyMsg) (DialogModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(d.input.Value())
		if value == "" {
			return d, nil // Don't submit empty input.
		}
		data := map[string]string{"name": value}
		if d.folderPath != "" {
			data["oldPath"] = d.folderPath
		}
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: d.dtype, confirmed: true, data: data}
		}
	case "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: d.dtype, confirmed: false}
		}
	default:
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return d, cmd
	}
}

// updateMove handles the folder selection list for move workflow.
func (d DialogModel) updateMove(msg tea.KeyMsg) (DialogModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if d.optionCursor > 0 {
			d.optionCursor--
		}
		return d, nil
	case "down", "j":
		if d.optionCursor < len(d.options)-1 {
			d.optionCursor++
		}
		return d, nil
	case "enter":
		selected := d.options[d.optionCursor]
		folder := ""
		if selected != "(root)" {
			folder = selected
		}
		data := map[string]string{
			"name":   d.workflowName,
			"folder": folder,
		}
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: d.dtype, confirmed: true, data: data}
		}
	case "esc":
		return d, func() tea.Msg {
			return dialogResultMsg{dtype: d.dtype, confirmed: false}
		}
	}
	return d, nil
}

// View renders the dialog content (without overlay — overlay is handled by model.go).
func (d DialogModel) View() string {
	s := d.theme.Styles()

	title := s.DialogTitle.Render(d.title)
	body := d.message

	var content string
	switch {
	case d.hasInput:
		content = lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			body, "",
			d.input.View(), "",
			s.Dim.Render("[enter] submit  [esc] cancel"),
		)
	case d.dtype == dialogMoveWorkflow:
		// Render folder list with cursor.
		var items []string
		for i, opt := range d.options {
			if i == d.optionCursor {
				items = append(items, s.Selected.Render("❯ "+opt))
			} else {
				items = append(items, "  "+opt)
			}
		}
		folderList := lipgloss.JoinVertical(lipgloss.Left, items...)
		content = lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			body, "",
			folderList, "",
			s.Dim.Render("[↑↓] select  [enter] move  [esc] cancel"),
		)
	default:
		// Simple y/n confirmation.
		content = lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			body, "",
			s.Dim.Render("[y]es  [n]o"),
		)
	}

	return s.DialogBox.Render(content)
}
