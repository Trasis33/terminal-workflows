---
phase: 01-foundation-data-layer
plan: 01
subsystem: database
tags: [go, yaml, cobra, xdg, goccy-go-yaml, norway-problem]

# Dependency graph
requires:
  - phase: none
    provides: first plan — no prior dependencies
provides:
  - Workflow domain model (struct with Name, Command, Description, Tags, Args)
  - Store interface (List, Get, Save, Delete)
  - YAMLStore implementation with Norway Problem protection
  - XDG config path resolution (WorkflowsDir, EnsureDir)
  - Cobra CLI skeleton (wf --help)
  - Go module with pinned dependencies
affects: [01-02, 01-03, 01-04, phase-2, phase-3]

# Tech tracking
tech-stack:
  added: [go-1.26, cobra-v1.10.2, goccy/go-yaml, adrg/xdg, testify-v1.11.1]
  patterns: [store-interface-pattern, xdg-config-resolution, slug-based-filenames, typed-string-yaml-fields]

key-files:
  created:
    - go.mod
    - go.sum
    - .gitignore
    - cmd/wf/main.go
    - cmd/wf/root.go
    - internal/config/config.go
    - internal/store/workflow.go
    - internal/store/store.go
    - internal/store/yaml.go
    - internal/store/yaml_test.go
  modified: []

key-decisions:
  - "Used goccy/go-yaml instead of gopkg.in/yaml.v3 — unmaintained since April 2025"
  - "Used adrg/xdg instead of os.UserConfigDir() — returns wrong path on macOS"
  - "Typed string struct fields eliminate Norway Problem without custom MarshalYAML"
  - "Slug-based filenames (lowercase, hyphens, strip special chars) for workflow YAML files"
  - "Store interface pattern allows future in-memory implementation for testing"

patterns-established:
  - "Store interface: all data access through Store interface (List/Get/Save/Delete)"
  - "XDG resolution: config paths via adrg/xdg, never os.UserConfigDir()"
  - "Typed YAML fields: all string fields as Go string type to prevent boolean coercion"
  - "Slug filenames: Workflow.Filename() for filesystem-safe names"
  - "t.TempDir() for test isolation"

# Metrics
duration: 9min
completed: 2026-02-21
---

# Phase 1 Plan 01: Go Scaffold + Data Layer Summary

**Go project scaffold with Cobra CLI, XDG config, Workflow domain model, and YAML store using goccy/go-yaml with Norway Problem protection — 12 round-trip tests passing**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-21T09:07:46Z
- **Completed:** 2026-02-21T09:17:14Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Go project scaffolded with Cobra CLI (`wf --help` works), XDG config resolution, and module with pinned dependencies
- Workflow domain model with Name, Command, Description, Tags, Args and slug-based filename generation
- YAMLStore implementing full CRUD with goccy/go-yaml — Norway Problem values ("no", "yes", "on", "off", "true", "false") survive round-trip without corruption
- 12 comprehensive tests covering round-trip, multiline block scalars, Norway Problem (10 variants), nested quotes, list/delete, slugification, and edge cases

## Task Commits

Each task was committed atomically:

1. **Task 1: Go project scaffold with Cobra CLI and XDG config** - `b2d9316` (feat)
2. **Task 2: Workflow domain model + YAML store with round-trip tests** - `21cabaa` (feat)

## Files Created/Modified
- `go.mod` - Go module definition with cobra, goccy/go-yaml, adrg/xdg, testify
- `go.sum` - Dependency checksums
- `.gitignore` - Binary, .DS_Store, IDE file patterns
- `cmd/wf/main.go` - Binary entry point calling rootCmd.Execute()
- `cmd/wf/root.go` - Cobra root command ("Terminal workflow manager")
- `internal/config/config.go` - XDG WorkflowsDir() and EnsureDir() functions
- `internal/store/workflow.go` - Workflow and Arg structs with Filename() slugification
- `internal/store/store.go` - Store interface (List, Get, Save, Delete)
- `internal/store/yaml.go` - YAMLStore with recursive directory walking (2-level depth)
- `internal/store/yaml_test.go` - 12 tests: round-trip, Norway Problem, multiline, nested quotes, list, delete, slug

## Decisions Made
- **goccy/go-yaml over gopkg.in/yaml.v3**: yaml.v3 is unmaintained since April 2025. goccy/go-yaml is actively maintained and handles typed string fields correctly.
- **adrg/xdg over os.UserConfigDir()**: os.UserConfigDir() returns `~/Library/Application Support` on macOS instead of `~/.config`. adrg/xdg correctly resolves XDG paths.
- **No custom MarshalYAML needed**: goccy/go-yaml auto-quotes standalone Norway Problem values ("no", "yes", etc.) when marshalling typed string fields. Longer strings containing these words are left unquoted but round-trip correctly via typed unmarshal.
- **Store interface pattern**: Enables future in-memory Store implementation for testing CLI commands without disk I/O.
- **Slug-based filenames**: Workflow names converted to lowercase-hyphenated slugs (e.g., "Deploy Staging" → "deploy-staging.yaml") for filesystem safety.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Installed Go via Homebrew**
- **Found during:** Task 1
- **Issue:** Go was not installed on the system — `go mod init` failed
- **Fix:** Ran `brew install go` (Go 1.26.0 darwin/arm64)
- **Files modified:** None (system-level)
- **Verification:** `go version` returns `go1.26.0 darwin/arm64`
- **Committed in:** Part of Task 1 environment setup

**2. [Rule 1 - Bug] Fixed .gitignore pattern ignoring cmd/wf/ directory**
- **Found during:** Task 1
- **Issue:** Pattern `wf` in .gitignore was too broad — matched `cmd/wf/` directory, preventing git add
- **Fix:** Changed to `/wf` (root-only anchor) so only the binary at project root is ignored
- **Files modified:** `.gitignore`
- **Verification:** `git add cmd/wf/` succeeds
- **Committed in:** `b2d9316` (Task 1 commit)

**3. [Rule 1 - Bug] Fixed TestListWorkflows test logic error**
- **Found during:** Task 2
- **Issue:** Test expected `store.List()` to return 2 items (flat only) even though nested `infra/deploy.yaml` existed in the same basePath tree. The test then created a `rootStore` with the identical `dir` path and expected 3 — contradictory since both stores point to the same directory.
- **Fix:** Restructured test: first assert 2 items before creating nested workflow, then assert 3 after creating it, plus verify subdirectory-scoped store returns 1.
- **Files modified:** `internal/store/yaml_test.go`
- **Verification:** `go test ./internal/store/... -v` — all 12 tests pass
- **Committed in:** `21cabaa` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 blocking)
**Impact on plan:** All fixes necessary for correctness. No scope creep.

## Issues Encountered
- testify transitive dependencies (go-spew, go-difflib, gopkg.in/yaml.v3) needed `go mod tidy` to populate go.sum — standard Go toolchain behavior, resolved by running `go mod tidy` before test execution.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Workflow model and YAML store are ready for CLI commands (Plan 01-03: wf add, wf edit, wf rm)
- Template engine (Plan 01-02) already completed — ready to integrate with CLI commands
- Store interface enables in-memory testing for CLI command tests
- No blockers for next plan

---
*Phase: 01-foundation-data-layer*
*Completed: 2026-02-21*
