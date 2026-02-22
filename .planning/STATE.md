# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-19)

**Core value:** Users can find and execute any saved command workflow in under 3 seconds
**Current focus:** Phase 3 in progress — Management TUI

## Current Position

Phase: 3 of 6 (Management TUI)
Plan: 2 of 5 in current phase
Status: In progress
Last activity: 2026-02-22 — Completed 03-02-PLAN.md (browse view + sidebar)

Progress: [██████████░░░] 77% (10/13 defined plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 10
- Average duration: ~4 minutes
- Total execution time: ~0.7 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 4/4 ✅ | ~20 min | ~5 min |
| 2. Quick Picker | 4/4 ✅ | ~25 min | ~6 min |
| 3. Management TUI | 2/5 🔧 | ~11 min | ~5.5 min |

**Recent Trend:**
- Last 5 plans: 02-02 ✅, 02-03 ✅, 02-04 ✅, 03-01 ✅, 03-02 ✅
- Trend: Consistent ~3-8 min/plan

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

### Pending Todos

None yet.

### Blockers/Concerns

- Research flags TIOCSTI deprecation as Phase 2 blocker — must use shell function wrappers, not ioctl ✅ (resolved: shell function wrappers implemented)
- Copilot CLI SDK is technical preview (Jan 2026) — re-assess stability before Phase 5 planning
- Stale form.go/form_test.go files from debug session keep reappearing in working tree — may need .gitignore or cleanup before Plan 03-03

## Session Continuity

Last session: 2026-02-22
Stopped at: Completed 03-02-PLAN.md — Phase 3 plan 2 of 5 complete (browse view + sidebar)
Resume file: None
