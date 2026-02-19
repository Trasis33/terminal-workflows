# Requirements: wf

**Defined:** 2026-02-19
**Core Value:** Users can find and execute any saved command workflow in under 3 seconds, with arguments filled in inline and the result pasted to their prompt.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Workflow Storage

- [ ] **STOR-01**: User can create a new workflow with command, description, and tags
- [ ] **STOR-02**: User can edit an existing workflow's command, description, tags, and arguments
- [ ] **STOR-03**: User can delete a workflow
- [ ] **STOR-04**: User can save multiline commands as a single workflow
- [ ] **STOR-05**: User can register the previous shell command as a new workflow
- [ ] **STOR-06**: Workflows are stored as human-readable YAML files in `~/.config/wf/`

### Parameters

- [ ] **PARM-01**: User can define `{{named}}` parameters in workflow commands
- [ ] **PARM-02**: User can set default values for parameters
- [ ] **PARM-03**: User can define enum parameters with predefined option lists
- [ ] **PARM-04**: User can define dynamic parameters populated by shell command output
- [ ] **PARM-05**: Named parameters used multiple times in a command auto-fill from a single input
- [ ] **PARM-06**: User fills parameters inline with the full command visible

### Search

- [ ] **SRCH-01**: User can fuzzy search workflows by name, description, tags, and command content
- [ ] **SRCH-02**: User can filter search results by tag before fuzzy matching
- [ ] **SRCH-03**: Search results display workflow name, description, tags, and command preview

### Quick Picker

- [ ] **PICK-01**: User can invoke a fuzzy picker overlay via shell keybinding
- [ ] **PICK-02**: Picker starts in under 100ms
- [ ] **PICK-03**: User can search, select a workflow, fill parameters, and paste to prompt in one flow
- [ ] **PICK-04**: User can copy a workflow command to clipboard instead of pasting to prompt

### Management TUI

- [ ] **MTUI-01**: User can launch a full-screen TUI for managing workflows
- [ ] **MTUI-02**: User can create, edit, and delete workflows from the TUI
- [ ] **MTUI-03**: User can browse workflows organized by folders and tags
- [ ] **MTUI-04**: User can customize TUI theme (colors, layout)

### Shell Integration

- [ ] **SHEL-01**: User can install shell integration via `wf init zsh/bash/fish`
- [ ] **SHEL-02**: Shell integration adds a keybinding to invoke the quick picker
- [ ] **SHEL-03**: Selected workflow command is pasted into the user's active prompt
- [ ] **SHEL-04**: Shell integration works with PowerShell on Windows
- [ ] **SHEL-05**: Binary works standalone on macOS, Linux, and Windows

### Organization

- [ ] **ORGN-01**: User can assign tags to workflows
- [ ] **ORGN-02**: User can organize workflows in folders on disk
- [ ] **ORGN-03**: User can share workflow collections via git

### AI Features

- [ ] **AIFL-01**: User can generate a workflow from a natural language description via Copilot SDK
- [ ] **AIFL-02**: User can auto-fill workflow metadata (name, description, arg types) via AI
- [ ] **AIFL-03**: AI features degrade gracefully when Copilot SDK is unavailable

### Import

- [ ] **IMPT-01**: User can import workflows from Pet TOML format
- [ ] **IMPT-02**: User can import workflows from Warp YAML format

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Community & Sync

- **COMM-01**: User can browse and import community-maintained workflow packs
- **COMM-02**: Workflows auto-sync on create/edit (without manual git push)

### Advanced Features

- **ADVN-01**: User can use workflows from within tmux panes
- **ADVN-02**: User can store expected output alongside commands for reference
- **ADVN-03**: User can mask sensitive parameter input (password type)
- **ADVN-04**: User can tab-complete file paths during parameter filling

### Extensibility

- **EXTN-01**: Plugin system for extending wf functionality

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Full shell replacement | Scope trap — users are loyal to their shells, wf integrates with them |
| Cloud sync service | Infrastructure burden — git handles sharing without vendor lock-in |
| Binary/database storage | Prevents version control and human editing of workflows |
| GUI/Electron app | Terminal users want terminal-native tools, not desktop apps |
| Real-time collaboration | Enterprise SaaS scope — share via git repos instead |
| Complex workflow orchestration (DAGs, conditionals) | Becomes a CI/CD tool — use task, just, or make for that |
| Shell history mining | Atuin/fzf already handle this — wf is for curated workflows |
| In-app execution sandboxing | Enormous complexity for marginal safety — show command, trust user |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| STOR-01 | — | Pending |
| STOR-02 | — | Pending |
| STOR-03 | — | Pending |
| STOR-04 | — | Pending |
| STOR-05 | — | Pending |
| STOR-06 | — | Pending |
| PARM-01 | — | Pending |
| PARM-02 | — | Pending |
| PARM-03 | — | Pending |
| PARM-04 | — | Pending |
| PARM-05 | — | Pending |
| PARM-06 | — | Pending |
| SRCH-01 | — | Pending |
| SRCH-02 | — | Pending |
| SRCH-03 | — | Pending |
| PICK-01 | — | Pending |
| PICK-02 | — | Pending |
| PICK-03 | — | Pending |
| PICK-04 | — | Pending |
| MTUI-01 | — | Pending |
| MTUI-02 | — | Pending |
| MTUI-03 | — | Pending |
| MTUI-04 | — | Pending |
| SHEL-01 | — | Pending |
| SHEL-02 | — | Pending |
| SHEL-03 | — | Pending |
| SHEL-04 | — | Pending |
| SHEL-05 | — | Pending |
| ORGN-01 | — | Pending |
| ORGN-02 | — | Pending |
| ORGN-03 | — | Pending |
| AIFL-01 | — | Pending |
| AIFL-02 | — | Pending |
| AIFL-03 | — | Pending |
| IMPT-01 | — | Pending |
| IMPT-02 | — | Pending |

**Coverage:**
- v1 requirements: 36 total
- Mapped to phases: 0
- Unmapped: 36

---
*Requirements defined: 2026-02-19*
*Last updated: 2026-02-19 after initial definition*
