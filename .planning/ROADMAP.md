# Roadmap: wf v1.1 Polish & Power

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped 2026-02-25)
- 🚧 **v1.1 Polish & Power** - Phases 7-11 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) - SHIPPED 2026-02-25</summary>

See `.planning/milestones/v1.0-ROADMAP.md` for full details.

- [x] **Phase 1: Foundation & Data Layer** - YAML storage, template engine, CLI CRUD
- [x] **Phase 2: Quick Picker & Shell Integration** - Fuzzy picker, param fill, paste-to-prompt
- [x] **Phase 3: Management TUI** - Full-screen browse, create/edit, folders/tags
- [x] **Phase 4: Advanced Parameters & Import** - Enum/dynamic params, register, Pet/Warp import
- [x] **Phase 5: AI Integration** - Generate from NL, auto-fill metadata, graceful degradation
- [x] **Phase 6: Distribution & Sharing** - Git-based sharing, PowerShell/Windows support

**Stats:** 6 phases, 28 plans, 36 requirements, 11,736 LOC

</details>

### 🚧 v1.1 Polish & Power (In Progress)

**Milestone Goal:** Improve manage UX with execute flow and parameter CRUD, add list picker variable type, resolve terminal compatibility issues, add syntax highlighting and smarter default handling.

- [x] **Phase 7: Polish & Terminal Compat** - Syntax highlighting, UX fixes, Warp keybinding (completed 2026-02-28)
- [x] **Phase 8: Smart Defaults** - Preserve and recall parameter defaults (completed 2026-03-01)
- [ ] **Phase 9: Execute in Manage** - Full execute flow inside manage TUI
- [ ] **Phase 10: Parameter CRUD & Per-Field AI** - Add/remove/edit parameters, per-field AI generate
- [ ] **Phase 11: List Picker** - New parameter type with shell command output selection

## Phase Details

### Phase 7: Polish & Terminal Compat
**Goal**: Users see polished, syntax-highlighted commands and can use wf reliably across all terminals including Warp
**Depends on**: Phase 6 (v1.0 complete)
**Requirements**: DISP-01, DISP-02, MGUX-01, MGUX-02, TERM-01, TERM-02
**Success Criteria** (what must be TRUE):
  1. User sees shell syntax highlighted with color in the manage preview pane and browse list
  2. User navigating to a folder in manage sees its contents immediately without pressing an extra key
  3. User scrolling the command preview panel hits a natural stop at the end of content (no blank overscroll)
  4. User on Warp terminal runs `wf init zsh` and gets a working keybinding without manual intervention
  5. User can run `wf init zsh --key ctrl+o` to bind wf to a custom keybinding of their choice
**Plans**: 3 plans
Plans:
- [x] 07-01-PLAN.md — Syntax highlighting package (Chroma v2 + lipgloss token styling)
- [x] 07-02-PLAN.md — Keybinding system + Warp terminal compatibility
- [x] 07-03-PLAN.md — Manage TUI integration (sidebar auto-filter, viewport preview, wf list coloring)

### Phase 8: Smart Defaults
**Goal**: Users never retype parameter values they've already entered — the system remembers and offers previous values as defaults
**Depends on**: Phase 7
**Requirements**: DFLT-01, DFLT-02
**Success Criteria** (what must be TRUE):
  1. User runs `wf register` or saves a command, and the original values from the command appear as parameter defaults in the saved workflow
  2. User filling parameters in the picker sees previous values pre-populated and can accept them with Enter instead of retyping
**Plans**: 1 plan
Plans:
- [x] 08-01-PLAN.md — Default capture in register + picker pre-fill with dim styling

### Phase 9: Execute in Manage
**Goal**: Users can test and execute workflows without leaving the manage TUI — complete param fill, clipboard copy, or paste-to-prompt
**Depends on**: Phase 7
**Requirements**: EXEC-01, EXEC-02, EXEC-03, EXEC-04
**Success Criteria** (what must be TRUE):
  1. User browsing workflows in manage can press a key to start the execute flow on any workflow
  2. User filling parameters inside manage sees the same inline param fill UX as the picker (full command visible, cursor on current param)
  3. User completing param fill in manage can copy the result to clipboard and continue managing workflows
  4. User completing param fill in manage can choose to exit manage and have the completed command pasted to their shell prompt
**Plans**: TBD

### Phase 10: Parameter CRUD & Per-Field AI
**Goal**: Users can fully manage parameters on existing workflows from within the manage TUI — add, remove, rename, change types, edit metadata — and use AI to generate individual field values
**Depends on**: Phase 9
**Requirements**: PCRD-01, PCRD-02, PCRD-03, PCRD-04, PCRD-05, PCRD-06, PCRD-07, PFAI-01, PFAI-02
**Success Criteria** (what must be TRUE):
  1. User can add a new parameter to an existing workflow and remove an existing parameter, all within the manage TUI edit view
  2. User can rename a parameter and change its type between plain, enum, dynamic, and list — the workflow command template updates accordingly
  3. User can edit parameter metadata: default values, enum options, and dynamic commands — without leaving the TUI
  4. User can trigger AI generation on a single field and see a loading indicator on that specific field while AI processes
**Plans**: TBD

### Phase 11: List Picker
**Goal**: Users can define parameters that run a shell command and present the output as a selectable list, with control over which column is extracted as the final value
**Depends on**: Phase 10
**Requirements**: LIST-01, LIST-02, LIST-03, LIST-04, LIST-05
**Success Criteria** (what must be TRUE):
  1. User can define a list picker parameter that runs a shell command and displays the output lines as a selectable list during param fill
  2. User can select one item from the list and have it inserted as the parameter value
  3. User can configure column extraction (field index + custom delimiter) so only a specific part of the selected line becomes the value
  4. User can configure header line skipping so informational header rows from command output are not selectable
**Plans**: TBD

## Progress

**Execution Order:** 7 → 8 → 9 → 10 → 11

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 28/28 | Complete | 2026-02-25 |
| 7. Polish & Terminal Compat | v1.1 | 3/3 | Complete | 2026-02-28 |
| 8. Smart Defaults | v1.1 | 1/1 | Complete | 2026-03-01 |
| 9. Execute in Manage | v1.1 | 0/TBD | Not started | - |
| 10. Parameter CRUD & Per-Field AI | v1.1 | 0/TBD | Not started | - |
| 11. List Picker | v1.1 | 0/TBD | Not started | - |

## Requirement Coverage

| Requirement | Phase |
|-------------|-------|
| DISP-01 | Phase 7 |
| DISP-02 | Phase 7 |
| MGUX-01 | Phase 7 |
| MGUX-02 | Phase 7 |
| TERM-01 | Phase 7 |
| TERM-02 | Phase 7 |
| DFLT-01 | Phase 8 |
| DFLT-02 | Phase 8 |
| EXEC-01 | Phase 9 |
| EXEC-02 | Phase 9 |
| EXEC-03 | Phase 9 |
| EXEC-04 | Phase 9 |
| PCRD-01 | Phase 10 |
| PCRD-02 | Phase 10 |
| PCRD-03 | Phase 10 |
| PCRD-04 | Phase 10 |
| PCRD-05 | Phase 10 |
| PCRD-06 | Phase 10 |
| PCRD-07 | Phase 10 |
| PFAI-01 | Phase 10 |
| PFAI-02 | Phase 10 |
| LIST-01 | Phase 11 |
| LIST-02 | Phase 11 |
| LIST-03 | Phase 11 |
| LIST-04 | Phase 11 |
| LIST-05 | Phase 11 |

**Coverage: 26/26 (100%)**

---
*Roadmap created: 2026-02-27*
