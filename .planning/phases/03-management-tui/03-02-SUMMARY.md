---
phase: 03-management-tui
plan: 02
subsystem: ui
tags: [bubbletea, lipgloss, sidebar, browse, fuzzy-search, preview, tui]

# Dependency graph
requires:
  - phase: 03-management-tui
    plan: 01
    provides: Theme, themeStyles, keyMap, root Model with view router, message types
  - phase: 02-quick-picker
    provides: picker.ParseQuery, picker.Search for fuzzy filtering
  - phase: 01-foundation
    provides: store.Store interface, store.Workflow struct
provides:
  - SidebarModel with folder tree / tag list dual-mode navigation
  - BrowseModel with two-pane layout, fuzzy search, preview pane
  - extractFolders/extractTags helpers for workflow metadata derivation
  - Root model wired to BrowseModel (replaces placeholder viewBrowse)
affects: [03-03, 03-04, 03-05]

# Tech tracking
tech-stack:
  added: [charmbracelet/bubbles/textinput]
  patterns: [child-model composition, virtual scrolling, sidebar filter message]

key-files:
  created:
    - internal/manage/sidebar.go
    - internal/manage/browse.go
  modified:
    - internal/manage/model.go
    - internal/manage/manage.go
    - internal/manage/model_test.go

key-decisions:
  - "03-02-D1: Dual-mode sidebar (folders/tags) via ToggleMode() with virtual root items"
  - "03-02-D2: Reuse picker.ParseQuery + picker.Search for fuzzy filtering in browse view"
  - "03-02-D3: browsetruncate local helper to avoid collision with unexported truncateStr in picker"
  - "03-02-D4: Folders derived implicitly from workflow name path prefixes (no explicit folder model)"

patterns-established:
  - "Child model composition: BrowseModel.Update() returns (BrowseModel, tea.Cmd), root routes via updateBrowse"
  - "Custom message type for parent communication: sidebarFilterMsg carries filter type/value"
  - "Virtual scrolling with ensureCursorVisible for large workflow lists"
  - "SetDimensions/UpdateData pattern for child models to receive layout and data updates"

# Metrics
duration: 8min
completed: 2026-02-22
---

# Phase 3 Plan 2: Browse View & Sidebar Summary

**Two-pane browse view with folder/tag sidebar, fuzzy search via picker reuse, workflow preview pane, and root model wiring replacing placeholder**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-22T20:09:24Z
- **Completed:** 2026-02-22T20:17:42Z
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified including go.mod/go.sum)

## Accomplishments
- SidebarModel with dual-mode (folders/tags), cursor navigation, virtual root items ("All Workflows"/"All Tags"), and sidebarFilterMsg for parent communication
- BrowseModel with sidebar + scrollable workflow list + preview pane, fuzzy search via picker.Search/ParseQuery reuse, folder/tag filtering
- Root model fully wired — BrowseModel replaces placeholder viewBrowse, propagates WindowSizeMsg and workflowsLoadedMsg
- extractFolders/extractTags helpers derive metadata from workflow names and tags
- 18 tests passing including browse wiring, data refresh, folder/tag extraction

## Task Commits

Each task was committed atomically:

1. **Task 1: SidebarModel with folder tree and tag list** - `6bdb3ca` (feat)
2. **Task 2: BrowseModel + root model wiring** - `ccb5bef` (feat)

## Files Created/Modified
- `internal/manage/sidebar.go` - NEW: SidebarModel with dual-mode folder/tag navigation, cursor movement, filter messages
- `internal/manage/browse.go` - NEW: BrowseModel with two-pane layout, fuzzy search, preview, keybinding hints
- `internal/manage/model.go` - Updated: browse field in Model, SetDimensions propagation, BrowseModel routing, extractFolders/extractTags helpers
- `internal/manage/manage.go` - Updated: New() constructs BrowseModel with folders/tags/theme/keys
- `internal/manage/model_test.go` - Updated: browse wiring tests, data refresh propagation, extractFolders/extractTags tests
- `go.mod` - golang.org/x/text updated to v0.23.0
- `go.sum` - Updated checksums

## Decisions Made
- [03-02-D1] Dual-mode sidebar (folders/tags) via ToggleMode() with virtual root items ("All Workflows" / "All Tags") at index 0
- [03-02-D2] Reuse picker.ParseQuery + picker.Search for fuzzy filtering — no duplicate search logic
- [03-02-D3] browsetruncate local helper to avoid collision with unexported truncateStr in picker package
- [03-02-D4] Folders derived implicitly from workflow name path prefixes (e.g., "infra/deploy/app" → ["infra", "infra/deploy"]) — no explicit folder model needed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed stale form.go from working tree**
- **Found during:** Task 2 (compilation)
- **Issue:** An untracked `form.go` file from a previous debug session was in `internal/manage/`, referencing undefined error variables (`errNameRequired`, `errNameNoSlash`, `errCommandRequired`), blocking compilation
- **Fix:** Removed `form.go` from the working tree (it belongs to Plan 03-03, not this plan)
- **Files modified:** None committed — file was untracked
- **Verification:** `go build ./...` succeeds
- **Committed in:** N/A (file removal, not committed)

**2. [Rule 3 - Blocking] Cleaned FormModel references from model.go**
- **Found during:** Task 2 (compilation)
- **Issue:** model.go had accumulated `FormModel` field, `updateForm()`, `NewFormModel()` calls, and form view routing from the same debug session — all referencing the removed form.go
- **Fix:** Stripped all FormModel references, keeping only BrowseModel wiring
- **Files modified:** `internal/manage/model.go`
- **Verification:** `go build ./...` and `go test ./...` pass
- **Committed in:** `ccb5bef` (part of Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 - blocking)
**Impact on plan:** Both auto-fixes necessary to unblock compilation. No scope creep — removed stale debug artifacts.

## Issues Encountered
- Stale `form.go` from a previous debug session kept reappearing in the working tree (possibly restored by editor file watchers). Required `rm -f` to definitively remove.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Browse view and sidebar complete — ready for 03-03-PLAN.md (create/edit forms with huh)
- BrowseModel dispatches `switchToCreateMsg`, `switchToEditMsg`, `showDeleteDialogMsg`, `moveWorkflowMsg` — Plan 03 will handle these
- SidebarModel sidebar filter flow working end-to-end with BrowseModel
- No blockers or concerns

---
*Phase: 03-management-tui*
*Completed: 2026-02-22*
