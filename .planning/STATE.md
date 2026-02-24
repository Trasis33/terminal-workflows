# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Users can find and execute any saved command workflow in under 3 seconds
**Current focus:** Phase 4 complete — Ready for Phase 5 (AI Integration)

## Current Position

Phase: 4 of 6 (Advanced Parameters & Import) — COMPLETE
Plan: 6 of 6 in current phase
Status: Phase complete
Last activity: 2026-02-24 — Completed 04-06-PLAN.md (wf import command with preview/conflict handling)

Progress: [███████████████████] 100% (19/19 defined plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 19
- Average duration: ~3.8 minutes
- Total execution time: ~1.23 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 4/4 | ~20 min | ~5 min |
| 2. Quick Picker | 4/4 | ~25 min | ~6 min |
| 3. Management TUI | 5/5 | ~43 min | ~9 min |
| 4. Advanced Params | 6/6 | ~21 min | ~4 min |

**Recent Trend:**
- Last 5 plans: 04-02, 04-03, 04-04, 04-05, 04-06
- Trend: Phase 4 plans averaging ~4 min (fast execution cycles)

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

### Pending Todos

None yet.

### Blockers/Concerns

- Research flags TIOCSTI deprecation as Phase 2 blocker — must use shell function wrappers, not ioctl (resolved: shell function wrappers implemented)
- Copilot CLI SDK is technical preview (Jan 2026) — re-assess stability before Phase 5 planning

## Session Continuity

Last session: 2026-02-24
Stopped at: Completed 04-06-PLAN.md — Phase 4 COMPLETE (6/6 plans done). Ready for Phase 5.
Resume file: None
