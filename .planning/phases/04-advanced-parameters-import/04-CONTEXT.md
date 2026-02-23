# Phase 4: Advanced Parameters & Import - Context

**Gathered:** 2026-02-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Extend the parameter system with enum and dynamic (shell-sourced) parameter types, add a `wf register` command to capture previous shell commands as workflows, and add `wf import` to convert Pet TOML and Warp YAML collections into wf-compatible workflows.

This phase does NOT add new UI views, AI features, or distribution capabilities.

</domain>

<decisions>
## Implementation Decisions

### Enum parameter syntax & UX
- **Dual definition supported:** Pipe-delimited inline syntax for quick definition (`{{env|dev|staging|*prod}}`), YAML args section for complex cases (descriptions per option, additional metadata)
- **Default marker:** Asterisk prefix marks the default option (e.g., `{{env|dev|staging|*prod}}` — `prod` is default). Without a marker, no pre-selection
- **Picker UX:** Vertical list selection with arrow keys and Enter to confirm — same pattern regardless of option count
- **Custom input:** Claude's discretion on whether to allow freetext fallback or strict option-only selection

### Dynamic parameter behavior
- **Dual definition supported:** Inline bang syntax for quick use (`{{branch!git branch --list}}`), YAML args section for complex or long commands
- **Execution timing:** Claude's discretion on eager vs lazy execution — optimize for perceived snappiness
- **Timeout:** 5 seconds with a visible spinner/loading indicator while the command runs
- **Failure handling:** Show error message, fall back to free-text input for that parameter — never block the user
- **Output parsing:** Each line of stdout becomes one selectable option

### Register previous command flow
- **Dual capture:** `wf register` with no args grabs the last command from shell history; `wf register 'cmd'` takes direct input
- **History browsing:** `wf register --pick` shows a list of recent commands (last 10-20) to choose from
- **Metadata collection:** Interactive prompts for name and description at minimum, tags optional — same pattern as `wf add`
- **Parameter auto-detection:** Auto-detect obvious patterns (IPs, ports, paths, URLs, environment-like values) and suggest them as `{{parameters}}`, user confirms or adjusts before saving
- **Shell history support:** Read from shell-specific history files (~/.zsh_history, ~/.bash_history, ~/.local/share/fish/fish_history)

### Import conflict handling
- **Preview by default:** `wf import` shows a dry-run preview (count, names, potential conflicts) before writing; `--force` flag skips preview
- **Name conflicts:** Per-conflict interactive prompt — user chooses skip, rename, or overwrite for each collision
- **Target folder:** `--folder` flag to specify destination folder, defaults to root workflows directory
- **Unmappable features:** Preserved as YAML comments in the imported workflow file for manual review (e.g., Pet's `output` field becomes `# output: ...`)
- **Parameter syntax translation:** Pet's `<param=default>` converted to `{{param:default}}`; Warp's `{{arg}}` maps directly

### Claude's Discretion
- Whether enum parameters allow freetext fallback or are strict option-only
- Eager vs lazy execution timing for dynamic parameter commands
- Exact auto-detection patterns for parameter recognition in `wf register`
- How many history entries to show in `--pick` mode (10-20 range)

</decisions>

<specifics>
## Specific Ideas

- Enum default uses asterisk prefix marker (`*option`) rather than positional convention — explicit is better
- Dynamic parameter failure degrades gracefully to text input — the user should never be stuck
- `wf register` should feel like a "quick save" — grab command, answer a couple prompts, done
- Import preserves unmappable data as YAML comments rather than silently dropping — nothing should be lost during migration

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-advanced-parameters-import*
*Context gathered: 2026-02-23*
