package picker

import "github.com/charmbracelet/lipgloss"

// Styles for the picker TUI.
// Uses subtle greens/cyans against dark terminal background.
var (
	// normalStyle is the default text style for result rows.
	normalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	// selectedStyle highlights the currently selected result row.
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("158")). // light mint green
			Bold(true)

	// highlightStyle marks fuzzy-matched characters in result names.
	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("49")). // bright cyan-green
			Bold(true)

	// tagStyle renders tag labels in brackets.
	tagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("73")) // muted teal

	// dimStyle renders secondary text like descriptions.
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

	// defaultTextStyle renders pre-filled defaults in parameter inputs.
	defaultTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// previewBorderStyle wraps the command preview pane.
	previewBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(0, 1)

	// searchPromptStyle renders the search prompt marker.
	searchPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("49")). // cyan-green
				Bold(true)

	// paramLabelStyle renders parameter labels in the fill view.
	paramLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Bold(true)

	// paramActiveStyle highlights the currently focused parameter input.
	paramActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("49")).
				Bold(true)

	// cursorStyle renders the selection cursor marker.
	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("49")).
			Bold(true)

	// hintStyle renders footer hint text.
	hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
