---
phase: 04-advanced-parameters-import
plan: "08"
subsystem: shell-integration
tags: [shell, zsh, bash, fish, sidecar, history, xdg]

# Dependency graph
requires:
  - phase: 02-quick-picker
    provides: "shell integration scripts (zsh.go, bash.go, fish.go) with picker bindings"
  - phase: 04-advanced-parameters-import
    provides: "register.go with lastFromHistory() using history.NewReader()"
provides:
  - "Reliable current-session command capture via sidecar file"
  - "Shell hooks (precmd/PROMPT_COMMAND/fish_postexec) that write last command to ~/.local/share/wf/last_cmd"
  - "Clear warning when shell integration is not active"
affects: [phase-05-ai-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sidecar file pattern for cross-process IPC between shell and CLI tool"
    - "XDG_DATA_HOME-aware path construction for both shell scripts and Go code"

key-files:
  modified:
    - "internal/shell/zsh.go"
    - "internal/shell/bash.go"
    - "internal/shell/fish.go"
    - "cmd/wf/register.go"

key-decisions:
  - "04-08-D1: Sidecar file path uses XDG_DATA_HOME with ~/.local/share fallback"
  - "04-08-D2: Warning printed to stderr (not stdout) to avoid polluting captured command output"

patterns-established:
  - "Sidecar file IPC: shell hooks write to known path, Go binary reads before HISTFILE fallback"

# Metrics
duration: 1min
completed: 2026-02-24
---

# Phase 4 Plan 8: Register Sidecar History Fix Summary

**Shell hooks write last command to XDG-aware sidecar file; register.go reads sidecar first with HISTFILE fallback warning**

## Performance

- **Duration:** 1 min
- **Started:** 2026-02-24T12:05:48Z
- **Completed:** 2026-02-24T12:07:09Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added precmd/PROMPT_COMMAND/fish_postexec hooks to all three shell scripts that write the last executed command to `~/.local/share/wf/last_cmd`
- Updated `lastFromHistory()` to read sidecar file first (immune to shell history flush timing) before falling back to `$HISTFILE`
- When sidecar is absent, emits clear stderr warning explaining the limitation and how to enable shell integration
- All paths are XDG_DATA_HOME-aware (both shell scripts and Go code)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add last-command sidecar hooks to shell integration scripts** - `bd91ebe` (feat)
2. **Task 2: Update register.go to read sidecar first, warn on fallback** - `53e858b` (feat)

## Files Created/Modified
- `internal/shell/zsh.go` - Added `_wf_precmd` hook writing `$history[1]` to sidecar via `add-zsh-hook precmd`
- `internal/shell/bash.go` - Added `_wf_precmd` hook writing `history 1` output to sidecar via `PROMPT_COMMAND`
- `internal/shell/fish.go` - Added `_wf_postexec` hook writing `$argv[1]` to sidecar via `fish_postexec` event
- `cmd/wf/register.go` - `lastFromHistory()` reads sidecar before `history.NewReader()`; added `path/filepath` import

## Decisions Made
- [04-08-D1] Sidecar file path uses `${XDG_DATA_HOME:-$HOME/.local/share}/wf/last_cmd` — consistent with fish history XDG convention already in codebase
- [04-08-D2] Warning printed to stderr via `fmt.Fprintln(os.Stderr, ...)` — avoids polluting the captured command string that flows through stdout

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Gap closure plan complete — UAT Test 6 (register last shell command) issue resolved
- Shell integration now provides both picker bindings and sidecar history capture
- Ready for Phase 5 (AI Integration)

---
*Phase: 04-advanced-parameters-import*
*Completed: 2026-02-24*
