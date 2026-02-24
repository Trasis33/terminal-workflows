# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Users can find and execute any saved command workflow in under 3 seconds
**Current focus:** Phase 4 in progress — Advanced Parameters & Import

## Current Position

Phase: 4 of 6 (Advanced Parameters & Import)
Plan: 1 of 6 in current phase
Status: In progress
Last activity: 2026-02-24 — Completed 04-01-PLAN.md (template parser extension for enum/dynamic params)

Progress: [██████████████░░░░░] 74% (14/19 defined plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 14
- Average duration: ~4.1 minutes
- Total execution time: ~0.95 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 4/4 | ~20 min | ~5 min |
| 2. Quick Picker | 4/4 | ~25 min | ~6 min |
| 3. Management TUI | 5/5 | ~43 min | ~9 min |
| 4. Advanced Params | 1/6 | ~2 min | ~2 min |

**Recent Trend:**
- Last 5 plans: 03-03, 03-04, 03-05, 04-01
- Trend: 04-01 completed in ~2 min (TDD parser extension, refactor skipped)

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

### Pending Todos

None yet.

### Blockers/Concerns

- Research flags TIOCSTI deprecation as Phase 2 blocker — must use shell function wrappers, not ioctl (resolved: shell function wrappers implemented)
- Copilot CLI SDK is technical preview (Jan 2026) — re-assess stability before Phase 5 planning

## Session Continuity

Last session: 2026-02-24
Stopped at: Completed 04-01-PLAN.md — Phase 4 in progress (1/6 plans done)
Resume file: None
