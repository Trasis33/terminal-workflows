package manage

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

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
func Run(s store.Store) (string, error) {
	workflows, err := s.List()
	if err != nil {
		return "", err
	}

	cfgDir := config.ConfigDir()
	ttyOut, ttyOutErr := openTTY()
	if ttyOutErr == nil {
		defer ttyOut.Close()
		r := lipgloss.NewRenderer(ttyOut, termenv.WithProfile(termenv.TrueColor))
		// Pre-set dark background to prevent the renderer from sending an
		// OSC 11 query to the terminal. The terminal's response arrives
		// through stdin and gets captured by focused textinputs as garbage
		// characters (the ANSI escape leak bug).
		r.SetHasDarkBackground(true)
		lipgloss.SetDefaultRenderer(r)
	}
	ttyIn, ttyInErr := openTTYInput()
	if ttyInErr == nil {
		defer ttyIn.Close()
	}

	theme, err := LoadTheme(cfgDir)
	if err != nil {
		theme = DefaultTheme()
	}

	m := New(s, workflows, theme, cfgDir)
	programOptions := []tea.ProgramOption{tea.WithAltScreen()}
	if ttyOutErr == nil {
		programOptions = append(programOptions, tea.WithOutput(ttyOut))
	}
	if ttyInErr == nil {
		programOptions = append(programOptions, tea.WithInput(ttyIn))
	}
	p := tea.NewProgram(m, programOptions...)
	final, err := p.Run()
	if err != nil {
		return "", err
	}

	fm, ok := final.(Model)
	if !ok {
		return "", nil
	}
	return fm.result, nil
}
