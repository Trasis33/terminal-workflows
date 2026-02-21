# Phase 2: Quick Picker & Shell Integration - Context

**Gathered:** 2026-02-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can invoke a fuzzy picker via keybinding, search across all workflows, fill parameters inline, and have the completed command pasted into their active shell prompt. Shell integration installed via `wf init zsh/bash/fish`. Binary works standalone without shell integration. Picker launches in under 100ms with 100+ workflows.

</domain>

<decisions>
## Implementation Decisions

### Picker layout & results
- Single line per result — compact, shows more results, fzf-style density
- Each line shows: name, description snippet, and tags — no command in the list
- Preview pane below the result list shows the full command of the currently highlighted workflow
- Tag filtering via `@tag` prefix syntax in the search input — type `@docker` to filter to docker-tagged workflows before fuzzy matching the rest of the query

### Parameter filling flow
- Inline editing within the rendered command — show the full command with parameter placeholders highlighted, user tabs between them to fill
- Parameters with defaults are pre-filled — user tabs through and edits only what they need to override
- No confirmation step after filling — completing the last parameter and pressing Enter/Tab pastes directly to shell
- Zero-parameter workflows paste instantly on select — no confirmation, fastest path

### Claude's Discretion
- Exact keybinding for shell integration (Ctrl+R replacement, Ctrl+W, etc.) — research what's conventional and least conflicting
- Shell paste mechanism (TIOCSTI is deprecated; shell function wrappers needed — see STATE.md blocker)
- Picker height/max visible results
- Color scheme and highlight styling
- Handling of multiline commands in preview pane
- What happens if there's existing text on the shell prompt line
- Empty state when no workflows exist
- Error states (corrupted YAML, missing files)

</decisions>

<specifics>
## Specific Ideas

No specific references — open to standard approaches. The general direction is fzf-like speed and compactness with the addition of a command preview pane and inline parameter editing.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-quick-picker-shell-integration*
*Context gathered: 2026-02-21*
