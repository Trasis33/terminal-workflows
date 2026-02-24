# Phase 5: AI Integration - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can generate workflows from natural language descriptions and auto-fill workflow metadata via the Copilot SDK. All AI features degrade gracefully when the SDK is unavailable — AI commands show a clear "unavailable" message, nothing else breaks.

</domain>

<decisions>
## Implementation Decisions

### Generation interaction flow
- Three entry points: CLI single-shot, CLI interactive, and TUI-integrated
- `wf generate "description"` — single-shot mode, generates workflow from description + optional context (tags, folder, target shell)
- `wf generate` (no args) — interactive mode, one clarifying question at a time until enough context gathered, then generates
- TUI-integrated — accessible from within `wf manage` as a menu action; two paths: (1) fill in a command and AI generates name/description, (2) provide a description and AI generates name/command
- Input accepts natural language description plus optional context hints (tags, folder, shell)

### Auto-fill trigger & scope
- Two entry points: standalone command for existing workflows + action in TUI
- All metadata fields fillable: name, description, tags, argument types & descriptions
- Fields fillable individually or all at once — user's choice, never automatic
- CLI: flags per field (e.g., `wf autofill my-workflow --name --tags`) with interactive prompt fallback when no flags specified
- Never auto-prompts on save — user must explicitly invoke auto-fill
- TUI: accessible as an action on any workflow (not triggered automatically)

### Unavailability behavior
- AI commands always visible in CLI help output and TUI — not hidden when SDK unavailable
- Lazy SDK check on first AI command invocation (not startup) — no startup penalty
- Error message: one-liner with install hint (e.g., "AI features require GitHub Copilot. Install with: ...")
- TUI shows AI actions normally; selecting one without SDK produces inline error/toast
- Non-AI features completely unaffected — zero coupling

### Output review & editing
- AI output lands in a pre-filled edit form, not a preview-then-decide flow
- CLI path: inline prompts (name, command, description, tags) pre-filled with AI values, user edits inline
- TUI path: existing create/edit form opens pre-filled, all fields visible and editable simultaneously
- Regenerate available with refinement notes — user can say "make it shorter", "add rollback step", etc.
- No separate preview step — edit form IS the review

### Model selection
- Per-task model configuration with global fallback (e.g., generation=gpt-4o, metadata-fill=gpt-4o-mini, fallback=gpt-4o-mini)
- Free/lightweight models preferred for simple tasks (name, description, tags); better models for command generation
- Config lives in config file (~/.config/wf/config.yaml) — source of truth
- TUI settings view can also edit model configuration (extends existing settings)
- No per-command --model flag override — config is sufficient

### Claude's Discretion
- Prompt engineering for generation and auto-fill (system prompts, few-shot examples)
- SDK client initialization and connection handling
- Rate limiting or token budget strategy
- Exact interactive mode question flow (what clarifying questions to ask)
- Error handling for partial AI failures (e.g., generates command but fails on tags)

</decisions>

<specifics>
## Specific Ideas

- TUI integration should support bidirectional use: user provides a command and AI fills metadata, OR user provides a description and AI generates the command — both from the same interface
- Model selection should make it easy to use cheap/free models for lightweight tasks and reserve expensive models for command generation
- Interactive mode should feel like a conversation, not a form — one question at a time, each answer informing the next

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-ai-integration*
*Context gathered: 2026-02-24*
