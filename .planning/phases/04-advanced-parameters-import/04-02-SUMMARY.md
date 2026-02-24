---
phase: 04-advanced-parameters-import
plan: 02
subsystem: history-parsing
tags: [zsh, bash, fish, history, parser, tdd]

dependency-graph:
  requires: []
  provides: [HistoryReader-interface, shell-detection, zsh-parser, bash-parser, fish-parser]
  affects: [04-05, 04-06]

tech-stack:
  added: []
  patterns: [interface-based-readers, factory-constructor, shared-helper-functions]

key-files:
  created:
    - internal/history/history.go
    - internal/history/zsh.go
    - internal/history/bash.go
    - internal/history/fish.go
    - internal/history/detect.go
    - internal/history/history_test.go
  modified: []

decisions:
  - id: 04-02-D1
    decision: "Parse fish history line-by-line, not with YAML parser"
    reason: "Fish uses ad-hoc pseudo-YAML that breaks real YAML parsers"
  - id: 04-02-D2
    decision: "Bash # lines that aren't pure digits treated as regular commands"
    reason: "Users may have commands starting with # in their history"
  - id: 04-02-D3
    decision: "Mixed zsh format = plain entries first, then extended (not interleaved)"
    reason: "Reflects real zsh behavior when EXTENDED_HISTORY is enabled mid-session"
  - id: 04-02-D4
    decision: "Extract shared lastN/last helpers instead of duplicating across readers"
    reason: "DRY — identical reverse+slice logic was in all three readers"

metrics:
  duration: "~6 min"
  completed: "2026-02-24"
  tests: 28
  test-coverage: "all parsers, detection, edge cases"
---

# Phase 4 Plan 2: Shell History Parsers Summary

**TDD shell history parsing for zsh/bash/fish with auto-detection — 28 tests, 3 parsers, shared reader helpers**

## What Was Built

New `internal/history/` package providing shell history file parsing for three shells:

- **HistoryReader interface** — `LastN(n)` returns last n commands newest-first, `Last()` returns most recent
- **Zsh parser** — extended format (`: timestamp:duration;command`), plain format, mixed format, multiline continuation, metafied byte decoding (`unmetafy`)
- **Bash parser** — plain format (one per line), timestamped format (`#epoch` + command), non-numeric `#` lines treated as commands
- **Fish parser** — line-by-line pseudo-YAML parsing (`- cmd:` / `when:` / `paths:` ignored)
- **Shell detection** — `DetectShell()` from `$SHELL` basename, `NewReader()` factory with `$HISTFILE` override
- **Shared helpers** — `lastN()` and `last()` eliminate duplication across all three reader implementations

## TDD Execution

| Phase | Commit | Description |
|-------|--------|-------------|
| RED | `0b967b5` | 28 failing tests covering all parsers, detection, edge cases |
| GREEN | `8008a6d` | All implementations — 28/28 tests pass, go vet clean |
| REFACTOR | `447042d` | Extract shared lastN/last helpers, merge duplicate zsh branches |

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 04-02-D1 | Parse fish line-by-line, not YAML | Fish pseudo-YAML breaks real parsers |
| 04-02-D2 | Non-numeric bash `#` lines = commands | Real history can contain `#comments` as commands |
| 04-02-D3 | Mixed zsh = plain-first then extended | Matches real zsh EXTENDED_HISTORY enable mid-session |
| 04-02-D4 | Shared lastN/last helpers in history.go | DRY — eliminated identical code across 3 readers |

## Deviations from Plan

None — plan executed exactly as written.

## Test Results

```
28/28 PASS  github.com/fredriklanga/wf/internal/history
go vet: clean
```

Key test coverage:
- Zsh: extended, plain, mixed, multiline, metafied, empty, LastN
- Bash: plain, timestamped, empty lines, non-numeric hash, LastN
- Fish: standard, no-when, paths-ignored, LastN
- Detection: zsh, fish, bash, empty, unknown fallback
- Edge cases: LastN > available, Last on empty, LastN(0)

## Next Phase Readiness

This package is consumed by:
- **04-05** (`wf register`) — uses `NewReader()` + `LastN()` to capture previous commands
- **04-06** (`wf import`) — may use detection for default shell context

No blockers for downstream plans.
