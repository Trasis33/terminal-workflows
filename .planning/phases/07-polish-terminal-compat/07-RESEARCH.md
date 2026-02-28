# Phase 7: Polish & Terminal Compat - Research

**Researched:** 2026-02-28
**Domain:** Shell syntax highlighting, TUI UX polish, terminal keybinding compatibility
**Confidence:** HIGH

## Summary

Phase 7 covers three distinct domains: (1) syntax highlighting shell commands using Chroma v2, (2) UX polish for the manage TUI (auto-filtering sidebar, scroll-bounded preview), and (3) Warp terminal compatibility with custom keybinding support.

The syntax highlighting work is well-supported by Chroma's `terminal256` formatter, which renders ANSI-colored output to a `strings.Builder` — ideal for Bubble Tea rendering. The shell commands stored in workflows use `bash`/`sh` syntax, so `lexers.Get("bash")` covers the core lexing. The key integration challenge is mapping Chroma's Pygments-based styles to wf's existing ANSI-256 theme colors, which requires building a custom `chroma.Style` per wf theme.

The UX polish is straightforward: the sidebar's `moveCursor()` already calculates position but only emits `sidebarFilterMsg` on Enter — adding auto-emit on cursor move is a small change. The preview pane currently has no scroll state (fixed 6-line height, simple truncation) — it needs a `viewport.Model` (already used in the picker) to support bounded scrolling with a position indicator.

The Warp compatibility requires converting the four shell scripts from `const` strings to `text/template` templates, adding `$TERM_PROGRAM` detection in `wf init`, and adding a `--key` flag with validation. The keybinding escape sequences differ per shell (`\C-x` for zsh/bash, `\cx` for fish, `Ctrl+x` for PowerShell).

**Primary recommendation:** Use Chroma v2 with `terminal256` formatter and custom style builder that maps wf themes to Chroma token colors. Use `text/template` for shell scripts. Use `viewport.Model` for scrollable preview.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- Full AST-level coloring: command names, flags, arguments, variable expansions, keywords (if/for/while), pipes, redirects, strings, comments — all distinctly colored
- Template parameters (e.g., `{{name}}`, `{{port}}`) rendered in a distinct accent color that clearly separates them from regular command text
- Color scheme follows the currently selected wf theme in settings (theme-aware, not hardcoded)
- Three surfaces get highlighting:
  1. **Manage preview pane** — full AST highlighting, theme-aware colors
  2. **Picker/browse preview** — full AST highlighting, theme-aware colors
  3. **`wf list` CLI output** — simpler ANSI-native highlighting that separates folder, workflow name, and tags (uses terminal-native colors, not wf theme)
- Selecting a folder in the sidebar **instantly filters** the workflow list — no extra keypress needed
- **Folder breadcrumb** shown at the top of the workflow list indicating the active filter
- **Explicit "All" entry** at the top of the sidebar folder list to clear the filter
- **Empty state message** when a selected folder has no workflows
- **Fixed-height preview pane** (not dynamic) — prevents layout destabilization
- Scrolling stops **tight to content** — no blank overscroll
- **Line count / percentage indicator** shown for scroll position
- **Auto-detect Warp** via `$TERM_PROGRAM` at `wf init` time
- **Default fallback key for Warp: Ctrl+O**
- **Stderr message** during `wf init` when Warp detected
- **Comment block** in generated shell init script explaining bound key
- **`--key` flag** accepts modifier+letter combos (ctrl+<letter>, alt+<letter>)
- **Block dangerous key combos:** ctrl+c, ctrl+d, ctrl+z, and others
- `--key` flag works for all terminals, not just Warp

### Claude's Discretion
- Exact syntax highlighting library choice (chroma, tree-sitter, or custom tokenizer)
- Specific color assignments per token type within the theme system
- Preview pane fixed height value (current 6 lines or adjusted)
- Exact format/placement of scroll percentage indicator
- Warp detection edge cases (e.g., SSH through Warp, nested terminals)
- Which specific dangerous key combos to block beyond ctrl+c/d/z

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DISP-01 | User sees syntax-highlighted shell commands in the manage preview pane | Chroma v2 `terminal256` formatter renders to `strings.Builder`, output usable in `renderPreview()`. Custom style maps wf theme colors to token types. |
| DISP-02 | User sees syntax-highlighted shell commands in the manage browse list | Same Chroma highlighting function used for browse list command display. Preview pane in picker also gets highlighting via shared `highlight` package. |
| MGUX-01 | User sees folder contents auto-displayed when navigating to a folder in manage | Sidebar `moveCursor()` emits `sidebarFilterMsg` on every move instead of only on Enter. "All" entry already exists at index 0. Breadcrumb rendered at top of list pane. |
| MGUX-02 | User can scroll the command preview panel without overscrolling past content | Replace raw string rendering in preview with `viewport.Model` (already used in picker). Set viewport content height to actual content lines. Viewport naturally stops at content boundary. |
| TERM-01 | User on Warp terminal is automatically offered a working keybinding | `$TERM_PROGRAM == "WarpTerminal"` detection in `wf init`. Shell scripts become `text/template` templates with `{{.Key}}` placeholder. Default Warp key: Ctrl+O. Stderr message informs user. |
| TERM-02 | User can configure a custom keybinding via `wf init --key` flag | `--key` flag on init command, parsed as `modifier+letter`. Validation blocks dangerous combos. Key converted to shell-specific escape format. |

</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/alecthomas/chroma/v2` | v2.21+ (latest) | Shell syntax highlighting | Pre-approved by user. Go-native, pure Go, no CGo. Supports `bash` lexer with AST-level token types. 256-color terminal formatter built in. Used by Hugo, Goldmark, and moor. |
| `github.com/charmbracelet/bubbles/viewport` | (bundled with bubbles v1.0.0) | Scrollable preview pane | Already used in picker's preview (`internal/picker/model.go:45`). Provides content-bounded scrolling out of the box. |
| `text/template` | stdlib | Shell script templating | Go stdlib. Shell scripts need `{{.Key}}` substitution for keybinding and `{{.Comment}}` for comment blocks. No external dep needed. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/spf13/pflag` | (bundled with cobra) | `--key` flag parsing | Already in use via Cobra. Add string flag to `initCmd`. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Chroma | tree-sitter (go-tree-sitter) | More accurate AST but requires CGo, adds ~10MB binary size, complex build. Chroma's bash lexer is sufficient for shell commands. |
| Chroma | Custom regex tokenizer | Simpler but fragile. Misses nested quoting, heredocs, variable expansion edge cases. Not worth hand-rolling. |
| text/template | fmt.Sprintf | Simpler for single substitution but shell scripts have multiple template points (key, comments, conditionals). Templates are clearer. |

**Installation:**
```bash
go get github.com/alecthomas/chroma/v2@latest
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── highlight/             # NEW: syntax highlighting package
│   ├── highlight.go       # HighlightShell(command, theme) → string
│   └── highlight_test.go  # Unit tests with known token outputs
├── shell/
│   ├── zsh.go             # MODIFY: const → template
│   ├── bash.go            # MODIFY: const → template  
│   ├── fish.go            # MODIFY: const → template
│   ├── powershell.go      # MODIFY: const → template
│   ├── keybinding.go      # NEW: key parsing, validation, escape conversion
│   └── keybinding_test.go # NEW: unit tests for key validation
├── manage/
│   ├── browse.go          # MODIFY: add viewport, syntax highlighting, breadcrumb
│   └── sidebar.go         # MODIFY: auto-filter on cursor move
├── picker/
│   └── model.go           # MODIFY: use highlight package for preview
└── ...
cmd/wf/
├── init_shell.go          # MODIFY: add --key flag, Warp detection
└── list.go                # MODIFY: add ANSI coloring
```

### Pattern 1: Shared Highlight Package
**What:** Centralized `internal/highlight` package that both manage and picker import
**When to use:** Whenever shell command text needs syntax coloring
**Example:**
```go
// internal/highlight/highlight.go
package highlight

import (
    "strings"

    "github.com/alecthomas/chroma/v2"
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
)

// Shell highlights a shell command string using Chroma and returns
// ANSI-colored text suitable for terminal rendering.
func Shell(command string, style *chroma.Style) string {
    lexer := lexers.Get("bash")
    if lexer == nil {
        return command // fallback: no highlighting
    }
    lexer = chroma.Coalesce(lexer)

    formatter := formatters.Get("terminal256")
    if formatter == nil {
        return command
    }

    iterator, err := lexer.Tokenise(nil, command)
    if err != nil {
        return command
    }

    var buf strings.Builder
    if err := formatter.Format(&buf, style, iterator); err != nil {
        return command
    }
    return buf.String()
}
```

### Pattern 2: Theme-to-Chroma Style Mapping
**What:** Convert wf's Theme colors to a Chroma style for consistent look
**When to use:** Before calling `highlight.Shell()`, build a Chroma style from active theme
**Example:**
```go
// internal/highlight/theme.go
package highlight

import (
    "github.com/alecthomas/chroma/v2"
)

// StyleFromTheme builds a Chroma style from wf theme colors.
// Maps wf theme ANSI colors to Chroma token types.
func StyleFromTheme(primary, secondary, tertiary, dim, text string) *chroma.Style {
    return chroma.MustNewStyle("wf-custom", chroma.StyleEntries{
        chroma.Keyword:            "#" + ansiToHex(primary),
        chroma.NameBuiltin:        "#" + ansiToHex(secondary),
        chroma.LiteralString:      "#" + ansiToHex(tertiary),
        chroma.Comment:            "#" + ansiToHex(dim),
        chroma.Operator:           "#" + ansiToHex(text),
        chroma.NameVariable:       "#" + ansiToHex(secondary),
        chroma.GenericInserted:    "#" + ansiToHex(primary), // template params
        // ... additional mappings
    })
}
```

**Important note on Chroma styles:** Chroma styles use hex color strings (e.g., `#ff0000`), not ANSI-256 codes. The `terminal256` formatter converts these back to the nearest ANSI-256 color. Since wf themes already use ANSI-256 codes, an `ansiToHex()` lookup table is needed. Alternatively, bypass Chroma's style system and manually apply lipgloss styles per token type by iterating tokens directly — this avoids the ANSI→hex→ANSI round-trip.

### Pattern 3: Direct Token Iteration (Recommended Alternative)
**What:** Skip Chroma's formatter, iterate tokens directly, apply lipgloss styles
**When to use:** When you want pixel-perfect control over colors using existing theme ANSI codes
**Why:** Avoids ANSI→hex→ANSI conversion. wf already has lipgloss styles per theme.
**Example:**
```go
func ShellWithLipgloss(command string, tokenStyles map[chroma.TokenType]lipgloss.Style) string {
    lexer := lexers.Get("bash")
    if lexer == nil {
        return command
    }
    lexer = chroma.Coalesce(lexer)

    iterator, err := lexer.Tokenise(nil, command)
    if err != nil {
        return command
    }

    var buf strings.Builder
    for _, token := range iterator.Tokens() {
        style, ok := tokenStyles[token.Type]
        if !ok {
            // Walk up token hierarchy: CommentSingle → Comment → Generic
            style = findParentStyle(token.Type, tokenStyles)
        }
        buf.WriteString(style.Render(token.Value))
    }
    return buf.String()
}
```
**This is the recommended approach.** It keeps color control within lipgloss (consistent with rest of TUI), avoids hex conversion, and lets each theme define token-type colors directly as ANSI-256 codes.

### Pattern 4: Shell Script Templating
**What:** Convert shell script constants to `text/template` with keybinding placeholder
**When to use:** All four shell scripts need the same pattern
**Example:**
```go
// internal/shell/zsh.go
package shell

import "text/template"

var ZshTemplate = template.Must(template.New("zsh").Parse(`# wf shell integration for zsh
# Usage: eval "$(wf init zsh)"
{{.Comment}}
_wf_picker() {
  local output
  output=$(wf pick)
  local ret=$?
  if [[ -n "$output" ]]; then
    LBUFFER="$output"
    RBUFFER=""
  fi
  zle reset-prompt
  return $ret
}
zle -N _wf_picker
bindkey -M emacs '{{.Key}}' _wf_picker
bindkey -M viins '{{.Key}}' _wf_picker
bindkey -M vicmd '{{.Key}}' _wf_picker
` + precmdBlock))

// Key format for zsh: \C-g for Ctrl+G, \eg for Alt+G
```

### Pattern 5: Keybinding Validation & Conversion
**What:** Parse `--key` value, validate safety, convert to shell-specific format
**When to use:** In `wf init` command handler
**Example:**
```go
// internal/shell/keybinding.go
package shell

import "fmt"

// Keybinding represents a parsed key combination.
type Keybinding struct {
    Modifier string // "ctrl" or "alt"
    Letter   string // single letter a-z
}

// ParseKey parses "ctrl+g" or "alt+f" into a Keybinding.
func ParseKey(input string) (Keybinding, error) { ... }

// Validate checks for dangerous combinations.
func (k Keybinding) Validate() error {
    dangerous := map[string]bool{
        "ctrl+c": true, "ctrl+d": true, "ctrl+z": true,
        "ctrl+s": true, // terminal flow control (XOFF)
        "ctrl+q": true, // terminal flow control (XON)
        "ctrl+\\": true, // SIGQUIT
    }
    key := k.Modifier + "+" + k.Letter
    if dangerous[key] {
        return fmt.Errorf("key %q conflicts with essential terminal function", key)
    }
    return nil
}

// ForZsh returns the zsh bindkey escape sequence.
func (k Keybinding) ForZsh() string {
    if k.Modifier == "ctrl" {
        return fmt.Sprintf("\\C-%s", k.Letter) // e.g., \C-g
    }
    return fmt.Sprintf("\\e%s", k.Letter) // Alt: \eg
}

// ForBash returns the bash bind escape sequence.
func (k Keybinding) ForBash() string {
    if k.Modifier == "ctrl" {
        return fmt.Sprintf("\\C-%s", k.Letter) // same as zsh
    }
    return fmt.Sprintf("\\e%s", k.Letter)
}

// ForFish returns the fish bind escape sequence.
func (k Keybinding) ForFish() string {
    if k.Modifier == "ctrl" {
        return fmt.Sprintf("\\c%s", k.Letter) // e.g., \co
    }
    return fmt.Sprintf("\\e%s", k.Letter)
}

// ForPowerShell returns the PowerShell key chord string.
func (k Keybinding) ForPowerShell() string {
    if k.Modifier == "ctrl" {
        return fmt.Sprintf("Ctrl+%s", strings.ToUpper(k.Letter)) // e.g., Ctrl+G
    }
    return fmt.Sprintf("Alt+%s", strings.ToUpper(k.Letter))
}
```

### Anti-Patterns to Avoid
- **Don't use Chroma's HTML formatter** for terminal output — it produces HTML tags, not ANSI. Use `terminal256` or direct token iteration.
- **Don't hardcode highlight colors** outside the theme system — all colors must flow from the active theme to stay consistent across theme switches.
- **Don't make preview height dynamic** — user explicitly rejected this because it pushes panels off-screen. Keep a fixed height constant.
- **Don't use `os.Getenv("TERM_PROGRAM")` at import time** — call it inside `wf init` handler so it's evaluated at shell setup time, not at binary build time.
- **Don't forget `chroma.Coalesce(lexer)`** — without coalescing, the lexer can produce extremely chatty token streams (single-character tokens), hurting performance and making styling inconsistent.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Shell syntax tokenization | Custom regex parser | Chroma v2 `lexers.Get("bash")` | Shell syntax has nested quoting, heredocs, arithmetic expansion, brace expansion. Chroma handles these correctly. A regex tokenizer will break on `echo "$(cat <<EOF\n...\nEOF)"`. |
| Scrollable viewport with content bounds | Manual scroll offset tracking | `bubbles/viewport.Model` | Already proven in picker. Handles resize, content-height clamping, and scroll indicators. Re-implementing means tracking `scrollOffset`, `contentHeight`, `viewportHeight`, clamp logic — all already done in viewport. |
| ANSI escape sequence generation | Manual `\033[...m` strings | lipgloss styles | lipgloss handles terminal capability detection, color downgrading (truecolor → 256 → 16), and reset sequences. Manual ANSI strings break on terminals with limited color support. |

**Key insight:** Shell syntax is deceptively complex. Even simple-looking commands like `cmd "hello $(echo world)"` involve nested tokenization (string → command substitution → nested string). Chroma's Pygments-derived bash lexer handles this; a custom tokenizer won't without significant effort.

## Common Pitfalls

### Pitfall 1: Chroma Token Hierarchy Mismatch
**What goes wrong:** Style entries defined for `chroma.Comment` don't apply to `chroma.CommentSingle` unless you handle the hierarchy.
**Why it happens:** Chroma has a hierarchical token type system. `CommentSingle` is a child of `Comment`. When using direct token iteration (Pattern 3), you must walk up the hierarchy to find a matching style.
**How to avoid:** Implement a `findParentStyle()` function that walks `token.Type.Parent()` until a match is found. Fall back to default text style.
**Warning signs:** Some token types render unstyled (plain white) while similar types are styled.

### Pitfall 2: Template Parameter Highlighting Conflict
**What goes wrong:** Chroma's bash lexer treats `{{name}}` as syntax errors or brace expansion, not as template parameters.
**Why it happens:** `{{...}}` is not bash syntax. The lexer produces error tokens or splits into multiple brace-related tokens.
**How to avoid:** Pre-process the command text: replace `{{param}}` with a sentinel (e.g., `__WF_PARAM_0__`), run through Chroma, then post-process to replace sentinels with accent-colored parameter names. Store a mapping of sentinel → original param name.
**Warning signs:** Template parameters show up as red error tokens or split across multiple styled spans.

### Pitfall 3: ANSI Escape Codes in Lipgloss Width Calculation
**What goes wrong:** `lipgloss.Width()` counts ANSI escape codes as visible characters, causing layout misalignment.
**Why it happens:** Chroma's `terminal256` formatter embeds raw ANSI escape sequences. Lipgloss's width calculation may not strip them all.
**How to avoid:** Use the direct token iteration approach (Pattern 3) where lipgloss generates all ANSI codes itself — it knows the width of its own styled strings. If using Chroma's formatter output, use `ansi.StringWidth()` from `charmbracelet/x/ansi` instead of `lipgloss.Width()`.
**Warning signs:** Preview pane content wraps unexpectedly or gets truncated at wrong positions.

### Pitfall 4: Preview Scroll State Reset on Cursor Move
**What goes wrong:** User scrolls down in preview, moves to next workflow, preview stays scrolled to a position that doesn't exist in new (shorter) content.
**Why it happens:** Viewport content changes but scroll position isn't reset.
**How to avoid:** Call `viewport.GotoTop()` whenever the selected workflow changes (in `updatePreview()`).
**Warning signs:** Preview shows blank content or partial last line after switching workflows.

### Pitfall 5: Warp SSH Passthrough
**What goes wrong:** `$TERM_PROGRAM` is "WarpTerminal" when SSH'd from Warp, but the remote shell doesn't have Warp's keybinding conflict.
**Why it happens:** Warp sets `TERM_PROGRAM` and it propagates via SSH env forwarding.
**How to avoid:** Only check `$TERM_PROGRAM` — the Ctrl+G conflict exists regardless of SSH because the terminal emulator (Warp) intercepts the key before it reaches the shell. So the detection is correct even over SSH.
**Warning signs:** None — this is actually correct behavior. The key is intercepted at the terminal level.

### Pitfall 6: Fish `\c` vs Zsh/Bash `\C-` Escape Format
**What goes wrong:** Using `\C-g` format in fish produces a literal backslash-C, not Ctrl+G.
**Why it happens:** Fish uses `\cg` (lowercase c, no dash) for control sequences. Zsh and Bash use `\C-g` (uppercase C, with dash).
**How to avoid:** Each shell's Keybinding method produces the correct format. Test with actual shell execution.
**Warning signs:** Keybinding doesn't work in fish but works in zsh/bash.

## Code Examples

### Chroma Token Iteration with Lipgloss
```go
// Verified from Chroma README and API docs
// Source: https://github.com/alecthomas/chroma

import (
    "strings"

    "github.com/alecthomas/chroma/v2"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/charmbracelet/lipgloss"
)

func highlightCommand(command string, styles map[chroma.TokenType]lipgloss.Style) string {
    lexer := lexers.Get("bash")
    if lexer == nil {
        return command
    }
    lexer = chroma.Coalesce(lexer)

    iterator, err := lexer.Tokenise(nil, command)
    if err != nil {
        return command
    }

    var buf strings.Builder
    for _, tok := range iterator.Tokens() {
        if s, ok := styles[tok.Type]; ok {
            buf.WriteString(s.Render(tok.Value))
        } else {
            // Walk up hierarchy
            parent := tok.Type.Parent()
            for parent != chroma.Token(0) {
                if s, ok := styles[parent]; ok {
                    buf.WriteString(s.Render(tok.Value))
                    break
                }
                parent = parent.Parent()
            }
            if parent == chroma.Token(0) {
                buf.WriteString(tok.Value) // unstyled fallback
            }
        }
    }
    return buf.String()
}
```

### Viewport-Based Preview Pane
```go
// Source: charmbracelet/bubbles/viewport (already in go.mod)
import "github.com/charmbracelet/bubbles/viewport"

// In BrowseModel struct, add:
type BrowseModel struct {
    // ... existing fields ...
    previewVP viewport.Model // replaces direct string rendering
}

// In updatePreview:
func (b *BrowseModel) updatePreview() {
    if len(b.filtered) > 0 && b.cursor < len(b.filtered) {
        wf := b.filtered[b.cursor]
        highlighted := highlightCommand(wf.Command, b.tokenStyles())
        b.previewVP.SetContent(highlighted)
        b.previewVP.GotoTop() // reset scroll on workflow change
    } else {
        b.previewVP.SetContent("")
    }
}
```

### Shell Script Template Rendering
```go
// In cmd/wf/init_shell.go
import (
    "bytes"
    "fmt"
    "os"
    "text/template"
    
    "github.com/fredriklanga/wf/internal/shell"
)

func renderInitScript(shellName string, key shell.Keybinding) (string, error) {
    var tmpl *template.Template
    var keyStr string
    
    switch shellName {
    case "zsh":
        tmpl = shell.ZshTemplate
        keyStr = key.ForZsh()
    case "bash":
        tmpl = shell.BashTemplate
        keyStr = key.ForBash()
    case "fish":
        tmpl = shell.FishTemplate
        keyStr = key.ForFish()
    case "powershell":
        tmpl = shell.PowerShellTemplate
        keyStr = key.ForPowerShell()
    }
    
    data := shell.TemplateData{
        Key:     keyStr,
        Comment: fmt.Sprintf("# Keybinding: %s+%s", key.Modifier, key.Letter),
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

### Warp Detection
```go
// In cmd/wf/init_shell.go
func detectWarp() bool {
    return os.Getenv("TERM_PROGRAM") == "WarpTerminal"
}
```

### Dangerous Key Validation
```go
// Comprehensive blocklist
var dangerousKeys = map[string]string{
    "ctrl+c":  "SIGINT (interrupt process)",
    "ctrl+d":  "EOF (close shell)",
    "ctrl+z":  "SIGTSTP (suspend process)",
    "ctrl+s":  "XOFF (freeze terminal output)",
    "ctrl+q":  "XON (resume terminal output)",
    "ctrl+\\":  "SIGQUIT (core dump)",
    "ctrl+[":  "Escape key equivalent",
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Chroma v1 (`github.com/alecthomas/chroma`) | Chroma v2 (`github.com/alecthomas/chroma/v2`) | 2023 | Import path changed, API modernized. Must use v2 import path. |
| Chroma styles in Go code | Chroma styles in XML | 2023 (v2) | Styles now defined as XML files. `chroma.MustNewStyle()` still works for programmatic styles. |
| Fish `\cg` only | Fish also supports `ctrl-g` (fish 4.0+) | 2024 | Human-readable keybinding syntax. But stick with `\cg` for backward compat with fish 3.x. |

**Deprecated/outdated:**
- Chroma v1 import path (`github.com/alecthomas/chroma`) — must use v2
- `lexer.Iterator(nil, []byte(code))` — v2 uses `lexer.Tokenise(nil, code)` with string input

## Open Questions

1. **Chroma bash lexer accuracy for template params**
   - What we know: `{{param}}` is not valid bash syntax, Chroma will tokenize it as error/brace tokens
   - What's unclear: Exact token types produced for `{{...}}` — need to test empirically
   - Recommendation: Implement sentinel replacement (pre-process `{{x}}` → `__WF_x__`, post-process back). Test with actual Chroma output to verify.

2. **Optimal preview pane height**
   - What we know: Current is 6 (4 content + 2 border). User wants scrollable with indicator.
   - What's unclear: Whether 6 is enough for a good scroll experience or should be 8-10
   - Recommendation: Start with 8 (6 content + 2 border) to show more context. Can adjust based on testing.

3. **Alt+key support across terminals**
   - What we know: `ctrl+letter` works reliably across all terminals. `alt+letter` may not work in some (macOS Terminal uses Alt for special characters by default).
   - What's unclear: How many users will try `alt+` combos and hit platform issues
   - Recommendation: Support alt+ in parsing but document that ctrl+ is recommended. Don't block alt+ combos, just warn in docs.

## Sources

### Primary (HIGH confidence)
- Chroma v2 README — API patterns (lexers, formatters, styles, Tokenise), supported formatters (terminal256, terminal, terminal16m), bash lexer availability
- Chroma v2 pkg.go.dev — `Tokenise()` signature, `TokenType.Parent()` hierarchy, `Coalesce()` usage
- charmbracelet/bubbles/viewport — already in `go.mod` via `bubbles v1.0.0`, used in `internal/picker/model.go:45`
- Existing codebase — all four shell scripts examined, sidebar/browse/theme code analyzed

### Secondary (MEDIUM confidence)
- Web search: Warp terminal `$TERM_PROGRAM` set to "WarpTerminal", Ctrl+G conflict with "Add Selection for Next Occurrence" built-in command
- Web search: Fish shell keybinding format `\cg` (confirmed by existing codebase fish.go)
- Web search: Dangerous terminal key combos (Ctrl+C/D/Z/S/Q/\\ well-documented TTY signals)

### Tertiary (LOW confidence)
- Warp SSH passthrough behavior — based on general understanding of terminal emulator key interception. The detection is likely correct (Warp intercepts keys before shell) but edge cases with tmux/screen inside Warp are untested.
- Chroma `terminal256` color accuracy — ANSI-256 color matching depends on terminal's actual color palette. Most modern terminals use standard xterm-256color palette.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Chroma v2 is well-documented, actively maintained, used by Hugo. API patterns verified from official README and pkg.go.dev.
- Architecture: HIGH — All patterns use libraries already in go.mod (viewport, lipgloss) or stdlib (text/template). Direct token iteration avoids color conversion issues.
- Pitfalls: HIGH — Template param conflict, token hierarchy, and shell escape format differences are well-understood from Pygments/Chroma documentation and shell man pages.

**Research date:** 2026-02-28
**Valid until:** 2026-03-28 (stable domain — syntax highlighting and shell keybindings don't change rapidly)
