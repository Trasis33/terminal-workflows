---
phase: 01-foundation-data-layer
plan: 02
subsystem: template-engine
tags: [go, regex, parser, renderer, tdd]

dependency_graph:
  requires: []
  provides:
    - "ExtractParams function for parsing {{param}} syntax"
    - "Render function for substituting parameters"
    - "Param type shared across template package"
  affects:
    - "01-03 (CLI commands will use Render for command execution)"
    - "01-04 (integration tests will exercise template engine)"
    - "Phase 2 (picker will call ExtractParams to identify params to fill)"

tech_stack:
  added: []
  patterns:
    - "Regex-based template parsing (regexp.MustCompile)"
    - "Package-level compiled regex for performance"
    - "Ordered deduplication with map+slice"

key_files:
  created:
    - internal/template/parser.go
    - internal/template/renderer.go
    - internal/template/template_test.go
  modified: []

decisions:
  - id: "01-02-D1"
    decision: "Split first colon only for name:default parsing"
    rationale: "Allows defaults containing colons (e.g., URLs like http://localhost:8080)"
  - id: "01-02-D2"
    decision: "Return nil slice (not empty slice) when no params found"
    rationale: "Idiomatic Go — nil slice equals empty in len/range checks, avoids allocation"
  - id: "01-02-D3"
    decision: "Last default wins for duplicate parameter names"
    rationale: "Per plan spec — {{a}} {{a:y}} yields Default:'y' since later occurrence refines"

metrics:
  duration: "~4 minutes"
  completed: "2026-02-21"
---

# Phase 01 Plan 02: Template Engine (Parser + Renderer) Summary

**Regex-based {{name}} and {{name:default}} template parser with substitution renderer, TDD-driven with 17 test cases**

## What Was Built

Two-function template engine for parameterized workflow commands:

- **`ExtractParams(command string) []Param`** — Extracts unique parameters from command strings using regex `\{\{([^}]+)\}\}`. Handles `{{name}}` and `{{name:default}}` syntax, deduplicates by name (last default wins), preserves first-appearance order.

- **`Render(command string, values map[string]string) string`** — Substitutes all `{{...}}` occurrences with provided values. Falls back to default if no value provided, leaves `{{name}}` placeholder if neither value nor default exists.

- **`Param` struct** — Shared type with `Name` and `Default` string fields.

## TDD Execution

### RED Phase (commit: 9a39d56)
Wrote 17 failing test cases (158 lines) covering:
- Basic parameter extraction (2 params)
- Default value extraction
- Deduplication (same name appears twice)
- No parameters / empty string (edge cases)
- Multiple parameters (4 params in one command)
- Default with spaces
- Duplicate with last-default-wins semantics
- Default containing colons (URL edge case)
- Basic substitution
- Value overrides default
- Default used when no value provided
- All occurrences substituted
- Missing param placeholder stays
- No-params passthrough
- Nil values map handling
- Mixed provided and default values

### GREEN Phase (commit: 55bd96a)
Implemented parser.go (54 lines) and renderer.go (25 lines). All 17 tests pass.

### REFACTOR Phase
Skipped — code was already minimal and clean (~79 lines total for both files).

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 01-02-D1 | Split on first colon only | Allows URL defaults like `http://localhost:8080` |
| 01-02-D2 | Return nil slice for no params | Idiomatic Go, avoids unnecessary allocation |
| 01-02-D3 | Last default wins for dupes | Plan spec: `{{a}} {{a:y}}` → Default:"y" |

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

```
=== RUN   TestExtractParams_BasicParams          --- PASS
=== RUN   TestExtractParams_WithDefaults          --- PASS
=== RUN   TestExtractParams_Deduplicated          --- PASS
=== RUN   TestExtractParams_NoParams              --- PASS
=== RUN   TestExtractParams_EmptyString           --- PASS
=== RUN   TestExtractParams_MultipleParams        --- PASS
=== RUN   TestExtractParams_DefaultWithSpaces     --- PASS
=== RUN   TestExtractParams_DuplicateLastDefaultWins --- PASS
=== RUN   TestExtractParams_DefaultWithColon      --- PASS
=== RUN   TestRender_BasicSubstitution            --- PASS
=== RUN   TestRender_ValueOverridesDefault        --- PASS
=== RUN   TestRender_DefaultUsedWhenNoValue       --- PASS
=== RUN   TestRender_AllOccurrencesSubstituted    --- PASS
=== RUN   TestRender_MissingParamPlaceholderStays --- PASS
=== RUN   TestRender_NoParamsPassthrough           --- PASS
=== RUN   TestRender_NilValues                    --- PASS
=== RUN   TestRender_MixedProvidedAndDefault      --- PASS
PASS — 17/17 tests, 0.24s
```

## Line Counts

| File | Lines | Minimum | Status |
|------|-------|---------|--------|
| parser.go | 54 | 30 | ✅ |
| renderer.go | 25 | 20 | ✅ |
| template_test.go | 158 | 100 | ✅ |

## Next Phase Readiness

Plan 01-03 (CLI commands) can now use:
- `template.ExtractParams()` to identify parameters in workflow commands
- `template.Render()` to substitute parameter values before execution
- `template.Param` type for parameter metadata

No blockers for downstream plans.
