# Roadmap: wf

## Overview

wf goes from zero to a fully usable terminal workflow manager in 6 phases. The first phase builds the data layer (YAML storage, template engine, config) that everything else depends on. Phase 2 delivers the core value — a fuzzy picker that finds and pastes workflows in under 3 seconds. Phase 3 adds the full management TUI for creating and organizing workflows. Phases 4-6 layer on advanced parameters, AI generation, and cross-platform distribution.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & Data Layer** - YAML storage, template engine, and project scaffold
- [x] **Phase 2: Quick Picker & Shell Integration** - Fuzzy search, parameter filling, paste-to-prompt
- [x] **Phase 3: Management TUI** - Full-screen workflow creation, editing, and browsing
- [x] **Phase 4: Advanced Parameters & Import** - Enum/dynamic params, Pet/Warp import, register previous command
- [x] **Phase 5: AI Integration** - Copilot SDK workflow generation and metadata auto-fill
- [ ] **Phase 6: Distribution & Sharing** - Cross-platform binary, shell completions, git-based sharing

## Phase Details

### Phase 1: Foundation & Data Layer
**Goal**: Users can create, read, update, and delete parameterized workflow YAML files from the CLI, with the template engine correctly parsing `{{named}}` parameters
**Depends on**: Nothing (first phase)
**Requirements**: STOR-01, STOR-02, STOR-03, STOR-04, STOR-06, PARM-01, PARM-02, PARM-05, ORGN-01, ORGN-02
**Success Criteria** (what must be TRUE):
  1. User can run `wf add` and create a workflow YAML file with name, command, description, and tags in `~/.config/wf/`
  2. User can run `wf edit` and modify an existing workflow's command, description, tags, and arguments
  3. User can run `wf rm` and delete a workflow, with the YAML file removed from disk
  4. User can save a multiline command as a single workflow and retrieve it intact (no YAML corruption)
  5. User can define `{{named}}` parameters with defaults in a workflow, and the template engine parses them correctly (including reuse of the same parameter name)
**Plans:** 4 plans
Plans:
  - [x] 01-01-PLAN.md — Go scaffold + Workflow model + YAML store with Norway Problem protection
  - [x] 01-02-PLAN.md — Template engine (parser + renderer) via TDD
  - [x] 01-03-PLAN.md — CLI commands (wf add, wf edit, wf rm)
  - [x] 01-04-PLAN.md — Integration tests + wf list + final verification

### Phase 2: Quick Picker & Shell Integration
**Goal**: Users can invoke a fuzzy picker via keybinding, search across all workflows, fill parameters inline, and have the completed command pasted into their active shell prompt
**Depends on**: Phase 1
**Requirements**: SRCH-01, SRCH-02, SRCH-03, PICK-01, PICK-02, PICK-03, PICK-04, PARM-06, SHEL-01, SHEL-02, SHEL-03, SHEL-05
**Success Criteria** (what must be TRUE):
  1. User can type `wf` (or press a keybinding) and a fuzzy picker appears inline, searching by name, description, tags, and command content
  2. User can filter search results by tag before fuzzy matching, and results display name, description, tags, and command preview
  3. User can select a workflow, fill parameters inline with the full command visible, and the completed command is pasted into their shell prompt
  4. Picker launches in under 100ms with 100+ workflows loaded
  5. User can install shell integration via `wf init zsh/bash/fish` and the binary works standalone on macOS, Linux, and Windows without shell integration
**Plans:** 4 plans
Plans:
  - [x] 02-01-PLAN.md — Install Phase 2 deps + search/filter TDD (fuzzy, tag prefix, WorkflowSource)
  - [x] 02-02-PLAN.md — Picker TUI model (StateSearch + StateParamFill + styles)
  - [x] 02-03-PLAN.md — Shell integration scripts (zsh/bash/fish) + wf init command
  - [x] 02-04-PLAN.md — wf pick command + clipboard + cross-compile + end-to-end verification

### Phase 3: Management TUI
**Goal**: Users can launch a full-screen TUI to create, edit, delete, and browse workflows organized by folders and tags, with a customizable theme
**Depends on**: Phase 1
**Requirements**: MTUI-01, MTUI-02, MTUI-03, MTUI-04
**Success Criteria** (what must be TRUE):
  1. User can run `wf manage` and see a full-screen TUI listing all workflows
  2. User can create, edit, and delete workflows entirely from within the TUI (no CLI fallback needed)
  3. User can browse workflows organized by folders and filter by tags within the TUI
  4. User can customize the TUI theme (colors, layout)
**Plans:** 5 plans
Plans:
  - [x] 03-01-PLAN.md — Theme system, keybindings, root model scaffold, public API
  - [x] 03-02-PLAN.md — Browse view with sidebar (folders/tags), workflow list, preview pane, fuzzy search
  - [x] 03-03-PLAN.md — Create/edit form view using huh form library
  - [x] 03-04-PLAN.md — Dialog overlays (delete, folder ops, move) and theme settings view
  - [x] 03-05-PLAN.md — Cobra command wiring (`wf manage`) and end-to-end verification

### Phase 4: Advanced Parameters & Import
**Goal**: Users can use enum and dynamic parameters in workflows, register previous shell commands as workflows, and import existing collections from Pet and Warp formats
**Depends on**: Phase 2, Phase 3
**Requirements**: PARM-03, PARM-04, STOR-05, IMPT-01, IMPT-02
**Success Criteria** (what must be TRUE):
  1. User can define enum parameters with a predefined list of options and select from them during parameter filling
  2. User can define dynamic parameters that populate options from a shell command's output at fill time
  3. User can run `wf register` (or similar) to capture their previous shell command and save it as a new workflow
  4. User can run `wf import` to convert a Pet TOML file or Warp YAML file into wf-compatible workflows
**Plans:** 8 plans (6 original + 2 gap closure)
Plans:
  - [x] 04-01-PLAN.md — Template parser extension for enum/dynamic params + Arg struct (TDD)
  - [x] 04-02-PLAN.md — Shell history parsers for zsh/bash/fish (TDD)
  - [x] 04-03-PLAN.md — Pet TOML and Warp YAML import converters (TDD)
  - [x] 04-04-PLAN.md — Paramfill UI extension for enum/dynamic selection
  - [x] 04-05-PLAN.md — `wf register` command with history capture and auto-detection
  - [x] 04-06-PLAN.md — `wf import` command with preview/conflict handling + end-to-end verification
  - [x] 04-07-PLAN.md — Pet TOML bare-string tag normalization (gap closure)
  - [x] 04-08-PLAN.md — Shell sidecar for `wf register` current-session capture (gap closure)

### Phase 5: AI Integration
**Goal**: Users can generate workflows from natural language descriptions and auto-fill metadata via the Copilot SDK, with graceful degradation when the SDK is unavailable
**Depends on**: Phase 1
**Requirements**: AIFL-01, AIFL-02, AIFL-03
**Success Criteria** (what must be TRUE):
  1. User can describe what they want in natural language and receive a generated workflow with correct command, parameters, and metadata
  2. User can auto-fill workflow metadata (name, description, argument types) from an existing command via AI
  3. All wf features work normally when the Copilot SDK is unavailable — AI commands show a clear "unavailable" message, nothing else breaks
**Plans:** 3 plans
Plans:
  - [x] 05-01-PLAN.md — AI package foundation (Generator interface, Copilot SDK, availability, prompts, config)
  - [x] 05-02-PLAN.md — CLI commands (`wf generate` + `wf autofill`) with graceful degradation
  - [x] 05-03-PLAN.md — TUI AI integration (keybindings, async execution, form pre-fill)

### Phase 6: Distribution & Sharing
**Goal**: Users can share workflow collections via git, use shell completions, and install wf on Windows with PowerShell support
**Depends on**: Phase 2
**Requirements**: ORGN-03, SHEL-04
**Success Criteria** (what must be TRUE):
  1. User can point wf at a git repo of workflows and search/use them alongside local workflows
  2. User can install and use shell integration with PowerShell on Windows
**Plans:** 4 plans
Plans:
  - [x] 06-01-PLAN.md — Source manager + git operations + config SourcesDir
  - [ ] 06-02-PLAN.md — RemoteStore + MultiStore (read-only remote, aggregating local+remote)
  - [ ] 06-03-PLAN.md — Source CLI commands + wire MultiStore into pick/manage/list
  - [ ] 06-04-PLAN.md — PowerShell integration + Windows build tags for openTTY

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6
Note: Phases 3 and 5 only depend on Phase 1, so they could run after Phase 2 in any order. Phase 4 depends on both 2 and 3.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & Data Layer | 4/4 | ✅ Complete | 2026-02-21 |
| 2. Quick Picker & Shell Integration | 4/4 | ✅ Complete | 2026-02-22 |
| 3. Management TUI | 5/5 | ✅ Complete | 2026-02-22 |
| 4. Advanced Parameters & Import | 8/8 | ✅ Complete | 2026-02-24 |
| 5. AI Integration | 3/3 | ✅ Complete | 2026-02-24 |
| 6. Distribution & Sharing | 1/4 | 🔄 In progress | - |

## Requirement Coverage

| Requirement | Phase |
|-------------|-------|
| STOR-01 | Phase 1 |
| STOR-02 | Phase 1 |
| STOR-03 | Phase 1 |
| STOR-04 | Phase 1 |
| STOR-05 | Phase 4 |
| STOR-06 | Phase 1 |
| PARM-01 | Phase 1 |
| PARM-02 | Phase 1 |
| PARM-03 | Phase 4 |
| PARM-04 | Phase 4 |
| PARM-05 | Phase 1 |
| PARM-06 | Phase 2 |
| SRCH-01 | Phase 2 |
| SRCH-02 | Phase 2 |
| SRCH-03 | Phase 2 |
| PICK-01 | Phase 2 |
| PICK-02 | Phase 2 |
| PICK-03 | Phase 2 |
| PICK-04 | Phase 2 |
| MTUI-01 | Phase 3 |
| MTUI-02 | Phase 3 |
| MTUI-03 | Phase 3 |
| MTUI-04 | Phase 3 |
| SHEL-01 | Phase 2 |
| SHEL-02 | Phase 2 |
| SHEL-03 | Phase 2 |
| SHEL-04 | Phase 6 |
| SHEL-05 | Phase 2 |
| ORGN-01 | Phase 1 |
| ORGN-02 | Phase 1 |
| ORGN-03 | Phase 6 |
| AIFL-01 | Phase 5 |
| AIFL-02 | Phase 5 |
| AIFL-03 | Phase 5 |
| IMPT-01 | Phase 4 |
| IMPT-02 | Phase 4 |

**Coverage: 36/36 (100%)**
