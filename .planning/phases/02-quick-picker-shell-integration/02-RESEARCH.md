# Phase 2: Quick Picker & Shell Integration - Research

**Researched:** 2026-02-21
**Domain:** Go TUI (Bubble Tea), fuzzy search, shell integration, clipboard
**Confidence:** HIGH (shell paste mechanism verified from fzf/atuin source; Bubble Tea verified from official docs; fuzzy library verified from README)

---

## Summary

Phase 2 is primarily a shell integration and TUI composition problem. The most critical decision — how to paste a completed command into the user's shell prompt — has a well-established solution: **shell function wrappers**. TIOCSTI is deprecated in Linux kernel 6.2+ and must not be used. Every production tool in this space (fzf, atuin) uses the same pattern: the binary outputs the selection to stderr/stdout, and a shell function catches it and writes it to `BUFFER`/`LBUFFER`/`READLINE_LINE`. This is verified directly from fzf and atuin source code.

The TUI layer uses the Charm stack (Bubble Tea + Bubbles + Lip Gloss), which is the established standard for Go terminal applications. For this phase, **do not use `bubbles/list`** — its built-in fuzzy filtering is inflexible and the fzf-style density layout requires custom rendering. Use `bubbles/textinput` + `bubbles/viewport` + `sahilm/fuzzy` directly. The picker model has two states: `StateSearch` and `StateParamFill`. State transitions drive everything.

Sub-100ms startup is achievable without heroics. A Go binary that reads YAML files only pays startup cost for binary loading + file I/O. With 100 workflows at ~2KB each, that's 200KB of YAML. On modern hardware this is well under 50ms. The main risks are: CGo dependencies (avoid them), large `init()` functions, and unnecessary allocations on the hot path.

**Primary recommendation:** Use shell function wrappers (LBUFFER for zsh, READLINE_LINE for bash, commandline -r for fish), sahilm/fuzzy for matching, Bubble Tea + textinput + viewport + lipgloss for TUI, atotto/clipboard for clipboard, and tea.WithAltScreen for the picker overlay.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/bubbletea` | latest (~v1.x) | TUI framework (Elm arch) | Industry standard for Go TUIs; 10K+ dependents |
| `github.com/charmbracelet/bubbles` | latest | textinput, viewport, key components | Official companion components for Bubble Tea |
| `github.com/charmbracelet/lipgloss` | latest | Styling, layout, borders | CSS-in-Go for terminal; standard in Charm stack |
| `github.com/sahilm/fuzzy` | latest (~v0.1.x) | Fuzzy matching with match positions | Zero deps; returns MatchedIndexes for highlighting; VSCode-style scoring |
| `github.com/atotto/clipboard` | latest | Clipboard read/write | Simple text clipboard; no CGo on macOS |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.design/x/clipboard` | v0.7.1 | Advanced clipboard with image/watch | Only if atotto/clipboard proves insufficient; requires CGo and X11 on Linux |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `sahilm/fuzzy` | `lithammer/fuzzysearch` | lithammer uses Levenshtein (slower, less intuitive for symbol matching); sahilm has char position data needed for highlighting |
| `sahilm/fuzzy` | `bubbles/list` built-in filter | bubbles/list filter is inflexible; can't search across name+desc+tags as one query |
| `atotto/clipboard` | `golang.design/x/clipboard` | golang.design requires CGo and is heavier; atotto is pure Go on macOS and uses xclip/xsel on Linux |
| `tea.WithAltScreen` | inline mode | Inline mode leaves garbage in scrollback; altscreen gives clean exit and full terminal width |

### Installation
```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/sahilm/fuzzy
go get github.com/atotto/clipboard
```

---

## Architecture Patterns

### Recommended Project Structure
```
cmd/wf/
├── pick.go             # wf pick command (TUI entry point)
├── init.go             # wf init zsh|bash|fish command (outputs shell script)
internal/
├── picker/
│   ├── model.go        # Main tea.Model: StateSearch + StateParamFill
│   ├── search.go       # Fuzzy match logic, tag filter (@tag prefix)
│   ├── result.go       # Result type with MatchedIndexes for highlighting
│   ├── paramfill.go    # Parameter fill sub-model (textinputs per param)
│   └── styles.go       # Lip Gloss styles
internal/
├── shell/
│   ├── zsh.go          # zsh init script template
│   ├── bash.go         # bash init script template
│   └── fish.go         # fish init script template
```

### Pattern 1: Shell Function Wrapper (THE Core Pattern)

**What:** The binary runs as a subprocess. The shell function captures its output and writes it to the line editor buffer. No ioctl, no TIOCSTI.

**How it works:**
1. Shell function invokes `wf pick` with stderr swapped to stdout (3>&1 1>&2 2>&3)
2. `wf pick` renders TUI on stderr (the actual terminal), outputs selected command on stdout
3. Shell function captures stdout, assigns to LBUFFER/READLINE_LINE/commandline

**Verified from fzf source (key-bindings.zsh):**
```bash
# ZSH — fzf pattern, verified from source
fzf-history-widget() {
  # Run TUI, capture output
  selected="$( ... fzf ...)"
  # Assign to ZLE buffer variables
  BUFFER="$selected"
  CURSOR=${#BUFFER}
  zle reset-prompt
}
zle -N fzf-history-widget
bindkey '^R' fzf-history-widget
```

**Verified from fzf source (key-bindings.bash):**
```bash
# BASH — fzf pattern, verified from source
__fzf_history__() {
  output=$(... fzf ...)
  READLINE_LINE=$output        # Set readline buffer
  READLINE_POINT=0x7fffffff   # Move cursor to end
}
bind -m emacs-standard -x '"\C-r": __fzf_history__'
```

**Verified from atuin source (atuin.fish):**
```fish
# FISH — atuin pattern, verified from source
function _atuin_search
  set ATUIN_H (ATUIN_QUERY=(commandline -b) atuin search -i 3>&1 1>&2 2>&3 | string collect)
  if test -n "$ATUIN_H"
    commandline -r "$ATUIN_H"   # Replace command line buffer
  end
  commandline -f repaint
end
```

**The `3>&1 1>&2 2>&3` pattern:** This swaps stdout and stderr so the TUI renders on the actual terminal (fd 2 = stderr = /dev/tty) and the final selection goes to fd 1 (stdout) which the shell function captures. This is the canonical approach for interactive TUI tools.

### Pattern 2: Binary Output Protocol

When the user confirms a selection, `wf pick` writes the completed command to stdout and exits. The shell function captures it.

For the "accept immediately" variant (atuin's `__atuin_accept__:` prefix):
```
# Normal: put command on prompt line
fmt.Fprintln(os.Stdout, completedCommand)

# Execute immediately (if we later want that feature):
fmt.Fprintf(os.Stdout, "__wf_accept__:%s\n", completedCommand)
```

For Phase 2, only the normal variant (paste to prompt) is needed. The shell function checks for the value and assigns it.

### Pattern 3: Bubble Tea Multi-State Picker Model

**Two-state model:** `StateSearch` → (user selects workflow with params) → `StateParamFill` → (all params filled, Enter) → output + quit.

```go
// Source: Bubble Tea tutorial + architecture patterns
type viewState int

const (
    StateSearch   viewState = iota
    StateParamFill
)

type Model struct {
    // Core state
    state       viewState
    workflows   []store.Workflow  // all workflows, loaded once at init

    // Search state
    query       string
    tagFilter   string            // parsed from @tag prefix
    searchInput textinput.Model
    results     []fuzzy.Match     // sahilm/fuzzy results
    cursor      int               // selected result index
    preview     viewport.Model    // command preview pane

    // Param fill state
    selected    *store.Workflow   // the chosen workflow
    params      []store.Param     // extracted params
    inputs      []textinput.Model // one per param
    focusedInput int              // tab index
    rendered    string            // live-rendered command

    // Layout
    width  int
    height int
}
```

**Init:** Load all workflows synchronously in `Init()` (fast enough for 100+ YAML files). Return `tea.WindowSizeMsg` command to get terminal dimensions.

**Update state machine:**
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        m.preview.Width = msg.Width
        m.preview.Height = 6  // fixed preview height

    case tea.KeyMsg:
        switch m.state {
        case StateSearch:
            return m.updateSearch(msg)
        case StateParamFill:
            return m.updateParamFill(msg)
        }
    }
    // Delegate to focused component
    ...
}
```

**Zero-param instant paste:** In `updateSearch`, when Enter is pressed and the selected workflow has no params, skip `StateParamFill` entirely — output directly and `tea.Quit`.

### Pattern 4: Tag Filter with @tag Prefix Parsing

Parse from the query string before fuzzy matching:

```go
// Source: design decision from CONTEXT.md
func parseQuery(raw string) (tagFilter, fuzzyQuery string) {
    // "@docker kubernetes" → tagFilter="docker", fuzzyQuery="kubernetes"
    // "deploy" → tagFilter="", fuzzyQuery="deploy"
    if strings.HasPrefix(raw, "@") {
        parts := strings.SplitN(raw[1:], " ", 2)
        tagFilter = parts[0]
        if len(parts) > 1 {
            fuzzyQuery = parts[1]
        }
        return
    }
    fuzzyQuery = raw
    return
}

func filterWorkflows(workflows []store.Workflow, tagFilter, query string) []fuzzy.Match {
    // Step 1: tag pre-filter
    candidates := workflows
    if tagFilter != "" {
        candidates = filterByTag(workflows, tagFilter)
    }
    // Step 2: fuzzy match on "name description tags command" concatenated
    return fuzzy.FindFrom(query, workflowSource(candidates))
}
```

**The `Source` interface for sahilm/fuzzy:**
```go
// Source: sahilm/fuzzy README (verified)
type workflowSource []store.Workflow

func (ws workflowSource) String(i int) string {
    w := ws[i]
    // Search across name + description + tags (not command — command is preview only)
    return w.Name + " " + w.Description + " " + strings.Join(w.Tags, " ")
}

func (ws workflowSource) Len() int { return len(ws) }
```

### Pattern 5: Param Fill with Tab Navigation

```go
// In StateParamFill, tabs cycle through inputs
case "tab", "shift+tab":
    // advance/retreat focus
    m.inputs[m.focusedInput].Blur()
    if msg.String() == "tab" {
        m.focusedInput = (m.focusedInput + 1) % len(m.inputs)
    } else {
        m.focusedInput = (m.focusedInput - 1 + len(m.inputs)) % len(m.inputs)
    }
    m.inputs[m.focusedInput].Focus()

case "enter":
    // On last param: render final command, output, quit
    if m.focusedInput == len(m.inputs)-1 {
        cmd := renderFinalCommand(m)
        fmt.Fprintln(os.Stdout, cmd)
        return m, tea.Quit
    }
    // Otherwise advance to next
    m.focusedInput++
    ...

// Live render on every keystroke
m.rendered = renderCommand(m.selected.Command, m.inputs)
```

### Pattern 6: wf init Shell Script Output

The `wf init zsh` command prints shell code to stdout that the user sources:

```go
// cmd/wf/init.go
func runInit(shell string) {
    switch shell {
    case "zsh":
        fmt.Print(zshScript)
    case "bash":
        fmt.Print(bashScript)
    case "fish":
        fmt.Print(fishScript)
    }
}
```

**zsh script (wf_zsh_script.go):**
```zsh
# Paste to prompt (no execute)
_wf_widget() {
  local output
  output=$(wf pick 3>&1 1>&2 2>&3)
  if [[ -n "$output" ]]; then
    LBUFFER="$output"
    RBUFFER=""
  fi
  zle reset-prompt
}
zle -N _wf_widget
bindkey '^G' _wf_widget    # Ctrl+G — safe, not used by default in zsh
```

**bash script:**
```bash
_wf_picker() {
  local output
  output=$(wf pick 3>&1 1>&2 2>&3)
  if [[ -n "$output" ]]; then
    READLINE_LINE="$output"
    READLINE_POINT=${#READLINE_LINE}
  fi
}
bind -m emacs-standard -x '"\C-g": _wf_picker'
```

**fish script:**
```fish
function _wf_picker
  set -l output (wf pick 3>&1 1>&2 2>&3 | string collect)
  if test -n "$output"
    commandline -r $output
  end
  commandline -f repaint
end
bind \cg _wf_picker
```

### Pattern 7: Existing text on shell prompt

**Problem:** User has typed `git push origin ` and then invokes the picker.

**How fzf/atuin handle it:**
- fzf-file-widget: `LBUFFER="${LBUFFER}$(__fzf_select__)"` — APPENDS to existing
- atuin: `LBUFFER=$output` — REPLACES entirely

**Recommendation:** Replace (atuin approach). `wf pick` is invoked to find and fill a complete command — replacing existing text is the expected behavior. If preserving is needed, pass existing BUFFER as an env var and prepopulate search. For Phase 2, replace is correct.

### Anti-Patterns to Avoid
- **TIOCSTI**: Deprecated, disabled in Linux 6.2+, security risk. Never use.
- **`print -z` in zsh**: An older zsh-specific approach — NOT portable, and doesn't work when called from a subprocess. The LBUFFER approach is correct.
- **`bubbles/list` built-in fuzzy**: Its filtering runs on item title only; can't search across name+desc+tags+command. Use sahilm/fuzzy directly.
- **tea.WithAltScreen as ProgramOption**: In Bubble Tea v2, use `tea.WithAltScreen()` as a `ProgramOption` (not the `Init()` command). This is the documented best practice.
- **Loading workflows in Update()**: Load once in `Init()` via a `tea.Cmd`, not on every keystroke.
- **Writing selected command to a temp file**: The stderr/stdout swap pattern is cleaner. Temp file approach introduces race conditions.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Fuzzy matching with scoring | Custom string matching | `sahilm/fuzzy` | Returns MatchedIndexes for character highlighting; correct scoring for adjacent chars |
| TUI input fields | Raw key handling | `bubbles/textinput` | Handles cursor, backspace, unicode, paste, scrolling on long values |
| Scrollable preview pane | Manual line clipping | `bubbles/viewport` | Handles scroll position, line wrapping, resize |
| Key binding declarations | `msg.String() == "ctrl+g"` | `bubbles/key.NewBinding` | Enables help text generation and remapping |
| Terminal width detection | os.Getenv("COLUMNS") | `tea.WindowSizeMsg` | Bubble Tea sends this automatically on startup and resize |
| Character highlighting in results | ANSI escape concatenation | `lipgloss.Style.Render()` | Handles truncation, width calculations, composability |

**Key insight:** Every problem in this phase has a Charm ecosystem solution. The only truly custom logic is: tag filter parsing, the @tag prefix syntax, and the workflow→string function for fuzzy. Everything else uses existing primitives.

---

## Common Pitfalls

### Pitfall 1: Forgetting the stderr/stdout swap
**What goes wrong:** `wf pick` renders TUI on stdout. Shell function captures it as the "output". The user sees no TUI, just a blank prompt.
**Why it happens:** Default TUI rendering goes to stdout. The shell function captures stdout for the result.
**How to avoid:** Run Bubble Tea with `tea.WithOutput(os.Stderr)` explicitly. Write the final selection to `os.Stdout`. In the shell script, invoke as `wf pick 3>&1 1>&2 2>&3` (swap stdout/stderr).
**Verified from:** atuin.zsh: `ATUIN_SHELL=zsh ATUIN_LOG=error ATUIN_QUERY=$BUFFER atuin search "${search_args[@]}" -i 3>&1 1>&2 2>&3`

### Pitfall 2: Alt Screen and Inline Mix-Up
**What goes wrong:** Using `tea.WithAltScreen()` but then the picker "disappears" when it exits — correct behavior, but the user wants to see what they selected. Alternatively, NOT using alt screen means garbage stays in scrollback.
**Why it happens:** Alt screen restores terminal state on exit — this IS correct for a picker. The selected command appears on the prompt, not on screen.
**How to avoid:** Use `tea.WithAltScreen()`. This is the correct UX — picker disappears, command appears on prompt.

### Pitfall 3: Bubble Tea v1 vs v2 API Changes
**What goes wrong:** `tea.Program.Start()` is deprecated; `tea.NewProgram().Run()` is the current API. `p.Kill()` vs `p.Quit()` semantics differ.
**How to avoid:** Use `p.Run()` which returns `(tea.Model, error)`. Check for breaking changes at `github.com/charmbracelet/bubbletea/releases`.

### Pitfall 4: sahilm/fuzzy with Empty Query
**What goes wrong:** `fuzzy.FindFrom("", source)` returns zero results instead of all results.
**Why it happens:** Empty string matches nothing in fuzzy algorithms.
**How to avoid:** When query is empty (and no tag filter), bypass fuzzy matching and return all workflows ranked by name. Only call `fuzzy.FindFrom` when `len(query) > 0`.

### Pitfall 5: Keybinding Conflicts
**What goes wrong:** Binding Ctrl+G in bash may conflict with readline's `abort` function (which is the default for Ctrl+G in emacs mode).
**Why it happens:** Ctrl+G = BEL in ASCII; readline uses it as abort.
**How to avoid:** Use `bind -m emacs-standard -x '"\C-g": _wf_picker'` which overrides readline's abort only in emacs mode — this is acceptable since `wf` is the main purpose. Alternatively use Ctrl+\ (less conflicting). Let users configure via environment variable.
**Recommended default:** `Ctrl+G` — conflicts only with readline's rarely-used "abort" command; frees Ctrl+R for history, Ctrl+T for fzf users.

### Pitfall 6: YAML Load Performance
**What goes wrong:** Loading 100+ YAML files in `Init()` takes >100ms.
**Why it happens:** Large `init()` functions, CGo YAML parsers, or parsing entire filesystem.
**How to avoid:** Use `goccy/go-yaml` (already in go.mod — pure Go, fast). Load all workflows at startup in a single pass. 100 files × 2KB = 200KB total — this is microseconds of I/O, not milliseconds. Profile before optimizing.

### Pitfall 7: Windows clipboard without CGo
**What goes wrong:** `atotto/clipboard` on Windows requires `golang.org/x/sys/windows` which may need CGo.
**Why it happens:** Windows clipboard API requires Win32 calls.
**How to avoid:** `atotto/clipboard` uses the Windows clipboard API via pure Go syscalls on modern Go. Test cross-compilation with `GOOS=windows GOARCH=amd64 go build`. This should work.

### Pitfall 8: textinput Focus Management
**What goes wrong:** Multiple textinput.Models exist (one per param) but all respond to keypresses simultaneously.
**Why it happens:** Bubble Tea routes all messages to all sub-models unless you guard by focus.
**How to avoid:** Only call `input.Update(msg)` for `m.inputs[m.focusedInput]`. Call `input.Blur()` on all others, `input.Focus()` on the current one.

---

## Code Examples

### Shell Script: zsh integration (full, production-ready)
```zsh
# Source: Adapted from fzf key-bindings.zsh + atuin.zsh (both verified)
# Usage: eval "$(wf init zsh)"  or  source <(wf init zsh)
_wf_picker() {
  local output
  # Swap fd: TUI on stderr (terminal), selection on stdout (captured)
  output=$(wf pick 3>&1 1>&2 2>&3)
  local ret=$?
  if [[ -n "$output" ]]; then
    LBUFFER="$output"   # Replace current prompt content
    RBUFFER=""
  fi
  zle reset-prompt
  return $ret
}
zle -N _wf_picker
bindkey -M emacs '^G' _wf_picker
bindkey -M viins '^G' _wf_picker
bindkey -M vicmd '^G' _wf_picker
```

### Shell Script: bash integration
```bash
# Source: Adapted from fzf key-bindings.bash (verified)
# Usage: eval "$(wf init bash)"
_wf_picker() {
  local output
  output=$(wf pick 3>&1 1>&2 2>&3)
  if [[ -n "$output" ]]; then
    READLINE_LINE="$output"
    READLINE_POINT=${#READLINE_LINE}   # Cursor to end
  fi
}
# Requires bash >= 4.0 for bind -x with READLINE_LINE
bind -m emacs-standard -x '"\C-g": _wf_picker'
bind -m vi-insert -x '"\C-g": _wf_picker'
```

### Shell Script: fish integration
```fish
# Source: Adapted from atuin.fish (verified)
# Usage: wf init fish | source
function _wf_picker
  set -l output (wf pick 3>&1 1>&2 2>&3 | string collect)
  if test -n "$output"
    commandline -r $output   # Replace line buffer
  end
  commandline -f repaint
end
bind \cg _wf_picker
# Also bind in vi insert mode
bind -M insert \cg _wf_picker
```

### Bubble Tea: Program startup with alt screen and stderr output
```go
// Source: Bubble Tea README + official API
// cmd/wf/pick.go
func runPick() error {
    workflows, err := loadAllWorkflows()
    if err != nil {
        return err
    }

    m := picker.New(workflows)
    p := tea.NewProgram(
        m,
        tea.WithAltScreen(),          // Clean overlay, restores on exit
        tea.WithOutput(os.Stderr),    // TUI renders to stderr (terminal)
        tea.WithInput(os.Stdin),      // Keys from stdin
    )

    final, err := p.Run()
    if err != nil {
        return err
    }

    // If user selected something, write to stdout (captured by shell function)
    if fm, ok := final.(picker.Model); ok && fm.Result != "" {
        fmt.Fprintln(os.Stdout, fm.Result)
    }
    return nil
}
```

### sahilm/fuzzy: FindFrom with Source interface
```go
// Source: sahilm/fuzzy README (verified)
// internal/picker/search.go
type workflowSource []store.Workflow

func (ws workflowSource) String(i int) string {
    w := ws[i]
    // Combine searchable fields — weights implicit in field order
    return w.Name + " " + w.Description + " " + strings.Join(w.Tags, " ")
}

func (ws workflowSource) Len() int { return len(ws) }

func Search(query string, tagFilter string, workflows []store.Workflow) []fuzzy.Match {
    // Tag pre-filter
    filtered := workflows
    if tagFilter != "" {
        filtered = filterByTag(workflows, tagFilter)
    }
    // Return all on empty query
    if query == "" {
        // Return synthetic matches for all, in original order
        matches := make([]fuzzy.Match, len(filtered))
        for i, w := range filtered {
            matches[i] = fuzzy.Match{Str: w.Name, Index: i}
        }
        return matches
    }
    return fuzzy.FindFrom(query, workflowSource(filtered))
}
```

### Lip Gloss: Result list row with highlighted chars
```go
// Source: Bubble Tea + Lip Gloss official docs patterns
// internal/picker/styles.go
var (
    normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
    selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
    highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
    tagStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
    dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func renderResultRow(w store.Workflow, match fuzzy.Match, isSelected bool, width int) string {
    // Highlight matched chars in name
    name := highlightMatchedChars(w.Name, match.MatchedIndexes, isSelected)
    tags := tagStyle.Render("[" + strings.Join(w.Tags, " ") + "]")
    desc := dimStyle.Render(truncate(w.Description, 40))

    row := fmt.Sprintf("%-30s %s %s", name, desc, tags)
    if isSelected {
        return selectedStyle.Render(row)
    }
    return normalStyle.Render(row)
}
```

### bubbles/textinput: Parameter fill input
```go
// Source: Bubbles README (textinput section, verified)
// internal/picker/paramfill.go
func newParamInput(param store.Param) textinput.Model {
    ti := textinput.New()
    ti.Placeholder = param.Name
    if param.Default != "" {
        ti.SetValue(param.Default)   // Pre-fill default
    }
    ti.CharLimit = 256
    return ti
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| TIOCSTI ioctl | Shell function wrappers (LBUFFER/READLINE_LINE/commandline) | Linux 6.2 (2023) | TIOCSTI hard-blocked by default; wrapper approach is universal |
| `tea.Program.Start()` | `tea.NewProgram().Run()` | Bubble Tea v1.x | `.Start()` deprecated; `.Run()` returns (Model, error) |
| Writing TUI result to temp file | fd swap (3>&1 1>&2 2>&3) | Standard Unix pattern | Cleaner, no file cleanup, atomic |
| Full fuzzy library with stemming | sahilm/fuzzy (symbol matching) | — | Symbol-style fuzzy (VSCode scoring) beats Levenshtein for identifier matching |

**Deprecated/outdated:**
- `print -z` in zsh: Pre-2020 approach. Only works in the current shell process, not from a subprocess. Do not use.
- `TIOCSTI`: Blocked in Linux kernel 6.2+. Do not use.
- `tea.Program.Start()`: Deprecated in favor of `.Run()`. Do not use.

---

## Open Questions

1. **Ctrl+G keybinding conflict in bash**
   - What we know: Ctrl+G is readline's `abort` function by default in emacs mode. It's rarely used.
   - What's unclear: Will users notice? Should we use Ctrl+\ instead, or make it configurable?
   - Recommendation: Use Ctrl+G as default. Document override via `WF_KEYBINDING` env var in the shell scripts. Ctrl+\ would avoid ALL conflicts but is harder to type.

2. **Windows shell integration**
   - What we know: PowerShell and CMD have different keybinding mechanisms. The bash/zsh/fish scripts won't work on Windows.
   - What's unclear: What shell does the requirement "Windows without shell integration" mean? Just that `wf list` and `wf add` work standalone?
   - Recommendation: SHEL-05 says "binary works standalone" — interpret as: binary compiles and runs on Windows for standalone use; shell integration only targets zsh/bash/fish (all Unix). No Windows shell scripts needed for Phase 2.

3. **Existing text on prompt + picker invocation**
   - What we know: fzf appends, atuin replaces. CONTEXT.md doesn't specify.
   - What's unclear: Should `wf pick` preserve existing prompt content?
   - Recommendation: Replace (atuin approach). Users invoke `wf` to find a command, not to append to partial text. If they typed `git push origin ` and want to find a workflow, they'll discard the partial command.

4. **Multiline commands in preview pane**
   - What we know: CONTEXT.md marks this as Claude's discretion.
   - Recommendation: Use `strings.ReplaceAll(cmd, "\n", " ↵ ")` for the preview pane, or render with actual newlines in the viewport. Viewport handles multi-line content natively. Use actual newlines for readability. Limit preview to 6 lines tall.

---

## Sources

### Primary (HIGH confidence)
- `https://raw.githubusercontent.com/junegunn/fzf/master/shell/key-bindings.zsh` — fzf zsh integration (LBUFFER pattern, zle widget)
- `https://raw.githubusercontent.com/junegunn/fzf/master/shell/key-bindings.bash` — fzf bash integration (READLINE_LINE pattern)
- `https://raw.githubusercontent.com/atuinsh/atuin/main/crates/atuin/src/shell/atuin.zsh` — atuin zsh integration (LBUFFER=$output, fd swap)
- `https://raw.githubusercontent.com/atuinsh/atuin/main/crates/atuin/src/shell/atuin.bash` — atuin bash integration (READLINE_LINE=$output)
- `https://raw.githubusercontent.com/atuinsh/atuin/main/crates/atuin/src/shell/atuin.fish` — atuin fish integration (commandline -r $output)
- `https://raw.githubusercontent.com/charmbracelet/bubbletea/master/README.md` — Bubble Tea official README (Model/Update/View API, tea.WithAltScreen, tea.NewProgram().Run())
- `https://raw.githubusercontent.com/charmbracelet/bubbles/master/README.md` — Bubbles official README (textinput, viewport, list, key components)
- `https://raw.githubusercontent.com/sahilm/fuzzy/master/README.md` — sahilm/fuzzy README (FindFrom, Source interface, MatchedIndexes, benchmarks)
- `go.mod` — Current project dependencies (confirmed no bubbletea/bubbles/lipgloss yet; needs adding)

### Secondary (MEDIUM confidence)
- WebSearch verified: golang.design/x/clipboard v0.7.1 (June 2025) vs atotto/clipboard (last 2021) — atotto sufficient for text-only; golang.design adds CGo dependency
- WebSearch verified: TIOCSTI disabled in Linux 6.2+ (2023) — confirmed by multiple sources including kernel changelog references
- WebSearch verified: Bubble Tea altscreen best practice — `tea.WithAltScreen()` as ProgramOption, not Init command

### Tertiary (LOW confidence)
- WebSearch: Sub-100ms Go startup — general Go optimization advice, not Bubble Tea specific. LOW confidence but aligns with expected binary startup times.
- WebSearch: Ctrl+G as recommended keybinding — no authoritative source confirms this is "convention"; it's a reasoned recommendation. User should be able to configure.

---

## Metadata

**Confidence breakdown:**
- Shell paste mechanism: HIGH — verified directly from fzf and atuin source code (both zsh, bash, fish)
- Standard Stack (bubbletea/bubbles/lipgloss/sahilm): HIGH — verified from official READMEs
- Clipboard (atotto): MEDIUM — no Context7 query done; official GitHub README shows text-only, pure Go on macOS
- Architecture patterns (multi-state model): HIGH — standard Elm architecture, verified from Bubble Tea docs
- Shell keybinding recommendation (Ctrl+G): LOW — reasoned analysis, not authoritative
- Sub-100ms startup: MEDIUM — general Go performance patterns; actual measurement needed

**Research date:** 2026-02-21
**Valid until:** 2026-03-21 (30 days — Charm stack is stable; shell integration patterns are very stable)
