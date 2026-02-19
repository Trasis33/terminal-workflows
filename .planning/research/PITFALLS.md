# Domain Pitfalls

**Domain:** Go TUI Terminal Workflow Manager (`wf`)
**Researched:** 2026-02-19
**Overall Confidence:** HIGH (multiple authoritative sources cross-referenced)

---

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

---

### Pitfall 1: Paste-to-Prompt via TIOCSTI is Dead

**What goes wrong:** Building a "paste command to user's shell prompt" feature using the `TIOCSTI` ioctl — the historically standard way to inject keystrokes into a terminal — silently fails on modern Linux kernels (6.2+, i.e., most distros from 2023 onward). The syscall is deprecated and disabled by default. OpenBSD killed it in 2017. Your core UX feature — select a workflow, paste it ready-to-edit into the prompt — doesn't work.

**Why it happens:** `TIOCSTI` was a security vulnerability (CVE-2023-28339) allowing privilege escalation via terminal input injection. The kernel team deprecated it, and `sysctl kernel.tiocsti_restrict = 1` is now default.

**Consequences:** The entire product value proposition breaks. Users select a workflow, the TUI exits... and nothing appears in their prompt. This is a showstopper.

**Prevention:**
- **Do NOT use `TIOCSTI`.** It is not a viable mechanism in 2025/2026.
- **Primary approach: Print to stdout, use shell function wrapper.** The `wf` binary prints the command to stdout. A shell function (`wf()`) wraps the binary and uses the output:
  ```bash
  # ~/.zshrc / ~/.bashrc
  wf() {
    local cmd
    cmd=$(command wf "$@")
    if [ -n "$cmd" ]; then
      print -z "$cmd"  # zsh: paste to prompt buffer
      # OR: READLINE_LINE="$cmd" READLINE_POINT=${#cmd}  # bash
    fi
  }
  ```
- **For fish:** `commandline -r "$cmd"` in a fish function.
- **Alternative: Clipboard + notification.** Copy to clipboard via OSC 52 escape sequence (widely supported in modern terminals), then tell the user to paste. Less elegant but universally works.
- **Alternative: `tmux send-keys`.** If inside tmux, send keys to the pane. Only works for tmux users.

**Detection:** Test on a recent Ubuntu/Fedora. If your paste mechanism doesn't work out-of-the-box, you've hit this.

**Phase relevance:** Phase 1 (Core). This must be designed correctly from day one. The shell integration architecture is the foundation.

**Confidence:** HIGH — kernel changelog, CVE record, multiple distro confirmations.

---

### Pitfall 2: Bubble Tea State Explosion in Multi-View TUI

**What goes wrong:** Starting with a single `Model` struct that holds all state for search view, detail view, execution view, settings, etc. The `Update()` function becomes a massive switch statement. Adding a new view requires touching every other view's logic. The `View()` function becomes a nested conditional nightmare. Testing becomes impossible.

**Why it happens:** Bubble Tea's Elm Architecture is elegant for simple apps but doesn't prescribe how to handle multi-view complexity. Developers start simple (as they should) but don't refactor to a component tree soon enough. By the time the codebase is painful, the refactor is expensive.

**Consequences:** 
- `Update()` function exceeds 300+ lines with interleaved concerns
- Impossible to test individual views in isolation
- Every new feature risks breaking existing views via shared state mutation
- Message routing becomes a source of subtle bugs

**Prevention:**
- **Tree of Models pattern from day one.** Each "screen" or "panel" is its own Model with its own `Update()` and `View()`. A root model delegates.
  ```go
  type AppModel struct {
      activeView  ViewType
      searchView  SearchModel
      detailView  DetailModel
      executeView ExecuteModel
  }
  
  func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
      switch m.activeView {
      case ViewSearch:
          updated, cmd := m.searchView.Update(msg)
          m.searchView = updated.(SearchModel)
          return m, cmd
      // ...
      }
  }
  ```
- **Define clear message types per view.** Don't reuse generic messages across views.
- **Use `tea.Batch` for combining commands**, never for combining messages.
- **Write `teatest` tests** for each sub-model independently.

**Detection:** If your main `Update()` function exceeds 100 lines, refactor immediately.

**Phase relevance:** Phase 1 (Core TUI scaffold). Establish the component tree pattern before building features.

**Confidence:** HIGH — Bubble Tea official examples, GitHub issues, community patterns.

---

### Pitfall 3: YAML "Norway Problem" Destroys Command Templates

**What goes wrong:** A user saves a workflow command like `redis-cli CONFIG SET appendonly no`. When stored in YAML, `no` is interpreted as boolean `false`. When read back, the command becomes `redis-cli CONFIG SET appendonly false` — silently corrupted. Similarly, `yes`, `on`, `off`, `y`, `n` are all YAML boolean landmines. Octal-looking strings like port `0755` become decimal `493`.

**Why it happens:** YAML 1.1 (which `go-yaml.v3` partially supports for backwards compatibility) treats many bare words as booleans or numbers. Even in YAML 1.2 mode, the behavior with unquoted values into `bool`-typed Go struct fields persists.

**Consequences:** Commands silently mutate. Users run corrupted commands. Trust in the tool is destroyed. This is particularly devastating for a workflow manager because **the entire point is faithfully storing and reproducing commands.**

**Prevention:**
- **Always store command strings in YAML using explicit quoting.** Use `yaml:",flow"` or custom marshalling to force double-quoted strings:
  ```go
  type Workflow struct {
      Command string `yaml:"command"`
  }
  ```
  With `go-yaml.v3`, string-typed fields decode correctly. But **marshalling** may produce unquoted output that other tools misparse.
- **Custom `MarshalYAML`** that forces double-quoting for command fields:
  ```go
  func (w Workflow) MarshalYAML() (interface{}, error) {
      // Use yaml.Node to force quoted scalar style
      node := &yaml.Node{
          Kind:  yaml.ScalarNode,
          Value: w.Command,
          Style: yaml.DoubleQuotedStyle,
      }
      return node, nil
  }
  ```
- **Never use `interface{}` for command fields.** Always use typed `string` fields so the decoder treats values as strings.
- **Add round-trip tests:** Marshal a command, unmarshal it, assert byte-for-byte equality. Test with: `yes`, `no`, `on`, `off`, `true`, `false`, `0755`, `1e10`, `null`, `~`, `NO` (Norway!).
- **Consider using `yaml.Node` API** for full control over serialization style.

**Detection:** Write a test that round-trips `command: "redis-cli CONFIG SET appendonly no"`. If it comes back as `false`, you've been bitten.

**Phase relevance:** Phase 1 (YAML storage). Must be correct from the first persisted workflow.

**Confidence:** HIGH — go-yaml.v3 docs, YAML spec, Norway Problem is well-documented.

---

### Pitfall 4: Shell Function Wrapper Varies Per Shell (zsh/bash/fish)

**What goes wrong:** Building the paste-to-prompt shell wrapper for one shell (usually zsh or bash) and assuming it works everywhere. Each shell has fundamentally different mechanisms:
- **zsh:** `print -z "command"` places text in the line editor buffer
- **bash:** `READLINE_LINE="command"; READLINE_POINT=${#command}` (only works in `bind -x` or readline context, not in a regular function without tricks)
- **fish:** `commandline -r "command"` via a fish function
- **POSIX sh:** No mechanism at all
- **Nushell, PowerShell, etc.:** Each completely different

**Why it happens:** There is no universal "paste to prompt" API across shells. Each shell's line editor is a separate implementation.

**Consequences:** Works on your machine (zsh), fails for half your users (bash), confuses fish users, and is impossible for exotic shells.

**Prevention:**
- **Provide shell-specific wrapper functions** that users source. Ship them in the binary (embed via `go:embed`) and provide an `wf init <shell>` command that prints the correct function:
  ```
  eval "$(wf init zsh)"   # in ~/.zshrc
  eval "$(wf init bash)"  # in ~/.bashrc
  wf init fish | source   # in ~/.config/fish/config.fish
  ```
- **For bash specifically:** The readline approach is tricky. A more reliable bash pattern uses `bind -x '"\C-x\C-w": wf_select'` with a helper function, or falls back to clipboard.
- **Always provide clipboard fallback.** If shell integration isn't set up, copy to clipboard and print instructions.
- **Detect the parent shell** at runtime via `$SHELL` or `/proc/$PPID/comm` to give helpful error messages.
- **Test in CI with all three major shells.** Docker containers with zsh, bash, and fish.

**Detection:** Run `wf` from bash on a fresh machine. If nothing appears in the prompt, you haven't handled this.

**Phase relevance:** Phase 1-2. Core shell wrapper for zsh in Phase 1, bash/fish in Phase 2.

**Confidence:** HIGH — shell documentation, fzf/zoxide/atuin all solve this same problem with similar approaches.

---

### Pitfall 5: Terminal State Corruption on Panic or Forced Exit

**What goes wrong:** Bubble Tea puts the terminal into raw mode (no echo, no line buffering, alternate screen buffer). If the program panics, is killed with SIGKILL, or crashes, the terminal is left in raw mode. The user's terminal becomes unusable — no echo, no cursor, can't type. They have to blindly type `reset` and hit enter.

**Why it happens:** Go's `recover()` in a deferred function can catch panics, but signal handling is imperfect. `SIGKILL` cannot be caught. Bubble Tea's cleanup runs on `tea.Quit` but not on unexpected termination paths.

**Consequences:** Users lose trust immediately. A single crash permanently associates your tool with "that thing that broke my terminal." Power users will avoid it.

**Prevention:**
- **Use `defer` to restore terminal state** at the outermost level:
  ```go
  func main() {
      // Capture original terminal state BEFORE tea.NewProgram
      oldState, _ := term.GetState(int(os.Stdin.Fd()))
      defer term.Restore(int(os.Stdin.Fd()), oldState)
      
      p := tea.NewProgram(model, tea.WithAltScreen())
      if _, err := p.Run(); err != nil {
          fmt.Fprintf(os.Stderr, "Error: %v\n", err)
          os.Exit(1)
      }
  }
  ```
- **Handle signals explicitly:** Catch `SIGINT`, `SIGTERM`, `SIGHUP` and ensure terminal restore runs.
- **Use `tea.WithAltScreen()` as a ProgramOption,** not `tea.EnterAltScreen` in Init(). The option approach has better cleanup behavior.
- **Add a recovery wrapper:**
  ```go
  defer func() {
      if r := recover(); r != nil {
          term.Restore(int(os.Stdin.Fd()), oldState)
          fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", r, debug.Stack())
          os.Exit(1)
      }
  }()
  ```
- **Test with `kill -9 <pid>`** to verify the terminal is recoverable.

**Detection:** Add a `panic("test")` to your Update function. Run the app. If you can't type after, you haven't handled this.

**Phase relevance:** Phase 1 (Core TUI setup). This must be in the initial `main()` scaffold.

**Confidence:** HIGH — Bubble Tea GitHub issues, terminal programming fundamentals.

---

## Moderate Pitfalls

Mistakes that cause delays or technical debt.

---

### Pitfall 6: Command Injection via Template Interpolation

**What goes wrong:** The workflow manager supports parameterized commands like `ssh {{.host}} -p {{.port}}`. A user provides `host` as `; rm -rf /` or `` `malicious` ``. If the tool naively interpolates and passes to `sh -c`, arbitrary code execution occurs.

**Why it happens:** Using `text/template` to build a command string, then passing it to `exec.Command("sh", "-c", renderedString)` is the natural but dangerous approach.

**Prevention:**
- **Never use `sh -c` with interpolated strings for execution.** Use `exec.Command` with argument slices:
  ```go
  // BAD: exec.Command("sh", "-c", fmt.Sprintf("ssh %s -p %s", host, port))
  // GOOD: exec.Command("ssh", host, "-p", port)
  ```
- **For paste-to-prompt (display-only):** The command is printed for the user to review before execution. This is inherently safer but still needs escaping for display. Use `shellescape.Quote()` from `github.com/alessio/shellescape`.
- **If you MUST support complex pipeline commands** (pipes, redirects), clearly separate "template resolution" from "command display." The resolved command is shown to the user who decides whether to execute.
- **Validate parameter values** against an allowlist of characters. Reject values containing shell metacharacters (`;`, `|`, `&`, `` ` ``, `$()`, etc.) unless the user explicitly opts in.

**Phase relevance:** Phase 2 (Parameterized workflows). Design the parameter interpolation system with security from the start.

**Confidence:** HIGH — OWASP command injection patterns, `os/exec` Go docs.

---

### Pitfall 7: XDG Config Path Surprise on macOS

**What goes wrong:** Using Go's `os.UserConfigDir()` returns `~/Library/Application Support` on macOS. CLI tool users on macOS universally expect `~/.config/wf/`. Your config file ends up in a deeply nested Finder-oriented path that CLI users never look in.

**Why it happens:** `os.UserConfigDir()` follows Apple's conventions, which are correct for GUI apps but wrong for CLI tools. It's "technically correct, but wrong in practice."

**Consequences:** Users can't find their config. They file issues. They create `~/.config/wf/` manually and wonder why it's not picked up. Other CLI tools (fzf, starship, lazygit) all use `~/.config/` on macOS, setting the user expectation.

**Prevention:**
- **Use `github.com/adrg/xdg`** which provides cross-platform XDG compliance:
  - Linux: `~/.config/wf/`
  - macOS: `~/.config/wf/` (XDG-aware, not Apple convention)
  - Windows: `%APPDATA%/wf/`
- **OR manually implement:** Check `$XDG_CONFIG_HOME` first, fall back to `~/.config/` on Unix-like systems, `%APPDATA%` on Windows.
- **Never hardcode paths.** Always resolve at runtime.
- **Create the directory if it doesn't exist** with `os.MkdirAll(configDir, 0755)`.
- **Document the config location** in `--help` output and `wf config path` subcommand.

**Phase relevance:** Phase 1 (Config storage). Decide this before writing any config-related code.

**Confidence:** HIGH — `adrg/xdg` docs, fzf/lazygit/starship conventions, os.UserConfigDir() Go docs.

---

### Pitfall 8: Bubble Tea Alternate Screen + Hybrid Mode Confusion

**What goes wrong:** The project wants a "hybrid interface" — sometimes inline (like fzf's `--height` mode where results appear below the prompt), sometimes fullscreen. Mixing `tea.WithAltScreen()` and inline mode in the same program causes rendering chaos: leftover content from alt screen bleeding into inline mode, scroll position confusion, and visual artifacts on rapid transitions.

**Why it happens:** Alternate screen and inline modes use fundamentally different terminal buffer mechanisms. The alternate screen is a separate buffer; inline mode renders in the main buffer. Transitioning between them mid-program is fragile across terminal emulators, especially macOS Terminal.app which fakes alt screen with blank lines.

**Consequences:** Visual glitches, leftover artifacts, inconsistent behavior across terminals. Users on iTerm2 see different behavior than users on Alacritty or Terminal.app.

**Prevention:**
- **Pick one mode per invocation.** Don't dynamically switch between alt screen and inline. Instead:
  - `wf` (no args / interactive search) → inline mode
  - `wf edit <name>` → could use alt screen if needed for multi-field editing
- **For inline/height mode:** Use `tea.WithHeight(n)` or render without `tea.WithAltScreen()`. The program runs inline in the terminal.
- **Avoid `tea.EnterAltScreen` / `tea.ExitAltScreen` as runtime commands.** Prefer `tea.WithAltScreen()` as a program option set once at startup.
- **Test on at least:** iTerm2, Terminal.app, Alacritty, and one Linux terminal (GNOME Terminal or Kitty).
- **Accept imperfection:** Terminal rendering will never be pixel-perfect across all emulators. Design for graceful degradation.

**Phase relevance:** Phase 1 (TUI mode decision). This architectural decision must be made upfront.

**Confidence:** HIGH — Bubble Tea GitHub issues on alt screen, terminal emulator documentation.

---

### Pitfall 9: Slow `Update()` / `View()` Freezes the Entire UI

**What goes wrong:** Performing file I/O, YAML parsing, fuzzy search computation, or API calls inside `Update()` or `View()`. Since Bubble Tea runs a single-threaded event loop, any blocking operation in these functions freezes the entire UI. The cursor stops blinking, keypresses queue up, the app appears hung.

**Why it happens:** Coming from web/mobile development where async is natural, developers forget that Bubble Tea's event loop is synchronous. The temptation to "just do a quick file read" in Update is strong.

**Consequences:** UI freezes for 50-500ms per operation. Users think the app crashed. Accumulated keypresses replay chaotically when the blocking operation completes.

**Prevention:**
- **All I/O goes through `tea.Cmd`:**
  ```go
  func loadWorkflows() tea.Msg {
      data, err := os.ReadFile(configPath)
      if err != nil {
          return errMsg{err}
      }
      var workflows []Workflow
      yaml.Unmarshal(data, &workflows)
      return workflowsLoadedMsg{workflows}
  }
  
  // In Update:
  case startLoadMsg:
      return m, loadWorkflows  // Returns a Cmd, doesn't block
  ```
- **Fuzzy search: Pre-compute on data load, debounce on keystroke.** Don't re-score all items on every keypress. Use a 50ms debounce timer.
- **For AI integration:** Always use `tea.Cmd` with HTTP client timeout. Show a spinner while waiting.
- **Profile with `pprof`** if the UI feels sluggish. Look for anything >16ms in Update/View.

**Phase relevance:** Phase 1 (Core architecture). Establish the Cmd pattern from the very first I/O operation.

**Confidence:** HIGH — Bubble Tea docs explicitly warn about this.

---

### Pitfall 10: Argument Parsing with Nested Quotes and Pipes in Stored Commands

**What goes wrong:** A user saves a command like:
```
docker exec -it mydb psql -c "SELECT * FROM users WHERE name = 'O''Brien'"
```
or:
```
cat file.txt | grep "pattern" | awk '{print $2}' > output.txt
```

When storing these in YAML, reading them back, and either displaying or executing them, the quoting gets mangled. Single quotes inside double quotes, escaped quotes, pipes, redirects, backticks, `$()` substitutions — each is a parsing minefield.

**Why it happens:** There are multiple levels of escaping: YAML encoding, Go string representation, shell interpretation. Each layer has its own escaping rules. A command passes through all three.

**Consequences:** Commands that worked when the user typed them don't work when retrieved from `wf`. Users lose trust and stop saving complex commands — exactly the ones they most need to save.

**Prevention:**
- **Store commands as opaque strings.** Don't try to parse, tokenize, or understand the command structure. It's a string in, string out.
- **Use YAML double-quoted style with proper escaping:**
  ```yaml
  command: "docker exec -it mydb psql -c \"SELECT * FROM users WHERE name = 'O''Brien'\""
  ```
- **Use `yaml.Node` with `DoubleQuotedStyle`** for marshalling to avoid ambiguity.
- **For the paste-to-prompt path:** Print the raw string exactly. Don't re-escape. The shell function wrapper handles it.
- **For direct execution:** If the tool offers "run now," use `exec.Command("sh", "-c", rawCommand)` — the user already wrote valid shell. Don't try to parse it into `exec.Command` arg slices.
- **Comprehensive round-trip test suite** with these cases:
  - Nested quotes: `echo "it's a \"test\""`
  - Pipes: `cat foo | grep bar | wc -l`
  - Redirects: `cmd > file 2>&1`
  - Subshells: `echo $(date)`
  - Backticks: `` echo `whoami` ``
  - Single quotes with special chars: `echo '$HOME is not expanded'`
  - Heredocs: `cat <<EOF ...`

**Phase relevance:** Phase 1 (YAML storage) + Phase 2 (Parameterized workflows).

**Confidence:** HIGH — shell escaping is a well-documented minefield, Go `os/exec` docs.

---

### Pitfall 11: Fuzzy Search Gets Slow with Thousands of Workflows

**What goes wrong:** For small workflow libraries (10-100 items), any fuzzy search algorithm is fast enough. But power users accumulate 500-2000+ workflows. Naive fuzzy matching (scoring every item on every keystroke) becomes visibly laggy, especially if the scoring algorithm is O(n*m) per item (where m = query length, n = candidate length).

**Why it happens:** Fuzzy search is inherently more expensive than exact matching. Without debouncing, every keypress triggers a full re-score. Without indexing or pruning, all items are scored even when most are obviously non-matching.

**Prevention:**
- **Use a proven fuzzy library.** `github.com/sahilm/fuzzy` is Go-native and performs well for in-memory collections. For fzf-like behavior, `github.com/junegunn/fzf/src/algo` can be used as a library (check license).
- **Debounce search input.** Don't search on every keypress. Wait 30-50ms after the last keystroke before triggering search:
  ```go
  case key.Matches(msg, searchKeys):
      m.query += msg.String()
      return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
          return doSearchMsg{query: m.query}
      })
  ```
- **Score incrementally.** If the user adds a character to the query, only re-score items that matched the previous prefix (prune the candidate set).
- **Pre-build an index** on app load. Tag-based filtering before fuzzy matching reduces the candidate set.
- **Benchmark with 2000 items.** If search latency exceeds 16ms (one frame), optimize.

**Phase relevance:** Phase 2-3. Works fine in Phase 1 with <100 items. Needs optimization by Phase 3 if used heavily.

**Confidence:** MEDIUM — performance characteristics depend on implementation. Benchmarking needed.

---

### Pitfall 12: GitHub Copilot SDK is Technical Preview — Not Production-Ready

**What goes wrong:** Planning the AI workflow generation feature around the GitHub Copilot CLI SDK (`github.com/github/copilot-cli-sdk-go`), which entered technical preview in January 2026. Building core features dependent on it, then discovering breaking API changes, missing features, or access restrictions.

**Why it happens:** Technical preview means: API may change, features may be removed, production use is not recommended. The SDK requires an active Copilot subscription, limiting the potential user base.

**Consequences:** AI features break on SDK updates. Users without Copilot subscriptions can't use the feature. The feature becomes a maintenance burden if the SDK is abandoned.

**Prevention:**
- **Design AI features behind a clean interface.** Abstract the AI provider:
  ```go
  type WorkflowGenerator interface {
      Generate(description string) ([]Workflow, error)
  }
  ```
- **Copilot SDK is ONE implementation** of this interface. Others: direct OpenAI API, Ollama (local), Anthropic, or no AI at all.
- **Make AI features strictly optional.** The tool must be 100% useful without AI. AI is a differentiator, not table stakes.
- **Pin SDK version** and don't auto-update. Test each version bump.
- **Implement graceful degradation:** If the SDK fails, show a helpful message, not a crash.
- **Ship AI features in a later phase** (Phase 3+), giving the SDK time to stabilize.

**Phase relevance:** Phase 3+ (AI features). Do NOT let this block Phase 1-2.

**Confidence:** MEDIUM — SDK just entered preview; stability unknown. Plan for it to be unstable.

---

## Minor Pitfalls

Mistakes that cause annoyance but are fixable.

---

### Pitfall 13: Value vs Pointer Receivers in Bubble Tea Models

**What goes wrong:** Using pointer receivers (`*Model`) for Bubble Tea's `Update()` method. This seems natural in Go but violates Bubble Tea's functional paradigm. The framework expects `Update` to return a new model value, not mutate in place. Using pointers can cause subtle bugs where the framework's internal state tracking diverges from the actual model state.

**Prevention:**
- **Always use value receivers** for `Init()`, `Update()`, and `View()`:
  ```go
  func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
  // NOT: func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
  ```
- Exception: If your model contains a large data structure (like a 2000-item workflow slice), store it behind a pointer in the model struct to avoid copying:
  ```go
  type Model struct {
      workflows *[]Workflow  // pointer to avoid copies
      // ... small fields use values
  }
  ```

**Phase relevance:** Phase 1. Get this right in the initial model definition.

**Confidence:** HIGH — Bubble Tea docs and examples consistently use value receivers.

---

### Pitfall 14: YAML Tabs vs Spaces Causes Cryptic Parse Errors

**What goes wrong:** Users hand-edit the YAML workflow file and use tabs for indentation. YAML strictly forbids tabs for indentation. The error message from `go-yaml.v3` is `"found character that cannot start any token"` — cryptic and unhelpful to users.

**Prevention:**
- **Validate YAML on load** and provide human-readable error messages:
  ```go
  if err != nil {
      if strings.Contains(err.Error(), "cannot start any token") {
          return fmt.Errorf("YAML parse error (likely tabs in indentation): %w\nHint: YAML requires spaces, not tabs, for indentation", err)
      }
  }
  ```
- **When writing YAML programmatically,** always use spaces (go-yaml does this by default).
- **Provide a `wf validate` command** that checks the workflow file for common issues.
- **Consider a `wf edit <name>` command** that opens a structured editor (TUI form) instead of raw YAML, avoiding the problem entirely.

**Phase relevance:** Phase 2 (when users start hand-editing files).

**Confidence:** HIGH — YAML spec, go-yaml behavior.

---

### Pitfall 15: Terminal Width Assumptions Break on Small/Large Terminals

**What goes wrong:** Hardcoding layout widths (e.g., "search results take 60 columns") or assuming a minimum terminal size. Users with narrow terminals (tmux splits, small monitors) see truncated or wrapped content. Users with ultra-wide terminals see awkward empty space.

**Prevention:**
- **Always use `tea.WindowSizeMsg`** to get actual terminal dimensions on startup and resize.
- **Design layouts with relative widths** using lipgloss:
  ```go
  func (m Model) View() string {
      width := m.width  // from WindowSizeMsg
      searchBox := lipgloss.NewStyle().Width(width - 4)
      // ...
  }
  ```
- **Set minimum usable width** (e.g., 40 columns) and show a "terminal too narrow" message below it.
- **Test at 80x24** (classic terminal), **120x40** (modern default), and **40x15** (tmux split).

**Phase relevance:** Phase 1 (layout system). Build responsive from the start.

**Confidence:** HIGH — standard TUI development practice.

---

### Pitfall 16: `go-yaml.v3` Silently Ignores Unknown Fields

**What goes wrong:** A user misspells a field in their workflow YAML (`commnad` instead of `command`). `go-yaml.v3` silently ignores unknown fields by default. The workflow loads with an empty command field. No error, no warning. The user doesn't discover the issue until they try to run the workflow.

**Prevention:**
- **Use `yaml.Decoder` with `KnownFields(true)`:**
  ```go
  decoder := yaml.NewDecoder(reader)
  decoder.KnownFields(true)
  err := decoder.Decode(&config)
  // Now returns error on unknown fields
  ```
- **Validate after unmarshalling:** Check required fields are non-empty.
- **Provide helpful error messages** that suggest the correct field name (Levenshtein distance).

**Phase relevance:** Phase 1 (YAML parsing). Enable strict parsing from the start.

**Confidence:** HIGH — go-yaml.v3 documentation.

---

### Pitfall 17: Bracketed Paste Mode Interference with Shell Wrappers

**What goes wrong:** The shell wrapper uses `print -z` (zsh) to paste the command to the prompt buffer. But if the user has Oh My Zsh with `bracketed-paste-magic` enabled, the pasted content may be double-pasted, delayed, or truncated. Fish may freeze on certain paste lengths over SSH via Windows Terminal.

**Prevention:**
- **Test shell wrappers with common shell frameworks:** Oh My Zsh, Prezto, Starship, fish with default config.
- **For zsh:** `print -z` bypasses bracketed paste entirely (it writes directly to the ZLE buffer, not via terminal paste). This is actually the correct approach.
- **For bash:** Using `READLINE_LINE` also bypasses bracketed paste.
- **Document known incompatibilities** if any are discovered during testing.

**Phase relevance:** Phase 2 (shell compatibility testing).

**Confidence:** MEDIUM — behavior varies across shell frameworks and terminal emulators. Needs empirical testing.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|---|---|---|
| **Phase 1: Core TUI scaffold** | Terminal state corruption on panic (#5) | `defer term.Restore()` in main() from day one |
| **Phase 1: Shell integration** | TIOCSTI is dead (#1), shell-specific wrappers (#4) | Shell function approach with `wf init <shell>` |
| **Phase 1: YAML storage** | Norway problem (#3), unknown fields (#16), nested quotes (#10) | Typed string fields, `DoubleQuotedStyle`, `KnownFields(true)`, round-trip tests |
| **Phase 1: TUI mode** | Alt screen vs inline confusion (#8) | Pick one mode per invocation, don't dynamically switch |
| **Phase 1: State management** | State explosion (#2) | Tree of models pattern from initial scaffold |
| **Phase 1: Event loop** | Blocking Update/View (#9) | All I/O via tea.Cmd from first implementation |
| **Phase 1: Config paths** | macOS XDG surprise (#7) | Use `adrg/xdg`, never `os.UserConfigDir()` |
| **Phase 2: Parameterized workflows** | Command injection (#6), quote mangling (#10) | Treat commands as opaque strings, escape display values |
| **Phase 2: Shell compat** | Bash/fish wrapper differences (#4), bracketed paste (#17) | Test all three shells in CI |
| **Phase 2: User editing** | Tabs in YAML (#14) | Validate + helpful error messages |
| **Phase 3: Search at scale** | Fuzzy search perf (#11) | Debounce + proven library + benchmark with 2000 items |
| **Phase 3: AI features** | Copilot SDK instability (#12) | Interface abstraction, optional feature, pin version |
| **All phases: TUI correctness** | Value vs pointer receivers (#13), terminal width (#15) | Value receivers, responsive layouts from start |

---

## Sources

### HIGH Confidence (Official docs, kernel changelog, library docs)
- Linux kernel TIOCSTI deprecation: kernel.org changelog for Linux 6.2, CVE-2023-28339
- Bubble Tea architecture: github.com/charmbracelet/bubbletea (official README, examples, issues)
- go-yaml.v3 docs: pkg.go.dev/gopkg.in/yaml.v3
- YAML 1.1/1.2 boolean handling: yaml.org/type/bool.html
- Go `os/exec` security: pkg.go.dev/os/exec
- `adrg/xdg` library: github.com/adrg/xdg
- Go `os.UserConfigDir()`: pkg.go.dev/os#UserConfigDir
- GitHub Copilot CLI SDK: github.com/github/copilot-cli-sdk-go (technical preview, January 2026)

### MEDIUM Confidence (Multiple community sources agreeing)
- Shell function wrapper patterns: fzf, zoxide, atuin implementations
- Fuzzy search performance: sahilm/fuzzy benchmarks, fzf algorithm documentation
- Oh My Zsh bracketed paste issues: github.com/ohmyzsh/ohmyzsh issues
- Terminal emulator differences: various terminal documentation

### LOW Confidence (Needs validation)
- Copilot SDK stability trajectory — too new to assess
- Fish shell freeze on specific paste lengths — reported but not widely confirmed
