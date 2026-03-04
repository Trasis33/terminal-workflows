# Phase 10 (Redo): Parameter CRUD & Per-Field AI on Create Workflow UI - Context

**Gathered:** 2026-03-04  
**Status:** Ready for planning

<domain>
## Phase Boundary

Keep the existing **Create Workflow** UI as the baseline and extend it to support full parameter CRUD and per-field AI.
Do not introduce or rely on a separate two-column param-editor shell for this phase.

</domain>

<decisions>
## Implementation Decisions

### Visual/layout baseline (non-negotiable)
- Reuse and extend the existing Create Workflow panel structure and styling.
- Keep colored section headers, comfortable spacing, and the large command input area.
- Workflow metadata and parameters must appear in one continuous editor surface.
- Do not replace the create-flow UI with a split-pane parameter-first layout.

### Section structure and continuity
- Render sections in this order: Workflow metadata -> Command -> Parameters.
- Parameters remain visibly present while editing (no disappearing/collapsing away from the user unexpectedly).
- Keep navigation and editing inside the same panel so the user does not feel context switches.

### Parameter CRUD behavior
- Support add/remove/rename/type-change/edit metadata within this same create-flow surface.
- Rename updates command template references and blocks duplicates.
- Type-change preserves compatible metadata and warns clearly when clearing incompatible metadata.

### Per-field AI behavior
- Place `G [AI]` affordances directly next to eligible parameter fields/actions.
- AI remains field-local and non-blocking (only target field shows loading/error).
- Require at least one seed field (Title, Description, or Command) before generation.
- Command AI updates should stage parameter suggestions in-editor before save.

### Save/cancel and validation behavior
- Keep staged edits with explicit Save/Cancel boundaries (no persistence until Save).
- Unsaved exit must provide: Save, Discard, Keep Editing.
- Validation runs inline during edits and again on Save.
- Navigation between fields is never blocked by errors; Save is blocked only for critical errors.

### Hard constraints to prevent interpretation drift
- If a solution does not visually/readably feel like an extension of Create Workflow, it is out of scope.
- "Unifying models" is acceptable only when it preserves the Create Workflow UI shell and visual language.
- Architecture choices must follow UX intent first for this phase.

</decisions>

<specifics>
## Specific Ideas

- Visual references from user:
  - `/Users/fredriklanga/Desktop/Screenshot 2026-03-04 at 12.56.48.png`
  - `/Users/fredriklanga/Desktop/Screenshot 2026-03-04 at 17.44.03.png`
- Keep parameter affordances obvious and readable (avoid flat all-white cramped presentation).

</specifics>

<deferred>
## Deferred Ideas

- Any redesign that introduces a new editor shell not based on Create Workflow.

</deferred>

---

*Phase: 10-parameter-crud-per-field-ai (redo context)*
*Context gathered: 2026-03-04*
