# Architecture Patterns — v1.1 Polish & Power

**Domain:** Go TUI Terminal Workflow Manager (v1.1 feature integration)
**Researched:** 2026-02-27
**Confidence:** HIGH (based on thorough reading of existing codebase + official library docs)

## Overview

This document analyzes how v1.1 "Polish & Power" features integrate with the existing `wf` codebase architecture. Each feature area is mapped to its integration points, new/modified components, data flow changes, and implementation approach.

### Existing Architecture (Quick Reference)

```
cmd/wf/
  pick.go          → launches picker via tea.NewProgram (alt-screen, /dev/tty output)
  main.go          → cobra CLI routing

internal/
  store/
    store.go       → Store interface: List/Get/Save/Delete
    workflow.go    → Workflow{Name,Command,Description,Tags,Args} + Arg struct
    yaml.go        → YAMLStore (file-per-workflow persistence)
    multi.go       → MultiStore (local + remote aggregation)

  template/
    parser.go      → ExtractParams() with priority: bang > pipe > colon > plain
    renderer.go    → Render() substitutes values into {{...}} placeholders

  picker/
    model.go       → Picker TUI (StateSearch → StateParamFill)
    paramfill.go   → Parameter fill: text input, enum cycling, dynamic commands
    search.go      → Fuzzy search with tag filtering
    styles.go      → Picker styles

  manage/
    manage.go      → Run() entry point, tea.NewProgram with alt-screen
    model.go       → Root Model with viewBrowse/viewCreate/viewEdit/viewSettings
    browse.go      → BrowseModel: sidebar, list, preview, search
    form.go        → FormModel: huh.Form with name/desc/command/tags/folder
    dialog.go      → DialogModel overlay system
    sidebar.go     → SidebarModel: folder/tag navigation
    settings.go    → SettingsModel: theme customization
    ai_actions.go  → AI generate/autofill commands
    keys.go        → Keybinding definitions
    theme.go       → Theme system
    styles.go      → themeStyles struct

  shell/
    zsh.go         → Ctrl+G binding, _wf_picker() ZLE widget
    bash.go        → Bash equivalent
    fish.go        → Fish equivalent

  config/
    config.go      → ConfigDir(), XDG paths
```

### Key Architectural Constraints

1. **huh v0.8 cannot add/remove fields dynamically.** Fields must exist at form creation; only property changes (title, options, visibility) are dynamic.
2. **Bubble Tea value-receiver pattern.** All `Update()`/`View()` use value receivers. Shared mutable state requires heap-allocated structs (see `formValues` pattern).
3. **Two separate tea.Program instances.** Picker (alt-screen + /dev/tty) and manage (alt-screen) are independent programs. They cannot communicate at runtime.
4. **`tea.ExecProcess` exists** for suspending alt-screen, running external process, resuming. Available in bubbletea v1.3.10.

---

## Feature 1: Parameter CRUD in Manage TUI

**Requirement:** Full CRUD on parameters in `wf manage` — add, remove, rename; change type/defaults/description/enum values.

### Current State

- `Workflow.Args []Arg` exists in the data model with `Name, Default, Description, Type, Options, DynamicCmd` fields.
- `FormModel` (form.go) manages name/description/command/tags/folder via `huh.Form` — **does NOT touch Args**.
- `saveWorkflow()` builds a `Workflow` from `formValues` but never populates `Args`.
- Template engine extracts params from command text at runtime — `Args` metadata is supplementary (used for persistence, not for form editing).

### Integration Challenge

huh v0.8 cannot dynamically add/remove form fields. Parameter CRUD requires a variable number of parameter entries, each with multiple sub-fields (name, type, default, options, description). This is fundamentally at odds with huh's static field model.

### Recommended Approach: Custom Param Editor Component

Build a custom Bubble Tea sub-model `ParamEditorModel` that replaces huh for parameter editing. This sits alongside (not inside) the huh form.

**Architecture:**

```
FormModel (form.go)
  ├── huh.Form           → name, description, command, tags, folder (unchanged)
  └── ParamEditorModel   → parameter CRUD (new custom component)
        ├── paramList []ParamEntry
        ├── cursor int
        ├── editingParam *ParamEditForm  (inline edit for selected param)
        └── mode: list | edit | add
```

**State machine for ParamEditorModel:**

```
List Mode (default)
  ↑↓ = navigate params
  'a' = add new param → Add Mode
  'e' / enter = edit selected → Edit Mode
  'd' = delete selected (with confirm)
  tab = return focus to huh form

Edit Mode
  Shows fields: name, type (text/enum/dynamic), default, description
  For enum: editable options list
  For dynamic: dynamic_cmd text input
  enter = save changes → List Mode
  esc = cancel → List Mode

Add Mode
  Same as Edit Mode but creates new entry
  enter = append to list → List Mode
```

### New Components

| Component | File | Purpose |
|-----------|------|---------|
| `ParamEditorModel` | `internal/manage/param_editor.go` | Custom Bubble Tea model for param CRUD |
| `ParamEditForm` | `internal/manage/param_edit_form.go` | Inline edit view for a single param |

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| `FormModel` | `internal/manage/form.go` | Embed `ParamEditorModel`, route messages, include in View, populate `Args` on save |
| `formValues` | `internal/manage/form.go` | Add `args []store.Arg` field (or manage separately in ParamEditorModel) |
| `saveWorkflow()` | `internal/manage/form.go` | Include `wf.Args` from ParamEditorModel state |
| `NewFormModel` | `internal/manage/form.go` | Pre-populate ParamEditorModel from `wf.Args` in edit mode |

### Data Flow

```
Edit workflow pressed:
  1. model.go sends switchToEditMsg{workflow}
  2. NewFormModel("edit", &wf, ...) creates FormModel
  3. ParamEditorModel initialized from wf.Args
  4. User edits params in ParamEditorModel (add/remove/rename/change type)
  5. User completes huh form (enter on last field)
  6. saveWorkflow() reads formValues + ParamEditorModel.Params()
  7. Builds Workflow with populated Args field
  8. Store.Save persists with full Args metadata
```

### Synchronization: Command Text ↔ Args

The template engine extracts params from command text. The Args array provides metadata. These can diverge. Strategy:

- **On form load (edit mode):** Extract params from command text. Merge with existing `wf.Args` — matching by name. Display merged result in ParamEditorModel.
- **On command text change:** Re-extract params. Add new ones to ParamEditorModel. Mark removed ones (warn, don't auto-delete).
- **On save:** Args array = ParamEditorModel state. No auto-sync needed — the user explicitly manages params.

### View Layout

```
┌─────────────────────────────────────────┐
│  Create/Edit Workflow                   │
├─────────────────────────────────────────┤
│  Name:        [________________]        │
│  Description: [________________]        │
│  Command:     [                ]        │
│               [                ]        │
│  Tags:        [________________]        │
│  Folder:      [________________]        │
├─────────────────────────────────────────┤
│  Parameters:                            │
│  ❯ env (enum: dev, staging, prod)       │
│    port (text, default: 3000)           │
│    branch (dynamic: git branch)         │
│                                         │
│  a add  e edit  d delete  tab form      │
└─────────────────────────────────────────┘
```

### Focus Management

Two-zone focus: huh form zone + param editor zone. Tab cycles between them. The `FormModel.Update()` must track which zone has focus and route messages accordingly.

```go
type formFocus int
const (
    focusHuhForm formFocus = iota
    focusParamEditor
)
```

### Build Complexity: HIGH

This is the most complex v1.1 feature. Requires: custom Bubble Tea component, focus management between two input systems (huh + custom), merge logic for command text ↔ args, and comprehensive testing.

---

## Feature 2: List Picker Variable Type

**Requirement:** New variable type that runs a shell command, shows output as selectable list, user picks one, and a specific field from the selected line is stored.

### Current State

- Template parser supports three types: `ParamText`, `ParamEnum`, `ParamDynamic`.
- Dynamic params (`{{name!command}}`) already run shell commands and populate options. User cycles through with ↑↓.
- Picker paramfill already handles dynamic option loading with spinner and fallback.

### How List Picker Differs from Dynamic

| Aspect | Dynamic (existing) | List Picker (new) |
|--------|-------------------|-------------------|
| Syntax | `{{name!command}}` | `{{name?command}}` or `{{name!command|field}}` |
| Display | Simple ↑↓ cycling through values | Full list with search/scroll |
| Selection | Replaces textinput value | Replaces textinput value |
| Field extraction | None (whole line) | Extract specific field (e.g., column 2) |
| Fallback | Text input on failure | Text input on failure |

### Recommended Syntax

Extend the bang syntax with field extraction: `{{name!command:fieldspec}}`

- `{{branch!git branch --list}}` — existing dynamic, returns whole line
- `{{container!docker ps --format "{{.Names}} {{.ID}}"::2}}` — list picker, extract field 2
- `{{pod!kubectl get pods -o name|/}}` — list picker with custom delimiter

**Alternative (cleaner):** Use a new sigil for list picker:

- `{{name?command}}` — list picker (whole line)
- `{{name?command|field_index}}` — list picker with field extraction

**Recommendation:** Use `?` sigil. Keeps the parser's priority system clean (add `?` check between `!` and `|`). The `!` sigil remains for simple dynamic (↑↓ cycling), `?` triggers the richer list picker UX.

### Integration Points

**Template parser (`internal/template/parser.go`):**

```go
// Add new type
ParamListPicker ParamType = 3

// Add to parseInner priority:
// 1. Bang (dynamic) — existing
// 2. Question mark (list picker) — NEW
// 3. Pipe (enum) — existing
// 4. Colon (default) — existing
// 5. Plain — existing
```

**Arg struct (`internal/store/workflow.go`):**

```go
type Arg struct {
    // ... existing fields ...
    FieldIndex int    `yaml:"field_index,omitempty"` // For list picker: which field to extract (1-indexed)
    Delimiter  string `yaml:"delimiter,omitempty"`   // For list picker: field delimiter (default: whitespace)
}
```

**Picker paramfill (`internal/picker/paramfill.go`):**

For list picker params, instead of simple ↑↓ cycling, show a scrollable list with fuzzy search within the options. This reuses existing dynamic command execution but adds:
- Scrollable viewport for long lists
- Optional fuzzy filtering within options
- Field extraction on selection

**Manage ParamEditorModel:**

Add "list" as a type option alongside text/enum/dynamic. Show field_index and delimiter fields when type is "list".

### New Components

| Component | File | Purpose |
|-----------|------|---------|
| `ParamListPicker` type constant | `internal/template/parser.go` | New param type |

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| `parseInner()` | `internal/template/parser.go` | Add `?` sigil handling before pipe check |
| `Param` struct | `internal/template/parser.go` | Add `FieldIndex int`, `Delimiter string` |
| `Arg` struct | `internal/store/workflow.go` | Add `FieldIndex`, `Delimiter` YAML fields |
| `initParamFill()` | `internal/picker/paramfill.go` | Handle `ParamListPicker` — load options, show scrollable list |
| `updateParamFill()` | `internal/picker/paramfill.go` | Add scrollable list navigation + search for list picker params |
| `viewParamFill()` | `internal/picker/paramfill.go` | Render list picker with scroll indicator |

### Data Flow

```
User selects workflow with {{container?docker ps}}:
  1. ExtractParams returns Param{Type: ParamListPicker, DynamicCmd: "docker ps"}
  2. initParamFill fires executeDynamic command
  3. Options load (e.g., 20 container names)
  4. viewParamFill renders scrollable list (not ↑↓ cycling)
  5. User can type to filter within options (inline fuzzy)
  6. User selects → field extraction applied → value stored
  7. Render substitutes selected value into command
```

### Build Complexity: MEDIUM

Extends existing dynamic param infrastructure. Main work is the scrollable list UI in paramfill and the parser extension.

---

## Feature 3: Execute Flow in Manage TUI

**Requirement:** Select a workflow in manage browse view → fill parameters → paste to prompt (or copy to clipboard). Full execute flow without leaving manage.

### Current State

- Manage TUI runs in alt-screen via `tea.NewProgram(m, tea.WithAltScreen())`.
- Picker runs as a separate `tea.Program` instance (also alt-screen, but with /dev/tty output).
- There is no connection between manage and the shell — manage doesn't output to stdout for shell capture.
- `tea.ExecProcess` can suspend alt-screen, run a command, and resume.

### The Core Problem

The manage TUI is a full-screen alt-screen program. To paste to prompt, the result needs to reach the shell function wrapper (`_wf_picker`) which captures stdout. But manage isn't launched via the shell wrapper — it's `wf manage`, not `wf pick`.

### Options Analysis

**Option A: Launch picker as subprocess via `tea.ExecProcess`**

```go
// In manage, when user presses 'x' (execute):
cmd := tea.ExecProcess(
    exec.Command("wf", "pick", "--workflow", selectedWorkflow.Name),
    func(err error) tea.Msg { return execDoneMsg{err} },
)
```

This suspends manage's alt-screen, runs `wf pick` (which renders to /dev/tty), captures the result, and resumes manage. But the result stays within the manage process — it doesn't reach the user's shell prompt.

**Option B: Quit manage with result, let shell wrapper handle it**

```go
// manage exits with a special exit code or writes result to a temp file
// Shell wrapper: if wf manage wrote to /tmp/wf-result, read it
```

This breaks the "stay in manage" requirement.

**Option C: Clipboard + notification (pragmatic)**

```go
// In manage, fill params inline, copy result to clipboard
clipboard.WriteAll(renderedCommand)
// Show flash: "Command copied to clipboard — paste with Ctrl+V"
```

User stays in manage. Command is available via paste. No shell wrapper needed.

**Option D: Inline param fill in manage, then exit to paste**

New view state `viewExecute` with inline param fill (reusing picker's `paramfill` logic). On completion, manage exits and outputs the result to stdout, which the shell wrapper captures — but only if manage was launched via a shell wrapper.

### Recommended Approach: Option C (Clipboard) + Option D (Exit-to-paste)

Implement both:

1. **Primary: Clipboard copy.** Press `x` on a workflow → if zero params, copy command to clipboard immediately. If params exist, show inline param fill (reusing paramfill UI adapted for manage), then copy result.

2. **Secondary: Exit-to-paste.** Press `X` (shift-x) → same flow but manage exits and prints result to stdout. If user launched manage via shell wrapper (`eval "$(wf init zsh)"`), it pastes to prompt. If not, it prints to terminal.

### Integration Points

**New view state:**

```go
const (
    viewBrowse   viewState = iota
    viewCreate
    viewEdit
    viewSettings
    viewExecute  // NEW: inline param fill for selected workflow
)
```

**Reusing picker paramfill logic:**

The `initParamFill`, `updateParamFill`, `viewParamFill` functions are in `internal/picker/paramfill.go` and tightly coupled to `picker.Model`. Two options:

1. **Extract shared param fill component** into `internal/paramfill/` package, used by both picker and manage.
2. **Duplicate with adaptation** — copy paramfill logic into manage with manage-specific adjustments.

**Recommendation:** Option 1 (extract). The paramfill logic is ~250 lines and has no picker-specific dependencies except styling. Extract into a shared package, parameterize styles.

### New Components

| Component | File | Purpose |
|-----------|------|---------|
| `ParamFillModel` | `internal/paramfill/model.go` | Extracted shared param fill component |
| `ExecuteModel` | `internal/manage/execute.go` | Manage-specific execute view wrapping ParamFillModel |

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| `model.go` | `internal/manage/model.go` | Add `viewExecute` state, `execute ExecuteModel` field, route messages |
| `browse.go` | `internal/manage/browse.go` | Add `x` key (clipboard execute) and `X` key (exit-to-paste) |
| `keys.go` | `internal/manage/keys.go` | Add execute keybindings |
| `Model.View()` | `internal/manage/model.go` | Add `viewExecute` rendering case |
| `picker/paramfill.go` | `internal/picker/paramfill.go` | Refactor to use shared ParamFillModel |
| `manage.go` / `Run()` | `internal/manage/manage.go` | Return result string for exit-to-paste mode |

### Data Flow (Clipboard Path)

```
User presses 'x' on workflow in browse:
  1. browse.go sends executeWorkflowMsg{workflow, mode: "clipboard"}
  2. model.go transitions to viewExecute
  3. ExecuteModel.Init() extracts params, fires dynamic commands
  4. User fills params (same UX as picker)
  5. On enter: template.Render(command, values)
  6. clipboard.WriteAll(result)
  7. Flash message: "Copied to clipboard"
  8. Return to viewBrowse
```

### Data Flow (Exit-to-Paste Path)

```
User presses 'X' on workflow in browse:
  1-5. Same as clipboard path
  6. Set Model.exitResult = renderedCommand
  7. Return tea.Quit
  8. Run() returns exitResult
  9. Calling code prints to stdout
  10. Shell wrapper captures and pastes to prompt
```

### Build Complexity: MEDIUM-HIGH

Requires extracting paramfill into shared package, creating execute view, and handling two output modes. The paramfill extraction is the biggest piece.

---

## Feature 4: Syntax Highlighting

**Requirement:** Syntax highlighting in workflow list rendering (command preview panes).

### Current State

- Preview pane in browse (`renderPreview()`) shows raw command text.
- Picker preview (`updatePreview()` / viewport) shows raw command text.
- No syntax highlighting anywhere.

### Library: `alecthomas/chroma`

**Why chroma:**
- Go-native, no external dependencies
- Terminal ANSI output formatters (8-color, 256-color, truecolor)
- Bash/Shell lexer (`lexers.Get("bash")`)
- Used by Hugo, Goldmark, and many Go projects
- Active maintenance

### Integration Points

**Shared highlight function:**

```go
// internal/highlight/highlight.go
func HighlightBash(code string, truecolor bool) string {
    lexer := lexers.Get("bash")
    style := styles.Get("monokai")  // or theme-matched
    formatter := formatters.Get("terminal256")
    // tokenize + format → ANSI string
}
```

**Where to apply:**
1. `manage/browse.go → renderPreview()` — command line in preview pane
2. `picker/model.go → updatePreview()` / viewport content — command preview
3. `manage/form.go → View()` — could highlight the command field (but huh controls rendering, so skip)
4. `manage/browse.go → renderList()` — inline command snippets (optional, may be noisy)

### New Components

| Component | File | Purpose |
|-----------|------|---------|
| `highlight` package | `internal/highlight/highlight.go` | Shared syntax highlighting wrapper |

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| `renderPreview()` | `internal/manage/browse.go` | Apply `highlight.HighlightBash()` to command display |
| `updatePreview()` | `internal/picker/model.go` | Apply highlighting to viewport content |
| `Theme` | `internal/manage/theme.go` | Add `SyntaxTheme string` field for chroma style selection |
| `go.mod` | `go.mod` | Add `github.com/alecthomas/chroma/v2` dependency |

### Performance Consideration

Chroma tokenization is fast (< 1ms for typical commands) but should not run on every keystroke during search. Strategy:
- Highlight only the **selected** workflow's preview (not all visible items)
- Cache highlighted output, invalidate on cursor change
- Preview updates on cursor move — acceptable latency

### Theme Integration

Map the manage theme colors to a chroma style, or provide a fixed set of chroma styles that pair well with the existing theme presets (dracula, monokai, catppuccin, etc.).

### Build Complexity: LOW

Small, self-contained feature. Add chroma dependency, write thin wrapper, apply in two preview locations.

---

## Feature 5: Variable Default Auto-Save

**Requirement:** When a user fills in parameter values during picker use, auto-save those values as defaults in the workflow's YAML config.

### Current State

- Picker paramfill collects values via textinput models.
- On enter, `template.Render()` substitutes values and sets `m.Result`.
- `m.Result` is printed to stdout. The picker exits. **No persistence of entered values.**
- Workflow `Args` have `Default` fields, but they're only populated during import or AI autofill, never during picker use.

### Integration Challenge

The picker currently has no write access to the store. It receives `[]store.Workflow` at creation and operates read-only. Auto-save requires:
1. Picker needs a `store.Store` reference (or a callback).
2. After param fill, update `workflow.Args[i].Default` for each param where the user entered a non-default value.
3. Save the updated workflow.

### Recommended Approach

Pass the store to the picker. After successful param fill (before `tea.Quit`), fire a background save command.

```go
// picker.New now accepts store
func New(workflows []store.Workflow, s store.Store) Model { ... }

// After render in updateParamFill enter handler:
m.Result = template.Render(m.selected.Command, values)
// Save defaults
return m, tea.Batch(
    tea.Quit,
    saveDefaultsCmd(m.selected, m.params, values, m.store),
)
```

**`saveDefaultsCmd` logic:**

```go
func saveDefaultsCmd(wf *store.Workflow, params []template.Param, values map[string]string, s store.Store) tea.Cmd {
    return func() tea.Msg {
        // Merge entered values into workflow Args as defaults
        for _, p := range params {
            v, ok := values[p.Name]
            if !ok || v == "" {
                continue
            }
            // Find or create matching Arg
            found := false
            for j := range wf.Args {
                if wf.Args[j].Name == p.Name {
                    wf.Args[j].Default = v
                    found = true
                    break
                }
            }
            if !found {
                wf.Args = append(wf.Args, store.Arg{Name: p.Name, Default: v})
            }
        }
        s.Save(wf)
        return nil // picker is quitting, msg won't be processed
    }
}
```

### Default Priority at Runtime

When filling params, defaults should come from (in priority order):
1. Workflow `Args[].Default` (persisted from previous runs)
2. Template inline default (`{{name:default_value}}`)
3. Empty (user must type)

Currently `initParamFill` uses `p.Default` from template parsing. Need to merge with `wf.Args` defaults:

```go
// In initParamFill, after extracting params from template:
for i, p := range m.params {
    // Check if workflow.Args has a saved default for this param
    for _, arg := range m.selected.Args {
        if arg.Name == p.Name && arg.Default != "" {
            m.params[i].Default = arg.Default // Saved default takes priority
            break
        }
    }
}
```

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| `picker.New()` | `internal/picker/model.go` | Accept `store.Store` parameter |
| `picker.Model` | `internal/picker/model.go` | Add `store store.Store` field |
| `initParamFill()` | `internal/picker/paramfill.go` | Merge `wf.Args` defaults with template defaults |
| `updateParamFill()` enter handler | `internal/picker/paramfill.go` | Fire `saveDefaultsCmd` on submit |
| `runPick()` | `cmd/wf/pick.go` | Pass store to `picker.New()` |
| `Workflow.Args` | already exists | No struct changes needed |

### Edge Cases

- **Remote workflows (read-only):** MultiStore's remote sources are read-only. Don't attempt save for remote workflows. Check `store.Save()` error gracefully.
- **Concurrent access:** User might be editing the same workflow in manage while picker saves defaults. The YAML file write is atomic (write-temp-rename), so last write wins. Acceptable for this use case.
- **Enum/dynamic params:** Should enum selections be saved as defaults? Yes — if the user consistently picks "staging", save it.

### Build Complexity: LOW-MEDIUM

Small code changes but touches a core flow (picker). Needs careful testing to avoid breaking the critical picker path.

---

## Feature 6: Warp Terminal Compatibility

**Requirement:** Fix Ctrl+G keybinding conflict with Warp terminal. Provide fallback keybinding.

### Current State

- Shell integration scripts bind `Ctrl+G` via `bindkey` (zsh), `bind` (bash).
- Warp terminal intercepts `Ctrl+G` for its own "Add Selection for Next Occurrence" / agent output toggle.
- Warp also ignores custom `bindkey` commands from shell integration scripts.
- Detection: `$TERM_PROGRAM` is set to `"WarpTerminal"` in Warp.

### Recommended Approach

1. **Detect Warp** in shell integration scripts via `$TERM_PROGRAM`.
2. **Use alternative keybinding** for Warp: `Ctrl+;` or `Ctrl+Space` or `Ctrl+O` (less commonly used).
3. **Make keybinding configurable** in `config.yaml`.

### Implementation

**Shell script changes (`internal/shell/zsh.go`):**

```bash
# In ZshScript:
if [[ "$TERM_PROGRAM" == "WarpTerminal" ]]; then
  # Warp intercepts Ctrl+G. Use Ctrl+Space as fallback.
  bindkey '^ ' _wf_picker  # Ctrl+Space
else
  bindkey -M emacs '^G' _wf_picker
  bindkey -M viins '^G' _wf_picker
  bindkey -M vicmd '^G' _wf_picker
fi
```

**Config-based override (`config.yaml`):**

```yaml
keybinding: "ctrl+g"  # or "ctrl+space", "ctrl+o", etc.
```

The shell script reads this:

```bash
_wf_keybind="${WF_KEYBIND:-^G}"
# ... use $_wf_keybind in bindkey
```

Or simpler: `wf init zsh` reads config and generates the script with the correct keybinding baked in.

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| `ZshScript` | `internal/shell/zsh.go` | Add Warp detection + fallback keybinding |
| `BashScript` | `internal/shell/bash.go` | Same Warp detection |
| `FishScript` | `internal/shell/fish.go` | Same Warp detection |
| `Config` | `internal/config/config.go` | Add `Keybinding string` field (optional) |
| `wf init` command | `cmd/wf/init.go` (if exists) | Read config keybinding, inject into script |

### Build Complexity: LOW

Simple conditional in shell scripts. Optional config field.

---

## Feature 7: Additional UX Fixes

### 7a. Folder Auto-Display in Manage

**Requirement:** Auto-display folder contents in manage without extra keypress.

**Current behavior:** Selecting a folder in sidebar sets a filter, but user must press Enter to apply.

**Fix:** In `SidebarModel.Update()`, on cursor move (↑↓), automatically emit `sidebarFilterMsg` for the current selection. This is a small change in `sidebar.go`.

**Modified:** `internal/manage/sidebar.go` — emit filter on cursor move, not just on Enter.

### 7b. Command Preview Overscroll Fix

**Requirement:** Fix command preview panel overscroll in manage view.

**Current behavior:** Preview pane in `renderPreview()` has a fixed height of 6 lines. Long commands overflow.

**Fix:** Either:
1. Truncate command display to available height with `...` indicator
2. Use a `viewport.Model` for the preview pane (scrollable)

**Recommendation:** Use viewport. The picker already uses `viewport.Model` for preview. Apply the same pattern in browse.

**Modified:** `internal/manage/browse.go` — add `viewport.Model` for preview, handle scroll keys in preview.

### 7c. Per-Field Generate Action

**Requirement:** Per-field Generate action for individual variables in manage.

**Current behavior:** AI autofill fills all fields at once.

**Fix:** In the ParamEditorModel (from Feature 1), add a `g` key that generates a value for the focused parameter using AI. This calls a targeted AI prompt: "Suggest a default value for parameter [name] in command [command]".

**Depends on:** Feature 1 (ParamEditorModel) being built first.

**Modified:** `internal/manage/param_editor.go` (new file), `internal/manage/ai_actions.go`.

---

## Suggested Build Order

Based on dependency analysis and risk:

```
Phase 1: Foundation fixes (low risk, unblock other features)
  ├── 1a. Warp compatibility (Feature 6) — standalone, no dependencies
  ├── 1b. Syntax highlighting (Feature 4) — standalone, add chroma dep
  ├── 1c. Folder auto-display (Feature 7a) — tiny sidebar.go change
  └── 1d. Preview overscroll fix (Feature 7b) — small browse.go change

Phase 2: Variable defaults (medium risk, core flow change)
  └── 2a. Auto-save defaults (Feature 5) — touches picker, needs testing

Phase 3: Execute flow (medium-high risk, architecture change)
  ├── 3a. Extract shared ParamFillModel from picker → internal/paramfill/
  ├── 3b. Build ExecuteModel in manage using shared ParamFillModel
  └── 3c. Refactor picker to use shared ParamFillModel

Phase 4: Parameter CRUD (highest risk, most new code)
  ├── 4a. Build ParamEditorModel (custom component)
  ├── 4b. Integrate into FormModel with focus management
  ├── 4c. Wire up save with Args persistence
  └── 4d. Per-field AI generate (Feature 7c) — depends on 4a

Phase 5: List picker variable type (medium risk, parser + UI)
  ├── 5a. Parser extension (? sigil, ParamListPicker type)
  ├── 5b. Scrollable list UI in paramfill
  └── 5c. Field extraction logic
```

### Build Order Rationale

1. **Phase 1 first:** Low-risk standalone fixes. Ship quick wins. Warp compat and syntax highlighting are independent.
2. **Phase 2 before Phase 3:** Auto-save defaults is a smaller change to the picker that validates the "picker writes to store" pattern before the larger execute flow refactor.
3. **Phase 3 before Phase 4:** Execute flow requires extracting `ParamFillModel` as a shared component. This refactoring must happen before the param CRUD feature adds its own param-related models, to avoid conflicting architectures.
4. **Phase 4 is highest risk:** Custom Bubble Tea component with focus management between two input systems. Benefits from all prior phases being stable.
5. **Phase 5 last:** List picker extends the paramfill system. If Phase 3 extracted it properly, Phase 5 just adds a new mode to the shared component. Also benefits from Phase 4's param metadata being available.

---

## Cross-Cutting Concerns

### Testing Strategy

| Feature | Test Approach |
|---------|---------------|
| Parameter CRUD | Unit test ParamEditorModel with teatest; round-trip: create params → save → reload → verify |
| List Picker | Unit test parser for `?` sigil; integration test with mock shell command |
| Execute Flow | Unit test ParamFillModel extraction; integration test clipboard write |
| Syntax Highlighting | Unit test highlight function output contains ANSI codes; snapshot test |
| Auto-Save Defaults | Integration test: pick workflow → fill params → verify YAML updated |
| Warp Compat | Manual test in Warp terminal; unit test script generation with TERM_PROGRAM set |

### State Management Patterns

All new models should follow the established pattern:
- Value receivers for `Init()`, `Update()`, `View()`
- Heap-allocated shared state for mutable data bound to UI widgets (see `formValues`)
- Message-based communication between parent/child models
- `tea.Cmd` for all I/O operations

### Backwards Compatibility

- **YAML format:** Adding `field_index` and `delimiter` fields to `Arg` uses `omitempty` — existing workflows load without changes.
- **Template syntax:** `?` sigil is new — no existing commands use it, so no conflicts.
- **Picker API:** `picker.New()` gaining a `store.Store` parameter is a breaking change for callers, but there's only one callsite (`pick.go`).

---

## Component Boundary Summary

### New Packages

| Package | Purpose | Dependencies |
|---------|---------|-------------|
| `internal/highlight/` | Syntax highlighting wrapper | `chroma/v2` |
| `internal/paramfill/` | Shared param fill component (extracted from picker) | `template`, `store`, `bubbles/textinput` |

### New Files

| File | Package | Purpose |
|------|---------|---------|
| `highlight.go` | `internal/highlight/` | `HighlightBash()` function |
| `model.go` | `internal/paramfill/` | Shared `ParamFillModel` |
| `param_editor.go` | `internal/manage/` | `ParamEditorModel` for CRUD |
| `execute.go` | `internal/manage/` | `ExecuteModel` wrapping shared param fill |

### Modified Files (by impact)

| File | Impact | Features Affected |
|------|--------|-------------------|
| `internal/manage/form.go` | HIGH | Parameter CRUD (embed ParamEditorModel, focus mgmt) |
| `internal/picker/paramfill.go` | HIGH | Refactor to use shared model, auto-save defaults, list picker |
| `internal/picker/model.go` | MEDIUM | Accept store, auto-save defaults |
| `internal/template/parser.go` | MEDIUM | List picker `?` sigil |
| `internal/store/workflow.go` | LOW | Add FieldIndex, Delimiter to Arg |
| `internal/manage/model.go` | MEDIUM | viewExecute state, execute routing |
| `internal/manage/browse.go` | MEDIUM | Execute keys, viewport preview, syntax highlighting |
| `internal/shell/zsh.go` | LOW | Warp detection |
| `internal/shell/bash.go` | LOW | Warp detection |
| `internal/shell/fish.go` | LOW | Warp detection |
| `internal/manage/sidebar.go` | LOW | Auto-filter on cursor move |
| `go.mod` | LOW | Add chroma dependency |

---

## Sources

- **Codebase analysis:** All files listed in "Existing Architecture" section read thoroughly (HIGH confidence)
- **huh v0.8 limitations:** Verified from codebase usage patterns and huh source (HIGH confidence)
- **`tea.ExecProcess`:** Bubble Tea v1.3.10 API, verified in bubbletea source (HIGH confidence)
- **Warp `$TERM_PROGRAM`:** Warp documentation and community reports (MEDIUM confidence)
- **`alecthomas/chroma`:** GitHub repository, used by Hugo (HIGH confidence)
- **Clipboard approach for execute:** `atotto/clipboard` already in go.mod, used by picker (HIGH confidence)
