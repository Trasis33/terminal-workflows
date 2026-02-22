# Phase 3: Management TUI - Research

**Researched:** 2026-02-22
**Domain:** Full-screen terminal UI for workflow CRUD, browsing, folder management, and theming
**Confidence:** HIGH

## Summary

The management TUI is a full-screen Bubble Tea application launched via `wf manage`. It provides a two-pane browse view (sidebar + workflow list + preview), full-screen form editing, folder/tag organization, and a customizable theme system. The existing codebase already demonstrates the core architectural pattern (viewState enum with child component routing) in `internal/picker/model.go`.

The standard approach uses the Charm ecosystem already in `go.mod`: Bubble Tea v1.3.10 for the application framework, Bubbles v1.0.0 for reusable components (list, textinput, textarea, viewport), lipgloss v1.1.0 for layout composition and styling, and the new addition of `charmbracelet/huh` for form handling. The huh library implements `tea.Model` and integrates directly into Bubble Tea apps, providing accessible forms with built-in validation and theming — avoiding the need to hand-roll complex form logic.

Key architectural insight: the management TUI is a multi-view application with 4-5 distinct views (browse, create, edit, settings, plus overlay dialogs). Each view is its own model with `Update`/`View` methods. The root model holds a `viewState` enum and routes messages to the active view. Layout composition uses `lipgloss.JoinHorizontal()` and `lipgloss.JoinVertical()` to assemble panes. Overlay dialogs render a dimmed background with `lipgloss.Place()` centering a dialog box.

**Primary recommendation:** Extend the existing picker's viewState pattern into a multi-view router with 4 child models (browse, form, settings, dialog), using huh for forms and the existing Store interface for all persistence.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| charmbracelet/bubbletea | v1.3.10 | TUI application framework (Elm architecture) | Already in go.mod. The Go standard for terminal apps. |
| charmbracelet/bubbles | v1.0.0 | Reusable TUI components (list, textinput, textarea, viewport) | Already in go.mod. Official companion to Bubble Tea. |
| charmbracelet/lipgloss | v1.1.0 | Terminal styling and layout composition | Already in go.mod. Handles colors, borders, pane layout. |
| charmbracelet/huh | v0.8.x | Form library (inputs, selects, confirms) | Official Charm form library. Implements tea.Model for embedding. Has built-in themes and validation. |
| sahilm/fuzzy | v0.1.1 | Fuzzy string matching | Already in go.mod. Used by picker. Reuse for management search. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| goccy/go-yaml | v1.19.2 | YAML parsing/writing | Already in go.mod. For theme config files. |
| adrg/xdg | v0.5.3 | XDG base directory resolution | Already in go.mod. Theme config path: `~/.config/wf/theme.yaml`. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| huh (forms) | Raw bubbles textinput/textarea | huh gives validation, tab navigation, accessibility for free. Raw bubbles = more control but significantly more code. Use huh. |
| sahilm/fuzzy | bubbles/list built-in filtering | bubbles/list has its own filter. But picker already uses sahilm/fuzzy with custom rendering — reuse that pattern for consistency. |

**Installation:**
```bash
go get github.com/charmbracelet/huh@latest
```

All other dependencies are already in `go.mod`.

## Architecture Patterns

### Recommended Package Structure
```
internal/manage/
├── model.go       # Root model, viewState enum, message routing, Init/Update/View
├── browse.go      # Browse view: sidebar + workflow list + preview pane
├── form.go        # Create/edit form using huh, tag autocomplete
├── settings.go    # Theme settings view with live preview
├── dialog.go      # Overlay dialogs (delete confirm, folder create/rename)
├── sidebar.go     # Sidebar component: folder tree + tag list (switchable)
├── styles.go      # Default styles, color constants (extends picker palette)
├── theme.go       # Theme struct, YAML load/save, preset themes
├── keys.go        # Keybinding definitions (key.Binding from bubbles/key)
└── manage.go      # Public API: New(), Run() — entry point for cobra command
```

### Pattern 1: Multi-View Router (Root Model)
**What:** Root model holds a `viewState` enum and child models. `Update()` routes messages to the active child. `View()` delegates rendering to the active child, then optionally overlays dialogs.
**When to use:** Always — this is the top-level architecture.
**Example:**
```go
// Source: Derived from existing internal/picker/model.go pattern
type viewState int

const (
    viewBrowse viewState = iota
    viewCreate
    viewEdit
    viewSettings
)

type Model struct {
    state     viewState
    prevState viewState // for returning from overlays

    // Child models
    browse   BrowseModel
    form     FormModel
    settings SettingsModel

    // Overlay (nil when not active)
    dialog *DialogModel

    // Shared state
    store     store.Store
    workflows []store.Workflow
    theme     Theme
    width     int
    height    int
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        // Propagate to all children
        // ...
    case tea.KeyMsg:
        // Dialog gets priority if active
        if m.dialog != nil {
            return m.updateDialog(msg)
        }
    }

    switch m.state {
    case viewBrowse:
        return m.updateBrowse(msg)
    case viewCreate, viewEdit:
        return m.updateForm(msg)
    case viewSettings:
        return m.updateSettings(msg)
    }
    return m, nil
}

func (m Model) View() string {
    var base string
    switch m.state {
    case viewBrowse:
        base = m.browse.View(m.theme, m.width, m.height)
    case viewCreate, viewEdit:
        base = m.form.View(m.theme, m.width, m.height)
    case viewSettings:
        base = m.settings.View(m.theme, m.width, m.height)
    }

    // Overlay dialog on top if active
    if m.dialog != nil {
        return m.renderOverlay(base, m.dialog.View(m.theme))
    }
    return base
}
```

### Pattern 2: Two-Pane Browse Layout with lipgloss
**What:** Sidebar (folder/tag tree) on the left, workflow list on the right, preview pane at the bottom. Composed using `lipgloss.JoinHorizontal()` and `lipgloss.JoinVertical()`.
**When to use:** The browse view.
**Example:**
```go
// Source: lipgloss layout utilities documentation
func (b BrowseModel) View(theme Theme, width, height int) string {
    sidebarWidth := 24
    listWidth := width - sidebarWidth - 3 // 3 for borders

    // Build sidebar
    sidebar := theme.SidebarStyle.
        Width(sidebarWidth).
        Height(height - 6). // reserve for preview + hints
        Render(b.sidebar.View())

    // Build workflow list
    list := theme.ListStyle.
        Width(listWidth).
        Height(height - 6).
        Render(b.list.View())

    // Join horizontally
    topPane := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, list)

    // Preview pane at bottom
    preview := theme.PreviewStyle.
        Width(width - 4).
        Render(b.preview.View())

    // Hints bar
    hints := theme.HintStyle.Render(b.hintsBar())

    return lipgloss.JoinVertical(lipgloss.Left, topPane, preview, hints)
}
```

### Pattern 3: Overlay Dialog with lipgloss.Place()
**What:** Render a dialog box centered over a dimmed background using `lipgloss.Place()`.
**When to use:** Delete confirmation, folder create/rename dialogs.
**Example:**
```go
// Source: lipgloss Place() documentation
func (m Model) renderOverlay(background, dialog string) string {
    // Dim the background by re-rendering it with a dim style
    // (or simply use the already-rendered background string)
    dimmed := lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Render(background)

    // Center the dialog over the full terminal
    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        dialog,
        lipgloss.WithWhitespaceChars("░"),
        lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
    )
}
```

### Pattern 4: huh Form Embedding
**What:** Use huh forms as child models within the management TUI for create/edit views.
**When to use:** Create and edit workflow forms.
**Example:**
```go
// Source: charmbracelet/huh README
import "github.com/charmbracelet/huh"

func newWorkflowForm(wf *store.Workflow, existingTags []string) *huh.Form {
    return huh.NewForm(
        huh.NewGroup(
            huh.NewInput().
                Title("Name").
                Value(&wf.Name).
                Validate(func(s string) error {
                    if s == "" {
                        return fmt.Errorf("name is required")
                    }
                    return nil
                }),
            huh.NewInput().
                Title("Description").
                Value(&wf.Description),
            huh.NewText().
                Title("Command").
                Value(&wf.Command).
                Lines(6),
            huh.NewMultiSelect[string]().
                Title("Tags").
                Options(huh.NewOptions(existingTags...)...).
                Value(&wf.Tags),
            huh.NewInput().
                Title("Folder").
                Value(&folderPath).
                Placeholder("e.g., infra/deploy"),
        ),
    )
}
```

Note: huh forms implement `tea.Model`, so they can be embedded in the root model's Update/View cycle. Call `form.Update(msg)` and `form.View()` from the parent.

### Pattern 5: Custom Message Types for View Transitions
**What:** Use custom tea.Msg types to signal view transitions, keeping child models decoupled from the root.
**When to use:** All view transitions (browse→edit, edit→browse, etc.).
**Example:**
```go
// Custom messages for view transitions
type switchToEditMsg struct{ workflow store.Workflow }
type switchToBrowseMsg struct{}
type showDeleteDialogMsg struct{ workflow store.Workflow }
type deleteConfirmedMsg struct{ workflow store.Workflow }
type refreshWorkflowsMsg struct{}

// In root Update:
case switchToEditMsg:
    m.form = newFormModel(msg.workflow, m.allTags())
    m.state = viewEdit
    return m, m.form.Init()
```

### Anti-Patterns to Avoid
- **God model:** Don't put all view logic in one giant model.go. Split into browse.go, form.go, etc.
- **Direct state mutation across views:** Use custom messages for view transitions, not direct field writes.
- **Ignoring WindowSizeMsg:** Every child that renders layout must receive and handle WindowSizeMsg. Propagate from root.
- **Hardcoded dimensions:** Always compute widths/heights relative to terminal size. Never use absolute numbers.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Form inputs with validation | Custom textinput + validation loop | `charmbracelet/huh` | Tab navigation, accessibility, validation, theming all built-in. Saves 500+ lines. |
| Multi-line text editing | Custom line buffer | `bubbles/textarea` | Handles word wrap, cursor movement, scrolling, undo. Extremely complex to replicate. |
| Scrollable content | Custom scroll offset tracking | `bubbles/viewport` | Handles page up/down, mouse scroll, content height calculation. |
| Fuzzy search | Custom substring matching | `sahilm/fuzzy` (already used) | Handles scoring, ranking, matched index tracking. |
| Keybinding help | Manual help string formatting | `bubbles/help` + `bubbles/key` | Renders keybinding help in short/full mode. Handles terminal width truncation. |
| Directory creation on save | Manual os.MkdirAll chains | `store.Save()` | YAMLStore already creates parent dirs. Just set the workflow name with folder prefix. |
| Slug generation | Manual string sanitization | `workflow.Filename()` | Already handles lowercasing, special chars, deduplication. |

**Key insight:** The Charm ecosystem (Bubble Tea + Bubbles + lipgloss + huh) provides composable building blocks. The management TUI is a composition of these blocks with routing logic, not a custom terminal rendering engine.

## Common Pitfalls

### Pitfall 1: Losing Focus State During View Transitions
**What goes wrong:** Switching from browse to edit and back causes text inputs to lose focus, cursor disappears, or multiple inputs have focus simultaneously.
**Why it happens:** Bubble Tea focus is managed manually. Each textinput/textarea has `.Focus()` and `.Blur()` methods that must be called explicitly.
**How to avoid:** When transitioning views, blur all inputs in the old view, focus the first input in the new view. Create helper methods: `focusFirstInput()`, `blurAll()`. Call them in the transition message handler.
**Warning signs:** Cursor blinking in the wrong input, typing not appearing, multiple cursors visible.

### Pitfall 2: WindowSizeMsg Not Propagated
**What goes wrong:** Child components render with wrong dimensions after terminal resize. Layout breaks, text overflows, panes overlap.
**Why it happens:** Only the root model receives `tea.WindowSizeMsg`. Children must receive it explicitly.
**How to avoid:** In root `Update()`, always handle `WindowSizeMsg` first, update `m.width`/`m.height`, then propagate to ALL child models (not just the active one — inactive views need correct dimensions when they become active).
**Warning signs:** Layout looks fine on startup but breaks after resize. Works in one terminal size but not another.

### Pitfall 3: Blocking Operations in Update()
**What goes wrong:** File I/O in `Update()` (saving workflows, reading theme files) blocks the UI, causing visible lag or freezing.
**Why it happens:** Bubble Tea runs on a single goroutine. `Update()` must return quickly.
**How to avoid:** Use `tea.Cmd` for any I/O operation. Return a command that performs the I/O, then handle the result message.
**Warning signs:** UI freezes briefly when saving/loading, sluggish response after operations.

```go
// Bad: blocking in Update
case saveMsg:
    m.store.Save(&wf) // blocks UI
    return m, nil

// Good: non-blocking with Cmd
case saveMsg:
    return m, func() tea.Msg {
        if err := m.store.Save(&wf); err != nil {
            return saveErrorMsg{err}
        }
        return saveSuccessMsg{}
    }
```

### Pitfall 4: Inconsistent Styling Across Views
**What goes wrong:** Different views use slightly different colors, borders, or spacing. The app feels stitched together rather than cohesive.
**Why it happens:** Developers define styles inline or per-view instead of using a centralized theme.
**How to avoid:** Define ALL styles in the Theme struct. Every view receives the Theme and uses only theme-provided styles. Never use `lipgloss.NewStyle()` directly in view rendering code — only in `theme.go` or `styles.go`.
**Warning signs:** Subtle color differences between views, different border styles on similar elements, inconsistent padding.

### Pitfall 5: Sidebar State Lost on Toggle
**What goes wrong:** Switching between folder view and tag view in the sidebar loses scroll position, selection state, or expansion state.
**Why it happens:** Recreating the sidebar model on toggle instead of maintaining both models.
**How to avoid:** Keep both folder and tag sidebar models alive simultaneously. The sidebar toggle switches which one's `View()` is rendered and which receives `Update()` messages, but both persist.
**Warning signs:** Scroll position resets when toggling views, previously expanded folders collapse.

### Pitfall 6: Move/Rename Without Store Coordination
**What goes wrong:** Moving a workflow to a new folder in the UI but the store doesn't handle the old file deletion, leaving orphaned files.
**Why it happens:** The Store interface has `Save()` and `Delete()` but no `Move()`. Moving requires explicit Delete-then-Save.
**How to avoid:** Implement move as: (1) save workflow with new name/path, (2) delete old name/path, (3) refresh workflow list. Wrap in a single method to ensure atomicity.
**Warning signs:** Duplicate workflows appearing, old files remaining after move.

## Code Examples

### Initializing the Management TUI from Cobra
```go
// Source: Existing pattern from internal/picker + cobra command
// File: cmd/manage.go (or added to existing root command)

func runManage(cmd *cobra.Command, args []string) error {
    s := store.NewYAMLStore(config.WorkflowsDir())
    workflows, err := s.List()
    if err != nil {
        return fmt.Errorf("loading workflows: %w", err)
    }

    theme, err := manage.LoadTheme(config.ConfigDir())
    if err != nil {
        theme = manage.DefaultTheme()
    }

    m := manage.New(s, workflows, theme)
    p := tea.NewProgram(m, tea.WithAltScreen()) // fullscreen
    _, err = p.Run()
    return err
}
```

### Theme Struct with YAML Config
```go
// Source: Derived from huh theme pattern + project YAML convention
// File: internal/manage/theme.go

type Theme struct {
    Name   string `yaml:"name"`
    Colors struct {
        Primary    string `yaml:"primary"`    // Main accent (default: "49")
        Secondary  string `yaml:"secondary"`  // Secondary accent (default: "158")
        Tertiary   string `yaml:"tertiary"`   // Tertiary accent (default: "73")
        Text       string `yaml:"text"`       // Normal text (default: "250")
        Dim        string `yaml:"dim"`        // Dimmed text (default: "242")
        Border     string `yaml:"border"`     // Border color (default: "238")
        Background string `yaml:"background"` // Background hint (default: "")
    } `yaml:"colors"`
    Borders struct {
        Style string `yaml:"style"` // "rounded", "normal", "thick", "double", "hidden"
    } `yaml:"borders"`
    Layout struct {
        SidebarWidth int  `yaml:"sidebar_width"` // default: 24
        ShowPreview  bool `yaml:"show_preview"`   // default: true
    } `yaml:"layout"`

    // Computed lipgloss styles (not serialized)
    styles themeStyles `yaml:"-"`
}

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
}

func DefaultTheme() Theme {
    t := Theme{Name: "default"}
    t.Colors.Primary = "49"    // cyan-green (matches picker)
    t.Colors.Secondary = "158" // mint (matches picker)
    t.Colors.Tertiary = "73"   // teal (matches picker)
    t.Colors.Text = "250"
    t.Colors.Dim = "242"
    t.Colors.Border = "238"
    t.Borders.Style = "rounded"
    t.Layout.SidebarWidth = 24
    t.Layout.ShowPreview = true
    t.computeStyles()
    return t
}

func (t *Theme) computeStyles() {
    border := lipgloss.RoundedBorder()
    switch t.Borders.Style {
    case "normal":
        border = lipgloss.NormalBorder()
    case "thick":
        border = lipgloss.ThickBorder()
    case "double":
        border = lipgloss.DoubleBorder()
    case "hidden":
        border = lipgloss.HiddenBorder()
    }

    t.styles.Sidebar = lipgloss.NewStyle().
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
}

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
```

### Keybinding Pattern with bubbles/key
```go
// Source: charmbracelet/bubbles key documentation
// File: internal/manage/keys.go

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
    Up        key.Binding
    Down      key.Binding
    Enter     key.Binding
    Back      key.Binding
    Create    key.Binding
    Edit      key.Binding
    Delete    key.Binding
    Move      key.Binding
    Search    key.Binding
    ToggleSidebar key.Binding
    Settings  key.Binding
    Quit      key.Binding
    Help      key.Binding
}

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
        Search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
        ToggleSidebar: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "folders/tags")),
        Settings: key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "theme")),
        Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
        Help:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
    }
}
```

### Folder Operations via Store
```go
// Source: Existing internal/store/yaml.go capabilities
// Folders are implicit in the store — they're directory paths in workflow names.

// Create folder: just os.MkdirAll (folder exists when workflow is saved into it)
func createFolder(baseDir, folderPath string) error {
    return os.MkdirAll(filepath.Join(baseDir, folderPath), 0755)
}

// List folders: walk the workflows directory for subdirectories
func listFolders(baseDir string) ([]string, error) {
    var folders []string
    err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
        if err != nil || !d.IsDir() || path == baseDir {
            return err
        }
        rel, _ := filepath.Rel(baseDir, path)
        depth := len(strings.Split(rel, string(filepath.Separator)))
        if depth > 2 {
            return filepath.SkipDir
        }
        folders = append(folders, rel)
        return nil
    })
    return folders, err
}

// Rename folder: os.Rename + update all workflow names in memory
func renameFolder(baseDir, oldPath, newPath string) error {
    return os.Rename(
        filepath.Join(baseDir, oldPath),
        filepath.Join(baseDir, newPath),
    )
}

// Delete folder: only if empty (os.Remove fails on non-empty)
func deleteEmptyFolder(baseDir, folderPath string) error {
    return os.Remove(filepath.Join(baseDir, folderPath))
}

// Move workflow: delete old + save new
func moveWorkflow(s store.Store, wf store.Workflow, newFolder string) error {
    oldName := wf.Name
    // Update name to include new folder prefix
    parts := strings.Split(wf.Name, "/")
    baseName := parts[len(parts)-1]
    wf.Name = newFolder + "/" + baseName

    if err := s.Save(&wf); err != nil {
        return err
    }
    return s.Delete(oldName)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual form handling with raw textinput | `charmbracelet/huh` forms library | huh v0.6+ (2024) | Forms get validation, tab navigation, theming for free. 5x less code. |
| lipgloss v0.x style API | lipgloss v1.0+ with `Width()`/`Height()` as methods | lipgloss v1.0 (2024) | Style methods are now chainable. `Copy()` no longer needed (styles are value types). |
| Manual help rendering | `bubbles/help` component | bubbles v0.16+ | Built-in short/full help display with keybinding integration. |
| Separate global styles | Theme struct pattern | Community standard (2024+) | Centralized theme enables runtime customization. |

**Deprecated/outdated:**
- `lipgloss.Copy()`: No longer needed. Styles are value types in v1.0+. Just reassign.
- `tea.WithInput()`/`tea.WithOutput()`: Replaced by `tea.WithInput()` on the `tea.Program` directly.
- `bubbles/list.DefaultStyles()`: Still works but should be overridden entirely for custom themes.

## Open Questions

Things that couldn't be fully resolved:

1. **huh form embedding vs standalone**
   - What we know: huh implements `tea.Model` and can run standalone or be embedded. The README shows both patterns.
   - What's unclear: Whether huh's internal key handling conflicts with the parent model when embedded (e.g., does huh consume `esc` before the parent can handle it?).
   - Recommendation: Test early. If key conflicts arise, use huh in standalone mode with `tea.WithAltScreen()` as a sub-program, or intercept keys before passing to huh.

2. **Tag autocomplete UX**
   - What we know: huh has `MultiSelect` for choosing from existing tags. User also wants to type new tags.
   - What's unclear: Whether huh supports a combo of MultiSelect + freeform input natively.
   - Recommendation: Implement tag input as a custom component: textinput with autocomplete dropdown (using fuzzy matching from sahilm/fuzzy). This is one case where hand-rolling is appropriate since huh doesn't cover this exact UX.

3. **Preset themes source**
   - What we know: CONTEXT.md says "preset themes available as starting points." huh has Charm, Dracula, Catppuccin, Base16 themes.
   - What's unclear: Whether to derive management TUI presets from huh's themes or create independent ones.
   - Recommendation: Create independent preset themes (Default/mint, Dark, Light, Dracula, Nord) since the management TUI theme struct has different fields than huh's theme. Embed as Go constants, not external files.

4. **Empty folder persistence**
   - What we know: CONTEXT.md says "empty folders remain visible." But YAMLStore only discovers folders by walking for .yaml files.
   - What's unclear: How to persist empty folders — they exist on disk but would be invisible to `List()`.
   - Recommendation: Add a `ListFolders()` method to the Store interface (or a helper function that walks for directories regardless of file content). Alternatively, create a `.keep` file in empty folders.

## Sources

### Primary (HIGH confidence)
- **Existing codebase** (`internal/picker/model.go`, `internal/store/`, `internal/config/`) — Verified patterns, styles, store interface
- **charmbracelet/bubbletea** GitHub README — Application architecture, Elm pattern, `tea.Cmd` usage
- **charmbracelet/bubbles** GitHub README — Available components: list, textinput, textarea, viewport, help, key
- **charmbracelet/lipgloss** GitHub README — Layout utilities: `JoinHorizontal`, `JoinVertical`, `Place`, border types
- **charmbracelet/huh** GitHub README — Form library: fields, themes, validation, `tea.Model` embedding

### Secondary (MEDIUM confidence)
- **Community patterns** (WebSearch: "Bubble Tea multi-view architecture", "lipgloss multi-pane layout") — Multi-view router pattern, overlay approach with `Place()`
- **huh theme system** — Theme struct with hierarchical lipgloss.Style fields (confirmed via README, detailed struct from GitHub source)

### Tertiary (LOW confidence)
- **Tag autocomplete UX** — No verified source for huh MultiSelect + freeform combo. Recommendation based on component capabilities analysis.
- **Overlay dimming** — lipgloss.Place() whitespace options confirmed, but full overlay-over-rendered-content requires testing (Place replaces content rather than overlaying).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All core libraries already in go.mod or from same Charm ecosystem. Versions verified from go.mod.
- Architecture: HIGH — Multi-view pattern verified in existing picker code. Layout composition from lipgloss docs.
- Pitfalls: MEDIUM — Based on community experience and architectural reasoning. Most are well-known Bubble Tea patterns, but some (e.g., huh key conflicts) need validation.
- Theme system: MEDIUM — YAML format aligns with project conventions (go-yaml already in go.mod). Theme struct pattern from huh. Details (exact fields, preset count) are recommendations.
- Code examples: MEDIUM — Derived from verified APIs but not compiled/tested. Structure is sound.

**Research date:** 2026-02-22
**Valid until:** 2026-03-22 (30 days — stable ecosystem, no major releases expected)
