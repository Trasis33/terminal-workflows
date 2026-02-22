package manage

import "github.com/charmbracelet/lipgloss"

// themeStyles holds pre-computed lipgloss styles derived from a Theme's
// color and border configuration. These are used throughout the management
// TUI for consistent rendering.
type themeStyles struct {
	Sidebar     lipgloss.Style
	List        lipgloss.Style
	Preview     lipgloss.Style
	Selected    lipgloss.Style
	Highlight   lipgloss.Style
	Tag         lipgloss.Style
	Dim         lipgloss.Style
	Hint        lipgloss.Style
	DialogBox   lipgloss.Style
	DialogTitle lipgloss.Style
	FormTitle   lipgloss.Style

	// Border variants for focused/unfocused panes.
	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style
}
