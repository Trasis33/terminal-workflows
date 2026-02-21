---
phase: 01-foundation-data-layer
plan: 03
subsystem: cli
tags: [go, cobra, cli, interactive-prompts, editor-integration]

dependency_graph:
  requires:
    - phase: 01-01
      provides: "Workflow model, Store interface, YAMLStore, XDG config"
    - phase: 01-02
      provides: "ExtractParams for auto-discovering {{params}} from commands"
  provides:
    - "wf add command with flags and interactive fallback"
    - "wf edit command with $EDITOR and quick flag updates"
    - "wf rm command with confirmation and --force"
    - "Exported WorkflowPath on YAMLStore"
  affects:
    - "01-04 (integration tests will exercise CLI commands)"
    - "Phase 2 (picker will invoke workflows created via these commands)"
    - "Phase 3 (management TUI will complement these CLI commands)"

tech-stack:
  added: []
  patterns:
    - "Interactive fallback: prompt for required fields when flags missing"
    - "$EDITOR integration: open YAML in user's editor with terminal takeover"
    - "Quick edit via flags: update specific fields without editor"

key-files:
  created:
    - cmd/wf/add.go
    - cmd/wf/edit.go
    - cmd/wf/rm.go
  modified:
    - cmd/wf/root.go
    - internal/store/yaml.go

key-decisions:
  - "Interactive prompts only shown when required fields (name/command) are missing — optional field prompts skipped in non-interactive mode"
  - "Exported WorkflowPath on YAMLStore to enable $EDITOR integration in edit command"
  - "wf edit re-extracts template params when command field is updated via flags"

patterns-established:
  - "Interactive fallback: flags first, prompt only when required fields missing"
  - "Editor integration: exec.Command with Stdin/Stdout/Stderr for terminal takeover"
  - "Quick edit: flag-based field updates without opening editor"
  - "Confirmation prompt: default to N, only proceed on y/yes"

duration: 3min
completed: 2026-02-21
---

# Phase 1 Plan 03: CLI Commands Summary

**Cobra CLI commands (wf add, wf edit, wf rm) with flags, interactive fallback, $EDITOR integration, and auto-extracted {{params}} from command strings**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-21T09:21:05Z
- **Completed:** 2026-02-21T09:24:27Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- `wf add` creates workflow YAML files with name, command, description, tags, and auto-extracted args from {{param}} syntax
- `wf edit` supports dual mode: $EDITOR for full editing, flags for quick field updates (--command, --description, --tag, --add-tag, --remove-tag)
- `wf rm` prompts for confirmation by default, `--force` skips it
- All commands print minimal one-liner feedback and error gracefully on invalid input

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement wf add command** - `a59b60a` (feat)
2. **Task 2: Implement wf edit and wf rm commands** - `cdb7373` (feat)

## Files Created/Modified
- `cmd/wf/add.go` - wf add command with flags, interactive fallback, and template param extraction
- `cmd/wf/edit.go` - wf edit command with $EDITOR mode and quick flag-based updates
- `cmd/wf/rm.go` - wf rm command with confirmation prompt and --force flag
- `cmd/wf/root.go` - Registers all subcommands, initializes YAMLStore via getStore()
- `internal/store/yaml.go` - Exported WorkflowPath method for editor integration

## Decisions Made
- **Interactive prompts gated on missing required fields:** Optional field prompts (description, tags) only shown when user enters interactive mode (name or command missing). Prevents unwanted prompts when all flags are provided.
- **Exported WorkflowPath:** Changed from unexported `workflowPath` to exported `WorkflowPath` on YAMLStore to enable the edit command's $EDITOR integration.
- **Re-extract params on command edit:** When `wf edit --command` updates the command field, template.ExtractParams is re-run to refresh the args list.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Exported WorkflowPath on YAMLStore**
- **Found during:** Task 2 (wf edit implementation)
- **Issue:** The edit command needs the filesystem path to open in $EDITOR, but `workflowPath` was unexported
- **Fix:** Renamed `workflowPath` to `WorkflowPath` (exported) on YAMLStore
- **Files modified:** internal/store/yaml.go
- **Verification:** `go build ./cmd/wf/` compiles, all existing tests pass
- **Committed in:** `cdb7373` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for editor integration. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All three CLI commands operational: add, edit, rm
- Ready for Plan 01-04 (integration tests + wf list + final verification)
- Store and template packages correctly integrated via CLI layer
- No blockers for next plan

---
*Phase: 01-foundation-data-layer*
*Completed: 2026-02-21*
