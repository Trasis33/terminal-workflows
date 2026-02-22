package manage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/picker"
	"github.com/fredriklanga/wf/internal/store"
)

// focusArea tracks which pane has keyboard focus in the browse view.
type focusArea int

const (
	focusList    focusArea = iota // workflow list has focus
	focusSidebar                  // sidebar has focus
)

// BrowseModel is the primary browse view with sidebar, workflow list,
// preview pane, and fuzzy search.
type BrowseModel struct {
	sidebar   SidebarModel
	focus     focusArea
	workflows []store.Workflow // full list (unfiltered)
	filtered  []store.Workflow // currently visible after filter + search
	cursor    int              // selected index in filtered list
	scrollOff int              // scroll offset for virtual scrolling

	searchInput textinput.Model
	searching   bool   // whether search input is active
	searchQuery string // current search text

	filterType  string // "folder", "tag", or "" (from sidebar)
	filterValue string // the folder path or tag name

	width  int
	height int
	theme  Theme
	keys   keyMap
}

// NewBrowseModel creates a browse view with workflows, folders, tags, and theme.
func NewBrowseModel(workflows []store.Workflow, folders []string, tags []string, theme Theme, keys keyMap) BrowseModel {
	ti := textinput.New()
	ti.Placeholder = "Search workflows..."
	ti.Prompt = "🔍 "
	ti.CharLimit = 128

	b := BrowseModel{
		sidebar:     NewSidebarModel(folders, tags, theme),
		focus:       focusList,
		workflows:   workflows,
		searchInput: ti,
		theme:       theme,
		keys:        keys,
	}
	b.applyFilter()
	return b
}

// SetDimensions updates layout dimensions for the browse view and its children.
func (b *BrowseModel) SetDimensions(width, height int) {
	b.width = width
	b.height = height

	sidebarWidth := b.theme.Layout.SidebarWidth
	// Sidebar gets its own height minus preview/hints
	listHeight := height - b.previewHeight() - 3        // hints + borders
	b.sidebar.SetDimensions(sidebarWidth, listHeight-4) // account for header/padding
}

// previewHeight returns the height allocated for the preview pane.
func (b BrowseModel) previewHeight() int {
	if !b.theme.Layout.ShowPreview {
		return 0
	}
	return 6 // fixed preview height (4 content + 2 border)
}

// listHeight returns the height available for the workflow list.
func (b BrowseModel) listHeight() int {
	h := b.height - b.previewHeight() - 3 // hints bar + spacing
	if b.searching {
		h -= 2 // search input + spacing
	}
	if h < 1 {
		h = 1
	}
	return h
}

// applyFilter recomputes filtered workflows from sidebar filter + search query.
func (b *BrowseModel) applyFilter() {
	// Step 1: Apply sidebar filter
	var prefiltered []store.Workflow
	switch b.filterType {
	case "folder":
		for _, wf := range b.workflows {
			if strings.HasPrefix(wf.Name, b.filterValue+"/") || wf.Name == b.filterValue {
				prefiltered = append(prefiltered, wf)
			}
		}
	case "tag":
		for _, wf := range b.workflows {
			for _, t := range wf.Tags {
				if strings.EqualFold(t, b.filterValue) {
					prefiltered = append(prefiltered, wf)
					break
				}
			}
		}
	default:
		prefiltered = b.workflows
	}

	// Step 2: Apply fuzzy search using picker.Search / picker.ParseQuery
	if b.searchQuery != "" {
		_, fuzzyQuery := picker.ParseQuery(b.searchQuery)
		matches := picker.Search(fuzzyQuery, "", prefiltered)
		result := make([]store.Workflow, 0, len(matches))
		for _, m := range matches {
			if m.Index >= 0 && m.Index < len(prefiltered) {
				result = append(result, prefiltered[m.Index])
			}
		}
		b.filtered = result
	} else {
		b.filtered = prefiltered
	}

	// Clamp cursor
	if b.cursor >= len(b.filtered) {
		b.cursor = len(b.filtered) - 1
	}
	if b.cursor < 0 {
		b.cursor = 0
	}
	b.scrollOff = 0
}

// Update handles messages for the browse view.
func (b BrowseModel) Update(msg tea.Msg) (BrowseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case sidebarFilterMsg:
		b.filterType = msg.filterType
		b.filterValue = msg.filterValue
		b.applyFilter()
		return b, nil

	case tea.KeyMsg:
		if b.searching {
			return b.updateSearch(msg)
		}

		// When sidebar has focus, route navigation keys to sidebar
		if b.focus == focusSidebar {
			return b.updateSidebarFocus(msg)
		}

		return b.updateListFocus(msg)
	}

	return b, nil
}

// updateListFocus handles keys when the workflow list has focus.
func (b BrowseModel) updateListFocus(msg tea.KeyMsg) (BrowseModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if b.cursor > 0 {
			b.cursor--
			b.ensureCursorVisible()
		}
		return b, nil

	case "down", "j":
		if b.cursor < len(b.filtered)-1 {
			b.cursor++
			b.ensureCursorVisible()
		}
		return b, nil

	case "/":
		b.searching = true
		b.searchInput.Focus()
		return b, textinput.Blink

	case "tab":
		b.sidebar.ToggleMode()
		b.filterType, b.filterValue = "", ""
		b.applyFilter()
		return b, nil

	case "left", "h":
		b.focus = focusSidebar
		return b, nil

	case "n":
		return b, func() tea.Msg { return switchToCreateMsg{} }

	case "e":
		if len(b.filtered) > 0 {
			return b, func() tea.Msg { return switchToEditMsg{workflow: b.filtered[b.cursor]} }
		}
		return b, nil

	case "d":
		if len(b.filtered) > 0 {
			wf := b.filtered[b.cursor]
			return b, func() tea.Msg { return showDeleteDialogMsg{workflow: wf} }
		}
		return b, nil

	case "m":
		if len(b.filtered) > 0 {
			wf := b.filtered[b.cursor]
			return b, func() tea.Msg { return moveWorkflowMsg{workflow: wf} }
		}
		return b, nil

	case "ctrl+t":
		return b, func() tea.Msg { return switchToSettingsMsg{} }

	case "q":
		return b, tea.Quit

	case "ctrl+c":
		return b, tea.Quit
	}

	return b, nil
}

// updateSidebarFocus handles keys when the sidebar has focus.
func (b BrowseModel) updateSidebarFocus(msg tea.KeyMsg) (BrowseModel, tea.Cmd) {
	switch msg.String() {
	case "right", "l", "esc":
		b.focus = focusList
		return b, nil

	case "tab":
		b.sidebar.ToggleMode()
		b.filterType, b.filterValue = "", ""
		b.applyFilter()
		return b, nil

	case "q":
		return b, tea.Quit

	case "ctrl+c":
		return b, tea.Quit

	default:
		var cmd tea.Cmd
		b.sidebar, cmd = b.sidebar.Update(msg)
		return b, cmd
	}
}

// updateSearch handles keys while the search input is active.
func (b BrowseModel) updateSearch(msg tea.KeyMsg) (BrowseModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		b.searching = false
		b.searchQuery = ""
		b.searchInput.SetValue("")
		b.searchInput.Blur()
		b.applyFilter()
		return b, nil

	case "enter":
		b.searching = false
		b.searchQuery = b.searchInput.Value()
		b.searchInput.Blur()
		// Keep current filter applied
		return b, nil

	default:
		var cmd tea.Cmd
		b.searchInput, cmd = b.searchInput.Update(msg)
		b.searchQuery = b.searchInput.Value()
		b.applyFilter()
		return b, cmd
	}
}

// ensureCursorVisible adjusts scroll offset to keep cursor in view.
func (b *BrowseModel) ensureCursorVisible() {
	visibleRows := b.listHeight()
	if b.cursor < b.scrollOff {
		b.scrollOff = b.cursor
	}
	if b.cursor >= b.scrollOff+visibleRows {
		b.scrollOff = b.cursor - visibleRows + 1
	}
}

// UpdateData refreshes workflow, folder, and tag data without losing browse state.
func (b *BrowseModel) UpdateData(workflows []store.Workflow, folders []string, tags []string) {
	b.workflows = workflows
	b.sidebar.UpdateData(folders, tags)
	b.applyFilter()
}

// View renders the full browse layout.
func (b BrowseModel) View() string {
	s := b.theme.Styles()

	sidebarWidth := b.theme.Layout.SidebarWidth

	// Calculate dimensions
	listPaneWidth := b.width - sidebarWidth - 4 // borders/padding
	if listPaneWidth < 20 {
		listPaneWidth = 20
	}
	lh := b.listHeight()

	// Render sidebar
	sidebarContent := b.sidebar.View()
	sidebarStyle := s.Sidebar.Width(sidebarWidth).Height(lh)
	if b.focus == focusSidebar {
		sidebarStyle = sidebarStyle.BorderForeground(lipgloss.Color(b.theme.Colors.Primary))
	}
	sidebarPane := sidebarStyle.Render(sidebarContent)

	// Render workflow list
	listContent := b.renderList(listPaneWidth, lh)
	listStyle := s.List.Width(listPaneWidth).Height(lh)
	listPane := listStyle.Render(listContent)

	// Join sidebar + list horizontally
	topPane := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, listPane)

	// Build vertical sections
	sections := []string{}

	// Search bar (only when searching)
	if b.searching {
		sections = append(sections, b.searchInput.View(), "")
	}

	sections = append(sections, topPane)

	// Preview pane
	if b.theme.Layout.ShowPreview {
		previewContent := b.renderPreview()
		previewWidth := b.width - 4
		if previewWidth < 20 {
			previewWidth = 20
		}
		previewPane := s.Preview.Width(previewWidth).Render(previewContent)
		sections = append(sections, previewPane)
	}

	// Hints bar
	sections = append(sections, b.renderHints(s))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderList renders the scrollable workflow list with cursor.
func (b BrowseModel) renderList(width, height int) string {
	s := b.theme.Styles()

	if len(b.filtered) == 0 {
		return s.Dim.Render("  No workflows found")
	}

	var rows []string
	end := b.scrollOff + height
	if end > len(b.filtered) {
		end = len(b.filtered)
	}

	for i := b.scrollOff; i < end; i++ {
		wf := b.filtered[i]
		isSelected := i == b.cursor

		// Build row
		var prefix string
		if isSelected {
			prefix = s.Selected.Render("❯ ")
		} else {
			prefix = "  "
		}

		// Workflow name
		name := wf.Name
		if isSelected {
			name = s.Selected.Render(name)
		}

		// Description (truncated)
		desc := browsetruncate(wf.Description, 30)
		if desc != "" {
			desc = " " + s.Dim.Render(desc)
		}

		// Tags
		var tags string
		if len(wf.Tags) > 0 {
			tags = " " + s.Tag.Render("["+strings.Join(wf.Tags, " ")+"]")
		}

		row := prefix + name + desc + tags

		// Truncate to width
		if lipgloss.Width(row) > width {
			row = row[:width-1] + "…"
		}

		rows = append(rows, row)
	}

	// Scroll indicator
	if len(b.filtered) > height {
		scrollInfo := s.Dim.Render(fmt.Sprintf(" %d/%d", b.cursor+1, len(b.filtered)))
		rows = append(rows, scrollInfo)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderPreview renders the preview pane showing the selected workflow's details.
func (b BrowseModel) renderPreview() string {
	s := b.theme.Styles()

	if len(b.filtered) == 0 || b.cursor >= len(b.filtered) {
		return s.Dim.Render("No workflow selected")
	}

	wf := b.filtered[b.cursor]

	var parts []string

	// Command
	parts = append(parts, s.Highlight.Render("Command: ")+wf.Command)

	// Folder (derived from name)
	if idx := strings.LastIndex(wf.Name, "/"); idx >= 0 {
		parts = append(parts, s.Dim.Render("Folder:  ")+wf.Name[:idx])
	}

	// Tags
	if len(wf.Tags) > 0 {
		parts = append(parts, s.Dim.Render("Tags:    ")+s.Tag.Render(strings.Join(wf.Tags, ", ")))
	}

	// Description
	if wf.Description != "" {
		parts = append(parts, s.Dim.Render("Desc:    ")+wf.Description)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// browsetruncate truncates a string to maxLen, adding "…" if truncated.
func browsetruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return s[:maxLen-1] + "…"
}

// renderHints renders context-sensitive keybinding hints.
func (b BrowseModel) renderHints(s themeStyles) string {
	var hints string
	if b.searching {
		hints = "esc cancel  enter confirm"
	} else if b.focus == focusSidebar {
		hints = "↑↓ navigate  enter filter  →/esc list  tab folders/tags  q quit"
	} else {
		hints = "n new  e edit  d delete  m move  / search  tab folders/tags  ←/h sidebar  ctrl+t theme  q quit"
	}
	return s.Hint.Render("  " + hints)
}
