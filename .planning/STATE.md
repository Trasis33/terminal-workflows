# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Users can find and execute any saved command workflow in under 3 seconds
**Current focus:** Phase 6 Distribution & Sharing — PowerShell + Windows support delivered

## Current Position

Phase: 6 of 6 (Distribution & Sharing)
Plan: 4 of 4 in current phase (plan 3 pending)
Status: In progress
Last activity: 2026-02-25 — Completed 06-04-PLAN.md (PowerShell + Windows build tags)

Progress: [███████████████████████████░] 96% (27/28 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 27
- Average duration: ~3.4 minutes
- Total execution time: ~1.5 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 4/4 | ~20 min | ~5 min |
| 2. Quick Picker | 4/4 | ~25 min | ~6 min |
| 3. Management TUI | 5/5 | ~43 min | ~9 min |
| 4. Advanced Params | 8/8 | ~23 min | ~3 min |
| 5. AI Integration | 3/3 | ~13 min | ~4 min |
| 6. Distribution | 3/4 | ~5 min | ~2 min |

**Recent Trend:**
- Last 5 plans: 05-03, 06-01, 06-02, 06-04
- Trend: Fast execution cycles (~1-2 min)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [01-01-D1] Used goccy/go-yaml instead of gopkg.in/yaml.v3 (unmaintained since April 2025)
- [01-01-D2] Used adrg/xdg instead of os.UserConfigDir() (returns wrong path on macOS)
- [01-01-D3] Typed string fields eliminate Norway Problem without custom MarshalYAML
- [01-01-D4] Store interface pattern for testable CRUD operations
- [01-01-D5] Slug-based filenames for workflow YAML files
- [01-02-D1] Split on first colon only for name:default parsing (allows URL defaults)
- [01-02-D2] Return nil slice for no params (idiomatic Go)
- [01-02-D3] Last default wins for duplicate parameter names
- [01-03-D1] Interactive prompts gated on missing required fields (no prompts when all flags provided)
- [01-03-D2] Exported WorkflowPath on YAMLStore for $EDITOR integration
- [01-03-D3] Re-extract template params when command field is updated via wf edit
- [01-04-D1] Integration tests bypass global state via newTestRoot() with temp-dir YAMLStore
- [01-04-D2] Compact wf list format: name, description, [tags] — one line, greppable
- [02-01-D1] Deferred Charm stack + clipboard deps to plans that import them (Go removes unused deps on tidy)
- [02-02-D1] Mint/cyan-green color palette (Color 49, 158, 73) — avoids purple-gradient cliche
- [02-02-D2] Single commit for both tasks — Go compilation requires all package files simultaneously
- [02-02-D3] Esc quits picker entirely (not back-to-search) — consistent across both states
- [02-02-D4] Enter on non-last param advances to next — reduces friction
- [02-03-D1] Ctrl+G as default keybinding — overrides readline's rarely-used "abort" in bash, safe in zsh/fish
- [02-03-D2] Replace existing prompt text (atuin approach) — wf outputs complete commands, not fragments
- [02-04-D1] Use /dev/tty for TUI output instead of fd swap — ZLE redirects fds before subprocess starts
- [02-04-D2] Ctrl+Y for in-picker clipboard copy with flash message
- [02-04-D3] Fallback to os.Stderr if /dev/tty unavailable (Windows compatibility)
- [03-01-D1] themeStyles struct kept in styles.go for separation; Theme and computeStyles() in theme.go
- [03-01-D2] Independent preset themes (not derived from huh) — management TUI theme struct has different fields
- [03-01-D3] Dialog overlay uses lipgloss.Place with whitespace chars for visual depth
- [03-02-D1] Dual-mode sidebar (folders/tags) via ToggleMode() with virtual root items
- [03-02-D2] Reuse picker.ParseQuery + picker.Search for fuzzy filtering in browse view
- [03-02-D3] browsetruncate local helper to avoid collision with unexported truncateStr in picker
- [03-02-D4] Folders derived implicitly from workflow name path prefixes (no explicit folder model)
- [03-03-D1] Esc handled in FormModel.Update before huh delegation — huh only uses ctrl+c for abort by default
- [03-03-D2] Folder extracted from workflow name via LastIndex('/') in edit mode — folder field separate from name
- [03-03-D3] huh.ThemeCharm() as form theme — independent of management TUI's custom theme system
- [03-03-D4] Single atomic commit due to parallel agent conflict — all form files committed together
- [03-04-D1] Single commit per task despite interleaved model.go changes — Go compilation requires all package files
- [03-04-D2] DialogModel returns dialogResultMsg via tea.Cmd — clean separation of UI and side effects
- [03-04-D3] Settings view uses itself as live preview — editing a color immediately changes the view
- [03-04-D4] Move dialog includes (root) as first option — users need to remove folder prefixes
- [03-05-D1] Shared *formValues struct for huh pointer stability across bubbletea value-copy cycles
- [03-05-D2] Settings keybinding changed from ctrl+t to S (shift-s) — avoids Aerospace/i3/sway conflicts
- [04-01-D1] parseInner returns Param directly instead of (name, def) tuple — cleaner API, all fields populated in one place
- [04-01-D2] ParamType uses iota for in-memory efficiency; Arg.Type uses string for YAML readability
- [04-02-D1] Parse fish history line-by-line, not with YAML parser — pseudo-YAML breaks real parsers
- [04-02-D2] Non-numeric bash # lines treated as regular commands
- [04-02-D3] Mixed zsh format = plain entries first then extended (not interleaved)
- [04-02-D4] Extract shared lastN/last helpers instead of duplicating across readers
- [04-04-D1] Parallel arrays for param state instead of new struct — keeps textinput[] and liveRender compatible
- [04-04-D2] 5-second timeout for dynamic commands via context.WithTimeout
- [04-04-D3] Failed dynamic params fall back to free-text input
- [04-05-D1] URLs detected first, then IPs/paths skip URL ranges to avoid duplicates
- [04-05-D2] Port detection requires 4-5 digits after colon (avoids false positives on short numbers)
- [04-06-D1] Warning map dual-key lookup (raw name + description) for Pet format compatibility
- [04-06-D2] YAML comment injection is best-effort (silent failure on file I/O errors)
- [04-06-D3] No root.go modification needed — 04-05 already wired importCmd alongside registerCmd
- [04-07-D1] Used interface{} + normalizeTag helper instead of custom UnmarshalTOML (go-toml v2 requires Decoder setup)
- [04-08-D1] Sidecar file path uses XDG_DATA_HOME with ~/.local/share fallback
- [04-08-D2] Warning printed to stderr (not stdout) to avoid polluting captured command output
- [05-01-D1] All ai package source files created in Task 1 — Go requires all package files to compile together
- [05-01-D2] AIModelConfig/AISettings/AppConfig types defined in config package to avoid circular imports
- [05-01-D3] ForTask() uses triple fallback: task-specific → Fallback → hardcoded gpt-4o-mini
- [05-01-D4] extractJSON helper handles markdown fences, raw JSON with prefix text, empty strings
- [05-01-D5] Copilot SDK API adapted from research — client.Start(ctx), session.Destroy() returns error, Data.Content is *string
- [05-02-D1] AI errors use fmt.Fprintf(os.Stderr) + return nil — prevents cobra usage help on AI failures
- [05-02-D2] Template params extracted from command first, merged with AI-suggested arg metadata
- [05-02-D3] Autofill shows current→suggested diff with accept/edit/skip per field
- [05-03-D1] G and A keybindings (uppercase) for AI generate/autofill — avoids conflicts with existing lowercase bindings
- [05-03-D2] Static loading indicator instead of animated spinner — acceptable for v1
- [05-03-D3] Transient aiError in hints bar cleared on next keypress — non-intrusive error display
- [05-03-D4] dialogAIGenerate routed through existing dialog system — no new infrastructure needed

- [06-01-D1] No new dependencies for source management — os/exec for git, existing goccy/go-yaml for config
- [06-02-D1] RemoteStore.Get uses List iteration (no slug-based path assumption for remote repos)
- [06-02-D2] Remote store failures log warnings to stderr but don't break List
- [06-02-D3] Sorted alias iteration for deterministic MultiStore.List ordering
- [06-04-D1] CONOUT$ for Windows openTTY output (not CONIN$ which is for input)
- [06-04-D2] WriteString instead of fmt.Print for shell scripts — avoids go vet false positives

### Pending Todos

None yet.

### Blockers/Concerns

- Copilot CLI SDK is technical preview (Jan 2026) — API may change; current implementation uses v0.1.25

## Session Continuity

Last session: 2026-02-25T13:09:14Z
Stopped at: Completed 06-04-PLAN.md — PowerShell integration + Windows build tags. Plan 06-03 (Source CLI commands) still pending.
Resume file: None
