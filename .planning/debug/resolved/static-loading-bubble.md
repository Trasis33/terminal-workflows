---
status: resolved
trigger: "Investigate issue: static-loading-bubble\n\n**Summary:** the loading spinner bubble does not move when loading. It is static from start to finish across the app, mostly in AI generation, and has behaved this way since implementation."
created: 2026-03-10T00:00:00Z
updated: 2026-03-10T13:05:40Z
---

## Current Focus

hypothesis: Confirmed; tick-driven spinner frames animate correctly and the user validated the fix in real usage.
test: Archive resolved session and record the completed fix.
expecting: Debug session is moved to resolved and commits capture code + docs history.
next_action: move debug file to resolved, commit code changes, then commit resolved debug doc

## Evidence

- timestamp: 2026-03-10T00:06:00Z
  checked: repository structure
  found: Project is a Go CLI/TUI app under internal/ and cmd/, not a web frontend codebase.
  implication: The loading bubble issue is likely in Bubble Tea/TUI rendering logic, not CSS or browser animation.

- timestamp: 2026-03-10T00:08:00Z
  checked: internal/manage/model.go View loading state
  found: Global AI loading UI renders literal text "⣾ Generating with AI..." or "⣾ Auto-filling metadata..." with no variable animation frame.
  implication: The main loading overlay is inherently static unless some other code swaps the glyph.

- timestamp: 2026-03-10T00:08:30Z
  checked: internal/manage/form.go loading indicators
  found: Per-field AI and autofill states render literal "⣾" and "⣾ Autofilling..." strings from fieldLoading/autofillLock booleans only.
  implication: Form-level loading indicators are also static by design.

- timestamp: 2026-03-10T00:09:00Z
  checked: internal/manage/ai_actions.go async AI commands
  found: AI commands only return final result messages; they do not emit periodic tick messages for animation.
  implication: While AI is in flight, Bubble Tea receives no animation-driving messages from this feature.

- timestamp: 2026-03-10T00:10:30Z
  checked: spinner/tick usage across codebase
  found: No spinner package usage exists, and no periodic tick/animation loop is wired for AI loading; only unrelated tea.Tick use is a flash timeout.
  implication: The static behavior is the direct result of missing animation state + tick scheduling, not a rendering bug.

- timestamp: 2026-03-10T00:22:30Z
  checked: internal/manage spinner implementation and tests
  found: Added shared spinner helper, wired spinnerTickMsg updates into global and form AI loading paths, and manage package tests pass after gofmt.
  implication: The code now has an explicit animation mechanism and regression coverage for frame advancement/reset.

## Symptoms

expected: The loading bubble should animate continuously while AI content is generating.
actual: The bubble is fully static for the entire loading period.
errors: No visible errors, console errors, or app/log errors seen.
reproduction: Reproduces on any AI generation flow.
started: It has never worked correctly; broken since first implementation.

## Eliminated

## Evidence

## Resolution

root_cause: AI loading UIs in internal/manage/model.go and internal/manage/form.go render a hardcoded single-frame spinner glyph ("⣾") and the manage TUI never schedules periodic update messages to advance animation frames.
fix: Added a shared tick-driven spinner helper and wired it into both global AI overlay loading and form-level AI loading/autofill states so the displayed Braille frame advances during loading and resets when complete.
verification: `gofmt -w internal/manage/spinner.go internal/manage/model.go internal/manage/form.go internal/manage/model_test.go internal/manage/form_test.go && go test ./internal/manage` passed, including new tests proving spinner frames advance on ticks and reset when loading ends; user then confirmed the loading bubble is fixed in real workflow usage.
files_changed: [internal/manage/spinner.go, internal/manage/model.go, internal/manage/form.go, internal/manage/model_test.go, internal/manage/form_test.go]
