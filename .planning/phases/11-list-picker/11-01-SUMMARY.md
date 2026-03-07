---
phase: 11-list-picker
plan: 01
subsystem: ui
tags: [go, yaml, params, list-picker, bubbletea]

# Dependency graph
requires:
  - phase: 10-02
    provides: "List parameter type in the custom parameter editor and soft-staged metadata behavior"
  - phase: 04-04
    provides: "Picker/manage param-fill runtime using template.Param metadata"
provides:
  - "Persisted list picker metadata on workflow args with YAML list_* keys"
  - "Shared param metadata overlay that makes saved arg type data authoritative at runtime"
  - "Centralized list command loading, header skipping, diagnostics, and literal field extraction helpers"
affects: [11-02, 11-03, picker, manage]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Overlay extracted template params with stored arg metadata by matching param name before param-fill UI setup"
    - "Preserve raw list rows through loading and defer literal delimiter extraction until after selection"

key-files:
  created:
    - "internal/params/metadata.go"
    - "internal/params/list_source.go"
    - "internal/params/extract.go"
    - "internal/params/metadata_test.go"
    - "internal/params/list_source_test.go"
    - "internal/params/param_type_test.go"
    - "internal/params/store_roundtrip_test.go"
  modified:
    - "internal/store/workflow.go"
    - "internal/template/parser.go"
    - "internal/picker/paramfill.go"
    - "internal/manage/execute_dialog.go"

key-decisions:
  - "Stored workflow arg metadata now overrides extracted runtime param type data by name, while inline defaults remain authoritative."
  - "List extraction stays literal via strings.Split with 1-based field indices and 0 meaning whole-row fallback."

patterns-established:
  - "Runtime param hydration: template.ExtractParams + params.OverlayMetadata before building inputs"
  - "List source helpers return short errors plus optional detail and explicit empty-after-skip state"

requirements-completed: [LIST-01, LIST-03, LIST-04, LIST-05]

# Metrics
duration: 6 min
completed: 2026-03-07
---

# Phase 11 Plan 1: Shared List Metadata Foundation Summary

**Persisted list picker arg metadata now survives workflow reloads and reaches picker/manage runtime through a shared overlay, with centralized list loading and literal field extraction helpers.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-07T19:09:30Z
- **Completed:** 2026-03-07T19:16:00Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Added `list_cmd`, `list_delimiter`, `list_field_index`, and `list_skip_header` persistence to workflow args.
- Introduced shared `internal/params` helpers for metadata overlay, command-backed row loading, header skipping, diagnostics, and deferred extraction.
- Locked overlay precedence, YAML persistence, extraction semantics, and scanner/header-skip behavior with focused tests.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add list metadata contracts and shared helper package** - `c6b7eba` (feat)
2. **Task 2: Add foundation tests for overlay and list parsing rules** - `c3e3264` (test)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `internal/store/workflow.go` - Persists list picker metadata on workflow args.
- `internal/template/parser.go` - Adds `ParamList`, runtime list fields, and string conversion helpers.
- `internal/params/metadata.go` - Overlays stored arg metadata onto extracted template params by name.
- `internal/params/list_source.go` - Centralizes command execution, row scanning, header skipping, and diagnostics.
- `internal/params/extract.go` - Implements literal delimiter extraction with retryable missing-field errors.
- `internal/picker/paramfill.go` - Switches picker runtime param hydration to the shared overlay path.
- `internal/manage/execute_dialog.go` - Switches manage execute dialog param hydration to the shared overlay path.
- `internal/params/metadata_test.go` - Covers overlay precedence and extraction behavior.
- `internal/params/list_source_test.go` - Covers header skipping, empty-after-skip, command failure detail, and scanner errors.
- `internal/params/store_roundtrip_test.go` - Verifies persisted YAML list metadata keys and round-trip behavior.

## Decisions Made
- Stored arg metadata is now authoritative for runtime param type/configuration, but inline template defaults still win over saved defaults.
- List extraction intentionally uses literal delimiters and 1-based field indices to match the plan scope and avoid quote-aware parsing complexity.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Wired picker and manage execution through shared metadata overlay**
- **Found during:** Task 1 (Add list metadata contracts and shared helper package)
- **Issue:** Creating `OverlayMetadata` alone would not make saved `type: list` args reach runtime because picker and manage execute paths still merged stored defaults only.
- **Fix:** Updated both param-fill entry points to hydrate params via `params.OverlayMetadata` before building inputs.
- **Files modified:** `internal/picker/paramfill.go`, `internal/manage/execute_dialog.go`
- **Verification:** `go build ./... && go test ./internal/params/... ./internal/template/... -count=1`
- **Committed in:** `c6b7eba` (part of Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Required to satisfy the core runtime truth that saved list metadata survives reload and affects execution. No scope creep.

## Issues Encountered

- Initial overlay test expected an inline enum without a starred default to keep `prod` as the default; corrected the fixture to use `*prod` so the test matched existing enum semantics.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Runtime and persistence foundation for list params is in place, so Phase 11-02 can focus on authoring UI for list command, delimiter, field index, and header skipping.
- Phase 11-03 can reuse the shared params helpers instead of duplicating command execution or parsing logic in picker/manage list selection flows.

## Self-Check: PASSED

---
*Phase: 11-list-picker*
*Completed: 2026-03-07*
