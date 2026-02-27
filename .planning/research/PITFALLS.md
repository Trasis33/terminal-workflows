# Domain Pitfalls — v1.1 "Polish & Power"

**Domain:** Go TUI Terminal Workflow Manager (`wf`)
**Researched:** 2026-02-27
**Milestone:** v1.1 — variable defaults, manage UX, list picker, terminal compat, parameter CRUD
**Overall Confidence:** HIGH (based on v1.0 codebase analysis + external research)

**Note:** This document covers pitfalls specific to the v1.1 feature set. The original v1.0 pitfalls document informed the initial build and remains valid for foundational concerns. This document focuses on what can go wrong when adding the new features to the existing codebase.

---

## Critical Pitfalls

Mistakes that cause rewrites, data loss, or major integration failures.

---

### Pitfall 1: Auto-Save Defaults via Non-Atomic `os.WriteFile` Corrupts YAML

**Feature:** Auto-save entered variable values as defaults
**Risk level:** CRITICAL — data loss

**What happens:** The auto-save feature writes updated defaults back to workflow YAML files every time a user fills parameters. `YAMLStore.Save()` uses `os.WriteFile()` directly (yaml.go:45). If the process is killed, the terminal crashes, or `SIGKILL` arrives mid-write, the YAML file is left half-written — corrupted. With auto-save happening on every workflow execution (potentially multiple times per minute), the window for corruption multiplies compared to the current manual save flow.

**Warning signs:**
- Zero-byte workflow files appearing after system crash
- YAML parse errors on previously-working workflows
- Partial content in workflow files (truncated mid-field)

**Prevention:**
- **Atomic write pattern: temp file + rename.** Write to a temporary file in the same directory, then `os.Rename()` to the target path. On POSIX, rename is atomic within the same filesystem:
  ```go
  func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
      dir := filepath.Dir(path)
      tmp, err := os.CreateTemp(dir, ".wf-*.tmp")
      if err != nil {
          return err
      }
      tmpPath := tmp.Name()
      defer os.Remove(tmpPath) // cleanup on error path
      
      if _, err := tmp.Write(data); err != nil {
          tmp.Close()
          return err
      }
      if err := tmp.Sync(); err != nil { // fsync before rename
          tmp.Close()
          return err
      }
      if err := tmp.Close(); err != nil {
          return err
      }
      return os.Rename(tmpPath, path)
  }
  ```
- **Debounce auto-saves.** Don't write on every parameter fill. Accumulate changes and write after a 2-second idle period, or write once when the picker exits after successful parameter fill. This reduces write frequency by 10-50x.
- **Only update the `default` field, not the entire workflow.** Parse the existing YAML, update only the `args[N].default` value, re-marshal. This minimizes the blast radius of any formatting changes.

**Detection:** Kill the `wf` process with `kill -9` during parameter fill. Check if the workflow file is corrupted.

**Phase:** Must be in the first plan that touches auto-save defaults.

**Confidence:** HIGH — `os.WriteFile` non-atomicity is well-documented; current code at yaml.go:45 confirmed.

---

### Pitfall 2: Auto-Save Defaults Trigger Full YAML Re-Marshal, Destroying Formatting

**Feature:** Auto-save entered variable values as defaults
**Risk level:** CRITICAL — user trust

**What happens:** The auto-save flow reads a workflow, modifies the `Default` field on one `Arg`, then calls `yaml.Marshal()` + `Save()`. `goccy/go-yaml` (like all YAML libraries) re-serializes the entire document. This means: comments are stripped, key ordering may change, quoting style may change, blank lines are removed. Users who hand-edit their YAML files (a core promise of the human-readable storage format) see their carefully formatted files mangled by a simple parameter fill.

**Warning signs:**
- Git diffs showing massive reformatting of unchanged fields after running a workflow
- Users reporting "wf is rewriting my files"
- Comment loss in workflow YAML

**Prevention:**
- **Surgical field update instead of full re-marshal.** For auto-save, parse the YAML into `yaml.Node` (AST), find the specific `args[N].default` node, update its value, and write the AST back. This preserves comments, ordering, and formatting:
  ```go
  // Pseudocode for surgical default update
  func updateArgDefault(filePath string, argName string, newDefault string) error {
      data, _ := os.ReadFile(filePath)
      var node yaml.Node
      yaml.Unmarshal(data, &node)
      // Walk AST to find args -> [name matches] -> default
      // Update only that scalar node's Value
      // Re-marshal from AST (preserves formatting)
  }
  ```
- **Note:** `goccy/go-yaml` does support node-level manipulation. Verify with `goccy/go-yaml`'s `ast` package (it has `ast.File`, `ast.MappingNode`, etc.) whether surgical updates are possible without full re-serialization.
- **Fallback if AST approach is too complex:** Write a separate `.defaults.yaml` file per workflow (or a single `defaults.yaml` in the config dir) that only stores `{workflow_name: {arg_name: last_value}}`. This completely avoids touching the workflow YAML.
- **Test:** Round-trip a hand-formatted YAML file (with comments, custom ordering, blank lines) through the auto-save flow. Assert the file is byte-for-byte identical except for the changed default.

**Detection:** Create a workflow YAML with inline comments and custom formatting. Run the workflow, fill params, check if the YAML file formatting changed.

**Phase:** Must be resolved before implementing auto-save. Consider the separate defaults file approach if AST manipulation proves fragile.

**Confidence:** HIGH — YAML re-marshal behavior is well-known; `goccy/go-yaml` AST support verified via repository.

---

### Pitfall 3: Parameter CRUD in huh Forms Requires Dynamic Field Rebuilding That huh Doesn't Support

**Feature:** Full CRUD on parameters in wf manage (add/remove/rename args)
**Risk level:** CRITICAL — architectural challenge

**What happens:** The current `FormModel` builds a `huh.Form` once at transition time (`buildForm()` in form.go:101). The form has a fixed set of fields: name, description, command, tags, folder. For parameter CRUD, you need to dynamically add/remove argument fields (name, type, default, options, description per arg). But `huh.Form` does not support adding or removing fields after construction. Calling `huh.NewForm()` with new groups/fields resets all form state — focus, cursor position, partially-entered values.

**Warning signs:**
- Form state (cursor position, partial input) lost when adding/removing an arg
- Flickering or visual reset when the form rebuilds
- Complex state management trying to preserve partial form state across rebuilds

**Prevention:**
- **Don't use huh for parameter CRUD.** Build a custom parameter editor as a separate view state (`viewEditParams`), not an extension of the existing huh form. Use raw `textinput.Model` and `list` components from Bubbles for the parameter list. This gives full control over add/remove/reorder without fighting huh's static form model.
- **Alternatively: Two-step form flow.** Step 1: Edit workflow metadata (name, command, tags — existing huh form). Step 2: Switch to a custom parameter editor view for arg CRUD. The `viewEdit` state becomes a mini-flow with its own sub-states.
- **The `formValues` pointer pattern must extend.** If args are edited, they need their own shared pointer struct (like `formValues` but for `[]Arg`). Since Bubble Tea copies models on every Update, the arg list must be heap-allocated:
  ```go
  type paramEditorValues struct {
      args *[]store.Arg // pointer — survives value copies
  }
  ```
- **Preserve the existing form for basic fields.** Don't try to cram arg CRUD into the existing 5-field form. Keep it clean.

**Detection:** Try adding a 6th field group to the existing huh form dynamically mid-edit. If state is lost, you've confirmed this pitfall.

**Phase:** Must be resolved in the plan for parameter CRUD. Likely its own phase/plan.

**Confidence:** HIGH — huh form construction is static (confirmed in form.go:155-163); no `AddField()` API exists.

---

### Pitfall 4: List Picker Variable Type Creates Unbounded Shell Execution Surface

**Feature:** List picker dynamic variable type (shell command → select → field extraction)
**Risk level:** CRITICAL — reliability + security boundary

**What happens:** The list picker variable type runs a shell command, presents output as a selectable list, then extracts a specific field from the selected line. This extends the existing `executeDynamic()` (paramfill.go:99) pattern but adds field extraction. The risks compound:
1. **Shell command hangs forever** if the 5-second timeout is too short for slow commands (e.g., `aws ec2 describe-instances`)
2. **Output is enormous** for commands producing thousands of lines (e.g., `find / -name "*.go"`)
3. **Field extraction is fragile** — users will specify field indices or delimiters that don't match output format, producing empty/wrong values silently
4. **Command fails differently across environments** — a `kubectl` command works on dev but fails on CI with no cluster context

**Warning signs:**
- TUI freezes for >5 seconds during parameter fill
- OOM when command output is very large
- Empty parameter values after field extraction fails silently
- Works on developer's machine, fails on team members' machines

**Prevention:**
- **Configurable timeout with sensible default.** Default to 10 seconds (not 5 — real-world commands like `docker ps` or `kubectl get pods` can take 3-5 seconds). Allow per-arg override:
  ```yaml
  args:
    - name: pod
      type: list
      dynamic_cmd: "kubectl get pods -o name"
      timeout: 30  # seconds
  ```
- **Output line limit.** Cap at 1000 lines. If the command produces more, truncate and show "(1000 of N lines — refine your command)" at the bottom of the picker. Don't try to load 50,000 lines into memory.
- **Explicit field extraction syntax.** Don't invent a new DSL. Use `awk`-compatible field references that users already know:
  ```yaml
  args:
    - name: container_id
      type: list
      dynamic_cmd: "docker ps"
      field: 1        # first whitespace-delimited field (like awk $1)
      # OR
      field: "0:12"   # character range (columns 0-12)
  ```
- **Show raw output if field extraction fails.** Don't silently produce empty values. If `field: 3` but the selected line has only 2 fields, use the entire line as the value and show a warning.
- **Preview the extracted value.** In the picker list, show both the full line and the extracted field value (highlighted) so users can verify before selecting.
- **Test with: empty output, single line, 1000+ lines, lines with varying field counts, unicode output, ANSI color codes in output.** Strip ANSI codes before field extraction.

**Detection:** Create a list picker with `dynamic_cmd: "find / -type f"` and observe behavior.

**Phase:** Core design decision for list picker implementation.

**Confidence:** HIGH — extends existing `executeDynamic()` pattern (paramfill.go:99); issues are architectural not speculative.

---

## Moderate Pitfalls

Mistakes that cause delays, bugs, or tech debt.

---

### Pitfall 5: Execute Flow in Manage TUI Conflicts with Alt-Screen Mode

**Feature:** Full execute flow inside wf manage (param fill, paste to prompt)
**Risk level:** MODERATE — architectural mismatch

**What happens:** The manage TUI runs with `tea.WithAltScreen()` (manage.go). The picker's paste-to-prompt mechanism prints the rendered command to stdout, which the shell wrapper captures. But in alt-screen mode, stdout goes to the alternate buffer — the shell wrapper never sees the output. If you exit alt-screen to print the command, there's a visible flash as the terminal switches buffers. If you try to write to `/dev/tty` directly (as the picker does), the shell wrapper can't capture it either because it only captures stdout.

**Warning signs:**
- Command "executed" from manage TUI but nothing appears in the shell prompt
- Visual flash/flicker when switching out of alt-screen
- Users confused about where the command went

**Prevention:**
- **Option A: Clipboard-first for manage execute.** From the manage TUI, copy the rendered command to clipboard (using `atotto/clipboard`, already a dependency) and show a toast-style notification "Copied to clipboard — paste with Ctrl+V". Don't try to match the picker's paste-to-prompt UX.
- **Option B: Exit manage, then print.** Set a result field on the Model, call `tea.Quit`, and after `p.Run()` returns, print the result to stdout (same pattern as the picker). The manage TUI closes, the command is pasted to the prompt. But this means the user leaves the manage TUI — acceptable UX trade-off?
- **Option C: Hybrid.** Offer both clipboard (stay in manage) and "run" (exit manage + paste to prompt). Let the user choose via keybinding.
- **Do NOT try to dynamically switch between alt-screen and inline mode** within the same `tea.Program`. This causes rendering chaos (per v1.0 pitfall #8).

**Detection:** From manage TUI, trigger execute flow and check if the command appears in the shell prompt.

**Phase:** Execute flow design.

**Confidence:** HIGH — alt-screen behavior is well-understood; picker's `/dev/tty` approach confirmed in codebase.

---

### Pitfall 6: Syntax Highlighting Adds Per-Frame Render Cost That Scales with Visible Workflows

**Feature:** Syntax highlighting in workflow list rendering
**Risk level:** MODERATE — performance

**What happens:** The browse view re-renders the visible workflow list on every `View()` call (browse.go:380-440). Adding syntax highlighting means applying ANSI color sequences to each command preview. If syntax highlighting is done naively (e.g., regex matching for shell keywords on every render), the cost scales linearly with visible items × command length. With 20 visible workflows of average 100-char commands, that's 2000 characters of regex matching per frame.

**Warning signs:**
- Perceptible lag when scrolling through workflow list
- CPU spikes during rapid scrolling (j/k held down)
- Frame drops visible as "tearing" or delayed cursor movement

**Prevention:**
- **Cache highlighted output.** Compute syntax highlighting once when a workflow enters the visible window, cache the styled string, invalidate only when the workflow list changes (not on every View call):
  ```go
  type highlightCache struct {
      mu    sync.Mutex
      cache map[string]string // command -> highlighted string
  }
  ```
- **Use a lightweight highlighter, not a full parser.** Don't pull in a tree-sitter binding or full lexer. Shell syntax highlighting for display purposes needs only: keywords (`if`, `for`, `do`, `done`), strings (single/double quoted), variables (`$VAR`, `${VAR}`), comments (`#`), pipes/redirects (`|`, `>`, `>>`, `<`), and commands (first word). A ~50-line regex-based highlighter is sufficient.
- **Limit highlighting to the preview pane, not the list.** The workflow list shows names + truncated descriptions (browse.go:393-431). Only the preview pane (browse.go:443-473) shows the full command. Highlight only there — it's a single command, not 20.
- **Benchmark:** Profile View() with and without highlighting. If the delta exceeds 2ms, the cache is mandatory.

**Detection:** Hold `j` key for 3 seconds with 50+ workflows. If scrolling is noticeably slower with highlighting enabled, optimize.

**Phase:** Syntax highlighting implementation.

**Confidence:** MEDIUM — performance impact depends on implementation. Benchmarking needed.

---

### Pitfall 7: Warp Terminal Compatibility Requires Multiple Workarounds

**Feature:** Warp terminal Ctrl+G compatibility + fallback keybinding
**Risk level:** MODERATE — terminal-specific

**What happens:** Warp terminal (as of Feb 2026) has multiple TUI compatibility issues:
1. **Ctrl+G keybinding doesn't trigger** in Warp's input model — Warp intercepts certain key combinations before they reach the shell.
2. **Kitty keyboard protocol bugs** — Warp's Feb 2026 update introduced `@` rendering problems with TUIs that use the Kitty keyboard protocol (which Bubble Tea v1 does NOT use, but v2 might).
3. **Non-English keyboard layouts** break shortcuts differently in Warp vs standard terminals.
4. **Warp's "blocks" model** wraps command output differently, which can affect inline picker rendering.

**Warning signs:**
- Keybinding doesn't trigger `wf` in Warp but works in iTerm2/Terminal.app
- Visual artifacts in Warp-specific output rendering
- User reports: "works everywhere except Warp"

**Prevention:**
- **Provide a configurable fallback keybinding.** Don't hardcode Ctrl+G. Let users set their own binding in the shell init script:
  ```bash
  # In wf init zsh output:
  WF_KEYBIND="${WF_KEYBIND:-\\C-g}"  # Default Ctrl+G, user can override
  bindkey "$WF_KEYBIND" _wf_widget
  ```
- **Document Warp-specific instructions.** Create a "Warp Terminal" section in docs with the specific keybinding that works (`Ctrl+Shift+G` or a custom sequence).
- **Test in Warp explicitly.** Add Warp to the test matrix. The `/dev/tty` approach for picker output may need special handling in Warp's environment.
- **Consider `TERM_PROGRAM` detection.** The env var `TERM_PROGRAM=WarpTerminal` identifies Warp. Use it to auto-select a compatible keybinding or show a setup hint on first run.
- **Don't try to "fix" Warp.** Warp is a moving target with its own keyboard model. Provide workarounds, not hacks.

**Detection:** Install `wf` in Warp terminal. Try Ctrl+G. If nothing happens, you need the fallback.

**Phase:** Terminal compatibility plan.

**Confidence:** MEDIUM — Warp behavior is based on user reports and search results, not official Warp documentation (which is limited for TUI compatibility).

---

### Pitfall 8: Preview Panel Overscroll Causes Cursor to Disappear Below Visible Area

**Feature:** Fix command preview panel overscroll in manage view
**Risk level:** MODERATE — UX bug

**What happens:** The browse view's preview pane (browse.go:363-371) uses a fixed height of 6 lines (`previewHeight()` returns 6). But commands can be much longer than 4 content lines. Currently the preview just truncates — but the actual bug being fixed likely involves the scroll offset calculation in `ensureCursorVisible()` (browse.go:306-314). The `listHeight()` calculation (browse.go:86-95) subtracts preview height and search height, but doesn't account for edge cases:
1. When searching is active, `listHeight` shrinks by 2, but `scrollOff` isn't reclamped
2. When the terminal is resized smaller, `scrollOff` can exceed the new visible area
3. The `cursor` can be valid but `scrollOff` can place it off-screen

**Warning signs:**
- Pressing `j`/`k` with no visible cursor movement (cursor is below the visible area)
- Scroll position jumps when toggling search on/off
- Different behavior at different terminal heights

**Prevention:**
- **Clamp `scrollOff` in `listHeight()` changes.** Whenever `listHeight()` changes (search toggle, resize), immediately re-run `ensureCursorVisible()`:
  ```go
  // After any operation that changes listHeight:
  b.ensureCursorVisible()
  ```
- **Add bounds checking in `renderList`.** Before rendering, verify:
  ```go
  if b.scrollOff < 0 {
      b.scrollOff = 0
  }
  maxScroll := len(b.filtered) - b.listHeight()
  if maxScroll < 0 {
      maxScroll = 0
  }
  if b.scrollOff > maxScroll {
      b.scrollOff = maxScroll
  }
  ```
- **For the preview pane itself:** If the command is longer than 4 lines, either truncate with "…" or make the preview scrollable using `viewport.Model` from Bubbles. A scrollable preview is more complex but gives better UX for long commands.
- **Test at minimum terminal height.** Set terminal to 15 rows. Browse 20 workflows. Toggle search. Resize to 10 rows. Verify cursor is always visible.

**Detection:** Open manage with 20+ workflows in a 24-row terminal. Toggle search on/off while cursor is near the bottom of the list.

**Phase:** Overscroll fix plan.

**Confidence:** HIGH — code paths confirmed in browse.go; `scrollOff` is never reclamped on `listHeight` changes.

---

### Pitfall 9: Folder Auto-Display Breaks "Sidebar as Filter" Mental Model

**Feature:** Auto-display folder contents in manage without extra keypress
**Risk level:** MODERATE — UX design

**What happens:** Currently, the sidebar requires pressing Enter on a folder to filter the workflow list (sidebar.go:103-106). The request is to auto-display folder contents when the folder is highlighted (cursor moves = immediate filter). But this changes the interaction model: the sidebar becomes a "live filter" instead of a "select then apply" filter. This creates problems:
1. **Rapid cursor movement causes filter churn.** Holding `j` to scroll through 10 folders triggers 10 filter-apply cycles. If `applyFilter()` is slow (fuzzy search on large lists), the UI stutters.
2. **Tag mode behavior inconsistency.** If folders auto-filter on cursor move, should tags also auto-filter? Users expect consistent behavior between the two modes.
3. **Loss of "browse sidebar without changing the list" behavior.** Currently users can browse the folder tree to understand structure without affecting the main list. Auto-display removes this ability.

**Warning signs:**
- Users complain the workflow list "jumps around" while browsing folders
- Performance issues with rapid folder switching
- Inconsistent behavior between folder and tag sidebar modes

**Prevention:**
- **Debounce the auto-filter.** Don't apply the filter on every cursor move. Wait 150-200ms after the last cursor movement before applying:
  ```go
  case "up", "k", "down", "j":
      s.moveCursor(delta)
      // Cancel previous debounce, start new one
      return s, tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
          return sidebarAutoFilterMsg{filterType: ft, filterValue: fv}
      })
  ```
- **Apply consistently to both folders and tags.** If folders auto-filter, tags must too.
- **Add a visual indicator** that shows "filtering by: docker/" in the list header so users understand why the list changed.
- **Keep Enter behavior.** Enter should still work as an explicit "lock this filter" action, distinguishing between "browsing sidebar" (auto-filter, tentative) and "committed filter" (Enter, sticky).

**Detection:** Navigate up and down through 10+ folders quickly. Check for visual stuttering or perceived lag in list updates.

**Phase:** Folder auto-display plan.

**Confidence:** MEDIUM — UX design concern, not a hard technical limitation. Debounce approach is straightforward.

---

### Pitfall 10: Per-Field AI Generate in Manage Creates N+1 API Call Pattern

**Feature:** Per-field Generate action for individual variables in manage
**Risk level:** MODERATE — UX + cost

**What happens:** The current AI autofill sends one request to fill all fields (name, description, tags, args) simultaneously. Per-field generate means: user focuses a specific arg, presses a key, and AI generates a value for just that field. This creates an N+1 API pattern — editing 5 args generates 5 separate API calls. Each call to Copilot SDK has latency (500ms-2s) and rate limits. Five sequential calls = 2.5-10 seconds of waiting for a single workflow edit.

**Warning signs:**
- User waits 5+ seconds to fill all parameters individually
- Rate limiting errors from Copilot SDK during rapid per-field generation
- Inconsistent AI output across fields (each call has different context)

**Prevention:**
- **Batch suggestion with field focus.** Instead of one API call per field, send the full workflow context and ask for suggestions for ALL unfilled fields. Cache the result. When the user triggers "generate" on field N, pull from the cached batch result:
  ```go
  type aiSuggestionCache struct {
      workflowCmd string
      suggestions map[string]string // arg_name -> suggested_value
      timestamp   time.Time
  }
  ```
- **Invalidate cache when the command changes.** If the user edits the command field, clear the suggestion cache.
- **Show all suggestions as ghost text / dimmed defaults.** Instead of per-field "generate", show AI-suggested values as placeholders for all args at once. User can accept (tab) or override (type).
- **Timeout handling.** If the API call takes >3 seconds, show the form without suggestions. Don't block the UI.
- **Budget consideration.** Each Copilot API call has token cost. Batch is 1 call; per-field is N calls. Batch is 1/N the cost.

**Detection:** Trigger per-field generate on 5 fields sequentially. Measure total latency and check for rate limiting.

**Phase:** Per-field AI generate design.

**Confidence:** MEDIUM — Copilot SDK rate limits are not well-documented (technical preview).

---

## Low-Risk Pitfalls

Mistakes that cause annoyance but are fixable.

---

### Pitfall 11: Default Value Consistency Between Runtime and Edit Time

**Feature:** Fix default value consistency at runtime/edit time
**Risk level:** LOW — data integrity

**What happens:** Defaults can currently be specified in three places:
1. **In the command template:** `{{name:default_value}}` (parsed by template/parser.go)
2. **In the args YAML field:** `args: [{name: "name", default: "default_value"}]` (stored in store/workflow.go)
3. **Auto-saved defaults** (new in v1.1)

When the picker fills parameters (`initParamFill` in paramfill.go:26-80), it uses `template.ExtractParams()` which reads from the command string. But the `store.Arg` struct may have a different default. If someone edits the command to change `{{env:prod}}` to `{{env:staging}}` but doesn't update the `args[].default` field, the picker uses "staging" (from command) while the manage form shows "prod" (from args YAML).

**Warning signs:**
- Different default values shown in picker vs manage TUI
- Auto-saved default overwritten by template default on next run
- User confusion: "I changed the default but it keeps using the old one"

**Prevention:**
- **Establish a single source of truth.** Recommendation: the `args[].default` field in YAML is authoritative. The command template `{{name:default}}` syntax is a convenience for initial creation only. On save, extract defaults from the command template and populate `args[].default`. On fill, always read from `args[].default`.
- **Merge strategy on param fill:**
  ```go
  // Priority: args[].default > template default > ""
  func resolveDefault(templateParam template.Param, storeArg *store.Arg) string {
      if storeArg != nil && storeArg.Default != "" {
          return storeArg.Default
      }
      return templateParam.Default
  }
  ```
- **Sync defaults when saving.** When the form saves a workflow, extract params from the command, merge with existing args, ensure `args[].default` matches `{{name:default}}` from the template. Surface conflicts to the user.
- **Auto-saved defaults update `args[].default`.** Don't create a third storage location.

**Detection:** Create a workflow with `{{env:prod}}` in command and `default: staging` in args YAML. Run the workflow. Which default appears?

**Phase:** Variable defaults consistency plan.

**Confidence:** HIGH — code paths confirmed across parser.go, paramfill.go, and workflow.go.

---

### Pitfall 12: Parameter CRUD State Management in Value-Copy Bubble Tea

**Feature:** Full CRUD on parameters in wf manage
**Risk level:** LOW — engineering complexity

**What happens:** Adding/removing/reordering args in the manage TUI means mutating a slice (`[]store.Arg`) across Bubble Tea's value-copy Update cycles. Since `Model.Update()` uses value receivers (per v1.0 architecture), the entire model is copied on every keystroke. If the arg slice is stored as a value in a sub-model, adding an arg in one Update cycle creates a new slice that may not propagate correctly through the model tree.

**Warning signs:**
- Arg added in the editor but gone after next keystroke
- Stale arg data after reordering
- Inconsistent state between View() and Update() cycles

**Prevention:**
- **Use the proven `formValues` pointer pattern.** The existing `formValues` struct (form.go:24-30) is heap-allocated via `vals *formValues`. Extend this pattern for args:
  ```go
  type paramEditorState struct {
      args   []store.Arg
      cursor int
      // ... other mutable state
  }
  
  type ParamEditorModel struct {
      state *paramEditorState // pointer — survives value copies
      theme Theme
      // ...
  }
  ```
- **Don't try to manage arg CRUD within the existing `FormModel`.** Create a separate `ParamEditorModel` with its own Update/View cycle.
- **Test: rapidly add/delete args while verifying count is correct.** Write a `teatest` that adds 5 args, deletes 2, reorders 1, and asserts the final arg list matches expectations.

**Detection:** Add an arg, press a key, check if the arg is still present in the View output.

**Phase:** Parameter CRUD implementation.

**Confidence:** HIGH — `formValues` pointer pattern already proven; extension is straightforward.

---

### Pitfall 13: ANSI Escape Codes in Dynamic Command Output Corrupt Field Extraction

**Feature:** List picker dynamic variable type
**Risk level:** LOW — data quality

**What happens:** Many commands produce colorized output by default when attached to a TTY (e.g., `ls --color=auto`, `grep --color=auto`, `kubectl` with `KUBECTL_COLOR=auto`). When the list picker runs a dynamic command via `sh -c`, the output may contain ANSI escape sequences (`\033[31m`, etc.). These invisible characters:
1. Break field extraction (field 1 includes `\033[31m` prefix)
2. Corrupt the selected value stored as a default
3. Display incorrectly in the picker list or mess up column alignment

**Warning signs:**
- Garbled characters in selected values
- Field extraction returns wrong content (offset by escape code length)
- Defaults stored with invisible ANSI codes

**Prevention:**
- **Strip ANSI escape codes from command output before processing:**
  ```go
  var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
  
  func stripANSI(s string) string {
      return ansiRegex.ReplaceAllString(s, "")
  }
  ```
  Apply to every line of dynamic command output before it enters the options list.
- **Set `NO_COLOR=1` in the subprocess environment** to disable color output at the source:
  ```go
  cmd := exec.CommandContext(ctx, "sh", "-c", command)
  cmd.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb")
  ```
  This follows the [NO_COLOR convention](https://no-color.org/) supported by many CLI tools.
- **Both approaches together** (belt and suspenders): set `NO_COLOR` to reduce ANSI output, strip remaining codes as a safety net.

**Detection:** Create a list picker with `dynamic_cmd: "ls --color=always"`. Check if the selected value contains escape codes.

**Phase:** List picker implementation.

**Confidence:** HIGH — ANSI in subprocess output is a well-known issue; `NO_COLOR` convention is widely adopted.

---

## Integration-Specific Pitfalls

Pitfalls arising from how v1.1 features interact with the existing v1.0 architecture.

---

### Pitfall 14: Adding `viewExecute` State Breaks Existing Key Routing Assumptions

**Feature:** Execute flow in manage TUI
**Risk level:** MODERATE — regression risk

**What happens:** The root model routes messages based on `viewState` (model.go:20-24). Adding `viewExecute` requires changes to: (a) the `viewState` enum, (b) the `Update()` switch statement, (c) the `View()` switch statement, (d) `WindowSizeMsg` handling, and (e) transition messages. But the existing code has implicit assumptions:
- `viewCreate` and `viewEdit` share the same `updateForm` handler (model.go:258-259)
- `KeyMsg` routing happens BEFORE state-specific routing (model.go:96-107)
- The dialog overlay takes priority regardless of state (model.go:98-100)

Adding `viewExecute` means: Does the execute view use the dialog overlay? Does Esc exit execute or does it go through the global handler? Can the user quit from execute view?

**Warning signs:**
- Esc key behavior is different in execute view vs other views
- Ctrl+C from execute view leaves the TUI in a broken state
- Dialog overlay doesn't render correctly during execute flow

**Prevention:**
- **Map the full key routing table before coding.** Document which keys do what in each state:
  | Key | Browse | Create/Edit | Settings | Execute (new) |
  |-----|--------|-------------|----------|---------------|
  | esc | — | → Browse | → Browse | → Browse |
  | ctrl+c | quit | quit | quit | quit |
  | enter | — | submit form | — | confirm exec |
- **Follow the existing pattern exactly.** New view states must have their own `updateExecute` handler, called from the same switch in `Update()`. Don't add special cases to the global key handler.
- **Add the new state AFTER the existing states in the iota** to avoid changing existing state values (matters if states are ever serialized or compared numerically):
  ```go
  const (
      viewBrowse   viewState = iota
      viewCreate
      viewEdit
      viewSettings
      viewExecute  // NEW — after existing states
  )
  ```
- **Write a state transition test.** Test that every transition (Browse → Execute → Browse, Browse → Execute → Quit) works correctly and doesn't leave the model in an inconsistent state.

**Detection:** Add `viewExecute`, press Esc, verify you return to Browse. Press Ctrl+C, verify clean exit.

**Phase:** Execute flow implementation.

**Confidence:** HIGH — model.go routing confirmed; integration points are clear.

---

### Pitfall 15: Auto-Save Defaults + Parameter CRUD Create Circular Save Conflicts

**Feature:** Auto-save defaults + parameter CRUD interaction
**Risk level:** MODERATE — data consistency

**What happens:** Two features independently modify the `args` field of a workflow YAML:
1. **Auto-save defaults:** Updates `args[N].default` after parameter fill (triggered from picker)
2. **Parameter CRUD:** Adds/removes/renames args (triggered from manage TUI)

Race condition scenario: User runs a workflow from the picker (auto-save updates defaults in background). Simultaneously, the manage TUI is open on the same workflow, and the user removes an arg. The auto-save writes defaults for an arg that no longer exists. Or worse: auto-save writes the file while manage is mid-edit, and the next manage save overwrites the auto-saved defaults.

**Warning signs:**
- Deleted args reappearing after running the workflow
- Defaults reverting to old values after editing in manage
- YAML file containing stale/orphaned arg entries

**Prevention:**
- **File-level locking for workflow YAML writes.** Use `flock` (on Unix) or a `.lock` file to prevent concurrent writes:
  ```go
  func (s *YAMLStore) Save(w *Workflow) error {
      lockPath := s.WorkflowPath(w.Name) + ".lock"
      lock, _ := os.Create(lockPath)
      defer lock.Close()
      defer os.Remove(lockPath)
      syscall.Flock(int(lock.Fd()), syscall.LOCK_EX)
      defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
      // ... existing save logic
  }
  ```
- **Read-modify-write for auto-save.** Don't cache the workflow struct. Read the current YAML, update only the default field, write back. This ensures auto-save picks up any CRUD changes made since the last read.
- **Sequence auto-saves after the picker exits, not during.** Don't write defaults while the user is still filling params. Write once at the end, after the command is rendered and about to be pasted.
- **In manage TUI, reload workflow from disk before opening edit form.** Don't rely on the in-memory `[]store.Workflow` slice which may be stale if auto-save modified the file.

**Detection:** Open manage TUI on a workflow. From another terminal, run the same workflow via picker (triggering auto-save). Go back to manage TUI and save. Check which defaults survived.

**Phase:** Must be considered when both auto-save and parameter CRUD are planned.

**Confidence:** MEDIUM — race condition is architectural, not yet observable (features don't exist yet). But the risk is real given `os.WriteFile` is non-atomic.

---

### Pitfall 16: `goccy/go-yaml` Marshal Changes Arg Field Ordering After CRUD Edits

**Feature:** Parameter CRUD + auto-save defaults
**Risk level:** LOW — cosmetic but trust-eroding

**What happens:** After parameter CRUD operations (add, remove, reorder args), the workflow is marshaled and saved. `goccy/go-yaml`'s `Marshal()` serializes struct fields in Go struct order, which may differ from the original YAML file's field order. Additionally, `omitempty` fields that were previously present (but empty) may disappear, and fields that were absent may appear. The resulting YAML diff is noisy even if the semantic content is unchanged.

**Warning signs:**
- Git diffs showing field reordering after editing args in manage
- `omitempty` fields appearing/disappearing unpredictably
- YAML output style changes (flow vs block) after round-trip

**Prevention:**
- **Accept and document this behavior.** Full re-marshal on CRUD operations is acceptable — the user explicitly edited the workflow. The concern is specifically for auto-save (Pitfall 2) where NO user edit should cause formatting changes.
- **Standardize field order in `store.Workflow`** struct tag order. Ensure the Go struct field order matches the desired YAML output order (name, command, description, tags, args).
- **For parameter CRUD specifically:** Since the user is explicitly editing, full re-marshal is fine. Just ensure the output is clean and consistent.
- **Test:** Add an arg via CRUD, save, check that existing fields (name, command, etc.) are in the expected order and style.

**Detection:** Create a workflow via YAML editor with specific field ordering. Edit args in manage. Check if non-arg fields moved.

**Phase:** Parameter CRUD implementation.

**Confidence:** HIGH — `goccy/go-yaml` marshaling behavior verified.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|---|---|---|
| **Auto-save defaults** | Non-atomic `os.WriteFile` (#1), YAML re-marshal destroys formatting (#2), default consistency (#11) | Atomic write pattern, surgical AST update or separate defaults file, single source of truth for defaults |
| **Parameter CRUD in TUI** | huh doesn't support dynamic fields (#3), value-copy state management (#12), save conflicts with auto-save (#15) | Custom param editor view (not huh), pointer pattern for mutable state, file-level locking |
| **List picker variable type** | Unbounded shell execution (#4), ANSI codes in output (#13) | Configurable timeout + line limit, strip ANSI + NO_COLOR env var |
| **Execute flow in manage** | Alt-screen conflicts with paste-to-prompt (#5), view state routing (#14) | Clipboard-first or exit-then-print, careful state enum extension |
| **Syntax highlighting** | Per-frame render cost (#6) | Cache highlighted output, limit to preview pane |
| **Warp terminal compat** | Ctrl+G interception (#7) | Configurable fallback keybinding, TERM_PROGRAM detection |
| **Overscroll fix** | ScrollOff not reclamped on height changes (#8) | Bounds checking in renderList, ensureCursorVisible on every height change |
| **Folder auto-display** | Filter churn on rapid cursor movement (#9) | Debounce 150ms, consistent behavior across folder/tag modes |
| **Per-field AI generate** | N+1 API calls (#10) | Batch suggestion with field focus, cache results |

---

## Sources

### HIGH Confidence (codebase analysis, official docs)
- `internal/store/yaml.go:45` — `os.WriteFile` confirmed non-atomic
- `internal/manage/form.go:101-166` — huh form built once at transition, no dynamic field API
- `internal/manage/form.go:24-50` — `formValues` pointer pattern for value-copy survival
- `internal/manage/model.go:20-24` — viewState enum, routing structure
- `internal/manage/browse.go:86-95, 306-314` — listHeight/scrollOff calculations
- `internal/picker/paramfill.go:99-132` — executeDynamic with 5-second timeout
- `internal/template/parser.go:33-58` — ExtractParams with template-level defaults
- `internal/store/workflow.go:18-25` — Arg struct with separate default field
- Go `os.Rename` atomicity — pkg.go.dev/os, POSIX spec
- `goccy/go-yaml` AST support — github.com/goccy/go-yaml `ast` package

### MEDIUM Confidence (cross-referenced from multiple sources)
- Warp terminal Ctrl+G interception — user reports, STATE.md blocker note
- Warp Kitty keyboard protocol bugs — Feb 2026 search results
- NO_COLOR convention — https://no-color.org/
- ANSI stripping regex — widely used pattern, standard escape code format

### LOW Confidence (needs validation)
- `goccy/go-yaml` AST surgical update capabilities — API exists, but precision of format preservation unverified
- Warp terminal specific keybinding workarounds — needs empirical testing
- Copilot SDK rate limits for per-field generation — not documented (technical preview)
