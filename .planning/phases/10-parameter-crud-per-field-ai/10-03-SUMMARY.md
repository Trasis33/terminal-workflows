---
phase: 10-parameter-crud-per-field-ai
plan: 03
subsystem: ui
tags: [bubbletea, ai, copilot, ghost-text, autofill, termenv]

# Dependency graph
requires:
  - phase: 10-parameter-crud-per-field-ai
    provides: "ParamEditorModel with full CRUD and sub-field editing (Plans 01+02)"
  - phase: 05-ai-integration
    provides: "Generator interface, Copilot SDK, Autofill API"
provides:
  - "Per-field AI generation via Ctrl+G on any editable field"
  - "Ghost text rendering on separate line with Enter accept / Esc dismiss"
  - "Autofill button to bulk-fill all empty fields"
  - "Command-to-params AI suggestion with navigable ghost params"
  - "Ghost param accept/dismiss navigation (Tab/Enter/Esc)"
  - "ANSI escape leak fix via termenv.WithProfile + SetHasDarkBackground"
  - "AI pipe death recovery via context.Background() subprocess lifecycle"
affects: [11-list-picker]

# Tech tracking
tech-stack:
  added: [termenv]
  patterns:
    - "Ghost text on separate line to avoid fixed-width textinput alignment issues"
    - "Subprocess lifecycle: context.Background() for long-lived CLI subprocesses"
    - "OSC query prevention: pre-set dark background to avoid terminal stdin pollution"
    - "Ghost param navigation: onGhostParams + ghostCursor state for navigable AI suggestions"

key-files:
  created:
    - internal/manage/ai_actions.go
  modified:
    - internal/manage/form.go
    - internal/manage/param_editor.go
    - internal/manage/model.go
    - internal/manage/keys.go
    - internal/manage/manage.go
    - internal/ai/availability.go
    - internal/ai/copilot.go

key-decisions:
  - "Ghost text rendered on separate line below field instead of inline suffix — fixed-width textinput pushes inline ghost text far right"
  - "GetGenerator uses context.Background() for client.Start() — subprocess must outlive individual request timeouts"
  - "termenv.WithProfile(TrueColor) + SetHasDarkBackground(true) prevents OSC 11 query that causes ANSI escape leak into textinputs"
  - "Pipe errors show user-friendly 'AI unavailable' message instead of raw Go error"
  - "Ghost params navigable via dedicated onGhostParams/ghostCursor state with Enter=accept, Esc=dismiss"

patterns-established:
  - "Ghost text pattern: store in map[string]string, render conditionally as separate styled row"
  - "Loading indicator pattern: fieldLoading map[string]bool with braille spinner character"
  - "Autofill lock pattern: form-level lock preventing edits during bulk AI generation"
  - "OSC prevention pattern: pre-set background detection to avoid terminal stdin pollution"

requirements-completed: [PFAI-01, PFAI-02]

# Metrics
duration: multi-session
completed: 2026-03-07
---

# Phase 10 Plan 03: Per-Field AI Generation Summary

**Per-field AI via Ctrl+G with ghost text suggestions, autofill button, command-to-params AI, and critical fixes for ANSI escape leak and AI subprocess lifecycle**

## Performance

- **Duration:** Multi-session (bugs required iterative debugging)
- **Started:** 2026-03-04
- **Completed:** 2026-03-07
- **Tasks:** 2 (Task 1: implementation, Task 2: human-verify checkpoint)
- **Files modified:** 8

## Accomplishments
- Per-field AI generation on all editable fields (name, description, tags, param defaults, enum options, dynamic commands) via Ctrl+G
- Ghost text rendered on separate line below field with accept/dismiss UX
- Autofill button fills all empty fields at once with per-field loading indicators and form lock
- Command-to-params AI suggests parameters with navigable ghost param list
- Fixed ANSI escape leak (OSC 11 query polluting stdin → textinputs showing garbage)
- Fixed AI subprocess death (request-scoped context killing Copilot CLI process)
- Human-verified end-to-end

## Task Commits

Each task was committed atomically:

1. **Task 1: Per-field AI, ghost text, autofill** - `297ef61` (feat)
2. **Fix: ANSI escape leak, textarea indent, AI pipe recovery** - `2438044` (fix)
3. **Fix: Ghost text line rendering, ghost param navigation, AI resilience** - `6771d7b` (fix)

## Files Created/Modified
- `internal/manage/ai_actions.go` - NEW: perFieldAICmd, autofillWorkflowCmd, suggestParamsCmd, message types
- `internal/manage/form.go` - Ghost text rendering (ghostLine), Autofill button, per-field AI trigger, fieldLoading state, pipe error UX
- `internal/manage/param_editor.go` - Ghost param navigation (onGhostParams, ghostCursor), accept/dismiss, ghost text per sub-field
- `internal/manage/model.go` - Route perFieldAIResultMsg, suggestParamsResultMsg to form
- `internal/manage/keys.go` - Added FormAI (Ctrl+G) keybinding
- `internal/manage/manage.go` - ANSI fix: termenv.WithProfile(TrueColor) + SetHasDarkBackground(true)
- `internal/ai/availability.go` - GetGenerator uses context.Background() for subprocess Start()
- `internal/ai/copilot.go` - resetBrokenInstance properly closes old client, isPipeError detection

## Decisions Made
- Ghost text rendered on separate line below field — the fixed-width textinput.View() fills its width, pushing inline ghost text far to the right
- GetGenerator uses context.Background() for client.Start() — the Copilot subprocess must outlive individual request timeouts; passing a request-scoped context with defer cancel() kills the subprocess when the 30s timeout fires
- termenv.WithProfile(TrueColor) + SetHasDarkBackground(true) prevents the renderer from sending an OSC 11 background color query; the terminal response arrives through stdin and gets captured by focused textinputs as garbage characters
- Pipe errors (file already closed, broken pipe, EOF) show "AI unavailable - Copilot connection failed. Try again" instead of raw Go error messages

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Critical Bug] ANSI escape code leak into textinputs**
- **Found during:** Task 1 (testing per-field AI in terminal)
- **Issue:** lipgloss.NewRenderer(ttyOut) sends OSC 11 query to detect background color; terminal response arrives through stdin and gets captured by focused textinputs
- **Fix:** Use termenv.WithProfile(termenv.TrueColor) and r.SetHasDarkBackground(true) to prevent the OSC query
- **Files modified:** internal/manage/manage.go
- **Committed in:** 2438044

**2. [Rule 2 - Critical Bug] AI subprocess killed by request timeout**
- **Found during:** Task 1 (AI generation failing with "file already closed")
- **Issue:** GetGenerator passed caller's request-scoped context to client.Start(); when request completed and cancel() fired, Go killed the Copilot subprocess; singleton then held dead pipes
- **Fix:** GetGenerator uses context.Background() for client.Start(); resetBrokenInstance properly Close()s old client
- **Files modified:** internal/ai/availability.go, internal/ai/copilot.go
- **Committed in:** 2438044

**3. [Rule 1 - Bug] Ghost text right-alignment**
- **Found during:** Task 1 (testing ghost text rendering)
- **Issue:** ghostSuffix appended ghost text after textinput.View() which renders a fixed-width field, pushing ghost text far right
- **Fix:** Renamed to ghostLine, rendered on separate row below the field
- **Files modified:** internal/manage/form.go
- **Committed in:** 6771d7b

**4. [Rule 3 - Blocking] Ghost params not interactable**
- **Found during:** Task 1 (testing command-to-params AI)
- **Issue:** Ghost params rendered but had no navigation — Tab stopped at Add Parameter, no way to accept/dismiss
- **Fix:** Added onGhostParams/ghostCursor state, updateGhostNavigation handler, highlighted focused ghost param with inline hints
- **Files modified:** internal/manage/param_editor.go
- **Committed in:** 6771d7b

---

**Total deviations:** 4 auto-fixed (2 critical bugs, 1 bug, 1 blocking)
**Impact on plan:** All fixes necessary for correct operation. No scope creep.

## Issues Encountered
- Multi-session debugging required for ANSI escape leak (root cause was subtle: OSC 11 response arriving through stdin)
- AI pipe death was a singleton lifecycle issue — the first call worked but subsequent calls failed because the subprocess was killed

## User Setup Required
None - requires existing GitHub Copilot authentication (configured in Phase 5).

## Next Phase Readiness
- Phase 10 is now complete — all 3 plans delivered
- Parameter CRUD and per-field AI fully operational
- Ready for Phase 11 (List Picker) which will add a new parameter type using the ParamEditorModel infrastructure

---
*Phase: 10-parameter-crud-per-field-ai*
*Completed: 2026-03-07*
