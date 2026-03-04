# Phase 10: Parameter CRUD & Per-Field AI - Context

**Gathered:** 2026-03-04 (updated)
**Status:** Ready for planning

<domain>
## Phase Boundary

Full parameter management (add/remove/rename/type-change/edit metadata) and per-field AI generation, all within the existing Create Workflow UI surface in the manage TUI. Do not introduce a separate editor shell or split-pane layout.

</domain>

<decisions>
## Implementation Decisions

### Visual/layout baseline (non-negotiable)
- Reuse and extend the existing Create Workflow panel structure and styling.
- Keep colored section headers, comfortable spacing, and the large command input area.
- Workflow metadata and parameters must appear in one continuous editor surface.
- Do not replace the create-flow UI with a split-pane parameter-first layout.

### Section structure and continuity
- Render sections in this order: Workflow metadata → Command → Parameters.
- Parameters remain visibly present while editing (no disappearing/collapsing away from the user unexpectedly).
- Keep navigation and editing inside the same panel so the user does not feel context switches.

### Parameter add/remove flow
- Both a visible "+ Add Parameter" button at the bottom of the param list AND a keyboard shortcut as accelerator.
- New parameter appends to the bottom of the list; cursor jumps to its name field immediately.
- Remove requires a confirm dialog before deleting (protect against accidental removal).
- A workflow can have zero parameters — removing the last param is allowed.

### Rename & type-change interaction
- Rename is inline: user edits the name field directly in-place.
- Live command template preview — user sees how `{{old_name}}` changes to `{{new_name}}` in real-time as they type.
- Duplicate name is blocked with an inline error: "Parameter name already exists."
- Type selector is inline on each parameter row (dropdown/selector showing current type: plain/enum/dynamic/list).
- Type-change uses soft staging: incompatible metadata is preserved in the editing session. If user switches enum→plain and back to enum before saving, enum options are still there. Only on Save does the system warn about clearing incompatible data, with options to save, discard changes, or keep editing.

### Per-field AI behavior
- All editable fields are AI-eligible: title, description, command template, parameter default values, enum options, dynamic commands.
- Command-to-params AI: user can trigger analysis of an existing command template to suggest parameters to extract.
- Two AI interaction modes:
  - **"Autofill" button** (global) — fills all empty fields at once using context from filled fields. Skips fields the user already filled.
  - **Per-field AI buttons** — user triggers generation on a single specific field, AI uses all other filled fields as context.
- AI suggestions appear as ghost text (dimmed). User accepts with Enter, dismisses with Esc.
- Require at least one seed field (title, description, or command) before generation is available.
- During Autofill: per-field loading spinners on each field being generated. The entire form is locked to the state sent to AI (user cannot edit mid-generation to prevent context mismatch).
- Per-field generation: only the target field shows loading; other fields remain visible but generation is non-blocking.

### Edit surface navigation
- Tab/Shift+Tab navigates between fields in order. Enter confirms a field and moves to next. Arrow keys navigate within a field.
- Parameters use collapsed rows showing name + type. User expands a param to see/edit all its fields (default, enum options, etc.).
- Accordion behavior: only one parameter can be expanded at a time. Expanding another collapses the current one.
- Bottom help bar showing keyboard shortcut hints (e.g., "Tab: next field | Ctrl+N: new param | Ctrl+D: delete param").

### Save/cancel and validation behavior
- Keep staged edits with explicit Save/Cancel boundaries (no persistence until Save).
- Unsaved exit must provide: Save, Discard, Keep Editing.
- Validation runs inline during edits and again on Save.
- Navigation between fields is never blocked by errors; Save is blocked only for critical errors.

### Hard constraints to prevent interpretation drift
- If a solution does not visually/readably feel like an extension of Create Workflow, it is out of scope.
- "Unifying models" is acceptable only when it preserves the Create Workflow UI shell and visual language.
- Architecture choices must follow UX intent first for this phase.

### Claude's Discretion
- Exact keyboard shortcuts for add/remove/other CRUD actions
- Loading skeleton/spinner design for AI generation
- Exact spacing, typography, and color choices within the existing design language
- Error state handling details
- How the "Autofill" button is positioned relative to the form
- Compression of the command template preview during rename

</decisions>

<specifics>
## Specific Ideas

- Visual references from user:
  - `/Users/fredriklanga/Desktop/Screenshot 2026-03-04 at 12.56.48.png`
  - `/Users/fredriklanga/Desktop/Screenshot 2026-03-04 at 17.44.03.png`
- Keep parameter affordances obvious and readable (avoid flat all-white cramped presentation).
- AI ghost suggestions should feel like browser autofill — present but not intrusive.
- Autofill should feel like "smart form completion" — enter a title and description, hit Autofill, and the command + params are suggested.
- Command-to-params extraction should feel like the reverse: paste a command, AI suggests which parts should be parameterized.

</specifics>

<deferred>
## Deferred Ideas

- Any redesign that introduces a new editor shell not based on Create Workflow.
- Parameter reorder (up/down to control fill order) — deferred per REQUIREMENTS.md (PCRD-08).

</deferred>

---

*Phase: 10-parameter-crud-per-field-ai*
*Context gathered: 2026-03-04*
