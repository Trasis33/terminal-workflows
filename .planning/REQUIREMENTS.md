# Requirements: wf v1.1 Polish & Power

**Defined:** 2026-02-27
**Core Value:** Users can find and execute any saved command workflow in under 3 seconds

## v1.1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Defaults

- [x] **DFLT-01**: User sees original command values preserved as parameter defaults when saving a command as a workflow
- [x] **DFLT-02**: User can accept default values in param fill without retyping previously used values

### Manage Execute

- [ ] **EXEC-01**: User can trigger execute flow on a workflow from the manage browse view
- [ ] **EXEC-02**: User can fill parameters inline within manage TUI (same UX as picker)
- [ ] **EXEC-03**: User can copy completed command to clipboard from manage and stay in manage
- [ ] **EXEC-04**: User can paste completed command to prompt by exiting manage after param fill

### Parameter CRUD

- [ ] **PCRD-01**: User can add a new parameter to an existing workflow in manage TUI
- [ ] **PCRD-02**: User can remove a parameter from an existing workflow in manage TUI
- [ ] **PCRD-03**: User can rename a parameter on an existing workflow in manage TUI
- [ ] **PCRD-04**: User can change a parameter's type (plain/enum/dynamic/list) in manage TUI
- [ ] **PCRD-05**: User can edit a parameter's default value in manage TUI
- [ ] **PCRD-06**: User can edit enum option values for an enum parameter in manage TUI
- [ ] **PCRD-07**: User can edit the dynamic command for a dynamic parameter in manage TUI

### Per-Field AI

- [ ] **PFAI-01**: User can trigger AI generation for a single field in the workflow edit form
- [ ] **PFAI-02**: User sees a loading indicator on the specific field while AI generates

### List Picker

- [ ] **LIST-01**: User can define a list picker parameter that runs a shell command and shows output as a selectable list
- [ ] **LIST-02**: User can select one item from the list during param fill
- [ ] **LIST-03**: User can configure column extraction so only a specific field from the selected line is used as the value
- [ ] **LIST-04**: User can configure a custom delimiter for column splitting
- [ ] **LIST-05**: User can configure header line skipping so column headers are not selectable

### Display

- [x] **DISP-01**: User sees syntax-highlighted shell commands in the manage preview pane
- [x] **DISP-02**: User sees syntax-highlighted shell commands in the manage browse list

### Manage UX

- [x] **MGUX-01**: User sees folder contents auto-displayed when navigating to a folder in manage
- [x] **MGUX-02**: User can scroll the command preview panel without overscrolling past content

### Terminal Compatibility

- [x] **TERM-01**: User on Warp terminal is automatically offered a working keybinding (Ctrl+G conflict detected and fallback applied)
- [x] **TERM-02**: User can configure a custom keybinding via `wf init --key` flag

## Future Requirements

Deferred to later milestones. Tracked but not in current roadmap.

### Variable Defaults (Extended)

- **DFLT-03**: User sees history of previous values per parameter (browser autofill style)
- **DFLT-04**: User can opt-out of auto-save for sensitive parameters

### Parameter CRUD (Extended)

- **PCRD-08**: User can reorder parameters via up/down to control fill order

### Display (Extended)

- **DISP-03**: User sees parameter placeholders highlighted distinctly from shell syntax
- **DISP-04**: User sees syntax highlighting in the picker search results

### Terminal Compatibility (Extended)

- **TERM-03**: User can configure keybinding via config YAML (not just init flag)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| In-TUI text editor for commands | Users have strong editor preferences; keep `$EDITOR` integration |
| Command execution history/analytics | Atuin's territory; wf is for curated workflows |
| Multi-select in list picker | Ambiguous for paste-to-prompt model |
| Param value validation (regex, range) | Over-engineering; let the shell report errors |
| Custom Chroma themes | 3-4 built-in style mappings sufficient |
| File picker param type | Massive undertaking; use dynamic param with `find` command |
| Date picker param type | Calendar widget in terminal is fragile; users type dates |
| Auto-save every executed value as new default | Modifies YAML on every run; prefer explicit defaults |
| Param reorder in v1.1 | Deferred to reduce CRUD complexity; add/remove/edit sufficient |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DFLT-01 | Phase 8 | Complete |
| DFLT-02 | Phase 8 | Complete |
| EXEC-01 | Phase 9 | Pending |
| EXEC-02 | Phase 9 | Pending |
| EXEC-03 | Phase 9 | Pending |
| EXEC-04 | Phase 9 | Pending |
| PCRD-01 | Phase 10 | Pending |
| PCRD-02 | Phase 10 | Pending |
| PCRD-03 | Phase 10 | Pending |
| PCRD-04 | Phase 10 | Pending |
| PCRD-05 | Phase 10 | Pending |
| PCRD-06 | Phase 10 | Pending |
| PCRD-07 | Phase 10 | Pending |
| PFAI-01 | Phase 10 | Pending |
| PFAI-02 | Phase 10 | Pending |
| LIST-01 | Phase 11 | Pending |
| LIST-02 | Phase 11 | Pending |
| LIST-03 | Phase 11 | Pending |
| LIST-04 | Phase 11 | Pending |
| LIST-05 | Phase 11 | Pending |
| DISP-01 | Phase 7 | Complete |
| DISP-02 | Phase 7 | Complete |
| MGUX-01 | Phase 7 | Complete |
| MGUX-02 | Phase 7 | Complete |
| TERM-01 | Phase 7 | Complete |
| TERM-02 | Phase 7 | Complete |

**Coverage:**
- v1.1 requirements: 26 total
- Mapped to phases: 26
- Unmapped: 0

---
*Requirements defined: 2026-02-27*
*Last updated: 2026-03-01 after phase 8 completion*
