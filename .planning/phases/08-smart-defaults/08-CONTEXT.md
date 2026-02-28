# Phase 8: Smart Defaults - Context

**Gathered:** 2026-02-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Users never retype parameter values they've already entered. When saving a command via `wf register`, the original literal values are preserved as parameter defaults. When filling parameters in the picker, defaults are pre-populated and can be accepted with Enter. Creating new param types, value history/recency tracking, or cross-workflow param sharing are out of scope.

</domain>

<decisions>
## Implementation Decisions

### Default capture on register
- When `wf register` detects values and the user accepts parameterization, the original literal value automatically becomes the `Args[].Default` field — no extra prompt
- Capture all values including potentially sensitive ones (tokens, keys) — user already approved the detection
- Inline template defaults (e.g. `{{host:10.0.0.1}}`) are the author's intent and are immutable during register — they take priority over captured values. Only editable via `wf manage` or direct YAML edit
- If an inline default already exists, the captured value is NOT written to `Args[].Default`
- When a user manually adds a new param (editing template to add `{{port}}`), prompt them for a default value during that flow

### Param fill UX
- Pre-filled input: the default value appears directly in the text input field, ready to accept with Enter
- Visual distinction: pre-filled defaults shown in dim/muted color, user-typed values in normal/bright color
- Edit behavior: cursor placed at end of default value for in-place editing (not replace-on-first-keystroke)
- Always show the param fill screen even when all params have defaults — user presses Enter through each to confirm or modify
- No auto-run or quick-confirm bypass for fully-defaulted workflows

### Enum default handling
- Keep existing `*` asterisk syntax for enum defaults (e.g. `{{env|dev|staging|*prod}}`)
- No change to enum default mechanism — this phase focuses on text param defaults

### Storage & persistence
- Defaults stored directly in the workflow YAML file's `Args[].Default` field — single source of truth, portable, version-controllable
- Per-workflow scoping only — a param named `host` in two different workflows maintains independent defaults
- No separate defaults store or history file

### Claude's Discretion
- YAML update strategy: whether to use AST-level surgical updates (goccy/go-yaml) or full marshal/unmarshal for writing defaults — choose based on feasibility and formatting preservation
- Exact dim/muted color choice for pre-filled defaults in the picker
- Error handling for edge cases (empty defaults, malformed YAML, etc.)

</decisions>

<specifics>
## Specific Ideas

- The flow should be zero-friction: `wf register` → detect → parameterize → defaults auto-captured. No extra steps for the user compared to today, but defaults "just appear" next time they use the workflow
- The param fill screen interaction model: Tab through params, Enter to accept default or after editing, same navigation as today but with pre-populated values

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-smart-defaults*
*Context gathered: 2026-02-28*
