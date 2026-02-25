---
phase: 06-distribution-sharing
plan: 01
subsystem: source-management
tags: [git, os-exec, xdg, yaml, source-sharing]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: config package with XDG paths, goccy/go-yaml for persistence
provides:
  - "Source Manager with CRUD operations (Add/Remove/Update/List)"
  - "Git operations wrapper (clone/pull/available) with timeouts and GIT_TERMINAL_PROMPT=0"
  - "config.SourcesDir() for data-home clone storage"
  - "UpdateResult diff reporting after source update"
affects: [06-02, 06-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "os/exec git wrapper with context timeouts and GIT_TERMINAL_PROMPT=0"
    - "Source config persistence via sources.yaml alongside clone directories"
    - "Snapshot-diff pattern for detecting YAML file changes after git pull"

key-files:
  created:
    - "internal/source/manager.go"
    - "internal/source/git.go"
  modified:
    - "internal/config/config.go"

key-decisions:
  - "No new dependencies — os/exec for git, existing goccy/go-yaml for config"

patterns-established:
  - "Source Manager pattern: CRUD on sources.yaml + git operations on clone dirs"
  - "GIT_TERMINAL_PROMPT=0 on all exec.Command calls to prevent auth hangs"

# Metrics
duration: 2min
completed: 2026-02-25
---

# Phase 6 Plan 1: Source Manager + Git Ops Summary

**Source manager CRUD with os/exec git wrapper, config.SourcesDir(), and snapshot-diff update reporting**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-25T13:05:19Z
- **Completed:** 2026-02-25T13:07:07Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Git operations wrapper with shallow clone, ff-only pull, availability check, and alias derivation
- Source Manager with Add (clone+register), Remove (cleanup+deregister), Update (pull+diff), List, and SourceDirs
- config.SourcesDir() and EnsureSourcesDir() using xdg.DataHome for clone storage

## Task Commits

Each task was committed atomically:

1. **Task 1: Config extension + Git operations wrapper** - `3983f52` (feat)
2. **Task 2: Source manager with CRUD operations** - `f3802df` (feat)

## Files Created/Modified
- `internal/config/config.go` - Added SourcesDir() and EnsureSourcesDir() for xdg.DataHome/wf/sources path
- `internal/source/git.go` - Git operations: gitAvailable, gitClone, gitPull, deriveAlias
- `internal/source/manager.go` - Source Manager: NewManager, Add, Remove, Update, List, SourceDirs with sources.yaml persistence

## Decisions Made
- No new dependencies added — os/exec for git operations and existing goccy/go-yaml for config persistence keeps the dependency graph clean

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Source manager foundation complete, ready for 06-02-PLAN.md (RemoteStore + MultiStore)
- Manager.SourceDirs() provides the alias→path map needed by RemoteStore

---
*Phase: 06-distribution-sharing*
*Completed: 2026-02-25*
