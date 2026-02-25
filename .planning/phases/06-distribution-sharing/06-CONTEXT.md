# Phase 6: Distribution & Sharing - Context

**Gathered:** 2026-02-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can share workflow collections via git repos and search/use them alongside local workflows. Users can install and use wf with PowerShell on Windows, including a native .exe binary. Shell completions for all supported shells.

Requirements: ORGN-03 (git-based sharing), SHEL-04 (PowerShell support)

</domain>

<decisions>
## Implementation Decisions

### Git repo source management
- `wf source add <git-url>` with optional `--name <alias>` flag; alias auto-derived from repo name if not provided
- Multiple sources allowed simultaneously (e.g., team repo + community repo + personal backup)
- Auto-detect repo structure — support both flat directories of YAML files and repos mirroring wf's local structure
- Authentication relies on user's existing git config (SSH keys, credential helpers) — no built-in token handling
- Source removal via `wf source remove <alias>`

### Remote vs local workflow UX
- Subtle source distinction — show source label in listings/search results only, not after selection
- Name conflicts resolved by namespacing: remote workflows prefixed with source alias (e.g., `team/deploy` vs `deploy`)
- All sources searched by default in picker and TUI; user can filter to specific source if needed
- Remote editing behavior: Claude's discretion (fork-to-local on edit is the reasonable default)

### Sync & update behavior
- Manual update only — user runs `wf source update [alias]` to pull latest from remote
- Offline: use cached clone seamlessly — remote workflows fully available from last pull
- Update feedback: show diff summary (new workflows, removed, updated)
- Clone storage location: Claude's discretion (XDG data directory is the natural fit given existing convention from [04-08-D1])

### PowerShell integration
- PowerShell 7+ (pwsh) only — no Windows PowerShell 5.1
- `wf init powershell` follows same pattern as other shells: Claude's discretion on exact approach
- Keybinding: Claude's discretion (Ctrl+G for consistency if no conflicts, otherwise PS-appropriate alternative)
- Full Windows support: ship .exe binary, handle Windows-specific paths, terminal I/O, and config directories
- Cross-compile with GOOS=windows alongside existing macOS/Linux targets

### Claude's Discretion
- Remote workflow edit behavior (fork-to-local recommended)
- Clone storage directory path
- PowerShell keybinding choice
- PowerShell init snippet approach
- Shell completion implementation details
- Windows-specific codepaths for paths, clipboard, and terminal handling

</decisions>

<specifics>
## Specific Ideas

- Source aliases auto-derived from repo name keeps the common case simple (`wf source add https://github.com/team/workflows` → alias "workflows")
- Namespace-based conflict resolution means users never lose access to either version
- Manual sync avoids any startup latency — critical given the 100ms picker launch target from Phase 2
- Existing XDG convention from Phase 4 sidecar files ([04-08-D1]) should guide clone storage location

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-distribution-sharing*
*Context gathered: 2026-02-25*
