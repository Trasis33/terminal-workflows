# Project Research Summary

**Project:** wf v1.1 (Terminal Workflow Manager)
**Domain:** Go TUI terminal workflow manager — v1.1 Polish & Power
**Researched:** 2026-02-27
**Confidence:** HIGH

## Executive Summary

v1.1 "Polish & Power" adds 9 features to a shipped v1.0 codebase (11,736 LOC). The existing Go/Bubble Tea v1/Lip Gloss/huh stack is sufficient for every feature — the **only new dependency is `alecthomas/chroma/v2` (v2.23.1)** for syntax highlighting. Critically, **Bubble Tea v2.0.0 shipped 4 days ago (Feb 23, 2026) and must be avoided for v1.1** — the migration would touch every file and the ecosystem is too fresh. All v1.1 features are fully achievable on the current v1 stack without compromise.

The features divide into three complexity tiers. **Low complexity:** syntax highlighting (chroma integration, ~20 lines), Warp keybinding fix (shell script conditionals), overscroll fix (bounds checking), folder auto-display (debounced filter). **Medium complexity:** auto-save variable defaults (picker needs store write access, atomic file writes required), execute flow in manage (extract shared `ParamFillModel` from picker, add clipboard + exit-to-paste modes), list picker variable type (new `?` sigil in template parser, scrollable selection UI). **High complexity:** parameter CRUD in manage TUI (custom `ParamEditorModel` because huh cannot dynamically add/remove fields, two-zone focus management between huh form and custom component). The per-field AI generate is medium complexity but depends on parameter CRUD being built first.

The most dangerous pitfalls cluster around **auto-save defaults**: non-atomic `os.WriteFile` risks YAML corruption on crash, and full YAML re-marshal destroys user formatting (comments, ordering, whitespace). The recommended mitigation is atomic writes (temp+rename) and either surgical AST-level updates via `goccy/go-yaml`'s `ast` package or a separate `defaults.yaml` file that avoids touching workflow files entirely. The second major risk is the **execute flow's alt-screen conflict** — manage runs fullscreen, so paste-to-prompt requires either clipboard copy (stay in manage) or exit-then-print (leave manage). Both modes should be offered. Build order is critical: extract the shared `ParamFillModel` before building parameter CRUD, or the two features will create conflicting architectures.

## Key Findings

### Recommended Stack Changes

**One new dependency, zero upgrades, zero removals.**

- **`alecthomas/chroma/v2` v2.23.1** (ADD): Syntax highlighting for shell commands in preview panes. Pure Go, bash lexer built-in, terminal ANSI formatters (256-color, truecolor). Used by Hugo, Goldmark, lazygit, nap. ~5MB binary size increase. Coexists cleanly with Lip Gloss (chroma outputs ANSI, Lip Gloss wraps for layout).
- **All existing dependencies PINNED at current versions**: Bubble Tea v1.3.10, Bubbles v1.0.0, huh v0.8.0, Lip Gloss v1.1.0, Cobra v1.10.2, goccy/go-yaml v1.19.2, sahilm/fuzzy v0.1.1, adrg/xdg v0.5.3.
- **DO NOT add**: Bubble Tea v2, glamour (overkill — wraps chroma with markdown overhead), atomicgo/keyboard (conflicts with BT input loop), viper/koanf (existing config.go + go-yaml is sufficient), any database for defaults.

### Expected Feature Patterns

**Must have (table stakes):**
- View/add/remove/edit parameters in manage TUI — every competitor relies on `$EDITOR` for this; structured in-TUI CRUD is novel
- Execute workflows from manage TUI (param fill + clipboard/paste-to-prompt) — Hoard, SnipKit do this; wf manage currently can't
- Syntax highlighting in command preview — raw text preview looks amateur; chroma integration is ~20 lines
- Warp terminal keybinding fix — Ctrl+G is intercepted by Warp; detect `$TERM_PROGRAM` and use fallback binding
- Auto-save variable defaults — no snippet manager does this; closest analogue is browser autofill
- Overscroll fix and folder auto-display — UX polish bugs that erode trust

**Should have (differentiators):**
- List picker variable type with column extraction — Navi's `--column` pattern is the gold standard; wf's current dynamic params show whole lines only
- Per-field AI generate — no snippet manager offers field-level AI; start with description-from-command (highest value, lowest risk)
- Configurable keybindings via config YAML — solves Warp and all future terminal conflicts permanently
- Param reorder via up/down in CRUD editor

**Defer (v2+):**
- In-TUI text editor for commands (scope creep — keep `$EDITOR` via keybinding)
- Command execution history/analytics (Atuin's territory)
- Multi-select in list picker (ambiguous for paste-to-prompt model)
- Param value validation (regex, range — let the shell report errors)
- Custom Chroma themes (3-4 built-in mappings sufficient)
- Param type: file picker, date picker (massive undertaking, workaround via dynamic commands)

### Architecture Approach

v1.1 introduces 2 new packages and 4 new files into the existing tree-of-models architecture. The key architectural move is **extracting `ParamFillModel` from `internal/picker/paramfill.go` into `internal/paramfill/model.go`** as a shared component that both picker and manage compose. This extraction must happen before parameter CRUD to prevent competing architectures.

**New packages:**
1. **`internal/paramfill/`** — Shared param fill component (extracted from picker's 250-line paramfill.go). Used by both picker and manage execute flow. Parameterized for styles.
2. **`internal/highlight/`** — Thin chroma wrapper. `HighlightBash(code string) string` function. Caches output, auto-detects dark/light background via `lipgloss.HasDarkBackground()`.

**New files in existing packages:**
3. **`internal/manage/param_editor.go`** — Custom `ParamEditorModel` for parameter CRUD. NOT a huh form — uses raw textinput + list navigation because huh cannot add/remove fields dynamically.
4. **`internal/manage/execute.go`** — `ExecuteModel` wrapping shared `ParamFillModel` with clipboard and exit-to-paste modes.

**Key integration decisions:**
- `FormModel` gains two-zone focus: huh form zone + param editor zone, toggled via Tab
- `picker.New()` gains `store.Store` parameter for auto-save defaults (only 1 callsite to update)
- Template parser adds `?` sigil for list picker type (priority between `!` and `|`)
- Manage model adds `viewExecute` state after existing states in iota
- Shell init scripts add `$TERM_PROGRAM` detection for Warp fallback keybinding

### Critical Pitfalls

1. **Non-atomic YAML writes corrupt workflow files** — `YAMLStore.Save()` uses `os.WriteFile()` directly. With auto-save running on every execution, crash during write = data loss. **Fix:** Atomic write pattern (temp file in same dir + `os.Rename()` + `fsync`). Must be implemented before auto-save ships.

2. **YAML re-marshal destroys user formatting** — Auto-save triggers `yaml.Marshal()` which strips comments, reorders keys, changes quoting. Users who hand-edit YAML (a core promise) see their files mangled. **Fix:** Either surgical AST update via `goccy/go-yaml`'s `ast` package (update only the `default` scalar node), or use a separate `~/.config/wf/defaults.yaml` file that never touches workflow YAML.

3. **huh cannot add/remove fields dynamically** — No `AddField()`/`RemoveField()` API. Building param CRUD inside huh requires full form rebuild which resets focus, cursor, and partial input. **Fix:** Build `ParamEditorModel` as a separate custom Bubble Tea component alongside the huh form. Don't fight huh's static model.

4. **Alt-screen blocks paste-to-prompt** — Manage TUI runs fullscreen. Stdout goes to alternate buffer, invisible to shell wrapper. **Fix:** Offer two modes: `x` = clipboard copy (stay in manage), `X` = exit manage + print to stdout (shell wrapper captures).

5. **Shell execution surface is unbounded** — List picker runs arbitrary shell commands. No line limit = OOM on `find /`. No timeout configurability. ANSI codes in output corrupt field extraction. **Fix:** Default 10s timeout (configurable per-arg), 1000-line output cap, `NO_COLOR=1` env var + ANSI stripping.

## Implications for Roadmap

### Phase 1: Foundation Fixes & Quick Wins
**Rationale:** Low-risk standalone changes that unblock users and improve perceived quality immediately. No architectural dependencies. Can ship independently.
**Delivers:** Working Warp keybinding, syntax-highlighted previews, fixed overscroll, auto-filtering folders.
**Features:** Syntax highlighting (chroma integration), Warp keybinding fix, overscroll fix, folder auto-display.
**Avoids:** Render cost pitfall (#6) — cache highlighted output, limit to preview pane only. Folder filter churn (#9) — debounce 150ms.

### Phase 2: Auto-Save Variable Defaults
**Rationale:** Smaller change to picker that validates "picker writes to store" pattern before the larger execute flow refactor. Must solve atomic writes and formatting preservation first — these are critical pitfalls that affect all subsequent features touching YAML persistence.
**Delivers:** "It remembers what I typed" — unique differentiator no competitor has. Establishes atomic write infrastructure.
**Features:** Auto-save entered values as defaults, default value consistency fix (single source of truth: `args[].default` > template inline > empty).
**Avoids:** Non-atomic writes (#1) — implement atomic write pattern. YAML re-marshal formatting destruction (#2) — use separate defaults file or surgical AST update. Save conflicts (#15) — read-modify-write, not cache-then-write.

### Phase 3: Execute Flow in Manage
**Rationale:** Requires extracting `ParamFillModel` as a shared component — this refactoring MUST happen before Phase 4's param CRUD adds its own param-related models. Users need to test workflows from manage before we give them CRUD to edit parameters.
**Delivers:** Select workflow in manage -> fill params -> clipboard copy or paste-to-prompt. Full execute loop without leaving manage.
**Features:** Full execute flow, shared ParamFillModel extraction.
**Avoids:** Alt-screen paste-to-prompt conflict (#5) — implement both clipboard and exit-to-paste modes. View state routing regression (#14) — map full key routing table before coding, follow existing pattern exactly.

### Phase 4: Parameter CRUD
**Rationale:** Highest complexity feature. Benefits from Phases 1-3 being stable. Requires custom Bubble Tea component because huh is architecturally incompatible. Per-field AI generate depends on this.
**Delivers:** Add/remove/rename/reorder parameters in manage TUI. Type switching (text/enum/dynamic/list). Per-field AI generate for individual variables.
**Features:** Full parameter CRUD, per-field AI generate, inline enum option editing.
**Avoids:** huh dynamic field problem (#3) — custom ParamEditorModel, not huh extension. Value-copy state (#12) — pointer pattern (`*paramEditorState`). N+1 AI calls (#10) — batch suggestion with cache, not per-field API calls.

### Phase 5: List Picker Variable Type
**Rationale:** Extends the paramfill system that was extracted in Phase 3 and the param metadata from Phase 4. Adds a new param type that benefits from the CRUD UI being available to configure it.
**Delivers:** `{{name?command}}` syntax for scrollable list selection with column extraction. Richer alternative to existing `{{name!command}}` cycling.
**Features:** List picker type, `?` sigil in parser, field extraction, scrollable list UI with fuzzy filtering.
**Avoids:** Unbounded shell execution (#4) — configurable timeout + 1000-line cap. ANSI corruption (#13) — `NO_COLOR=1` + strip ANSI before field extraction.

### Phase Ordering Rationale

- **Phase 1 first:** Zero dependencies on other features. Ships quick wins that improve daily UX. Warp fix removes a user blocker.
- **Phase 2 before Phase 3:** Auto-save is a smaller, contained change that proves "picker writes to store" before the larger execute flow extraction. Atomic write infrastructure established here protects all later YAML write paths.
- **Phase 3 before Phase 4:** The ParamFillModel extraction in Phase 3 is a prerequisite. If Phase 4 builds its own param-related models first, we get competing architectures that need expensive reconciliation.
- **Phase 4 before Phase 5:** List picker is a new param type. The CRUD UI from Phase 4 provides the configuration interface for list picker metadata (field_index, delimiter). Without CRUD, users must hand-edit YAML to configure list pickers.
- **Per-field AI generate in Phase 4** (not separate): It's a small extension of the ParamEditorModel with an AI prompt per field. Shipping it with CRUD avoids a separate plan for a thin feature.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Auto-Save Defaults):** `goccy/go-yaml` AST manipulation for surgical updates is unverified in practice. The `ast` package API exists but format preservation precision is unknown. May need to fall back to separate defaults file approach.
- **Phase 4 (Parameter CRUD):** Two-zone focus management between huh form and custom component has no established pattern. Needs prototyping. The `formValues` pointer pattern is proven but extending it to a dynamic list requires careful design.
- **Phase 5 (List Picker):** Template parser syntax (`?` sigil) design needs validation — ensure no conflicts with existing command strings. Field extraction UX (showing extracted value vs full line) needs user testing.

Phases with standard patterns (skip deep research):
- **Phase 1 (Foundation Fixes):** Chroma integration is well-documented (Hugo, lazygit, nap all use it). Warp detection is a simple env var check. Overscroll and auto-display are small logic fixes.
- **Phase 3 (Execute Flow):** Extracting a shared component + adding a view state follows established Bubble Tea patterns. Clipboard via `atotto/clipboard` already in go.mod.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Only 1 new dependency (chroma v2.23.1, verified). All existing deps pinned. BT v2 avoidance is well-reasoned. |
| Features | HIGH | Verified against Pet, Navi, Hoard, SnipKit, Warp READMEs. Feature gaps and anti-features clearly mapped. Per-field AI is novel (no precedent). |
| Architecture | HIGH | Based on thorough reading of 11,736 LOC codebase. Integration points verified at file:line level. huh v0.8 limitation confirmed. |
| Pitfalls | HIGH | 16 pitfalls identified. Critical pitfalls (#1-#4) have proven solutions. `os.WriteFile` non-atomicity confirmed at yaml.go:45. |

**Overall confidence:** HIGH

### Gaps to Address

- **`goccy/go-yaml` AST surgical update precision:** The `ast` package exists but whether it preserves comments, ordering, and whitespace through an update-then-marshal cycle is unverified. Test this early in Phase 2; if it fails, use separate defaults file.
- **Warp keybinding empirical testing:** Which fallback keybinding (`Ctrl+Space`, `Ctrl+/`, `Ctrl+O`) actually works in Warp needs testing in Warp itself. Research identified the problem and detection mechanism but not the verified solution.
- **Copilot SDK rate limits for per-field AI:** The SDK is in technical preview. Rate limits for rapid successive calls (batch vs N+1) are undocumented. The batch-with-cache mitigation should work regardless.
- **Two-zone focus management UX:** No established pattern exists for splitting focus between a huh form and a custom Bubble Tea component in the same view. Needs prototyping in Phase 4.

## Sources

### Primary (HIGH confidence)
- Chroma v2.23.1 releases: github.com/alecthomas/chroma/releases (Jan 23, 2026)
- Chroma Go API: pkg.go.dev/github.com/alecthomas/chroma/v2 (TTY formatters confirmed)
- huh v0.8.0 releases: github.com/charmbracelet/huh/releases (Oct 14, 2025, custom Field types, no AddField API)
- Bubble Tea v2.0.0 announcement: charm.land (Feb 23, 2026, breaking changes documented)
- Pet, Navi, Hoard, SnipKit GitHub READMEs (feature comparison, parameter patterns)
- Project codebase: go.mod, yaml.go, form.go, paramfill.go, browse.go, keys.go, model.go (verified at line level)
- Go `os.Rename` atomicity: pkg.go.dev/os, POSIX spec
- `goccy/go-yaml` AST package: github.com/goccy/go-yaml (ast.File, ast.MappingNode exist)

### Secondary (MEDIUM confidence)
- Warp terminal Ctrl+G interception: GitHub issues #500, #537 (user reports)
- Warp `$TERM_PROGRAM=WarpTerminal`: multiple community sources
- NO_COLOR convention: https://no-color.org/
- Bubble Tea dynamic form patterns: community examples

### Tertiary (LOW confidence)
- `goccy/go-yaml` AST format preservation precision: API exists, practical fidelity unverified
- Warp-specific fallback keybinding that works: needs empirical testing
- Copilot SDK rate limits for per-field generation: undocumented (technical preview)

---
*Research completed: 2026-02-27*
*Ready for roadmap: yes*
