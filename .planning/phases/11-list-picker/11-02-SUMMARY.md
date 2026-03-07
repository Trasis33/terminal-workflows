---
phase: 11-list-picker
plan: 02
subsystem: ui
tags: [go, bubbletea, manage, list-picker, parameter-editor]

# Dependency graph
requires:
  - phase: 11-01
    provides: "Persisted list metadata fields plus runtime overlay helpers for list params"
  - phase: 10-02
    provides: "ParamEditorModel soft-staging, sub-field routing, and save-time ToArgs cleanup"
provides:
  - "Manage editor controls for list command, delimiter, field index, and header skip"
  - "Soft-staged list metadata that survives type switches and persists only for type:list"
  - "Manage form regression coverage for list save/reload round trips"
affects: [11-03, manage, list-picker]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "List params extend ParamEditorModel sub-field navigation with dedicated textinputs for command and extraction metadata"
    - "List metadata remains in-memory during type changes and is emitted only through ParamEditorModel.ToArgs when the saved type is list"

key-files:
  created: []
  modified:
    - internal/manage/param_editor.go
    - internal/manage/param_editor_test.go
    - internal/manage/form_test.go

key-decisions:
  - "List field index is author-facing 1-based, with blank or 0 preserved as explicit whole-row fallback."
  - "List command, delimiter, field index, and header skip stay isolated from enum/dynamic metadata and soft-stage across type switches until save."

patterns-established:
  - "Expanded list param rows render four dedicated metadata controls beneath the default field in the custom manage editor"
  - "Form round-trip coverage validates hydrated list args directly through NewFormModel and saveWorkflow without special-case form wiring"

requirements-completed: [LIST-01, LIST-03, LIST-04, LIST-05]

# Metrics
duration: 6 min
completed: 2026-03-07
---

# Phase 11 Plan 02: Manage List Authoring Summary

**Manage workflow editing now supports full list-picker metadata authoring with persisted command, delimiter, 1-based field extraction, and header skipping controls.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-07T19:20:08Z
- **Completed:** 2026-03-07T19:26:30Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added dedicated manage editor inputs for list command, delimiter, field index, and header skip metadata.
- Preserved list settings during in-memory type switching while keeping enum and dynamic metadata isolated on save.
- Added regression tests proving list args hydrate into the editor and round-trip through save/reload with exact persisted values.

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend ParamEditorModel with list metadata fields** - `c30bc60` (feat)
2. **Task 2: Persist list metadata through the manage form** - `f709701` (test)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `internal/manage/param_editor.go` - Adds list-specific inputs, sub-field routing, validation warnings, rendering, and `ToArgs()` persistence.
- `internal/manage/param_editor_test.go` - Covers list field visibility, soft staging, whole-row fallback, and list validation parsing.
- `internal/manage/form_test.go` - Verifies list args hydrate from existing workflows and persist exact metadata through save/reload.

## Decisions Made
- Used dedicated list metadata inputs instead of reusing enum or dynamic fields so list authoring stays explicit and semantically separate.
- Kept field index author-facing and 1-based while preserving `0` as the stored whole-row fallback, matching Phase 11 extraction semantics from Plan 01.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Manage authoring for list params is complete, so Phase 11-03 can focus entirely on runtime list selection UX in picker and manage execute flows.
- Existing list metadata overlay and extraction helpers from 11-01 now have matching author-side configuration and regression coverage.

## Self-Check: PASSED

---
*Phase: 11-list-picker*
*Completed: 2026-03-07*
