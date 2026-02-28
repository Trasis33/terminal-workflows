# Phase 7: Polish & Terminal Compat - Context

**Gathered:** 2026-02-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Users see polished, syntax-highlighted shell commands and can use wf reliably across all terminals including Warp. Covers: shell syntax highlighting in TUI views and CLI output, manage UX refinements (folder navigation, scroll boundaries), and Warp terminal compatibility with custom keybinding support. No new features — this is polish and compatibility for the existing v1.0.

</domain>

<decisions>
## Implementation Decisions

### Syntax highlighting style
- Full AST-level coloring: command names, flags, arguments, variable expansions, keywords (if/for/while), pipes, redirects, strings, comments — all distinctly colored
- Template parameters (e.g., `{{name}}`, `{{port}}`) rendered in a distinct accent color that clearly separates them from regular command text
- Color scheme follows the currently selected wf theme in settings (theme-aware, not hardcoded)
- Three surfaces get highlighting:
  1. **Manage preview pane** — full AST highlighting, theme-aware colors
  2. **Picker/browse preview** — full AST highlighting, theme-aware colors
  3. **`wf list` CLI output** — simpler ANSI-native highlighting that separates folder, workflow name, and tags (uses terminal-native colors, not wf theme)

### Folder auto-display behavior
- Selecting a folder in the sidebar **instantly filters** the workflow list — no extra keypress needed
- **Folder breadcrumb** shown at the top of the workflow list indicating the active filter (e.g., "docker/" or "All Workflows")
- **Explicit "All" entry** at the top of the sidebar folder list to clear the filter and show all workflows
- **Empty state message** when a selected folder has no workflows (e.g., "No workflows in this folder")

### Preview scroll boundaries
- **Fixed-height preview pane** (not dynamic) — prevents layout destabilization where other panels get pushed off-screen
- Scrolling stops **tight to content** — last line of the command reaches the bottom of the viewport, no blank overscroll
- **Line count / percentage indicator** shown (e.g., in the preview pane border or corner) to communicate scroll position and signal more content exists

### Warp detection & keybinding
- **Auto-detect Warp** via `$TERM_PROGRAM` at `wf init` time — if "WarpTerminal" is detected, automatically use fallback key instead of Ctrl+G
- **Default fallback key for Warp: Ctrl+O** — rarely conflicts with shell functionality
- **Inform user via both channels:**
  1. **Stderr message** during `wf init` execution (e.g., "Warp detected: binding to Ctrl+O instead of Ctrl+G")
  2. **Comment block** in the generated shell init script explaining the bound key and how to change it
- **`--key` flag** accepts modifier+letter combos (ctrl+<letter>, alt+<letter>)
- **Block dangerous key combos:** ctrl+c, ctrl+d, ctrl+z, and other keys that would break shell editing
- `--key` flag works for all terminals, not just Warp — any user can customize their keybinding

### Claude's Discretion
- Exact syntax highlighting library choice (chroma, tree-sitter, or custom tokenizer)
- Specific color assignments per token type within the theme system
- Preview pane fixed height value (current 6 lines or adjusted)
- Exact format/placement of scroll percentage indicator
- Warp detection edge cases (e.g., SSH through Warp, nested terminals)
- Which specific dangerous key combos to block beyond ctrl+c/d/z

</decisions>

<specifics>
## Specific Ideas

- The `wf list` CLI output should use simpler highlighting than TUI — just visually separate folder, workflow name, and tags with terminal-native ANSI colors
- Preview pane dynamic height was explicitly rejected because it causes the top of the app to disappear (pushes other panels off-screen)
- The "All" sidebar entry is the primary way to clear folder filters (not Escape)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-polish-terminal-compat*
*Context gathered: 2026-02-28*
