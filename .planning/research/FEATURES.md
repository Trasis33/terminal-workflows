# Features Research: v1.1 Polish & Power

**Researched:** 2026-02-27
**Confidence:** HIGH (verified against Pet, Navi, Hoard, SnipKit GitHub READMEs; Charm/huh ecosystem; Chroma docs; Warp terminal compatibility reports; multiple search cross-references)

---

## 1. Parameter CRUD in TUI

### How other tools handle it

**No snippet manager has in-TUI parameter editing.** This is a universal gap:

| Tool | Parameter Editing | Approach |
|------|------------------|----------|
| **Pet** | Edit raw TOML file | `pet edit` opens `$EDITOR`. Parameters are inline in the command string `<param=default>`. No structured arg metadata. |
| **Navi** | Edit `.cheat` files in `$EDITOR` | Variables defined as `$ var: command` lines in cheatsheet files. No UI for editing variable definitions. |
| **Hoard** | CLI flags at save time | `#param` markers in command strings. Parameters are purely positional markers — no type, no default, no enum. |
| **SnipKit** | Config YAML files | Parameters defined in YAML with type info (enum, password, path). Edited by hand in config files. |
| **Warp Workflows** | Warp's GUI editor | Web-based editor for workflow YAML. Arg metadata (name, description, default) edited through Warp Drive web UI. Not in terminal. |

**Key insight:** Every tool either relies on `$EDITOR` for raw file editing or has no parameter editing at all. wf already has an `Args` struct in YAML with name/type/default/options/dynamic_cmd — building CRUD on top of this structured data is genuinely novel.

### Table stakes

| Feature | Why Expected | Complexity | Depends On |
|---------|-------------|------------|------------|
| View existing params in edit form | Users need to see what params exist | Low | Existing form infrastructure |
| Add new parameter (name + type) | Core CRUD | Medium | Dynamic form rebuilding |
| Remove parameter | Core CRUD | Medium | Dynamic form rebuilding |
| Change parameter default value | Most basic edit | Low | Existing huh input binding |
| Change parameter type (text→enum→dynamic) | Users experiment with param types | Medium | Form field replacement |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Inline enum option editing (add/remove options) | No tool does this in TUI | High | Needs dynamic list within a form field |
| Type-specific sub-forms (show options field only for enum, command field only for dynamic) | Contextual UI reduces confusion | Medium | Conditional field visibility in huh |
| Auto-detect params from command | Parse `{{...}}` and pre-populate arg list | Low | Already have `ExtractParams()` |
| Param reorder via up/down | Control parameter fill order | Medium | List manipulation in form |

### Complexity notes

**The hard part is dynamic form rebuilding in huh.** Research confirms: `huh` does NOT have `AddField()` / `RemoveField()` APIs. The pattern is:

1. Store field data in your own model (not in huh)
2. Rebuild the entire `huh.Form` when fields change
3. Re-initialize with `form.Init()`
4. Re-bind all `Value()` pointers carefully (slice reallocation invalidates pointers)

**Recommendation:** Build a dedicated `ParamEditorModel` as a separate Bubble Tea model (not a huh form). Use a simple list with keybindings for add/edit/remove. When editing a single param, open a small huh form for that param's properties. This avoids the dynamic-form-in-huh problem entirely.

**Estimated complexity: MEDIUM-HIGH.** The individual operations are simple, but the UX choreography (list ↔ edit transitions, keeping command string in sync with args list) requires careful state management.

---

## 2. List Picker Dynamic Variable Type

### How other tools handle it (especially Navi's pattern)

**Navi is the gold standard here.** Its `$ var: command --- fzf-options` pattern is the most mature approach:

```
# Select a Docker container
$ container: docker ps --format '{{.ID}}\t{{.Names}}\t{{.Image}}' --- --column 2 --header-lines 0

# Select a Kubernetes pod
$ pod: kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase --- --column 1
```

**Key Navi features:**
- Command output piped to fzf for interactive selection
- `--column N` extracts a specific column from multi-column output (user sees full line, but only column N is substituted)
- `--multi` allows selecting multiple items
- `--preview` shows additional context
- `--delimiter` controls column splitting
- `--header-lines` skips header rows

**Pet** has no list picker equivalent — dynamic params don't exist.

**SnipKit** has enum params (pre-defined list) but no shell-command-driven list selection.

**wf's current `{{param!command}}`** already runs the command and presents options — but it shows each line as a single opaque option. There's no column extraction or field mapping.

### Table stakes

| Feature | Why Expected | Complexity | Depends On |
|---------|-------------|------------|------------|
| Run shell command, show output lines as selectable list | Already exists via `ParamDynamic` | Done | - |
| User selects one line from the list | Already exists in picker paramfill | Done | - |
| Timeout + error handling for slow/failed commands | Already exists (5s timeout) | Done | - |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Column extraction (`--column N`) | Show full context, store specific field | Medium | Parse multi-column output, extract on selection |
| Delimiter specification | Handle tab-separated, colon-separated, etc. | Low | Split on configurable delimiter |
| Header line skipping | Don't show header row as selectable option | Low | Skip first N lines of output |
| Preview of full row when column-extracted | User sees what they're selecting | Medium | Show all columns, highlight extracted one |
| Template syntax for list picker | e.g., `{{param!command:column:delimiter}}` or separate YAML fields | Medium | Syntax design decision |

### Complexity notes

**Template syntax design is the key decision.** Two approaches:

**Option A: Extend bang syntax inline**
```
{{container!docker ps --format '{{.ID}} {{.Names}}':2}}
```
Problem: Nested `{{` breaks the regex parser. Escaping is ugly.

**Option B: Define in YAML args (recommended)**
```yaml
args:
  - name: container
    type: list
    dynamic_cmd: "docker ps --format '{{.ID}} {{.Names}}'"
    column: 2
    delimiter: " "
    header_lines: 1
```
The command string just uses `{{container}}` — the list picker behavior is defined in the arg metadata.

**Recommendation: Option B.** It keeps the template syntax clean, leverages the existing `Arg` struct, and avoids parser ambiguity. The `type: list` distinguishes from `type: dynamic` (which shows all options as-is without column extraction).

**Estimated complexity: MEDIUM.** The command execution and list display already exist. The new parts are: column parsing, a new `ParamList` type, and the YAML schema for the list picker metadata.

---

## 3. Execute Flow in Manage TUI

### How other tools handle it

| Tool | Execute from Browse? | How? |
|------|---------------------|------|
| **Pet** | `pet exec` is a separate CLI command | No browse view — `pet list` shows snippets, `pet exec` uses fzf for selection + execution |
| **Navi** | Navi IS the execution interface | Single-mode: browse cheats → fill params → execute. No separate "management" mode |
| **Hoard** | TUI has execute action | Select snippet in TUI → press Enter → Hoard fills params and executes or copies to clipboard |
| **SnipKit** | Yes, from main interface | Browse → select → fill params → execute in subshell |
| **nap** | No — view/edit only | Copy to clipboard, no execution flow |

**Key UX pattern from Hoard/SnipKit:** Browse view → press Enter (or dedicated key) → inline parameter fill → output action (execute or copy).

**IDE command palette analogy:** VS Code's command palette lets you both browse AND execute. The pattern is: list → select → action. Users expect this in any list-based tool.

### Table stakes

| Feature | Why Expected | Complexity | Depends On |
|---------|-------------|------------|------------|
| Press Enter/x on workflow in browse → enter param fill | "I can see it but can't run it" is frustrating | Medium | Reuse picker's paramfill logic |
| Same param fill UX as picker (live preview, enum cycling, dynamic loading) | Consistency between picker and manage | Medium | Shared paramfill model |
| Result goes to clipboard / paste-to-prompt | Same output path as picker | Low | Existing clipboard/paste code |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| "Try it" mode from edit form | Edit workflow, immediately test it without leaving manage | High | Requires form→paramfill→form navigation |
| Quick re-execute with last values | Remember last used values per workflow, one-key re-run | Medium | Requires value persistence (see section 5) |

### Complexity notes

**The main challenge is embedding the picker's paramfill flow into the manage TUI's model tree.** The manage TUI uses a state machine (`viewBrowse`, `viewForm`, `viewSettings`, `viewDialog`). Adding `viewExecute` as a new state that instantiates a paramfill model is architecturally clean.

**Key design decisions:**
1. After execution completes, return to browse (not quit manage TUI)
2. The paramfill model needs to be adapted — currently it calls `tea.Quit` on Enter. In manage context, it should send a message instead
3. Clipboard paste works, but paste-to-prompt requires quitting the TUI first (manage runs in alternate screen)

**Recommendation:** Support two modes: (a) copy to clipboard and return to browse, (b) quit manage and paste to prompt (like picker does). Let the user choose with different keys (Enter = clipboard + stay, Ctrl+Enter = paste-to-prompt + quit).

**Estimated complexity: MEDIUM.** The hard parts (paramfill UI, dynamic param execution, live rendering) are already built. This is primarily wiring and state management.

---

## 4. Syntax Highlighting for Command Snippets

### How TUI tools handle it

**The Go ecosystem has a clear solution:** `alecthomas/chroma`.

| Library | Purpose | How to Use |
|---------|---------|------------|
| **Chroma** | Syntax highlighting engine | Tokenize code → ANSI-colored string. Supports `bash`, `sh`, `zsh`, 250+ languages. Has terminal formatters (`terminal`, `terminal256`, `terminal16m`). |
| **Glamour** | Markdown rendering with highlighting | Renders fenced code blocks with syntax highlighting. Uses Chroma under the hood. Overkill for inline command highlighting. |
| **Lipgloss** | TUI styling | Does not do syntax highlighting, but provides the framing/borders around highlighted content. |

**How other TUI tools do it:**
- **nap** (Go, Charm ecosystem): Uses Chroma for syntax highlighting in its snippet preview pane
- **glow** (Go, Charm): Uses Glamour for markdown rendering which includes code block highlighting
- **lazygit** (Go): Uses Chroma for diff syntax highlighting

**Integration pattern for wf:**
```go
import (
    "github.com/alecthomas/chroma/v2"
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
)

func highlightCommand(cmd string) string {
    lexer := lexers.Get("bash")
    formatter := formatters.Get("terminal256")
    style := styles.Get("monokai") // or match TUI theme
    iterator, _ := lexer.Tokenise(nil, cmd)
    var buf bytes.Buffer
    formatter.Format(&buf, style, iterator)
    return buf.String()
}
```

### Table stakes

| Feature | Why Expected | Complexity | Depends On |
|---------|-------------|------------|------------|
| Highlight commands in preview pane | Command preview without highlighting looks like raw text dump | Low | Add Chroma dependency |
| Highlight in browse list (at least the command portion) | Aids visual scanning | Medium | Truncation + ANSI width calculation |
| Match highlighting theme to TUI theme | Jarring if colors clash | Low | Map TUI theme to Chroma style |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Highlight `{{param}}` placeholders distinctly from shell syntax | Show params as visually distinct from the command | Medium | Custom token overlay on Chroma output |
| Highlight in picker search results | Visual polish in the speed-critical path | Low | Same Chroma integration |
| Auto-detect language (bash vs python vs yaml) | Multi-language command support | Low | Chroma's `lexers.Analyse()` |

### Complexity notes

**This is straightforward.** Chroma is well-maintained, pure Go, and the integration is ~20 lines. The two gotchas:

1. **ANSI width calculation:** Lipgloss's `Width()` counts visible characters, not ANSI escape sequences. When truncating highlighted strings for the list view, you must use `ansi.StringWidth()` from `charmbracelet/x/ansi` (or `muesli/ansi`), not `len()`.

2. **Parameter placeholder highlighting:** After Chroma highlights the shell syntax, overlay `{{param}}` matches with a distinct style (e.g., bold cyan). This requires post-processing the Chroma output or using a two-pass approach: first find param positions, then highlight the rest with Chroma, then re-insert styled param tokens.

**Estimated complexity: LOW-MEDIUM.** Core highlighting is trivial. Param overlay and ANSI-aware truncation add some complexity.

---

## 5. Auto-Saving Variable Defaults from User Input

### How other tools handle it

**Almost no snippet manager does this.** It's a gap across the ecosystem:

| Tool | Remembers Values? | How? |
|------|-------------------|------|
| **Pet** | No | `<param=default>` is static in the TOML file. User input is ephemeral. |
| **Navi** | No | Variable values are never persisted. Each invocation starts fresh. Confirmed limitation in docs. |
| **Hoard** | No | `#param` markers have no default mechanism at all. |
| **SnipKit** | Partial | Pre-defined values in config, but no auto-save of user input. |
| **Warp** | No | Arg defaults are static in workflow YAML. |
| **Shell history** | Implicitly | fzf-history and atuin remember full commands, but not individual parameter values. |

**The closest analogue is browser autofill** — remember what the user typed last time for a given field. This is a genuinely differentiating feature.

### Table stakes

| Feature | Why Expected | Complexity | Depends On |
|---------|-------------|------------|------------|
| Save last-used value as new default in YAML | Users expect "it remembers what I typed" | Medium | Write-back to workflow YAML |
| Show saved default in param fill UI | Feedback that the value was remembered | Low | Already shows defaults |
| Opt-out per workflow or per param | Some params should NOT auto-save (passwords, one-time tokens) | Low | Flag in Arg struct |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| History of previous values (not just last) | Pick from recent values like browser autofill | High | Separate history store |
| Per-workflow value history with timestamp | "What did I use last Tuesday?" | High | Overkill for v1.1 |

### Complexity notes

**The key decision: where to persist defaults.**

**Option A: Update the workflow YAML file (recommended for v1.1)**
After successful param fill, write the entered values back as defaults in the YAML args section. Simple, transparent, works with git sharing. Downside: modifies the workflow file on every execution.

**Option B: Separate defaults file**
Store last-used values in a separate file (e.g., `~/.config/wf/.defaults/workflow-name.yaml`). Doesn't modify the workflow. Downside: another file to manage, doesn't sync via git (which may be a feature).

**Option C: Inline in command string**
Update `{{param}}` → `{{param:last_value}}` in the command string. Simple but modifies the command template, which feels wrong.

**Recommendation: Option A for v1.1, with Option B as a future enhancement.** Option A is simpler and users can see the defaults in their YAML files. Add a `no_autosave: true` flag to `Arg` for sensitive params.

**Estimated complexity: MEDIUM.** The save-back logic is simple. The tricky part is: (1) not saving if the user didn't change anything, (2) handling the case where the command template has inline defaults that differ from the Args section, (3) not modifying shared/remote workflow files.

---

## 6. Per-Field AI Generation

### How the ecosystem handles AI in snippets

No snippet manager offers per-field AI generation. The landscape:

| Tool | AI Feature | Granularity |
|------|-----------|-------------|
| **Hoard** | ChatGPT integration | Generate entire new command from prompt |
| **SnipKit** | SnipKit Assistant | Generate entire parameterized script |
| **Warp** | Warp AI | Generate command from natural language (whole command) |
| **wf v1.0** | Copilot generation + autofill | Generate entire workflow OR auto-fill metadata (name, description, arg types) |

**Per-field generation is novel.** The concept: focus cursor on a specific field in the edit form, press a key, and AI fills just that field based on context from other fields.

### Differentiators (likely unique)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Generate description from command | "Explain what this command does" — one of the most useful AI fills | Low | Single LLM call with command as context |
| Generate command from description | "Write a command that does X" | Medium | Already exists as whole-workflow generation |
| Suggest tags from command + description | Auto-categorize workflows | Low | Simple classification prompt |
| Suggest default value for a param | "What's a typical value for `port`?" | Medium | Needs param name + command context |
| Suggest enum options for a param | "What are common values for `environment`?" → `dev, staging, prod` | Medium | Classification + domain knowledge |
| Suggest dynamic command for a param | "How do I list Docker containers?" → `docker ps --format '{{.Names}}'` | High | Requires understanding the param's semantic role |

### Complexity notes

**The infrastructure already exists** (Copilot SDK integration, AI actions in manage TUI). Per-field generation requires:

1. A focused-field-aware trigger (know which field the cursor is on)
2. Context assembly (other field values + the command template)
3. Targeted prompt engineering (different prompt per field type)
4. Result injection into the specific field

**The biggest risk is latency.** AI generation takes 1-3 seconds. In a form context, this needs a loading indicator on the specific field without blocking other field navigation.

**Recommendation:** Start with the highest-value, lowest-complexity fields: description-from-command and tags-from-command. These are useful, fast to prompt, and low risk.

**Estimated complexity: MEDIUM.** Infrastructure exists. New work is: per-field prompt templates, loading state per field, field-level AI trigger in form context.

---

## 7. Terminal Compatibility (Warp Keybindings)

### Warp-specific issues

**Confirmed issues with Warp Terminal and TUI apps:**

1. **Ctrl+G interception:** Warp may intercept `Ctrl+G` before it reaches the shell/TUI. This is wf's default picker hotkey. Multiple GitHub issues confirm Warp intercepts certain Ctrl combinations.

2. **bindkeys not forwarded:** Warp has a long-standing issue (since 2021, still active in 2025/2026) where user-defined shell `bindkeys` are ignored. This means `bindkey '^G' wf-picker` may silently fail in Warp.

3. **Subshell Warpification:** Warp automatically "Warpifies" subshells, which can interfere with TUI rendering and input handling. When wf's picker or manage TUI runs in a Warp subshell, input may be eaten by Warp's own completion/AI system.

4. **Block-based input model:** Warp's block model treats terminal output differently from traditional terminals. Full-screen TUIs (alternate screen) generally work, but overlay-style UIs may have rendering issues.

### Fallback keybinding patterns

| Approach | How | Pros | Cons |
|----------|-----|------|------|
| **Multiple default bindings** | Register `Ctrl+G` AND `Ctrl+/` AND `Ctrl+Space` | At least one usually works | Overrides more shell functions |
| **User-configurable keybinding** | `wf init --key "ctrl+k"` | User chooses what works | Requires user action |
| **Warp-specific detection** | Check `$TERM_PROGRAM == "WarpTerminal"` and use alternative binding | Automatic, transparent | Fragile if Warp changes env vars |
| **Warp Workflows integration** | Register as a Warp Workflow instead of shell keybinding | Native Warp experience | Loses cross-terminal compatibility |
| **Escape sequence fallback** | Use less common sequences less likely to be intercepted | More compatible | Harder to discover |

### Table stakes

| Feature | Why Expected | Complexity | Depends On |
|---------|-------------|------------|------------|
| Detect Warp terminal at `wf init` time | Don't silently fail | Low | Check `$TERM_PROGRAM` |
| Offer alternative keybinding for Warp users | Working hotkey is essential | Low | Parameterize keybinding in init |
| Document Warp compatibility in `--help` | Users need to know | Low | - |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Auto-detect and auto-configure for Warp | Zero friction for Warp users | Medium | Test which bindings actually work |
| `wf init --terminal warp` preset | One command to configure | Low | Template with known-good Warp bindings |

### Complexity notes

**This is a configuration/documentation problem, not a code problem.** The actual fix is:

1. Add `$TERM_PROGRAM` detection in `wf init`
2. When Warp detected, suggest/use an alternative keybinding (e.g., `Ctrl+/` or `Ctrl+Space`)  
3. Add a `--key` flag to `wf init` for manual override
4. Document in README and `wf init --help`

**Estimated complexity: LOW.** Mostly shell script changes in the init templates.

---

## Anti-Features

Things to deliberately NOT build in v1.1, with reasoning:

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **In-TUI text editor for commands** | Users have strong editor preferences. Building a terminal editor is a massive scope creep | Keep the existing multi-line `huh.Text` for short edits. For complex commands, add "open in $EDITOR" keybinding (like Pet's `pet edit`) |
| **Command execution history/analytics** | Turns wf into a shell history manager (Atuin's territory). Scope creep, storage overhead | Stick to paste-to-prompt model. Shell history handles execution tracking. |
| **Real-time collaboration on workflows** | SaaS territory. Git sharing is sufficient for team use | Keep git-based sharing. It's simpler and more reliable. |
| **Param type: "file picker"** | Building a TUI file browser is a massive undertaking | Users can use `{{path!find . -type f -maxdepth 2}}` with dynamic params. Or the new list picker with column extraction. |
| **Param type: "date picker"** | Calendar widget in terminal is fragile and rarely needed | Users type dates as text. Could suggest format in placeholder. |
| **Workflow chaining/DAGs** | Becomes a CI/CD tool. Out of scope per PROJECT.md | Individual workflow execution only. Users can chain in their shell. |
| **Custom Chroma themes** | Diminishing returns. 3-4 built-in styles sufficient | Map existing TUI themes to Chroma styles. Don't add a Chroma theme picker. |
| **Full undo/redo in param editor** | Over-engineering for the scale of edits | Simple delete + re-add is sufficient. Esc to cancel. |
| **Multi-select in list picker** | Navi has this via `--multi`, but for wf's paste-to-prompt model, multi-select creates ambiguity (paste multiple commands?) | Support single-select only. Multi-select is for execution-oriented tools. |
| **Param value validation (regex, range)** | Over-engineering. Shell will report errors when the command runs | Keep it simple: text input with optional default. Let the command fail naturally. |

---

## Feature Dependencies

```
Parameter CRUD ─────┐
                     ├──→ List Picker Type (new "list" type needs CRUD to configure)
Template Parser ─────┘

Picker ParamFill ───→ Execute in Manage (reuse paramfill model)

Chroma Integration ──→ Syntax Highlighting (browse + preview + picker)

Copilot SDK ─────────→ Per-Field AI Generation (extend existing AI actions)

Shell Init Scripts ──→ Warp Keybinding Fix (modify init templates)

Auto-Save Defaults ──→ (independent, but benefits from Execute in Manage flow)
```

## MVP Recommendation for v1.1

**Phase 1 (highest impact, lowest risk):**
1. Execute flow in manage TUI — closes the biggest UX gap
2. Warp keybinding fix — removes a blocker for Warp users
3. Syntax highlighting — highest visual impact for lowest effort

**Phase 2 (medium complexity, high value):**
4. Auto-save defaults — unique differentiator, medium effort
5. Parameter CRUD — powerful but complex; separate ParamEditor model

**Phase 3 (highest complexity):**
6. List picker variable type — needs template/YAML schema work + new ParamList type
7. Per-field AI generation — nice-to-have, extend existing infrastructure

**Ordering rationale:**
- Execute flow before param CRUD (users need to test their edits)
- Warp fix early (blocker removal)
- Syntax highlighting early (low effort, high perceived quality)
- Auto-save after execute flow (execute provides the save trigger)
- List picker after param CRUD (list picker is a new param type that benefits from the CRUD UI)

---

## Sources

**Verified (HIGH confidence):**
- Pet GitHub README: https://github.com/knqyf263/pet — TOML snippet format, `<param=default>` syntax, `pet edit` command
- Navi GitHub README & docs: https://github.com/denisidoro/navi — `$ var: command --- fzf-options` syntax, `--column`, `--multi`, variable expansion
- Hoard GitHub: https://github.com/Hyde46/hoard — `#param` syntax, ChatGPT integration
- SnipKit GitHub: https://github.com/lemoony/snipkit — parameter types (enum, password, path), SnipKit Assistant
- Chroma GitHub: https://github.com/alecthomas/chroma — terminal formatters, lexer API, style system
- Charm huh GitHub: https://github.com/charmbracelet/huh — `OptionsFunc`, dynamic forms, no AddField/RemoveField API
- Warp docs: https://docs.warp.dev — subshell Warpification, keybinding configuration, known issues

**Cross-referenced (MEDIUM confidence):**
- Warp Terminal keybinding conflicts with TUI apps — multiple GitHub issues and community reports
- Bubble Tea dynamic form patterns — community examples and search results
- nap TUI snippet manager — uses Chroma for highlighting

**Assessed (LOW confidence):**
- Per-field AI generation patterns — no established precedent found; assessment based on architectural reasoning from existing wf Copilot integration
