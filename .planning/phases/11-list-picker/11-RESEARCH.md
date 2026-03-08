# Phase 11: List Picker - Research

**Researched:** 2026-03-07
**Domain:** Bubble Tea v1 single-select list picker parameters in Go
**Confidence:** MEDIUM

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Show each selectable option as the full original output row, not just the extracted value.
- If header lines are skipped, hide them entirely from the picker instead of showing them as disabled or as helper text.
- Use numbered selection during param fill so users choose by index.
- After selection, confirm the final extracted value that will be inserted.
- Authors provide the list source as a single shell command.
- Column extraction is configured with both a field index and a custom delimiter.
- If column extraction is not configured, use the whole selected line as the final value.
- Header skipping should support skipping a configurable number of leading lines; a simple first-line skip can be treated as a convenience form of that same behavior.
- Large result sets should support search/filtering during selection.
- Typing should instantly filter the visible list rather than only jumping between matches.
- Rows in large lists should use trimmed previews so they stay scannable while still preserving context.
- When the list is filtered, renumber the visible results so numbered selection remains simple.
- If no selectable rows remain after header skipping, show a clear empty-state message rather than a blank list.
- If the shell command fails, show an error and stop the param fill flow rather than falling back to manual entry or auto-retrying.
- If parsing settings make extraction unusable for a selected row, show a parse error and let the user pick again.
- Command failures should show a short readable message by default, with more detailed diagnostics available on request.

### Claude's Discretion
- Exact presentation of trimmed row previews.
- Exact search matching rules and interaction details beyond instant filtering.
- Exact mechanism for revealing additional command error details on request.
- Exact wording and visual treatment of confirmations, empty states, and parse errors.

### Deferred Ideas (OUT OF SCOPE)
- Multi-select list picker behavior came up in requirements context, but it remains out of scope for this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LIST-01 | User can define a list picker parameter that runs a shell command and shows output as a selectable list | Extend `store.Arg`/param editor metadata, add shared command execution pipeline, add list-picker substate for picker/manage |
| LIST-02 | User can select one item from the list during param fill | Use dedicated filtered single-select view with visible numbering, cursor navigation, numeric direct selection, and confirmation |
| LIST-03 | User can configure column extraction so only a specific field from the selected line is used as the value | Add deferred extraction helper using raw row + delimiter + 1-based field index; retry on parse failure |
| LIST-04 | User can configure a custom delimiter for column splitting | Persist delimiter in YAML metadata and use literal `strings.Split` extraction semantics |
| LIST-05 | User can configure header line skipping so column headers are not selectable | Apply header skipping before building selectable rows; show explicit empty state if nothing remains |
</phase_requirements>

## Summary

Phase 11 should be planned as an extension of the existing custom Bubble Tea param-fill architecture, not as a new dependency-driven UI rewrite. The codebase already standardizes on Bubble Tea v1, Bubbles `textinput`, Lipgloss styling, and `sahilm/fuzzy`-based filtering. That stack is sufficient to implement the required list picker without introducing `bubbles/list` or `bubbles/table` as core dependencies.

The most important architectural discovery is that current param fill only merges stored defaults from `workflow.Args`; it does **not** merge stored type metadata (`Type`, `DynamicCmd`, etc.) into extracted template params. For list picker to work reliably from Parameter CRUD, Phase 11 must first introduce an authoritative metadata-overlay step by param name. Without that, a saved `type: list` parameter will not affect picker behavior.

The other major planning concern is execution/parsing robustness. Current dynamic loaders in both `internal/picker/paramfill.go` and `internal/manage/execute_dialog.go` duplicate shell-command execution, use `bufio.Scanner`, and do not check `scanner.Err()`. Phase 11 should centralize command execution, row parsing, header skipping, extraction, and diagnostics in shared helpers, then keep Bubble Tea models focused on UI state.

**Primary recommendation:** Reuse the existing custom Bubble Tea + `textinput` + `sahilm/fuzzy` pattern, add shared list-source/extraction helpers, and treat stored arg metadata as the source of truth for `type: list` behavior.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/bubbletea` | v1.3.10 | TUI state machine | Already the project's terminal architecture and explicitly pinned to v1 |
| `github.com/charmbracelet/bubbles/textinput` | v1.0.0 | Filter input and editor fields | Already used throughout picker/manage; supports focus, cursor, validation, suggestions |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Terminal styling | Existing picker/manage visuals already depend on it |
| `github.com/sahilm/fuzzy` | v0.1.1 | Instant fuzzy filtering | Already used in picker search; returns match positions and ranks |
| Go stdlib `os/exec`, `bufio`, `strings` | Go 1.25 project / current docs verified | Shell execution, row scanning, delimiter splitting | No extra dependency needed for command execution and row parsing |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/bubbles/viewport` | v1.0.0 | Scrollable long preview/detail areas | Reuse for expanded error diagnostics if needed |
| `github.com/charmbracelet/bubbles/list` | v1.0.0 | Built-in list UI with filtering | Only if custom rendering becomes too expensive; not recommended as primary path |
| `github.com/charmbracelet/bubbles/table` | v1.0.0 | Tabular row rendering | Only if a true multi-column table becomes necessary later; not needed for this phase |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom `textinput` + filtered slice + Lipgloss rows | `bubbles/list` | `bubbles/list` gives built-in filtering, but current codebase already custom-renders search/results and needs visible renumbering plus tight param-fill integration |
| Full-row list rows | `bubbles/table` | Table renders columns well but does not solve instant filtering or numeric visible-index UX by itself |

**Installation:**
```bash
# No additional dependencies required for the recommended path.
```

## Architecture Patterns

### Recommended Project Structure
```text
internal/
├── params/             # shared param metadata merge + list source helpers
│   ├── metadata.go     # merge template params with stored Arg metadata
│   ├── list_source.go  # run command, split rows, skip headers, diagnostics
│   ├── extract.go      # delimiter/field extraction helpers
│   └── *_test.go
├── picker/
│   ├── paramfill.go    # uses shared helpers, owns picker TUI state
│   └── list_state.go   # focused list-picker subview/state if split out
└── manage/
    ├── execute_dialog.go  # reuses shared helpers
    └── param_editor.go    # adds list metadata fields
```

### Pattern 1: Metadata Overlay by Param Name
**What:** Extract placeholder names from the command template, then overlay matching `workflow.Args` metadata by name so stored `type`, defaults, and list config drive runtime behavior.
**When to use:** Always before param fill in picker and manage execute dialog.
**Example:**
```go
// Project pattern derived from current picker/manage flow.
type FilledParam struct {
	Name           string
	Type           string
	Default        string
	ListCmd        string
	ListDelimiter  string
	ListFieldIndex int
	ListSkipHeader int
}

// Source refs:
// - internal/picker/paramfill.go
// - internal/manage/execute_dialog.go
```

### Pattern 2: Command → Raw Rows → Selection → Extraction
**What:** Keep raw command output rows intact for display, filter those rows, let the user choose from raw rows, then extract the final value only after selection.
**When to use:** For all `type: list` params.
**Why:** Full-row display, header hiding, parse-error retry, and confirmation all depend on preserving the raw row.
**Example:**
```go
type ListRow struct {
	Raw     string // full original row
	Preview string // trimmed UI preview only
}

// On confirm:
// 1. pick raw row
// 2. extract final value
// 3. show confirmation with extracted value
```

### Pattern 3: Dedicated List-Picker Substate
**What:** Treat list selection as its own subview/state within param fill instead of trying to force it into the current inline 5-option enum renderer.
**When to use:** Any list param with loading, filtering, numbering, empty/error handling, and retry.
**Example:**
```go
// Recommended state shape.
type listPickerState struct {
	loading      bool
	filterInput  textinput.Model
	allRows      []ListRow
	visibleRows  []ListRow
	cursor       int
	numberBuffer string
	parseError   string
	confirmValue string
}
```

### Pattern 4: Soft-Staged List Metadata in Param Editor
**What:** Follow the existing Phase 10 soft-staging rule: preserve incompatible metadata while editing, strip it only in `ToArgs()` save output.
**When to use:** When switching between `text`, `enum`, `dynamic`, and `list` in `ParamEditorModel`.
**Example:**
```go
// Source pattern: internal/manage/param_editor.go
// Recommendation: add list metadata fields to paramEntry,
// preserve them in-memory when type changes away from list,
// and emit warnings in ValidateForSave as needed.
```

### Anti-Patterns to Avoid
- **Do not reuse the current inline enum/dynamic option renderer:** it does not satisfy instant filtering, visible renumbering, or confirmation.
- **Do not extract values before selection:** you would lose the full original row and make parse-error retry awkward.
- **Do not treat command failure like dynamic-param failure:** list type must stop the flow, not fall back to manual entry.
- **Do not keep numbering tied to original rows after filter:** numbers must be based on the currently visible list.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Fuzzy matching | Custom scoring/ranking engine | `sahilm/fuzzy` | Already used in-project, returns ranked matches and matched rune positions |
| Text input/cursor behavior | Raw key-by-key editor logic | `bubbles/textinput` | Existing codebase standard; handles focus, cursor, editing, width |
| Shell execution lifecycle | Ad-hoc process management | `exec.CommandContext` + `Output()` | Gives timeout, kill-on-context, and `*exec.ExitError` stderr details |
| Row trimming/display styling | Manual ANSI juggling | Existing Lipgloss row rendering patterns | Keeps visuals consistent with current picker/manage |

**Key insight:** The hard part of this phase is state orchestration and metadata correctness, not terminal widget availability. Reuse the existing stack and centralize the new non-UI logic.

## Common Pitfalls

### Pitfall 1: Stored Arg Metadata Is Ignored at Runtime
**What goes wrong:** A param saved as `type: list` in YAML behaves like plain text because runtime param fill only extracts placeholders from the command template and merges stored defaults.
**Why it happens:** Current code in picker/manage overlays only `Default`, not `Type` or other metadata.
**How to avoid:** Introduce a shared metadata-merge step by param name before any param-fill UI is built.
**Warning signs:** `type: list` is visible in manage editor, but picker still shows a normal text field.

### Pitfall 2: `bufio.Scanner` Has a 64 KiB Token Limit
**What goes wrong:** Long rows can fail to scan, producing incomplete or empty option sets.
**Why it happens:** `bufio.Scanner` defaults to `MaxScanTokenSize` (64 KiB), and current dynamic loaders do not check `scanner.Err()`.
**How to avoid:** Either call `scanner.Buffer(..., max)` and always check `scanner.Err()`, or switch to `bufio.Reader` line reads if very long rows are plausible.
**Warning signs:** Large commands appear to return fewer rows than expected or inexplicably empty results.

### Pitfall 3: Empty Delimiter Does Not Mean “No Split”
**What goes wrong:** `strings.Split(row, "")` splits after each UTF-8 sequence, which is not the intended “whole line” behavior.
**Why it happens:** That is the documented behavior of `strings.Split` with an empty separator.
**How to avoid:** Treat extraction as enabled only when `field_index > 0` **and** `delimiter != ""`; otherwise use the whole row.
**Warning signs:** Extracted values become single characters.

### Pitfall 4: Command Failure and Empty Results Are Different States
**What goes wrong:** A shell failure, header-skip exhaustion, and successful zero rows get conflated into one generic blank list.
**Why it happens:** The current dynamic flow only distinguishes success vs fallback-to-text.
**How to avoid:** Model separate states: `loadError`, `emptyAfterHeaders`, `parseError`, `confirming`.
**Warning signs:** User sees “no options” when the real problem is a failed command.

### Pitfall 5: Visible Numbering Must Be Recomputed After Filtering
**What goes wrong:** The UI shows filtered rows but accepts original indices, causing incorrect selection.
**Why it happens:** Original and filtered slices diverge once search is active.
**How to avoid:** Always parse numeric input against the current visible slice.
**Warning signs:** Choosing `2` after filtering inserts a row that is no longer the second visible entry.

### Pitfall 6: Parse Errors Must Return to Selection, Not Exit the Flow
**What goes wrong:** Selecting a row with too few fields aborts the fill flow or silently inserts the raw row.
**Why it happens:** Extraction is treated as part of final submission rather than a retryable selection step.
**How to avoid:** Keep the list open, show the parse error inline, and let the user choose again.
**Warning signs:** One bad row forces a full restart.

## Code Examples

Verified patterns from official sources:

### Run a shell command with timeout and readable stderr
```go
// Source: https://pkg.go.dev/os/exec
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "sh", "-c", command)
stdout, err := cmd.Output()
if err != nil {
	var exitErr *exec.ExitError
	shortMsg := err.Error()
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		detail := strings.TrimSpace(string(exitErr.Stderr))
		_ = detail // show on demand
	}
	return shortMsg
}

_ = stdout
```

### Reuse the project's fuzzy filtering style
```go
// Source: https://pkg.go.dev/github.com/sahilm/fuzzy
type rowSource []ListRow

func (rs rowSource) String(i int) string { return rs[i].Raw }
func (rs rowSource) Len() int            { return len(rs) }

matches := fuzzy.FindFrom(filter, rowSource(allRows))
```

### Defer field extraction until selection confirmation
```go
// Source: https://pkg.go.dev/strings
func extractValue(raw, delim string, fieldIndex int) (string, error) {
	if fieldIndex <= 0 || delim == "" {
		return raw, nil
	}
	parts := strings.Split(raw, delim)
	if fieldIndex > len(parts) {
		return "", fmt.Errorf("field %d not found", fieldIndex)
	}
	return strings.TrimSpace(parts[fieldIndex-1]), nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `bubbles` v1 as latest | `bubbles` v2 exists, but this project stays on v1 | v2 released Feb 24, 2026 | Do not plan a v2 migration into this phase |
| Inline enum/dynamic mini-list in param fill | Dedicated filtered subview for list params | Needed now for LIST-01..05 | Supports numbering, filter, retry, confirmation |
| Stored defaults only overlaid at runtime | Full stored arg metadata overlay by name | Required for list type | Makes Parameter CRUD authoritative for param behavior |

**Deprecated/outdated:**
- Reintroducing `charmbracelet/huh`: removed in Phase 10; current editor architecture is custom Bubble Tea.
- Manual-entry fallback on list command failure: explicitly forbidden by locked decisions.

## Open Questions

1. **What should field index base be in author-facing UI?**
   - What we know: requirement says “field index,” but does not specify 0-based vs 1-based.
   - What's unclear: exact UX wording.
   - Recommendation: use **1-based** indexing in UI and store `0` as “unset” so YAML `omitempty` remains useful.

2. **Should delimiter parsing support quoted CSV-style fields in this phase?**
   - What we know: locked scope requires custom delimiter + field index, but says nothing about quote-aware parsing.
   - What's unclear: whether users expect `"a,b"` to remain one field.
   - Recommendation: keep this phase literal-delimiter-based with `strings.Split`; document quoted/escaped parsing as future work if needed.

## Sources

### Primary (HIGH confidence)
- Project code inspection:
  - `internal/picker/paramfill.go` - current dynamic loading, inline option UI, default-only overlay
  - `internal/manage/execute_dialog.go` - duplicated dynamic execution path in manage flow
  - `internal/manage/param_editor.go` - current `list` type placeholder and soft-staging pattern
  - `internal/store/workflow.go` - current persisted arg schema lacks list metadata
  - `internal/picker/search.go` - existing `sahilm/fuzzy` integration pattern
- `https://pkg.go.dev/github.com/charmbracelet/bubbles/textinput` - focus/input model API
- `https://pkg.go.dev/github.com/sahilm/fuzzy` - `FindFrom`, ranking, match metadata
- `https://pkg.go.dev/os/exec` - `CommandContext`, `Output`, `ExitError.Stderr`
- `https://pkg.go.dev/bufio` - `Scanner`, buffer limit, `Scanner.Buffer`, `Scanner.Err`
- `https://pkg.go.dev/strings` - `Split`, `TrimSpace`, empty-separator semantics
- `https://pkg.go.dev/github.com/charmbracelet/bubbles/list` - verified alternative list component capabilities
- `https://pkg.go.dev/github.com/charmbracelet/bubbles/table` - verified alternative table component capabilities
- `https://github.com/charmbracelet/bubbles/releases` - confirms v2 release exists while project remains on v1

### Secondary (MEDIUM confidence)
- None.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - current dependencies already exist in `go.mod` and official docs confirm APIs
- Architecture: MEDIUM - reuse path is clear, but list substate and metadata merge require non-trivial integration across picker/manage/editor
- Pitfalls: HIGH - most are directly evidenced by current code and verified stdlib behavior

**Research date:** 2026-03-07
**Valid until:** 2026-04-06
