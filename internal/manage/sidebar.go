package manage

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// sidebarMode determines which view the sidebar displays.
type sidebarMode int

const (
	sidebarFolders sidebarMode = iota
	sidebarTags
)

// sidebarFilterMsg is sent when the user selects a folder or tag in the sidebar.
type sidebarFilterMsg struct {
	filterType  string // "folder", "tag", or "" (all)
	filterValue string // folder path or tag name
}

// SidebarModel provides a switchable sidebar with folder tree and tag list views.
// Both views persist simultaneously — toggling switches the active view without
// recreating models or losing cursor state (research pitfall #5).
type SidebarModel struct {
	mode sidebarMode

	folders      []string // sorted folder paths (relative)
	tags         []string // deduplicated, sorted tag names
	folderCursor int      // 0 = "All Workflows" virtual root
	tagCursor    int      // 0 = "All Tags" virtual root

	width  int
	height int
	theme  Theme
}

// NewSidebarModel creates a sidebar with folder and tag data.
func NewSidebarModel(folders []string, tags []string, theme Theme) SidebarModel {
	sortedFolders := make([]string, len(folders))
	copy(sortedFolders, folders)
	sort.Strings(sortedFolders)

	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)

	return SidebarModel{
		mode:    sidebarFolders,
		folders: sortedFolders,
		tags:    sortedTags,
		theme:   theme,
	}
}

// ToggleMode switches between folder and tag views.
func (s *SidebarModel) ToggleMode() {
	if s.mode == sidebarFolders {
		s.mode = sidebarTags
	} else {
		s.mode = sidebarFolders
	}
}

// SetDimensions updates the sidebar's allocated dimensions.
func (s *SidebarModel) SetDimensions(width, height int) {
	s.width = width
	s.height = height
}

// SelectedFilter returns the current selection as a filter type and value.
// Returns ("", "") for "All Workflows" or "All Tags".
func (s SidebarModel) SelectedFilter() (filterType string, filterValue string) {
	switch s.mode {
	case sidebarFolders:
		if s.folderCursor == 0 || len(s.folders) == 0 {
			return "", ""
		}
		return "folder", s.folders[s.folderCursor-1]
	case sidebarTags:
		if s.tagCursor == 0 || len(s.tags) == 0 {
			return "", ""
		}
		return "tag", s.tags[s.tagCursor-1]
	}
	return "", ""
}

// Update handles keyboard navigation within the sidebar.
func (s SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			s.moveCursor(-1)
			ft, fv := s.SelectedFilter()
			return s, func() tea.Msg {
				return sidebarFilterMsg{filterType: ft, filterValue: fv}
			}
		case "down", "j":
			s.moveCursor(1)
			ft, fv := s.SelectedFilter()
			return s, func() tea.Msg {
				return sidebarFilterMsg{filterType: ft, filterValue: fv}
			}
		case "enter":
			ft, fv := s.SelectedFilter()
			return s, func() tea.Msg {
				return sidebarFilterMsg{filterType: ft, filterValue: fv}
			}
		}
	}
	return s, nil
}

// moveCursor moves the cursor by delta within the current mode's list bounds.
func (s *SidebarModel) moveCursor(delta int) {
	switch s.mode {
	case sidebarFolders:
		maxIdx := len(s.folders) // +1 for "All Workflows" at index 0
		s.folderCursor += delta
		if s.folderCursor < 0 {
			s.folderCursor = 0
		}
		if s.folderCursor > maxIdx {
			s.folderCursor = maxIdx
		}
	case sidebarTags:
		maxIdx := len(s.tags) // +1 for "All Tags" at index 0
		s.tagCursor += delta
		if s.tagCursor < 0 {
			s.tagCursor = 0
		}
		if s.tagCursor > maxIdx {
			s.tagCursor = maxIdx
		}
	}
}

// View renders the sidebar content based on the active mode.
func (s SidebarModel) View() string {
	styles := s.theme.Styles()

	var header string
	var items []string

	switch s.mode {
	case sidebarFolders:
		header = styles.Highlight.Render("📁 Folders")
		items = s.renderFolders(styles)
	case sidebarTags:
		header = styles.Highlight.Render("🏷  Tags")
		items = s.renderTags(styles)
	}

	parts := []string{header, ""}
	parts = append(parts, items...)

	// Fill remaining height with empty lines for stable layout
	contentLines := len(parts)
	if s.height > contentLines {
		for i := 0; i < s.height-contentLines; i++ {
			parts = append(parts, "")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderFolders builds the folder tree view with cursor highlighting.
func (s SidebarModel) renderFolders(styles themeStyles) []string {
	var items []string

	// Virtual root: "All Workflows"
	label := "All Workflows"
	if s.folderCursor == 0 {
		items = append(items, styles.Selected.Render("❯ "+label))
	} else {
		items = append(items, "  "+label)
	}

	for i, folder := range s.folders {
		depth := strings.Count(folder, "/")
		indent := strings.Repeat("  ", depth+1)

		// Display just the last segment for readability
		name := folder
		if idx := strings.LastIndex(folder, "/"); idx >= 0 {
			name = folder[idx+1:]
		}

		line := fmt.Sprintf("%s%s", indent, name)

		if i+1 == s.folderCursor { // +1 because index 0 is "All Workflows"
			items = append(items, styles.Selected.Render("❯ "+strings.TrimLeft(line, " ")))
		} else {
			items = append(items, line)
		}
	}

	return items
}

// renderTags builds the tag list view with cursor highlighting.
func (s SidebarModel) renderTags(styles themeStyles) []string {
	var items []string

	// Virtual root: "All Tags"
	label := "All Tags"
	if s.tagCursor == 0 {
		items = append(items, styles.Selected.Render("❯ "+label))
	} else {
		items = append(items, "  "+label)
	}

	for i, tag := range s.tags {
		if i+1 == s.tagCursor { // +1 because index 0 is "All Tags"
			items = append(items, styles.Selected.Render("❯ "+tag))
		} else {
			items = append(items, "  "+tag)
		}
	}

	return items
}

// UpdateData refreshes folder/tag data without losing cursor state.
// Clamps cursors if the new data is shorter.
func (s *SidebarModel) UpdateData(folders []string, tags []string) {
	sortedFolders := make([]string, len(folders))
	copy(sortedFolders, folders)
	sort.Strings(sortedFolders)

	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)

	s.folders = sortedFolders
	s.tags = sortedTags

	// Clamp cursors
	if s.folderCursor > len(s.folders) {
		s.folderCursor = len(s.folders)
	}
	if s.tagCursor > len(s.tags) {
		s.tagCursor = len(s.tags)
	}
}
