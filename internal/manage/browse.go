package manage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredriklanga/wf/internal/highlight"
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

	previewVP   viewport.Model
	tokenStyles highlight.TokenStyles

	aiError string // transient AI error, cleared on next key press

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
		tokenStyles: highlight.TokenStylesFromColors(
			theme.Colors.Primary,
			theme.Colors.Secondary,
			theme.Colors.Tertiary,
			theme.Colors.Dim,
			theme.Colors.Text,
		),
		previewVP: viewport.New(0, 4),
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

	previewW := width - 8 // preview border + padding
	if previewW < 20 {
		previewW = 20
	}
	b.previewVP.Width = previewW
	b.previewVP.Height = 4
	b.updatePreviewContent()
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
	b.updatePreviewContent()
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
		// Clear transient AI error on any key press.
		b.aiError = ""

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
			b.updatePreviewContent()
		}
		return b, nil

	case "down", "j":
		if b.cursor < len(b.filtered)-1 {
			b.cursor++
			b.ensureCursorVisible()
			b.updatePreviewContent()
		}
		return b, nil

	case "J":
		b.previewVP.LineDown(1)
		return b, nil

	case "K":
		b.previewVP.LineUp(1)
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

	case "S":
		return b, func() tea.Msg { return switchToSettingsMsg{} }

	case "G":
		return b, func() tea.Msg { return showAIGenerateDialogMsg{} }

	case "A":
		if len(b.filtered) > 0 {
			wf := b.filtered[b.cursor]
			return b, func() tea.Msg { return showAIAutofillDialogMsg{workflow: wf} }
		}
		return b, nil

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

	sections = append(sections, b.renderBreadcrumb())
	sections = append(sections, topPane)

	// Preview pane
	if b.theme.Layout.ShowPreview {
		previewContent := b.renderPreviewPane()
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
		if b.filterType == "folder" {
			return s.Dim.Render("  No workflows in " + b.filterValue + "/")
		}
		if b.filterType == "tag" {
			return s.Dim.Render("  No workflows tagged " + b.filterValue)
		}
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

func (b BrowseModel) renderBreadcrumb() string {
	s := b.theme.Styles()
	switch b.filterType {
	case "folder":
		return s.Highlight.Render("Folder: " + b.filterValue + "/")
	case "tag":
		return s.Tag.Render("Tag: " + b.filterValue)
	default:
		return s.Dim.Render("All Workflows")
	}
}

func (b *BrowseModel) updatePreviewContent() {
	s := b.theme.Styles()

	if len(b.filtered) == 0 || b.cursor >= len(b.filtered) {
		b.previewVP.SetContent(s.Dim.Render("No workflow selected"))
		return
	}

	wf := b.filtered[b.cursor]

	var parts []string

	// Command
	parts = append(parts, highlight.Shell(wf.Command, b.tokenStyles))

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

	b.previewVP.SetContent(lipgloss.JoinVertical(lipgloss.Left, parts...))
	b.previewVP.GotoTop()
}

func (b BrowseModel) renderPreviewPane() string {
	content := b.previewVP.View()
	if b.previewVP.TotalLineCount() <= b.previewVP.Height {
		return content
	}
	percent := int(b.previewVP.ScrollPercent() * 100)
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	indicator := b.theme.Styles().Dim.Render(fmt.Sprintf("Scroll %d%%", percent))
	return lipgloss.JoinVertical(lipgloss.Left, content, indicator)
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
	// Show transient AI error instead of normal hints if set.
	if b.aiError != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		return errStyle.Render("  ⚠ " + b.aiError)
	}

	var hints string
	if b.searching {
		hints = "esc cancel  enter confirm"
	} else if b.focus == focusSidebar {
		hints = "↑↓ navigate  enter filter  →/esc list  tab folders/tags  q quit"
	} else {
		hints = "n new  e edit  d delete  m move  G generate  A autofill  / search  J/K preview scroll  tab folders/tags  ←/h sidebar  S settings  q quit"
	}
	return s.Hint.Render("  " + hints)
}
