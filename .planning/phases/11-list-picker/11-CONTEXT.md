# Phase 11: List Picker - Context

**Gathered:** 2026-03-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a single-select list picker parameter type that runs a shell command during param fill, presents the resulting output as selectable options, and lets the parameter author control how the selected row becomes the final value through column extraction, custom delimiter, and header-line skipping. Multi-select and other new picker capabilities are out of scope for this phase.

</domain>

<decisions>
## Implementation Decisions

### User-facing selection experience
- Show each selectable option as the full original output row, not just the extracted value.
- If header lines are skipped, hide them entirely from the picker instead of showing them as disabled or as helper text.
- Use numbered selection during param fill so users choose by index.
- After selection, confirm the final extracted value that will be inserted.

### Parameter author setup
- Authors provide the list source as a single shell command.
- Column extraction is configured with both a field index and a custom delimiter.
- If column extraction is not configured, use the whole selected line as the final value.
- Header skipping should support skipping a configurable number of leading lines; a simple first-line skip can be treated as a convenience form of that same behavior.

### Large result handling
- Large result sets should support search/filtering during selection.
- Typing should instantly filter the visible list rather than only jumping between matches.
- Rows in large lists should use trimmed previews so they stay scannable while still preserving context.
- When the list is filtered, renumber the visible results so numbered selection remains simple.

### Empty and failure states
- If no selectable rows remain after header skipping, show a clear empty-state message rather than a blank list.
- If the shell command fails, show an error and stop the param fill flow rather than falling back to manual entry or auto-retrying.
- If parsing settings make extraction unusable for a selected row, show a parse error and let the user pick again.
- Command failures should show a short readable message by default, with more detailed diagnostics available on request.

### Claude's Discretion
- Exact presentation of trimmed row previews.
- Exact search matching rules and interaction details beyond instant filtering.
- Exact mechanism for revealing additional command error details on request.
- Exact wording and visual treatment of confirmations, empty states, and parse errors.

</decisions>

<specifics>
## Specific Ideas

- The picker should optimize for fast terminal selection: visible numbered choices, immediate filtering, and a confirmation that emphasizes the final inserted value rather than the raw row.
- Parsing controls should stay explicit for authors: command, delimiter, field index, and header skipping are the meaningful knobs that need to be available.

</specifics>

<deferred>
## Deferred Ideas

- Multi-select list picker behavior came up in requirements context, but it remains out of scope for this phase.

</deferred>

---

*Phase: 11-list-picker*
*Context gathered: 2026-03-07*
