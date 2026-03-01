# Phase 9: Execute in Manage - Context

**Gathered:** 2026-03-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can trigger the param fill flow on any workflow from the manage TUI browse view, fill parameters, and then either copy the completed command to clipboard (staying in manage) or exit manage with the command pasted to their shell prompt. This is not about "executing" commands — it's about resolving parameterized commands and getting them out via clipboard or paste-to-prompt, same as `wf pick` but from within manage.

</domain>

<decisions>
## Implementation Decisions

### Trigger & entry point
- Enter key triggers the param fill / action flow on the currently selected workflow in browse
- Works on any visible workflow regardless of folder, tag filter, or search state
- Zero-param workflows skip param fill entirely and go straight to the action menu

### Param fill integration
- Param fill appears as a dialog overlay, centered over the browse view
- Browse view is dimmed behind the dialog but remains visible for spatial context
- Dialog is medium-sized (~60-70% terminal width), grows vertically as needed for params
- Dialog title shows the workflow name so the user knows what they're filling
- Param fill UX is adapted for the dialog context (not a direct copy of the picker's layout) — command preview, inputs, and navigation should fit the dialog form factor

### Post-fill actions
- After param fill completes (or immediately for zero-param workflows), the dialog shows the final rendered command and an action menu
- Three actions available: Copy to clipboard / Paste-to-prompt / Cancel
- Copy to clipboard: dialog closes, user returns to browse, flash message confirms "Copied!"
- Paste-to-prompt: manage exits cleanly and the completed command is placed on the user's shell prompt line (same stdout mechanism as `wf pick` shell integration)
- Cancel: dialog closes, user returns to browse, no action taken

### State & navigation flow
- Esc cancels the param fill dialog at any point, all param fill state is discarded
- Browse state is fully preserved on return from dialog (cursor position, sidebar selection, active search query)
- Flash message (~1.5s) confirms clipboard copy after returning to browse
- Enter hint is added to the browse view hints bar so users can discover the feature

### Claude's Discretion
- Exact dialog border/styling within the existing manage theme system
- How to adapt the picker's param fill layout for the dialog (vertical arrangement, preview placement)
- Internal param fill navigation (tab/shift-tab/enter/up-down for enums) — match picker conventions where sensible
- Flash message styling and duration
- How the action menu is rendered inside the dialog (list, buttons, key hints)

</decisions>

<specifics>
## Specific Ideas

- The paste-to-prompt mechanism should use the same stdout approach as `wf pick` — manage writes the final command to stdout, and the shell integration function captures it and places it on the prompt line
- The dialog overlay approach is consistent with manage's existing dialog system (delete confirm, folder ops, move, AI generate) — reuse that infrastructure
- Param fill should support all three param types already in the picker: text (with defaults), enum (up/down selection), and dynamic (shell command populated)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-execute-in-manage*
*Context gathered: 2026-03-01*
