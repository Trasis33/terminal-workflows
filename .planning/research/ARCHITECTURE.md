# Architecture Patterns

**Domain:** Go TUI Terminal Workflow Manager (hybrid: quick picker + full TUI)
**Researched:** 2026-02-19
**Confidence:** HIGH (verified with official Bubble Tea repo, multiple sources cross-referenced)

## Recommended Architecture

### Overview: Dual-Entry Elm Architecture with Shared Core

The `wf` application has two distinct user-facing modes that share the same underlying data and logic layers. This demands a clean separation between the **TUI layer** (Bubble Tea models) and the **core domain** (workflow storage, search, template engine, shell integration). The two modes are:

1. **Quick Picker** (`wf` or hotkey) — Inline, sub-100ms, fuzzy search + select + paste-to-prompt
2. **Full TUI** (`wf manage`) — Alt-screen, CRUD, tagging, AI-assisted creation

Both modes are separate `tea.Program` instances (not one app with two screens) because they have fundamentally different terminal requirements: quick picker runs inline (no alt-screen), full TUI takes over with `tea.WithAltScreen()`.

```
                    +---------+      +----------+
                    | wf      |      | wf manage|
                    | (quick) |      | (full)   |
                    +----+----+      +-----+----+
                         |                 |
                    +----v----+      +-----v----+
                    | Picker  |      | Manager  |
                    | Model   |      | Model    |
                    | (inline)|      | (altscr) |
                    +----+----+      +-----+----+
                         |                 |
                    +----v-----------------v----+
                    |     Core Domain Layer      |
                    |                            |
                    |  +--------+  +---------+   |
                    |  | Store  |  | Search  |   |
                    |  | (YAML) |  | (Fuzzy) |   |
                    |  +--------+  +---------+   |
                    |  +--------+  +---------+   |
                    |  |Template|  | Shell   |   |
                    |  | Engine |  | Bridge  |   |
                    |  +--------+  +---------+   |
                    +----------------------------+
```

### Component Boundaries

| Component | Responsibility | Communicates With | Package |
|-----------|---------------|-------------------|---------|
| **CLI Router** | Parses subcommands (`wf`, `wf manage`, `wf add`, etc.), dispatches to correct mode | Picker Model, Manager Model | `cmd/wf/` |
| **Picker Model** | Quick-picker TUI: fuzzy search, workflow selection, argument fill, outputs result | Store, Search, Template Engine | `internal/tui/picker/` |
| **Manager Model** | Full TUI: lists workflows, CRUD forms, tag management, AI creation UI | Store, Search, Template Engine, AI Service | `internal/tui/manager/` |
| **Store** | YAML file CRUD operations, watches `~/.config/wf/` for changes, loads/saves workflow definitions | Filesystem | `internal/store/` |
| **Search** | Fuzzy matching across workflow name, description, tags, command content. Returns ranked results | Store (reads workflow index) | `internal/search/` |
| **Template Engine** | Parses `{{arg}}` placeholders in commands, extracts parameter definitions, renders final command string with user-provided values | None (pure logic) | `internal/template/` |
| **Shell Bridge** | Generates shell plugin scripts (zsh/bash/fish), handles paste-to-prompt output mechanism | Template Engine (for final command) | `internal/shell/` |
| **AI Service** | Copilot SDK integration for AI-assisted workflow creation and natural language to command translation | External API (Copilot SDK) | `internal/ai/` |
| **Config** | Loads app configuration from `~/.config/wf/config.yaml`, handles XDG paths | Filesystem | `internal/config/` |
| **Styles** | Shared Lip Gloss styles for consistent theming across both TUI modes | None (constants) | `internal/ui/styles/` |
| **Components** | Shared Bubble Tea sub-models (argument form, workflow card, tag pills) used by both Picker and Manager | Styles | `internal/ui/components/` |

### Data Flow

#### Quick Picker Flow (Critical Path: sub-100ms to interactive)

```
User types hotkey (e.g., Ctrl+Space)
    |
    v
Shell plugin (zsh widget / bash bind / fish function)
  calls: wf --picker
    |
    v
CLI Router --> Picker Model.Init()
    |
    +--> Store.LoadIndex() [YAML -> in-memory, ~10-30ms]
    |
    v
Picker Model renders inline fuzzy search
    |
    v (user types)
Search.Fuzzy(query) --> ranked results
    |
    v (user selects)
Template Engine.ExtractParams(workflow.Command)
    |
    v (if params exist)
Argument fill form (inline prompts)
    |
    v
Template Engine.Render(command, args) --> final command string
    |
    v
Picker Model returns tea.Quit
    |
    v
CLI prints final command to stdout
    |
    v
Shell plugin captures stdout, sets LBUFFER/READLINE_LINE/commandline
    |
    v
Command appears on user's prompt line, ready to execute
```

#### Full TUI Flow

```
User runs: wf manage
    |
    v
CLI Router --> Manager Model.Init() with tea.WithAltScreen()
    |
    +--> Store.LoadAll() [full workflow data]
    |
    v
Manager Model renders full-screen UI:
  +-- Sidebar (folders/tags navigation)
  +-- Main panel (workflow list with search)
  +-- Detail panel (selected workflow info)
    |
    v (user action: create/edit/delete)
CRUD operations --> Store.Save/Delete/Update
    |
    v (if AI creation)
AI Service.Generate(prompt) --> new workflow draft
    |
    v
Store.Save(workflow)
```

#### Shell Integration Data Flow (Paste-to-Prompt)

This is the most architecturally critical piece. Based on how fzf and Atuin implement this pattern:

```
Shell Plugin Script (sourced in .zshrc / .bashrc / config.fish):
  |
  +-- Defines a shell function/widget
  |
  +-- On hotkey press:
  |     1. Captures current prompt state (LBUFFER in zsh, READLINE_LINE in bash)
  |     2. Runs: selected=$(wf --picker)
  |     3. Sets prompt buffer to $selected
  |     4. Resets prompt display
  |
  +-- Shell-specific mechanisms:
        zsh:  zle widget + LBUFFER assignment + zle reset-prompt
        bash: bind -x + READLINE_LINE/READLINE_POINT assignment
        fish: bind function + commandline -r
```

**Confidence: HIGH** - This is exactly how fzf, Atuin, and similar tools implement paste-to-prompt. Verified across multiple sources including Atuin's official documentation and fzf's shell integration scripts.

## Recommended Directory Structure

```
wf/
├── cmd/
│   └── wf/
│       └── main.go                 # Entry point, CLI routing (cobra or minimal)
│
├── internal/
│   ├── config/
│   │   └── config.go               # XDG paths, app config loading
│   │
│   ├── store/
│   │   ├── store.go                # Store interface
│   │   ├── yaml.go                 # YAML file implementation
│   │   ├── index.go                # In-memory search index
│   │   └── workflow.go             # Workflow domain model
│   │
│   ├── search/
│   │   └── fuzzy.go                # Fuzzy search implementation
│   │
│   ├── template/
│   │   ├── parser.go               # Extract {{params}} from commands
│   │   └── renderer.go             # Substitute params with values
│   │
│   ├── shell/
│   │   ├── bridge.go               # Shell integration orchestration
│   │   ├── zsh.go                  # Zsh plugin script generator
│   │   ├── bash.go                 # Bash plugin script generator
│   │   └── fish.go                 # Fish plugin script generator
│   │
│   ├── ai/
│   │   └── copilot.go              # Copilot SDK integration
│   │
│   ├── tui/
│   │   ├── picker/
│   │   │   ├── model.go            # Quick picker root model
│   │   │   ├── update.go           # Message handling
│   │   │   └── view.go             # Inline rendering
│   │   │
│   │   └── manager/
│   │       ├── model.go            # Full TUI root model
│   │       ├── update.go           # Message routing to child models
│   │       ├── view.go             # Screen composition
│   │       └── screens/            # Sub-screens (list, edit, create, etc.)
│   │           ├── list.go
│   │           ├── edit.go
│   │           └── create.go
│   │
│   └── ui/
│       ├── styles/
│       │   └── theme.go            # Lip Gloss style definitions
│       │
│       ├── keys/
│       │   └── keys.go             # Key binding definitions
│       │
│       └── components/
│           ├── searchbar.go        # Fuzzy search input component
│           ├── argform.go          # Parameter fill form
│           ├── workflowcard.go     # Workflow display card
│           └── tagpills.go         # Tag display/selection
│
├── shell/                          # Shell plugin scripts (installed by user)
│   ├── wf.zsh
│   ├── wf.bash
│   └── wf.fish
│
├── go.mod
├── go.sum
└── Makefile
```

## Patterns to Follow

### Pattern 1: Separate tea.Program Instances for Each Mode

**What:** Run quick picker and full TUI as completely separate `tea.Program` instances, not as different "screens" within one program.

**Why:** The quick picker runs inline (no alt-screen, renders within current terminal flow) and must start in <100ms. The full TUI takes over the entire terminal with alt-screen. These are fundamentally different terminal modes. Combining them in one program adds unnecessary complexity and startup overhead.

**When:** Always. This is the core architectural decision.

**Example:**
```go
// cmd/wf/main.go
func main() {
    if isPickerMode() {
        // Inline mode, no alt-screen, minimal init
        m := picker.New(store, search)
        p := tea.NewProgram(m)
        result, err := p.Run()
        // Print selected command to stdout for shell capture
        fmt.Print(result.(picker.Model).SelectedCommand())
    } else if isManageMode() {
        // Full-screen mode
        m := manager.New(store, search, aiService)
        p := tea.NewProgram(m, tea.WithAltScreen())
        p.Run()
    }
}
```

### Pattern 2: Tree of Models with Message Routing

**What:** The Manager model (full TUI) acts as a root model that embeds child models (list screen, edit screen, create screen). Messages flow down from parent to active child. Children communicate up via custom message types.

**When:** Full TUI mode with multiple screens/views.

**Example:**
```go
// internal/tui/manager/model.go
type Screen int

const (
    ScreenList Screen = iota
    ScreenEdit
    ScreenCreate
)

type Model struct {
    activeScreen Screen
    list         list.Model       // child model
    edit         edit.Model       // child model
    create       create.Model     // child model
    store        *store.Store
    width, height int
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Global key handling (quit, etc.)
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }

    // Route to active screen
    var cmd tea.Cmd
    switch m.activeScreen {
    case ScreenList:
        m.list, cmd = m.list.Update(msg)
    case ScreenEdit:
        m.edit, cmd = m.edit.Update(msg)
    case ScreenCreate:
        m.create, cmd = m.create.Update(msg)
    }
    return m, cmd
}
```

**Confidence: HIGH** - This is the officially recommended pattern from Bubble Tea documentation and examples (composable-views, views examples). Multiple community sources confirm.

### Pattern 3: Core Domain as Plain Go (No TUI Dependency)

**What:** The store, search, template, and shell packages have zero dependency on Bubble Tea. They are pure Go packages that the TUI models call into. This means:
- They can be unit tested without any TUI harness
- They can be reused by non-TUI commands (e.g., `wf run <name>` for scriptable execution)
- Startup cost is minimized (no TUI init needed for simple operations)

**When:** Always.

**Example:**
```go
// internal/store/store.go — no tea imports anywhere in this package
type Store struct {
    basePath string
    index    *Index
}

func (s *Store) LoadIndex() error { ... }
func (s *Store) Search(query string) []Workflow { ... }
func (s *Store) Get(id string) (*Workflow, error) { ... }
func (s *Store) Save(w *Workflow) error { ... }
func (s *Store) Delete(id string) error { ... }
```

### Pattern 4: Lazy Initialization for Sub-100ms Startup

**What:** For the quick picker, defer everything that isn't immediately needed. Load only the search index (not full workflow details). Parse YAML lazily. Don't initialize AI service at all.

**When:** Quick picker mode.

**Strategy:**
```
Startup budget: 100ms total
  - Go binary start:     ~5ms  (native compiled)
  - Parse CLI flags:     ~1ms
  - Load config:         ~3ms  (single small YAML file)
  - Load search index:   ~15-30ms (scan workflow files, extract name/tags/description)
  - Init Bubble Tea:     ~5ms
  - First render:        ~2ms
  - Total:               ~31-46ms  ✓ Well under 100ms
```

**Critical:** The search index should be a lightweight structure (names, tags, first line of description) loaded from a pre-built index file or quick YAML scan — NOT full workflow parsing.

**Confidence: MEDIUM** - Timing estimates are based on Go binary startup benchmarks and typical YAML parse times. Actual performance depends on number of workflows and disk speed. Needs validation with real benchmarks.

### Pattern 5: Custom Template Syntax (Not `text/template`)

**What:** Use a simple custom parser for `{{argument_name}}` placeholders rather than Go's `text/template`. The project description specifies `{{arguments}}` syntax with text + enum types — this is much simpler than Go's template language and avoids confusing users with Go template semantics.

**When:** Always for the template/parameter system.

**Rationale:**
- `text/template` uses `{{.FieldName}}` — the dot prefix is confusing for non-programmers
- `text/template` supports conditionals, loops, pipes — unnecessary complexity for command arguments
- A custom parser is ~50 lines of Go (regex or simple state machine)
- User-facing syntax: `{{name}}` for text, `{{env|dev|staging|prod}}` for enums

**Example:**
```go
// internal/template/parser.go
type Param struct {
    Name    string
    Type    ParamType // Text or Enum
    Options []string  // For enum type
}

func ExtractParams(command string) []Param {
    // Parse {{name}} and {{name|opt1|opt2|opt3}}
    re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
    // ...
}

func Render(command string, values map[string]string) string {
    // Replace {{name}} with values["name"]
    // ...
}
```

### Pattern 6: Shell Plugin as Generated Script

**What:** Ship shell plugin scripts as static files bundled with the binary, exposed via `wf init zsh|bash|fish`. Users add `eval "$(wf init zsh)"` to their `.zshrc`. The scripts define a widget/function that calls `wf --picker` and captures stdout.

**When:** Always for shell integration.

**Example (generated zsh script):**
```zsh
# Output of: wf init zsh
_wf_widget() {
    local selected
    selected=$(wf --picker --query="${LBUFFER}" 2>/dev/null)
    if [[ -n "$selected" ]]; then
        LBUFFER="${selected}"
    fi
    zle reset-prompt
}
zle -N _wf_widget
bindkey '^[w' _wf_widget  # Alt+W default, configurable
```

**Confidence: HIGH** - This is exactly how fzf, Atuin, zoxide, and other shell-integrated Go tools work. Well-established pattern.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Single tea.Program with Mode Switching

**What:** Running both quick picker and full TUI as different "views" within one `tea.Program` instance.

**Why bad:**
- Alt-screen toggling within a running program is fragile
- Quick picker startup includes full TUI initialization overhead
- Error in one mode can crash the other
- Testing becomes coupled

**Instead:** Two separate `tea.Program` instances dispatched by the CLI router.

### Anti-Pattern 2: Global Mutable State

**What:** Using package-level variables for workflow data, search state, or configuration.

**Why bad:**
- Makes testing impossible without careful setup/teardown
- Race conditions if async commands access global state
- Can't run multiple tests in parallel

**Instead:** Pass dependencies via constructors. Each model receives its `Store`, `Search`, etc. through `New()` functions.

### Anti-Pattern 3: Business Logic in Update()

**What:** Putting YAML parsing, file I/O, or search algorithms directly inside Bubble Tea `Update()` methods.

**Why bad:**
- `Update()` runs on every message — must be fast
- Mixing IO with UI logic makes both untestable
- Blocks the render loop during file operations

**Instead:** Use `tea.Cmd` for all side effects. Put business logic in the core domain packages (`store`, `search`). Return commands from `Update()` that call into core packages asynchronously.

```go
// Bad: IO in Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    workflows, _ := os.ReadDir("~/.config/wf/workflows/")  // BLOCKS!
    // ...
}

// Good: IO via tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, m.store.LoadAsync()  // Returns tea.Cmd
}

func (s *Store) LoadAsync() tea.Cmd {
    return func() tea.Msg {
        workflows, err := s.loadFromDisk()
        return WorkflowsLoadedMsg{workflows, err}
    }
}
```

### Anti-Pattern 4: Over-Nesting Models

**What:** Creating deeply nested model hierarchies (root -> screen -> panel -> section -> component -> sub-component) where every layer routes messages.

**Why bad:**
- Message routing boilerplate explodes
- Performance degrades with deep delegation chains
- Debugging becomes painful (which layer ate my message?)

**Instead:** Keep nesting to 2 levels max: Root Model -> Screen/Component Models. Use flat composition where possible. A screen model can directly embed `bubbles` components without wrapping them in custom routing layers.

### Anti-Pattern 5: Using `text/template` for User-Facing Command Parameters

**What:** Using Go's `text/template` engine for the `{{argument}}` syntax in workflow commands.

**Why bad:**
- Go templates use `{{.FieldName}}` — the dot confuses end users
- Go templates execute arbitrary logic (security risk if user can craft templates)
- Go templates have complex error messages that leak implementation details
- Overkill for simple string substitution

**Instead:** Custom 50-line parser for `{{name}}` and `{{name|option1|option2}}` syntax.

## Scalability Considerations

| Concern | 10 workflows | 100 workflows | 1,000+ workflows |
|---------|-------------|--------------|------------------|
| **Startup time** | <20ms (trivial) | <50ms (scan YAML headers) | Needs pre-built index file; lazy load on first query |
| **Search speed** | Instant (<1ms) | Instant (<5ms) | Still fast with `sahilm/fuzzy` (<20ms for 1K items) |
| **File organization** | Single flat directory | Subdirectories by tag/folder | Same, with optional index cache |
| **Memory** | <1MB | <5MB | <20MB (holding all command strings in memory) |
| **YAML storage** | One file per workflow | One file per workflow | Consider workflow bundles or single-file collections |

**Recommendation:** Start with one-file-per-workflow. The filesystem IS the database. Only add indexing/caching if benchmarks show startup exceeds 100ms threshold (unlikely under ~500 workflows).

## Build Order (Dependency Graph)

Components should be built in this order based on dependencies:

```
Phase 1: Foundation (no TUI, no Bubble Tea dependency)
  ├── internal/config/        # XDG paths, config loading
  ├── internal/store/         # Workflow model + YAML CRUD
  └── internal/template/      # {{param}} parser + renderer

Phase 2: Search
  └── internal/search/        # Fuzzy search over store data
      (depends on: store for Workflow model)

Phase 3: Quick Picker TUI
  ├── internal/ui/styles/     # Shared Lip Gloss styles
  ├── internal/ui/components/ # Search bar, arg form
  ├── internal/tui/picker/    # Quick picker model
  └── cmd/wf/ (picker mode)   # CLI entry point
      (depends on: store, search, template, styles, components)

Phase 4: Shell Integration
  └── internal/shell/         # zsh/bash/fish plugin scripts
      (depends on: picker being functional via stdout)

Phase 5: Full TUI
  └── internal/tui/manager/   # Full management interface
      (depends on: store, search, template, styles, components)

Phase 6: AI Features
  └── internal/ai/            # Copilot SDK integration
      (depends on: store for saving generated workflows)
```

**Rationale for ordering:**
- **Phase 1 first** because everything depends on the data model and storage
- **Phase 2 before Phase 3** because the picker needs fuzzy search
- **Phase 3 before Phase 4** because shell integration wraps the picker
- **Phase 3 before Phase 5** because the quick picker is the core value prop and simpler to build (validates the architecture)
- **Phase 6 last** because AI is a differentiator, not table stakes, and requires external API setup

## Key Architecture Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Two `tea.Program` instances | Yes | Inline vs alt-screen are incompatible terminal modes |
| Core domain separate from TUI | Yes | Testability, reusability, startup performance |
| Custom template parser | Yes, over `text/template` | Simpler UX, no security concerns, fits domain better |
| YAML file storage | One file per workflow | Simple, version-controllable, human-editable, filesystem is the DB |
| Fuzzy search library | `sahilm/fuzzy` | Embeddable (unlike fzf binary), designed for code/filename matching, lightweight |
| Shell integration | `eval "$(wf init zsh)"` pattern | Proven pattern (fzf, Atuin, zoxide), per-shell widgets |
| XDG compliance | `~/.config/wf/` via `adrg/xdg` | Cross-platform, follows conventions |
| CLI framework | Minimal (Cobra or even stdlib `flag`) | Simple command routing only; keep startup fast |

## Sources

- Charmbracelet Bubble Tea README and examples — https://github.com/charmbracelet/bubbletea (HIGH confidence)
- Bubble Tea v1.3.10 (latest release as of research date) — official repo (HIGH confidence)
- Charmbracelet Bubbles component library — https://github.com/charmbracelet/bubbles (HIGH confidence)
- Atuin shell integration documentation and zsh/bash/fish examples — https://atuin.sh (HIGH confidence)
- fzf shell integration scripts — https://github.com/junegunn/fzf (HIGH confidence)
- Go `text/template` official documentation — https://pkg.go.dev/text/template (HIGH confidence)
- XDG Base Directory Specification — https://specifications.freedesktop.org/basedir-spec/ (HIGH confidence)
- Go project layout conventions (cmd/internal/pkg) — multiple community sources (MEDIUM confidence, no official Go spec but widely adopted)
- `sahilm/fuzzy` library — https://github.com/sahilm/fuzzy (MEDIUM confidence, needs version verification)
- `adrg/xdg` library — https://github.com/adrg/xdg (MEDIUM confidence, needs version verification)
- Sub-100ms startup timing estimates — based on Go compilation characteristics and general benchmarks (MEDIUM confidence, needs validation)
