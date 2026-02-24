# Requirements: wf

**Defined:** 2026-02-19
**Core Value:** Users can find and execute any saved command workflow in under 3 seconds, with arguments filled in inline and the result pasted to their prompt.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Workflow Storage

- [x] **STOR-01**: User can create a new workflow with command, description, and tags
- [x] **STOR-02**: User can edit an existing workflow's command, description, tags, and arguments
- [x] **STOR-03**: User can delete a workflow
- [x] **STOR-04**: User can save multiline commands as a single workflow
- [x] **STOR-05**: User can register the previous shell command as a new workflow
- [x] **STOR-06**: Workflows are stored as human-readable YAML files in `~/.config/wf/`

### Parameters

- [x] **PARM-01**: User can define `{{named}}` parameters in workflow commands
- [x] **PARM-02**: User can set default values for parameters
- [x] **PARM-03**: User can define enum parameters with predefined option lists
- [x] **PARM-04**: User can define dynamic parameters populated by shell command output
- [x] **PARM-05**: Named parameters used multiple times in a command auto-fill from a single input
- [x] **PARM-06**: User fills parameters inline with the full command visible

### Search

- [x] **SRCH-01**: User can fuzzy search workflows by name, description, tags, and command content
- [x] **SRCH-02**: User can filter search results by tag before fuzzy matching
- [x] **SRCH-03**: Search results display workflow name, description, tags, and command preview

### Quick Picker

- [x] **PICK-01**: User can invoke a fuzzy picker overlay via shell keybinding
- [x] **PICK-02**: Picker starts in under 100ms
- [x] **PICK-03**: User can search, select a workflow, fill parameters, and paste to prompt in one flow
- [x] **PICK-04**: User can copy a workflow command to clipboard instead of pasting to prompt

### Management TUI

- [x] **MTUI-01**: User can launch a full-screen TUI for managing workflows
- [x] **MTUI-02**: User can create, edit, and delete workflows from the TUI
- [x] **MTUI-03**: User can browse workflows organized by folders and tags
- [x] **MTUI-04**: User can customize TUI theme (colors, layout)

### Shell Integration

- [x] **SHEL-01**: User can install shell integration via `wf init zsh/bash/fish`
- [x] **SHEL-02**: Shell integration adds a keybinding to invoke the quick picker
- [x] **SHEL-03**: Selected workflow command is pasted into the user's active prompt
- [ ] **SHEL-04**: Shell integration works with PowerShell on Windows
- [x] **SHEL-05**: Binary works standalone on macOS, Linux, and Windows

### Organization

- [x] **ORGN-01**: User can assign tags to workflows
- [x] **ORGN-02**: User can organize workflows in folders on disk
- [ ] **ORGN-03**: User can share workflow collections via git

### AI Features

- [x] **AIFL-01**: User can generate a workflow from a natural language description via Copilot SDK
- [x] **AIFL-02**: User can auto-fill workflow metadata (name, description, arg types) via AI
- [x] **AIFL-03**: AI features degrade gracefully when Copilot SDK is unavailable

### Import

- [x] **IMPT-01**: User can import workflows from Pet TOML format
- [x] **IMPT-02**: User can import workflows from Warp YAML format

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
| STOR-01 | Phase 1 | Complete |
| STOR-02 | Phase 1 | Complete |
| STOR-03 | Phase 1 | Complete |
| STOR-04 | Phase 1 | Complete |
| STOR-05 | Phase 4 | Complete |
| STOR-06 | Phase 1 | Complete |
| PARM-01 | Phase 1 | Complete |
| PARM-02 | Phase 1 | Complete |
| PARM-03 | Phase 4 | Complete |
| PARM-04 | Phase 4 | Complete |
| PARM-05 | Phase 1 | Complete |
| PARM-06 | Phase 2 | Complete |
| SRCH-01 | Phase 2 | Complete |
| SRCH-02 | Phase 2 | Complete |
| SRCH-03 | Phase 2 | Complete |
| PICK-01 | Phase 2 | Complete |
| PICK-02 | Phase 2 | Complete |
| PICK-03 | Phase 2 | Complete |
| PICK-04 | Phase 2 | Complete |
| MTUI-01 | Phase 3 | Complete |
| MTUI-02 | Phase 3 | Complete |
| MTUI-03 | Phase 3 | Complete |
| MTUI-04 | Phase 3 | Complete |
| SHEL-01 | Phase 2 | Complete |
| SHEL-02 | Phase 2 | Complete |
| SHEL-03 | Phase 2 | Complete |
| SHEL-04 | Phase 6 | Pending |
| SHEL-05 | Phase 2 | Complete |
| ORGN-01 | Phase 1 | Complete |
| ORGN-02 | Phase 1 | Complete |
| ORGN-03 | Phase 6 | Pending |
| AIFL-01 | Phase 5 | Complete |
| AIFL-02 | Phase 5 | Complete |
| AIFL-03 | Phase 5 | Complete |
| IMPT-01 | Phase 4 | Complete |
| IMPT-02 | Phase 4 | Complete |

**Coverage:**
- v1 requirements: 36 total
- Mapped to phases: 36
- Unmapped: 0

---
*Requirements defined: 2026-02-19*
*Last updated: 2026-02-24 after Phase 5 completion*
