---
phase: "04-advanced-parameters-import"
plan: "06"
subsystem: "cli-import"
tags: ["cobra", "import", "pet", "warp", "yaml-comments", "conflict-resolution"]

dependency-graph:
  requires: ["04-01", "04-03"]
  provides: ["wf-import-command", "pet-import-cli", "warp-import-cli", "yaml-comment-injection"]
  affects: []

tech-stack:
  added: []
  patterns: ["cobra-subcommand-tree", "interactive-conflict-resolution", "yaml-comment-injection"]

key-files:
  created:
    - "cmd/wf/import.go"
  modified: []

decisions:
  - id: "04-06-D1"
    decision: "Warning map indexes by both raw name and description for Pet format compatibility"
    reason: "Pet warnings use snippet description as identifier, but workflow Name is slugified — dual-key lookup covers both"
  - id: "04-06-D2"
    decision: "YAML comment injection is best-effort (silent failure)"
    reason: "Import should not fail if comment prepending encounters a file error — the workflow is already saved"
  - id: "04-06-D3"
    decision: "importCmd wired in root.go by prior plan (04-05) alongside registerCmd"
    reason: "04-05 added both commands to root.go init() in a single commit — no root.go changes needed here"

metrics:
  duration: "~8 min"
  completed: "2026-02-24"
---

# Phase 4 Plan 6: wf import Command with Preview/Conflict Handling Summary

**One-liner:** Cobra import command with Pet/Warp subcommands, dry-run preview table, interactive skip/rename/overwrite conflict resolution, and YAML comment injection for unmappable fields.

## What Was Done

### Task 1: wf import command with preview and conflict handling
**Commit:** `ad45dba`

Created `cmd/wf/import.go` (269 lines) implementing the full import command:

**Command structure:**
- `wf import` — parent command with `--force` and `--folder` persistent flags
- `wf import pet <file>` — imports from Pet TOML snippet files
- `wf import warp <file>` — imports from Warp YAML workflow files

**Flow (shared via `runImport()`):**
1. Opens source file and calls the appropriate `Importer.Import()`
2. Reports parse results (count, warnings, errors)
3. Detects conflicts by checking `store.Get()` for each workflow name
4. Shows preview table with new/conflict status per workflow
5. Interactive conflict resolution: skip, rename (with new name prompt), or overwrite
6. Saves workflows via `store.Save()`
7. Injects YAML comment headers for unmappable fields (author, shells, output, etc.)

**Key implementation details:**
- `buildWarningMap()` extracts quoted names from warning strings and indexes by both raw name and slugified form (handles Pet's description-vs-slug mismatch)
- `injectComments()` reads the saved YAML file back, prepends `# Imported from <format> — unmappable fields:` header, writes back — all best-effort
- `--folder` flag prepends folder path to workflow names (creates subdirectory structure)
- `importWorkflowEntry` struct tracks each workflow through the pipeline with conflict flag

### Task 2: End-to-end verification
**No commit** (verification only, no code changes)

Verified all Phase 4 features working together:
- **Enum parameters:** `{{env|dev|staging|*prod}}` syntax preserved in YAML, default extracted correctly
- **Dynamic parameters:** `{{branch!git branch --list}}` syntax preserved in YAML
- **Register command:** `wf register 'curl ...'` captures command, detects URL parameter, saves workflow
- **Pet import:** 2 workflows parsed, 1 warning (output field), YAML comments injected correctly
- **Warp import:** 1 workflow parsed, 3 warnings (author, author_url, shells), YAML comments injected correctly
- **Folder import:** `--folder imported` creates subdirectory with prefixed workflow names
- **Full test suite:** `go test ./...` — all 9 packages pass
- **Vet:** `go vet ./...` — clean
- **Build:** `go build ./cmd/wf/` — clean

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Pet warning map key mismatch**
- **Found during:** Task 1 development
- **Issue:** `buildWarningMap` extracted quoted names from warning strings. For Pet, warnings use the snippet description (e.g., "Deploy to environment") but the workflow Name is slugified ("deploy-to-environment"). Warning lookup by `wf.Name` always missed.
- **Fix:** Added fallback lookup by `wf.Description` in the save loop; `buildWarningMap` already indexes by the raw quoted name which matches the description
- **Files modified:** `cmd/wf/import.go`
- **Commit:** `ad45dba`

**2. [Deviation] root.go already modified by prior plan**
- **Found during:** Task 1 commit
- **Issue:** Plan specified modifying `cmd/wf/root.go` to wire `importCmd`, but commit `5f3a5d9` (04-05) already added both `importCmd` and `registerCmd` to root.go
- **Impact:** No root.go changes needed — only `cmd/wf/import.go` was created
- **No fix needed** — existing wiring was correct

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 04-06-D1 | Warning map dual-key lookup (raw name + description) | Pet warnings use description; workflow Name is slugified. Dual-key covers both formats |
| 04-06-D2 | YAML comment injection is best-effort (silent failure) | Import should succeed even if comment file I/O fails |
| 04-06-D3 | No root.go modification needed | 04-05 already wired importCmd alongside registerCmd |

## Next Phase Readiness

**Phase 4 is now COMPLETE.** All 6 plans executed successfully.

All Phase 4 success criteria met:
1. ✅ Enum parameters with predefined option lists and selection during filling
2. ✅ Dynamic parameters that populate from shell command output at fill time
3. ✅ `wf register` captures previous shell commands and saves as workflows
4. ✅ `wf import` converts Pet TOML and Warp YAML files into wf workflows

**Ready for Phase 5 (AI Integration):** No blockers. Copilot CLI SDK stability should be re-assessed before planning.
