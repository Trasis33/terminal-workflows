# Stack Research: v1.1 Polish & Power

**Project:** wf (terminal workflow manager)
**Researched:** 2026-02-27
**Confidence:** HIGH (core recommendations) / MEDIUM (Bubble Tea v2 timing)
**Scope:** Stack additions/changes for v1.1 features only. See v1.0 STACK.md in git history for full foundation stack.

---

## Critical Context: Bubble Tea v2 Just Shipped

Bubble Tea v2.0.0 was released **4 days ago** (Feb 23, 2026). This is a major breaking change with:
- New import paths: `charm.land/bubbletea/v2` (was `github.com/charmbracelet/bubbletea`)
- Key handling rewrite: `KeyPressMsg`/`KeyReleaseMsg` replace `KeyMsg`; `key.Code`/`key.Text` replace `key.Type`/`key.Runes`
- Declarative view API: `View() View` struct replaces `View() string`
- Paste handling: dedicated `PasteMsg`/`PasteStartMsg`/`PasteEndMsg` replace `KeyMsg` with paste flag
- Bubbles v2 and Lip Gloss v2 shipped alongside; huh v2 exists at `github.com/charmbracelet/huh/v2`

**Recommendation: DO NOT upgrade to v2 for v1.1.** The v2 ecosystem is 4 days old. Migration would touch every file in the 11,736 LOC codebase. v1 remains fully functional and supported. All v1.1 features are achievable on the current v1 stack. Plan v2 migration as a separate v2.0 effort after ecosystem stabilizes (3-6 months).

---

## New Dependencies Needed

### 1. `alecthomas/chroma/v2` — Syntax Highlighting

| Attribute | Value |
|-----------|-------|
| **Package** | `github.com/alecthomas/chroma/v2` |
| **Version** | **v2.23.1** (released Jan 23, 2026) |
| **Purpose** | Syntax highlighting for command preview in browse/picker views |
| **Confidence** | HIGH — verified via GitHub releases page |

**Why this library:**
- The standard Go syntax highlighter — used by Hugo, Goldmark, Glamour, and many others
- Built-in ANSI terminal formatters: `TTY` (8-color), `TTY256` (256-color), `TTY16m` (true-color)
- Bash/shell lexer included out of the box — exactly what workflow commands need
- Pure Go, no CGO dependencies
- Actively maintained with releases every 2-3 weeks

**Why NOT glamour (v0.10.0):**
- Glamour is a full Markdown renderer that wraps chroma internally
- We need to highlight individual command strings, not render markdown documents
- Glamour adds unnecessary weight (markdown parsing, table rendering, heading styling)
- Chroma is the right layer of abstraction for this use case

**Integration with existing stack:**

Chroma outputs raw ANSI escape sequences. Lip Gloss also outputs ANSI. They coexist cleanly:
1. Render chroma output first → produces ANSI-colored string
2. Wrap with Lip Gloss for layout (borders, padding, width) → Lip Gloss preserves embedded ANSI

```go
import (
    "bytes"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/styles"
)

func highlightCommand(command string) string {
    lexer := lexers.Get("bash")
    if lexer == nil {
        return command // fallback to plain text
    }
    style := styles.Get("monokai") // match to wf theme
    formatter := formatters.Get("tty256")
    
    iterator, err := lexer.Tokenise(nil, command)
    if err != nil {
        return command
    }
    var buf bytes.Buffer
    formatter.Format(&buf, style, iterator)
    return buf.String()
}
```

**Where it applies in the codebase:**
- `internal/manage/browse.go` → `renderPreview()` (line 443) — command in preview pane
- `internal/manage/browse.go` → `renderList()` (line 379) — optional inline command snippets
- `internal/picker/paramfill.go` → `viewParamFill()` (line 256) — live render preview
- `internal/picker/model.go` → picker preview pane

**Cost:** ~5MB added to binary (chroma lexer definitions are embedded). Acceptable for the UX improvement. Use `lexers.Get("bash")` directly instead of `lexers.Analyse()` to avoid loading all lexers.

**Installation:**
```bash
go get github.com/alecthomas/chroma/v2@v2.23.1
```

---

## Existing Stack Sufficient — No New Dependencies

### 2. Parameter CRUD in Manage TUI → Existing `huh` v0.8.0 + custom model

**No new library needed.** The existing stack has everything required.

**Current state (verified in go.mod):**
- `huh v0.8.0` — already a dependency, specifically added "ability for users to create and manage their own Field types"
- `FormModel` in `internal/manage/form.go` already uses huh's `NewInput`, `NewText`, `NewGroup`, and `SuggestionsFunc`
- The tree-of-models pattern in `internal/manage/model.go` already routes between `BrowseModel`, `FormModel`, and dialog sub-models

**Architecture for parameter CRUD:**

Parameters need a custom Bubble Tea model (not a huh form), because the primary UX is a navigable list of params with add/remove/reorder — not a linear form flow.

```
Add to state machine in model.go:
  stateBrowse → stateEdit → stateParamList → stateParamEdit
```

| Component | Implementation | Library |
|-----------|---------------|---------|
| Parameter list (browse/navigate/delete) | Custom `ParamListModel` following `BrowseModel` scrollable list pattern | `bubbles/key` for keybindings, `lipgloss` for rendering |
| Parameter edit form (name, type, default, enum options, description) | `huh.Form` with dynamic groups | `huh` with `WithHideFunc` for type-specific fields |
| Type switching (plain → enum → dynamic) | Rebuild huh form or use `WithHideFunc` on groups | `huh` |

**huh dynamic fields pattern for param type switching:**
```go
huh.NewGroup(
    huh.NewInput().Title("Enum Options (pipe-separated)").Value(&opts),
).WithHideFunc(func() bool {
    return *paramType != "enum"
}, paramType)

huh.NewGroup(
    huh.NewInput().Title("Shell Command").Value(&dynCmd),
).WithHideFunc(func() bool {
    return *paramType != "dynamic"
}, paramType)
```

**Why NOT a new form library:** huh already supports dynamic fields. Adding another form library creates inconsistent UX and API surface.

### 3. List Picker Variable Type → Extend existing `paramfill.go`

**No new library needed.** This extends the existing dynamic parameter pattern.

**Current state (verified in codebase):**
- `internal/picker/paramfill.go` already handles dynamic params: executes shell commands (line 99-132), displays scrollable option lists with cursor navigation (line 300-355), handles loading/error states
- `template.Param` already has `DynamicCmd` field for shell commands

**What changes:**
- Extend `template.Param` with `FieldSeparator` (string) and `FieldIndex` (int) attributes
- After shell command output loads, if `FieldSeparator` is set, split each line and display as columns
- Store only the selected field (by index) as the param value
- New YAML syntax: `{{name!command|separator|field_index}}`

All achievable within the existing `bubbles/textinput` + custom rendering in `paramfill.go`.

### 4. Execute Flow in Manage TUI → Refactor existing `paramfill`

**No new library needed.** This is an architectural refactoring.

**Current state:**
- `internal/picker/paramfill.go` contains the complete param fill state machine (350+ lines)
- Tightly coupled to `picker.Model` via direct field access (`m.params`, `m.paramInputs`, etc.)

**Plan:**
1. Extract param fill into `internal/paramfill/model.go` — a standalone `tea.Model`
2. Both `picker.Model` and `manage.Model` compose this model
3. Add `stateExecute` to manage's state machine
4. On enter from browse → transition to param fill → on complete → paste to prompt

### 5. Variable Defaults (Auto-save) → Existing `goccy/go-yaml`

**No new library needed.**

**Pattern:**
- Store last-used values in `~/.config/wf/defaults.yaml` using `adrg/xdg` for path resolution
- Key by workflow name + param name: `{"workflow/name": {"param1": "last_value"}}`
- Load in `initParamFill()`, save on successful execution
- Use existing `goccy/go-yaml` for read/write

### 6. Per-field AI Generate → Existing Copilot SDK

**No new library needed.**

`internal/ai/` package already handles Copilot SDK integration. `internal/manage/ai_actions.go` already has AI action patterns. Per-field generation is a prompt engineering change — craft a prompt for a single variable given workflow context, call existing `ai.Generator`.

### 7. Warp Terminal Ctrl+G Compatibility → `os.Getenv` + configurable keybindings

**No new library needed.**

**The problem (verified):**
- Warp Terminal reserves Ctrl+G for `editor_view:add_next_occurrence`
- Warp intercepts this keystroke before TUI apps receive it (confirmed via GitHub issues #500, #537)
- Users can reconfigure in Warp Settings → Keyboard Shortcuts, but we shouldn't require this

**Solution — terminal detection + fallback keybinding:**

```go
// Terminal detection via standard env var
func isWarpTerminal() bool {
    return os.Getenv("TERM_PROGRAM") == "WarpTerminal"
}
```

**Two-layer approach:**

1. **Immediate fix:** Detect Warp and use alternative keybinding (e.g., `ctrl+a g` sequence or `alt+g`)
2. **Proper fix:** Make keybindings user-configurable via settings YAML

The existing `internal/manage/keys.go` already defines `keyMap` with `key.NewBinding()` (verified — 74 lines, clean pattern). Adding configurable overrides means:
- Load keybinding overrides from config YAML
- Apply overrides in `defaultKeyMap()` before returning
- Settings UI (`internal/manage/settings.go`) exposes keybinding customization

**Why NOT `atomicgo/keyboard` or other keybinding libraries:**
- Bubble Tea already handles all terminal input via its own input layer
- `charmbracelet/bubbles/key` (already imported in `keys.go`) provides `key.NewBinding` with multi-key support
- Adding another input library would conflict with Bubble Tea's event loop
- The problem is Warp intercepting keystrokes — no Go library can solve terminal-level interception

### 8. Auto-display Folder Contents + Overscroll Fix → Existing stack

**No new library needed.** Pure view logic fixes.

- **Auto-display:** `sidebarFilterMsg` pattern already exists (sidebar.go line 20-24). Wire up auto-selection when navigating folders.
- **Overscroll:** `ensureCursorVisible()` (browse.go line 306-313) needs bounds checking refinement when filtered list shrinks.

---

## What NOT to Add (And Why)

| Library | Why NOT |
|---------|---------|
| **Bubble Tea v2** | Released 4 days ago (Feb 23, 2026). Massive breaking changes touch every file. Wait 3-6 months. |
| **charmbracelet/glamour** | Renders full Markdown documents. We need chroma directly for shell command highlighting — glamour wraps chroma with unnecessary markdown overhead. |
| **atomicgo/keyboard** | Conflicts with Bubble Tea's input loop. BT handles all terminal input already. |
| **charmbracelet/x/input** | Part of v2 ecosystem. Not compatible with v1 stack we're staying on. |
| **viper/koanf (config libs)** | Overkill for keybinding overrides. Existing `goccy/go-yaml` + `internal/config/config.go` handles YAML config already. |
| **bbolt/sqlite (persistence)** | For variable defaults, a simple YAML file is sufficient. No database needed at this scale. |
| **Any alternative TUI framework** | All features achievable within BT v1. No reason to split the architecture. |

---

## Version Pinning Summary

| Dependency | Current | Target for v1.1 | Action |
|------------|---------|-----------------|--------|
| `bubbletea` | v1.3.10 | v1.3.10 | **Keep** — do not upgrade to v2 |
| `bubbles` | v1.0.0 | v1.0.0 | **Keep** — do not upgrade to v2 |
| `huh` | v0.8.0 | v0.8.0 | **Keep** — has dynamic field support needed |
| `lipgloss` | v1.1.0 | v1.1.0 | **Keep** — do not upgrade to v2 |
| `cobra` | v1.10.2 | v1.10.2 | **Keep** — unchanged |
| `goccy/go-yaml` | v1.19.2 | v1.19.2 | **Keep** — unchanged |
| `sahilm/fuzzy` | v0.1.1 | v0.1.1 | **Keep** — unchanged |
| `adrg/xdg` | v0.5.3 | v0.5.3 | **Keep** — unchanged |
| **`chroma/v2`** | — | **v2.23.1** | **ADD** — only new dependency |

**Installation for v1.1:**
```bash
go get github.com/alecthomas/chroma/v2@v2.23.1
```

---

## Chroma Theme Mapping

Match chroma styles to the existing theme system in `internal/manage/theme.go`:

| wf Theme Tone | Recommended Chroma Style | Notes |
|---------------|-------------------------|-------|
| Dark terminal | `monokai` or `dracula` | High contrast on dark backgrounds |
| Light terminal | `github` or `autumn` | Readable on light backgrounds |
| Auto-detect | Use `lipgloss.HasDarkBackground()` | Already available in lipgloss v1 |

```go
func chromaStyleForTheme(theme Theme) string {
    if theme.Colors.Background == "" {
        // Auto-detect
        if lipgloss.HasDarkBackground() {
            return "monokai"
        }
        return "github"
    }
    // Map from theme config
    return theme.ChromaStyle // add to Theme struct
}
```

---

## Key Architectural Decisions for Roadmap

### Decision 1: Extract paramfill before duplicating

The execute flow in manage TUI requires the same param fill logic that exists in picker. **Extract first, then compose.** Don't copy-paste 350 lines of paramfill.go.

### Decision 2: Custom model for param list, huh for param edit

Parameter CRUD has two distinct UX patterns:
- **List browsing** (navigate, add, delete, reorder) → Custom `tea.Model` with keyboard nav
- **Field editing** (name, type, default, description) → `huh.Form` for structured input

Don't try to force huh into a list manager role. huh excels at linear form flows, not dynamic lists.

### Decision 3: Config-driven keybindings over terminal detection

Terminal detection (`TERM_PROGRAM == "WarpTerminal"`) is a stopgap. The proper solution is user-configurable keybindings stored in config YAML. This solves:
- Warp Ctrl+G conflicts
- Any other terminal-specific conflicts
- Power user customization
- Future terminal compatibility without code changes

---

## Sources

| Source | Type | Confidence |
|--------|------|------------|
| Chroma releases: github.com/alecthomas/chroma/releases | Official releases page, verified v2.23.1 (Jan 23, 2026) | HIGH |
| Chroma Go API: pkg.go.dev/github.com/alecthomas/chroma/v2 | Official docs, TTY formatters confirmed | HIGH |
| Glamour releases: github.com/charmbracelet/glamour/releases | Official, v0.10.0 (Apr 16, 2025) | HIGH |
| huh releases: github.com/charmbracelet/huh/releases | Official, v0.8.0 (Oct 14, 2025), custom Field types confirmed | HIGH |
| Bubble Tea v2 announcement: charm.land | Official, released Feb 23, 2026 | HIGH |
| BT v2 breaking changes: github.com/charmbracelet/bubbletea | Import paths, key handling, view API documented | HIGH |
| Warp keybindings: docs.warp.dev | Ctrl+G = add_next_occurrence, customizable via Settings | HIGH |
| Warp keybinding issues: github.com/warpdotdev/Warp/issues/500, #537 | bindkey interception confirmed by users | MEDIUM |
| TERM_PROGRAM detection | Standard env var, Warp sets "WarpTerminal" — verified via multiple sources | HIGH |
| Project go.mod | Verified current dependency versions directly from codebase | HIGH |
| Project source code | Verified existing patterns in keys.go, form.go, paramfill.go, browse.go | HIGH |
