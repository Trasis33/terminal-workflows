# Phase 1: Foundation & Data Layer - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can create, read, update, and delete parameterized workflow YAML files from the CLI, with the template engine correctly parsing `{{named}}` parameters. This phase delivers the data layer and basic CLI commands — no fuzzy picker, no TUI, no shell integration.

</domain>

<decisions>
## Implementation Decisions

### Workflow file structure
- One file per workflow — no multi-workflow collection files
- Multiline commands stored using YAML block scalar (`|`) to preserve newlines natively
- Fields and filename convention at Claude's discretion (see below), but must include at minimum: name, command, description, tags
- Both inline `{{name}}` in command strings and a separate `args` section for detailed parameter definitions (defaults, descriptions)

### CLI interaction style
- `wf add`: flags for all fields + interactive prompts as fallback when flags are missing
- `wf edit`: open in `$EDITOR` by default, flags for quick field updates (e.g., `wf edit deploy --tag docker`)
- `wf rm`: confirmation prompt by default, `--force` flag to skip confirmation
- Feedback after operations: minimal one-liner ("Created deploy-staging", "Deleted deploy-staging")

### Tag & folder organization
- Tags: flat list of strings (e.g., `[docker, deploy, staging]`) — no hierarchy in Phase 1, extend later if needed
- Folders: filesystem directories under `~/.config/wf/workflows/` (e.g., `~/.config/wf/workflows/infra/docker/deploy.yaml`)
- Folder nesting: max 2 levels deep
- Root storage path: `~/.config/wf/workflows/`

### Parameter syntax & defaults
- Parameters defined inline in command string: `{{name}}` or `{{name:default_value}}`
- Detailed parameter definitions in a separate `args` section of the YAML (name, default, description)
- Same parameter name used multiple times = one parameter, fills all occurrences
- When rendering: always prompt the user for every parameter, showing default as suggestion
- Missing parameter with no default: warn but still render with placeholder visible in output

### Claude's Discretion
- Exact YAML fields beyond the core four (name, command, description, tags) — include what's relevant for Phase 1 functionality
- Filename convention for workflow files (name-based slug, ID prefix, etc.)
- Exact interactive prompt flow for `wf add`
- Error message wording and formatting
- YAML marshalling approach for the Norway Problem (typed string fields, DoubleQuotedStyle — per STATE.md blocker)

</decisions>

<specifics>
## Specific Ideas

- Workflows stored at `~/.config/wf/workflows/` following XDG convention
- YAML block scalars (`|`) chosen specifically for multiline readability — users should be able to open workflow files in any editor and understand them immediately
- The `wf edit` command should feel like `kubectl edit` — open the resource in your editor, save, done
- Dual parameter definition (inline + args section) gives quick workflows simplicity while allowing power users to add descriptions and types to parameters

</specifics>

<deferred>
## Deferred Ideas

- Full TUI editing experience for `wf edit` — Phase 3 (Management TUI) will provide this
- Hierarchical/nested tags — evaluate need after Phase 2 usage
- Enum and dynamic parameter types — Phase 4

</deferred>

---

*Phase: 01-foundation-data-layer*
*Context gathered: 2026-02-20*
