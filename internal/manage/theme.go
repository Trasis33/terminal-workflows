package manage

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
)

// Theme defines the visual configuration for the management TUI.
// It can be loaded from / saved to a YAML file for user customization.
type Theme struct {
	Name    string       `yaml:"name"`
	Colors  themeColors  `yaml:"colors"`
	Borders themeBorders `yaml:"borders"`
	Layout  themeLayout  `yaml:"layout"`

	// Computed lipgloss styles — not serialized to YAML.
	styles themeStyles `yaml:"-"`
}

// themeColors holds color values as ANSI color strings.
type themeColors struct {
	Primary    string `yaml:"primary"`    // Main accent
	Secondary  string `yaml:"secondary"`  // Secondary accent
	Tertiary   string `yaml:"tertiary"`   // Tertiary accent
	Text       string `yaml:"text"`       // Normal text
	Dim        string `yaml:"dim"`        // Dimmed/muted text
	Border     string `yaml:"border"`     // Border color
	Background string `yaml:"background"` // Background hint (empty = terminal default)
}

// themeBorders configures border appearance.
type themeBorders struct {
	Style string `yaml:"style"` // "rounded", "normal", "thick", "double", "hidden"
}

// themeLayout configures structural dimensions.
type themeLayout struct {
	SidebarWidth int  `yaml:"sidebar_width"` // Width of sidebar pane
	ShowPreview  bool `yaml:"show_preview"`  // Whether to show command preview
}

// Styles returns the computed lipgloss styles for this theme.
// Call computeStyles() first if the theme was deserialized from YAML.
func (t Theme) Styles() themeStyles {
	return t.styles
}

// computeStyles converts the theme's color/border configuration into
// lipgloss.Style values ready for rendering.
func (t *Theme) computeStyles() {
	border := borderFromString(t.Borders.Style)

	t.styles.Sidebar = lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(t.Colors.Border)).
		Padding(0, 1)

	t.styles.List = lipgloss.NewStyle().
		Padding(0, 1)

	t.styles.Preview = lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(t.Colors.Border)).
		Padding(0, 1)

	t.styles.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Secondary)).
		Bold(true)

	t.styles.Highlight = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Primary)).
		Bold(true)

	t.styles.Tag = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Tertiary))

	t.styles.Dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Dim))

	t.styles.Hint = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Dim))

	t.styles.DialogBox = lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(t.Colors.Primary)).
		Padding(1, 2).
		Width(50)

	t.styles.DialogTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Primary)).
		Bold(true)

	t.styles.FormTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Colors.Primary)).
		Bold(true).
		Padding(0, 0, 1, 0)

	t.styles.ActiveBorder = lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(t.Colors.Primary))

	t.styles.InactiveBorder = lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(t.Colors.Border))
}

// borderFromString maps a border style name to a lipgloss.Border value.
func borderFromString(s string) lipgloss.Border {
	switch s {
	case "normal":
		return lipgloss.NormalBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	default: // "rounded" or unrecognized
		return lipgloss.RoundedBorder()
	}
}

// DefaultTheme returns the default theme using the mint/cyan-green palette
// that matches the picker's color scheme.
func DefaultTheme() Theme {
	t := Theme{
		Name: "default",
		Colors: themeColors{
			Primary:   "49",  // bright cyan-green
			Secondary: "158", // light mint green
			Tertiary:  "73",  // muted teal
			Text:      "250",
			Dim:       "242",
			Border:    "238",
		},
		Borders: themeBorders{
			Style: "rounded",
		},
		Layout: themeLayout{
			SidebarWidth: 24,
			ShowPreview:  true,
		},
	}
	t.computeStyles()
	return t
}

// LoadTheme reads a theme from <configDir>/theme.yaml.
// Returns an error if the file does not exist or cannot be parsed.
func LoadTheme(configDir string) (Theme, error) {
	path := filepath.Join(configDir, "theme.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return Theme{}, err
	}
	t.computeStyles()
	return t, nil
}

// SaveTheme writes a theme to <configDir>/theme.yaml.
func SaveTheme(configDir string, t Theme) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "theme.yaml"), data, 0644)
}

// --- Preset Themes ---

// PresetDark returns a dark theme with cool blue-grey tones.
func PresetDark() Theme {
	t := Theme{
		Name: "dark",
		Colors: themeColors{
			Primary:   "75",  // cornflower blue
			Secondary: "110", // steel blue
			Tertiary:  "67",  // slate blue
			Text:      "252",
			Dim:       "245",
			Border:    "240",
		},
		Borders: themeBorders{Style: "rounded"},
		Layout:  themeLayout{SidebarWidth: 24, ShowPreview: true},
	}
	t.computeStyles()
	return t
}

// PresetLight returns a light-friendly theme with warm tones.
func PresetLight() Theme {
	t := Theme{
		Name: "light",
		Colors: themeColors{
			Primary:   "25", // dark blue
			Secondary: "30", // dark cyan
			Tertiary:  "65", // dark slate
			Text:      "234",
			Dim:       "245",
			Border:    "250",
		},
		Borders: themeBorders{Style: "rounded"},
		Layout:  themeLayout{SidebarWidth: 24, ShowPreview: true},
	}
	t.computeStyles()
	return t
}

// PresetDracula returns a Dracula-inspired theme.
func PresetDracula() Theme {
	t := Theme{
		Name: "dracula",
		Colors: themeColors{
			Primary:   "141", // purple
			Secondary: "84",  // green
			Tertiary:  "212", // pink
			Text:      "253",
			Dim:       "103", // comment grey
			Border:    "61",  // muted purple
		},
		Borders: themeBorders{Style: "rounded"},
		Layout:  themeLayout{SidebarWidth: 24, ShowPreview: true},
	}
	t.computeStyles()
	return t
}

// PresetNord returns a Nord-inspired theme with arctic, north-bluish colors.
func PresetNord() Theme {
	t := Theme{
		Name: "nord",
		Colors: themeColors{
			Primary:   "110", // nord blue
			Secondary: "108", // nord green
			Tertiary:  "139", // nord purple
			Text:      "254",
			Dim:       "244",
			Border:    "60", // nord polar night
		},
		Borders: themeBorders{Style: "rounded"},
		Layout:  themeLayout{SidebarWidth: 24, ShowPreview: true},
	}
	t.computeStyles()
	return t
}

// presetRegistry maps preset names to their constructor functions.
var presetRegistry = map[string]func() Theme{
	"default": DefaultTheme,
	"dark":    PresetDark,
	"light":   PresetLight,
	"dracula": PresetDracula,
	"nord":    PresetNord,
}

// PresetNames returns the list of available preset theme names.
func PresetNames() []string {
	return []string{"default", "dark", "light", "dracula", "nord"}
}

// PresetByName returns a preset theme by name.
// Returns the theme and true if found, or a zero Theme and false if not.
func PresetByName(name string) (Theme, bool) {
	fn, ok := presetRegistry[name]
	if !ok {
		return Theme{}, false
	}
	return fn(), true
}
