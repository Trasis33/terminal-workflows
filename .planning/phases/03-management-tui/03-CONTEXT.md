# Phase 3: Management TUI - Context

**Gathered:** 2026-02-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Full-screen TUI launched via `wf manage` for creating, editing, deleting, and browsing workflows organized by folders and tags, with a customizable theme. This is the management companion to the quick picker — for organizing and authoring workflows, not for searching and executing them.

</domain>

<decisions>
## Implementation Decisions

### Browse & navigation layout
- Two-pane layout: sidebar on left, workflow list on right
- Sidebar is switchable between folder view and tag view (toggle, not combined)
- Workflow list shows medium density: name + description + tags on one line
- Bottom preview pane shows full detail (command, args, tags, folder) for the currently selected workflow — always visible
- Full fuzzy search within the management TUI, same quality as the picker

### In-TUI editing experience
- Full-screen form for create and edit — replaces the browse view, dedicated editing screen
- Built-in multi-line text area for command editing (no $EDITOR handoff)
- Tag input with autocomplete from existing tags — suggests matches as you type, allows new tags
- Delete uses a confirmation dialog overlay showing the workflow name

### Organization & filtering
- Folders managed both in-sidebar (keybindings for create/rename/delete) and on-save (type a new folder path to auto-create)
- Move workflows between folders via quick-move shortcut (key + folder picker) and also editable in the edit form
- Empty folders remain visible — they're part of the organizational structure

### Theme customization
- Deep customization: colors, borders, spacing, component visibility — all configurable
- Default theme extends the existing mint/cyan-green palette (ANSI 49, 158, 73) from the picker
- In-TUI settings view for previewing and applying themes — not config-file-only
- Preset themes available as starting points

### Claude's Discretion
- Theme config file format (YAML, TOML, or other — pick what fits best with existing config patterns)
- Keybinding design for sidebar management, quick-move, view switching
- Exact preview pane layout and field ordering
- Loading states, error feedback, and empty state messaging
- Form field ordering and validation approach

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-management-tui*
*Context gathered: 2026-02-22*
