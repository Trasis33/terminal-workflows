package manage

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keybindings for the management TUI.
type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Back          key.Binding
	Create        key.Binding
	Edit          key.Binding
	Delete        key.Binding
	Move          key.Binding
	Search        key.Binding
	ToggleSidebar key.Binding
	Settings      key.Binding
	Quit          key.Binding
	Help          key.Binding
	FolderCreate  key.Binding
	FolderRename  key.Binding
	FolderDelete  key.Binding
}

// defaultKeyMap returns the default keybinding configuration.
func defaultKeyMap() keyMap {
	return keyMap{
		Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),

		Create: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Edit:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Move:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move")),

		Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		ToggleSidebar: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle sidebar")),
		Settings:      key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "theme")),

		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),

		FolderCreate: key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "new folder")),
		FolderRename: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename folder")),
		FolderDelete: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete folder")),
	}
}

// ShortHelp returns the keybindings shown in the short help view.
// Implements the bubbles/help.KeyMap interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Create, k.Delete, k.Help, k.Quit}
}

// FullHelp returns the full set of keybindings for the expanded help view.
// Implements the bubbles/help.KeyMap interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Create, k.Edit, k.Delete, k.Move},
		{k.Search, k.ToggleSidebar, k.Settings},
		{k.FolderCreate, k.FolderRename, k.FolderDelete},
		{k.Help, k.Quit},
	}
}
