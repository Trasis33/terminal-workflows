---
phase: 03-management-tui
plan: 04
subsystem: manage-tui
tags: [dialog, overlay, settings, theme, bubbletea]

dependency_graph:
  requires: ["03-01", "03-02"]
  provides: ["dialog-overlays", "settings-view", "theme-customization"]
  affects: ["03-05"]

tech_stack:
  added: []
  patterns: ["overlay-dialog-model", "settings-field-list", "live-theme-preview"]

key_files:
  created:
    - internal/manage/dialog.go
    - internal/manage/settings.go
  modified:
    - internal/manage/model.go
    - internal/manage/model_test.go

decisions:
  - id: "03-04-D1"
    decision: "Single atomic commit per task despite model.go having changes for both — Go compilation requires all package files simultaneously"
    rationale: "Same pattern as 03-03-D4; settings.go needed for model.go to compile"
  - id: "03-04-D2"
    decision: "DialogModel returns dialogResultMsg via tea.Cmd rather than direct model mutation"
    rationale: "Clean separation: dialog handles UI, root model handles side effects (store/OS ops)"
  - id: "03-04-D3"
    decision: "Settings view uses itself as live preview — theme being edited styles the settings view"
    rationale: "No separate preview pane needed; editing a color immediately changes the view"
  - id: "03-04-D4"
    decision: "Move dialog includes (root) as first option for moving workflows to top level"
    rationale: "Users need a way to remove folder prefixes, not just move between folders"

metrics:
  duration: "~5 minutes"
  completed: "2026-02-22"
---

# Phase 3 Plan 4: Dialog Overlays & Settings View Summary

**One-liner:** DialogModel with 5 overlay types (delete/folder-CRUD/move) and SettingsModel for live-preview theme customization with preset cycling.

## What Was Built

### Task 1: Dialog Overlays

**DialogModel** (`dialog.go`, 280 lines) — a self-contained overlay dialog supporting 5 dialog types:

- **dialogDeleteConfirm** — "Delete 'name'? This cannot be undone." with y/n prompt
- **dialogFolderCreate** — text input for new folder name
- **dialogFolderRename** — text input pre-filled with existing folder name
- **dialogFolderDelete** — confirmation for empty folder deletion
- **dialogMoveWorkflow** — scrollable folder list with cursor, includes (root) option

Each dialog type has a dedicated constructor (`NewDeleteDialog`, etc.) and returns `dialogResultMsg` with type, confirmed status, and key-value data. Root model handles results: store.Delete for workflows, os.MkdirAll/Rename/Remove for folders, save+delete for move.

**model.go changes:**
- Replaced `dialogState` struct with `*DialogModel` pointer
- Added `handleDialogResult()` for all 5 dialog result types
- Added `handleFolderDialogMsg()` for folder action routing
- Dialog gets message priority when active (checked before view routing)
- Overlay rendering via existing `renderOverlay()` with `lipgloss.Place()`

### Task 2: Settings View

**SettingsModel** (`settings.go`, 405 lines) — navigable field list for theme editing:

- **Presets row** — left/right arrows cycle through 5 presets with instant apply
- **Color fields** — 6 ANSI color values with rendered swatches (██), enter-to-edit with text input
- **Border style** — editable border type (rounded/normal/thick/double/hidden)
- **Layout fields** — sidebar width (numeric), show preview (boolean toggle)
- **Actions** — Save (persists to theme.yaml) and Cancel (reverts to original)

Live preview: the settings view renders using the theme being edited, so any color/preset change is immediately visible in the view itself.

**model.go changes:**
- Added `settings SettingsModel` field
- `switchToSettingsMsg` creates `NewSettingsModel` with theme copy
- `themeSavedMsg` reloads saved theme and rebuilds browse model with new styles
- `viewSettings` state routes to `settings.Update()` and `settings.View()`

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 03-04-D1 | Single commit per task despite interleaved model.go changes | Go compilation requires all package files; same as 03-03-D4 |
| 03-04-D2 | DialogModel returns dialogResultMsg via tea.Cmd | Clean separation: dialog handles UI, root model handles side effects |
| 03-04-D3 | Settings view uses itself as live preview | Editing a color immediately changes the view; no separate preview needed |
| 03-04-D4 | Move dialog includes (root) as first option | Users need to remove folder prefixes, not just move between folders |

## Deviations from Plan

None — plan executed exactly as written.

## Test Coverage

Added 9 new tests to `model_test.go`:
- `TestDialogDeleteRendersName` — delete dialog shows workflow name and y/n prompt
- `TestDialogFolderCreateHasInput` — folder create dialog has text input
- `TestDialogMoveShowsFolders` — move dialog lists (root) + available folders
- `TestDialogDeleteConfirmResult` — y key produces confirmed dialogResultMsg
- `TestDialogEscCancels` — esc produces non-confirmed result
- `TestDialogMoveNavigation` — up/down navigation, enter confirms with selected folder
- `TestSettingsModelView` — settings view renders all sections and fields
- `TestSettingsPresetCycling` — right arrow changes preset and marks dirty
- `TestSettingsCancelRevertsTheme` — esc reverts to original theme values

## Verification Results

1. ✅ `go build ./internal/manage/...` — compiles
2. ✅ `go test ./...` — all tests pass (36 manage tests, all packages green)
3. ✅ Delete dialog renders centered with workflow name visible
4. ✅ Folder create dialog accepts text input
5. ✅ Move dialog lists available folders with (root) option
6. ✅ Settings view shows all theme fields (presets, colors, borders, layout)
7. ✅ Preset selection changes colors live
8. ✅ Theme save produces valid YAML via SaveTheme()

## Commits

| Hash | Message |
|------|---------|
| 7f8bd8b | feat(03-04): add dialog overlays for delete, folder ops, and move workflow |
| 302d699 | feat(03-04): add settings view for theme customization with live preview |

## Next Phase Readiness

Plan 03-05 (final polish, help, status bar) can proceed. All interactive overlay and settings components are complete.
