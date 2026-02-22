---
phase: 02-quick-picker-shell-integration
plan: 04
subsystem: cli, picker
tags: [cobra, bubbletea, clipboard, cross-compile, tty, shell-integration]

# Dependency graph
requires:
  - phase: 02-02
    provides: Picker TUI model with search and param fill states
  - phase: 02-03
    provides: Shell integration scripts for zsh/bash/fish and wf init command
provides:
  - "wf pick Cobra command wiring picker model to tea.Program"
  - "Clipboard support via --copy flag and Ctrl+Y in-picker hotkey"
  - "Cross-platform binary (macOS, Linux, Windows)"
  - "Full end-to-end flow: keybinding → picker → selection → prompt paste"
affects: [phase-3-management-tui, phase-6-distribution]

# Tech tracking
tech-stack:
  added: [atotto/clipboard]
  patterns: [/dev/tty for TUI output in piped contexts, stderr fallback for Windows]

key-files:
  created: [cmd/wf/pick.go]
  modified: [cmd/wf/root.go, internal/picker/model.go, go.mod, internal/shell/zsh.go, internal/shell/bash.go, internal/shell/fish.go]

key-decisions:
  - "02-04-D1: Use /dev/tty for TUI output instead of fd swap — ZLE redirects fds before subprocess, breaking 3>&1 pattern"
  - "02-04-D2: Ctrl+Y for in-picker clipboard copy with flash message"
  - "02-04-D3: Fallback to os.Stderr if /dev/tty unavailable (Windows compatibility)"

patterns-established:
  - "/dev/tty pattern: Open /dev/tty directly for TUI rendering when stdout is captured by shell, matching fzf approach"
  - "Dual clipboard paths: --copy flag for scripting, Ctrl+Y for interactive use"

# Metrics
duration: ~15min
completed: 2026-02-22
---

# Phase 2 Plan 4: wf pick Command + Clipboard + Cross-Compile Summary

**wf pick wired to Bubble Tea picker via /dev/tty with clipboard support (--copy + Ctrl+Y), cross-compiled for 5 targets**

## Performance

- **Duration:** ~15 min (across checkpoint interaction)
- **Started:** 2026-02-22
- **Completed:** 2026-02-22
- **Tasks:** 3 (1 auto + 1 verification-only + 1 checkpoint)
- **Files modified:** 7

## Accomplishments
- `wf pick` Cobra command launches Bubble Tea picker, outputs selected command to stdout for shell capture
- Clipboard support: `--copy` flag copies to clipboard, Ctrl+Y copies highlighted workflow with flash message
- Cross-compilation verified for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
- Full end-to-end flow verified: `wf init zsh` → Ctrl+G → picker → selection → command on prompt
- Resolved ZLE fd redirection conflict by using /dev/tty for TUI output (matching fzf approach)

## Task Commits

Each task was committed atomically:

1. **Task 1: wf pick command with clipboard support** — `8ba0587` (feat)
2. **Task 2: Cross-compilation verification** — *(verification only — no commit needed)*
3. **Task 3: End-to-end picker verification** — *(checkpoint — human verified)*

**Bug fix during checkpoint:** `15d2fd0` (fix) — Use /dev/tty for TUI output instead of fd swap

**Plan metadata:** *(pending — this commit)*

## Files Created/Modified
- `cmd/wf/pick.go` — Cobra `pickCmd` wiring picker model to tea.Program with --copy flag
- `cmd/wf/root.go` — Added `pickCmd` registration in init()
- `internal/picker/model.go` — Added Ctrl+Y clipboard copy with flash message, /dev/tty TUI output
- `go.mod` — Added atotto/clipboard dependency
- `internal/shell/zsh.go` — Simplified to plain stdout capture (removed fd swap)
- `internal/shell/bash.go` — Simplified to plain stdout capture (removed fd swap)
- `internal/shell/fish.go` — Simplified to plain stdout capture (removed fd swap)

## Decisions Made

1. **[02-04-D1] Use /dev/tty for TUI output instead of fd swap** — ZLE redirects file descriptors before subprocess starts, which broke the `3>&1 1>&2 2>&3` fd swap pattern from research. Opening `/dev/tty` directly (like fzf does) ensures the TUI always renders to the real terminal regardless of shell redirection. Shell scripts simplified to plain stdout capture.

2. **[02-04-D2] Ctrl+Y for in-picker clipboard copy with flash message** — Non-blocking alternative to `--copy` flag. Users can copy any highlighted workflow without quitting the picker. Flash message provides visual confirmation.

3. **[02-04-D3] Fallback to os.Stderr if /dev/tty unavailable** — Windows doesn't have `/dev/tty`, so the picker falls back to `os.Stderr` for TUI output, maintaining cross-platform compatibility.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TUI rendering broken under ZLE fd redirection**
- **Found during:** Task 3 checkpoint (human verification)
- **Issue:** Shell integration fd swap (`3>&1 1>&2 2>&3`) broke under ZLE because zsh redirects fds before subprocess starts — TUI rendered to captured stdout instead of terminal
- **Fix:** Switched to opening `/dev/tty` directly for TUI rendering (matching fzf approach). Simplified shell integration scripts to plain stdout capture.
- **Files modified:** internal/picker/model.go, internal/shell/zsh.go, internal/shell/bash.go, internal/shell/fish.go
- **Verification:** Full end-to-end flow works — picker renders correctly, command appears on prompt
- **Committed in:** `15d2fd0`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Essential fix for correct shell integration operation. The /dev/tty approach is more robust than fd swap and matches established tools (fzf). No scope creep.

## Issues Encountered
None beyond the fd swap issue (documented above as deviation).

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- **Phase 2 complete** — All 4 plans executed, all requirements met (SRCH-01/02/03, PICK-01/02/03/04, PARM-06, SHEL-01/02/03/05)
- Ready for Phase 3 (Management TUI) or Phase 5 (AI Integration) — both depend only on Phase 1
- No blockers or concerns for next phase

---
*Phase: 02-quick-picker-shell-integration*
*Completed: 2026-02-22*
