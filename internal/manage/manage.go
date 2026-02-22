package manage

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/config"
	"github.com/fredriklanga/wf/internal/store"
)

// New creates a new management TUI model.
func New(s store.Store, workflows []store.Workflow, theme Theme, configDir string) Model {
	keys := defaultKeyMap()
	folders := extractFolders(workflows)
	tags := extractTags(workflows)

	return Model{
		state:     viewBrowse,
		store:     s,
		workflows: workflows,
		theme:     theme,
		keys:      keys,
		configDir: configDir,
		browse:    NewBrowseModel(workflows, folders, tags, theme, keys),
	}
}

// Run launches the management TUI as a full-screen alt-screen program.
// This is the main entry point called by the cobra command.
func Run(s store.Store) error {
	workflows, err := s.List()
	if err != nil {
		return err
	}

	cfgDir := config.ConfigDir()

	theme, err := LoadTheme(cfgDir)
	if err != nil {
		theme = DefaultTheme()
	}

	m := New(s, workflows, theme, cfgDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
