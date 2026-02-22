---
phase: 02-quick-picker-shell-integration
plan: 03
subsystem: shell
tags: [zsh, bash, fish, shell-integration, cobra, keybinding]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: CLI scaffold with Cobra rootCmd
provides:
  - Shell integration scripts (zsh, bash, fish) as Go constants
  - wf init command for outputting shell scripts
affects: [02-04-wf-pick-command, 06-distribution]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shell function wrapper pattern for TUI-to-prompt paste (fd swap 3>&1 1>&2 2>&3)"
    - "Go string constants for shell scripts (no template rendering needed)"

key-files:
  created:
    - internal/shell/zsh.go
    - internal/shell/bash.go
    - internal/shell/fish.go
    - cmd/wf/init_shell.go
  modified:
    - cmd/wf/root.go

key-decisions:
  - "Ctrl+G as default keybinding — least conflicting across shells"
  - "Replace (not append) existing prompt text — matches atuin behavior"

patterns-established:
  - "Shell function wrapper: binary outputs to stdout, shell function captures and assigns to buffer variable"
  - "fd swap pattern (3>&1 1>&2 2>&3): TUI renders on stderr, selection captured on stdout"

# Metrics
duration: 2min
completed: 2026-02-22
---

# Phase 2 Plan 3: Shell Integration Scripts Summary

**Shell integration scripts for zsh/bash/fish with Ctrl+G keybinding and fd swap pattern, plus `wf init` CLI command**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-22T09:23:35Z
- **Completed:** 2026-02-22T09:25:08Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Created shell integration scripts for zsh (LBUFFER), bash (READLINE_LINE), and fish (commandline -r)
- All scripts use fd swap (3>&1 1>&2 2>&3) for clean TUI/output separation
- Ctrl+G bound in emacs, vi-insert, and vi-command modes across all shells
- `wf init zsh|bash|fish` command outputs the correct script to stdout with clear error for unsupported shells

## Task Commits

Each task was committed atomically:

1. **Task 1: Shell script constants** - `ef1cbba` (feat)
2. **Task 2: wf init command** - `853c0ad` (feat)

## Files Created/Modified
- `internal/shell/zsh.go` - Zsh init script as Go string constant (LBUFFER, zle widget, bindkey)
- `internal/shell/bash.go` - Bash init script as Go string constant (READLINE_LINE, bind -x)
- `internal/shell/fish.go` - Fish init script as Go string constant (commandline -r, string collect)
- `cmd/wf/init_shell.go` - Cobra command: wf init [zsh|bash|fish] with ValidArgs and ExactArgs(1)
- `cmd/wf/root.go` - Added initCmd to root command

## Decisions Made
- [02-03-D1] Ctrl+G as default keybinding — overrides readline's rarely-used "abort" in bash, safe in zsh/fish
- [02-03-D2] Replace existing prompt text (atuin approach) — wf outputs complete commands, not fragments

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Shell integration scripts ready for use once `wf pick` command exists (02-04)
- `wf init` command registered and functional
- Ready for 02-04-PLAN.md (wf pick command + clipboard + cross-compile + e2e verification)

---
*Phase: 02-quick-picker-shell-integration*
*Completed: 2026-02-22*
