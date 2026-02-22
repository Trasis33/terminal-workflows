package manage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// settingsSection identifies which section of the settings view is active.
type settingsSection int

const (
	sectionPresets settingsSection = iota
	sectionColors
	sectionBorders
	sectionLayout
	sectionActions
)

// settingsField represents a single editable field in the settings view.
type settingsField struct {
	section settingsSection
	label   string
	key     string // internal key for identifying the field
}

// SettingsModel manages the theme customization view with live preview.
type SettingsModel struct {
	theme         Theme // the theme being edited (copy)
	originalTheme Theme // for cancel/revert
	presetNames   []string
	presetIndex   int // currently selected preset

	fields  []settingsField // all navigable fields
	cursor  int             // selected field index
	editing bool            // whether actively editing a value
	input   textinput.Model // for editing color/numeric values

	configDir string
	width     int
	height    int
	dirty     bool // whether changes have been made
}

// NewSettingsModel creates a settings view with a copy of the current theme.
func NewSettingsModel(theme Theme, configDir string) SettingsModel {
	ti := textinput.New()
	ti.CharLimit = 16

	presets := PresetNames()
	presetIdx := 0
	for i, n := range presets {
		if n == theme.Name {
			presetIdx = i
			break
		}
	}

	m := SettingsModel{
		theme:         theme,
		originalTheme: theme,
		presetNames:   presets,
		presetIndex:   presetIdx,
		input:         ti,
		configDir:     configDir,
	}
	m.fields = m.buildFields()
	return m
}

// buildFields constructs the ordered list of navigable settings fields.
func (m SettingsModel) buildFields() []settingsField {
	return []settingsField{
		{sectionPresets, "Preset", "preset"},
		{sectionColors, "Primary", "primary"},
		{sectionColors, "Secondary", "secondary"},
		{sectionColors, "Tertiary", "tertiary"},
		{sectionColors, "Text", "text"},
		{sectionColors, "Dim", "dim"},
		{sectionColors, "Border", "border"},
		{sectionBorders, "Style", "border_style"},
		{sectionLayout, "Sidebar Width", "sidebar_width"},
		{sectionLayout, "Show Preview", "show_preview"},
		{sectionActions, "Save", "save"},
		{sectionActions, "Cancel", "cancel"},
	}
}

// getFieldValue returns the current value for a settings field key.
func (m SettingsModel) getFieldValue(key string) string {
	switch key {
	case "preset":
		return m.presetNames[m.presetIndex]
	case "primary":
		return m.theme.Colors.Primary
	case "secondary":
		return m.theme.Colors.Secondary
	case "tertiary":
		return m.theme.Colors.Tertiary
	case "text":
		return m.theme.Colors.Text
	case "dim":
		return m.theme.Colors.Dim
	case "border":
		return m.theme.Colors.Border
	case "border_style":
		return m.theme.Borders.Style
	case "sidebar_width":
		return strconv.Itoa(m.theme.Layout.SidebarWidth)
	case "show_preview":
		if m.theme.Layout.ShowPreview {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// setFieldValue applies a new value for a settings field key.
func (m *SettingsModel) setFieldValue(key, value string) {
	m.dirty = true
	switch key {
	case "primary":
		m.theme.Colors.Primary = value
	case "secondary":
		m.theme.Colors.Secondary = value
	case "tertiary":
		m.theme.Colors.Tertiary = value
	case "text":
		m.theme.Colors.Text = value
	case "dim":
		m.theme.Colors.Dim = value
	case "border":
		m.theme.Colors.Border = value
	case "border_style":
		valid := map[string]bool{"rounded": true, "normal": true, "thick": true, "double": true, "hidden": true}
		if valid[value] {
			m.theme.Borders.Style = value
		}
	case "sidebar_width":
		if w, err := strconv.Atoi(value); err == nil && w > 10 && w < 60 {
			m.theme.Layout.SidebarWidth = w
		}
	case "show_preview":
		m.theme.Layout.ShowPreview = value == "true"
	}
	m.theme.computeStyles()
}

// Update processes messages for the settings view.
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		if m.editing {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.editing {
		return m.updateEditing(keyMsg)
	}

	return m.updateNavigating(keyMsg)
}

// updateNavigating handles keys when navigating the settings list.
func (m SettingsModel) updateNavigating(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	field := m.fields[m.cursor]

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(m.fields)-1 {
			m.cursor++
		}
		return m, nil

	case "left", "h":
		if field.key == "preset" {
			if m.presetIndex > 0 {
				m.presetIndex--
				m.applyPreset()
			}
			return m, nil
		}
		if field.key == "show_preview" {
			m.theme.Layout.ShowPreview = !m.theme.Layout.ShowPreview
			m.dirty = true
			m.theme.computeStyles()
			return m, nil
		}
		return m, nil

	case "right", "l":
		if field.key == "preset" {
			if m.presetIndex < len(m.presetNames)-1 {
				m.presetIndex++
				m.applyPreset()
			}
			return m, nil
		}
		if field.key == "show_preview" {
			m.theme.Layout.ShowPreview = !m.theme.Layout.ShowPreview
			m.dirty = true
			m.theme.computeStyles()
			return m, nil
		}
		return m, nil

	case "enter":
		switch field.key {
		case "save":
			return m, m.saveTheme()
		case "cancel":
			return m, func() tea.Msg { return switchToBrowseMsg{} }
		case "preset", "show_preview":
			// These use left/right, not enter-to-edit.
			return m, nil
		default:
			// Enter edit mode.
			m.editing = true
			m.input.SetValue(m.getFieldValue(field.key))
			m.input.Focus()
			return m, textinput.Blink
		}

	case "s":
		return m, m.saveTheme()

	case "esc":
		// Revert to original theme and go back.
		m.theme = m.originalTheme
		return m, func() tea.Msg { return switchToBrowseMsg{} }
	}

	return m, nil
}

// updateEditing handles keys while editing a field value.
func (m SettingsModel) updateEditing(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		field := m.fields[m.cursor]
		value := strings.TrimSpace(m.input.Value())
		if value != "" {
			m.setFieldValue(field.key, value)
		}
		m.editing = false
		m.input.Blur()
		return m, nil

	case "esc":
		m.editing = false
		m.input.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// applyPreset loads a preset theme and applies it live.
func (m *SettingsModel) applyPreset() {
	name := m.presetNames[m.presetIndex]
	preset, ok := PresetByName(name)
	if !ok {
		return
	}
	m.theme = preset
	m.dirty = true
}

// saveTheme persists the theme to disk and returns to browse.
func (m SettingsModel) saveTheme() tea.Cmd {
	return func() tea.Msg {
		if err := SaveTheme(m.configDir, m.theme); err != nil {
			return saveErrorMsg{err: fmt.Errorf("save theme: %w", err)}
		}
		return themeSavedMsg{}
	}
}

// View renders the settings view with all editable fields.
func (m SettingsModel) View() string {
	s := m.theme.Styles()

	title := s.FormTitle.Render("Theme Settings")

	var sections []string
	sections = append(sections, title, "")

	currentSection := settingsSection(-1)
	for i, field := range m.fields {
		// Section headers.
		if field.section != currentSection {
			currentSection = field.section
			switch currentSection {
			case sectionPresets:
				sections = append(sections, s.Highlight.Render("  Presets"))
			case sectionColors:
				sections = append(sections, "", s.Highlight.Render("  Colors"))
			case sectionBorders:
				sections = append(sections, "", s.Highlight.Render("  Borders"))
			case sectionLayout:
				sections = append(sections, "", s.Highlight.Render("  Layout"))
			case sectionActions:
				sections = append(sections, "")
			}
		}

		isSelected := i == m.cursor
		row := m.renderField(field, isSelected, s)
		sections = append(sections, row)
	}

	// Dirty indicator.
	if m.dirty {
		sections = append(sections, "", s.Tag.Render("  ● unsaved changes"))
	}

	// Hints.
	sections = append(sections, "",
		s.Hint.Render("  ↑↓ navigate  ←→ preset/toggle  enter edit  s save  esc back"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// renderField renders a single settings field row.
func (m SettingsModel) renderField(field settingsField, selected bool, s themeStyles) string {
	prefix := "    "
	if selected {
		prefix = s.Selected.Render("  ❯ ")
	}

	switch field.key {
	case "preset":
		// Render preset names inline with arrows.
		var parts []string
		for i, name := range m.presetNames {
			if i == m.presetIndex {
				parts = append(parts, s.Selected.Render(name))
			} else {
				parts = append(parts, s.Dim.Render(name))
			}
		}
		return prefix + "◂ " + strings.Join(parts, "  ") + " ▸"

	case "save":
		label := "[Save]"
		if selected {
			return prefix + s.Selected.Render(label)
		}
		return prefix + label

	case "cancel":
		label := "[Cancel]"
		if selected {
			return prefix + s.Selected.Render(label)
		}
		return prefix + label

	default:
		value := m.getFieldValue(field.key)
		labelStr := fmt.Sprintf("%-15s", field.label+":")

		if selected && m.editing {
			return prefix + labelStr + m.input.View()
		}

		// Color swatch for color fields.
		valueDisplay := value
		if field.section == sectionColors {
			swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(value)).Render("██")
			valueDisplay = fmt.Sprintf("[%s] %s", value, swatch)
		} else {
			valueDisplay = fmt.Sprintf("[%s]", value)
		}

		if selected {
			return prefix + s.Selected.Render(labelStr) + valueDisplay
		}
		return prefix + labelStr + valueDisplay
	}
}
