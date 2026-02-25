---
status: diagnosed
phase: 04-advanced-parameters-import
source: [04-01-SUMMARY.md, 04-02-SUMMARY.md, 04-03-SUMMARY.md, 04-04-SUMMARY.md, 04-05-SUMMARY.md, 04-06-SUMMARY.md]
started: 2026-02-24T11:00:00Z
updated: 2026-02-24T11:15:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Enum Parameter Syntax in Workflow
expected: Create a workflow with an enum parameter using `wf add`. Use the syntax `{{env|dev|staging|*prod}}` in the command field. The workflow should save successfully and `wf list` should show it. Running `cat` on the YAML file should show the enum syntax preserved with the default marker (*prod).
result: pass

### 2. Dynamic Parameter Syntax in Workflow
expected: Create a workflow with a dynamic parameter using `wf add`. Use the syntax `{{branch!git branch --list}}` in the command field. The workflow should save successfully with the dynamic command syntax preserved in the YAML file.
result: pass

### 3. Enum Parameter Selection in Picker
expected: Run `wf pick` (or `wf`), select the workflow with the enum parameter. During parameter filling, you should see a vertical list of options (dev, staging, prod) with a `❯` cursor. Arrow keys should cycle through options. The default option (prod) should be pre-selected. The live command preview should update as you change selection.
result: pass

### 4. Dynamic Parameter Loading in Picker
expected: Run `wf pick` and select the workflow with the dynamic parameter (`git branch`). You should briefly see a "Loading..." state while the shell command executes. Once loaded, a list of your git branches should appear as selectable options with arrow-key navigation.
result: pass

### 5. Dynamic Parameter Failure Fallback
expected: Create a workflow with a dynamic parameter that runs a command that will fail (e.g., `{{val!nonexistent-command-xyz}}`). When selecting this workflow in the picker, after the command fails, you should see a free-text input field instead of a list — allowing you to type the value manually.
result: pass

### 6. Register Last Shell Command
expected: Run any command in your shell (e.g., `echo hello world`), then run `wf register`. It should capture your last shell command and prompt you for a name and other metadata. After saving, `wf list` should show the new workflow.
result: issue
reported: "I ran 'echo hello world', but it registered 'npm run dev' which has been run previously"
severity: major

### 7. Register with Auto-Detection
expected: Run `wf register 'curl https://api.example.com/users'`. The tool should detect the URL as a parameterizable pattern and offer to convert it to a `{{url}}` parameter. You can accept or decline. After saving, the workflow should contain the parameterized (or original) command.
result: pass

### 8. Register with --pick Mode
expected: Run `wf register --pick`. You should see a list of recent shell history commands to choose from. Selecting one should proceed with the normal register flow (name, description, auto-detection).
result: pass

### 9. Import from Pet TOML
expected: Create a simple Pet-format TOML file with a snippet (e.g., `[[snippets]]` with description, command, tag, output). Run `wf import pet <file>`. You should see a preview table showing workflows to import, then they should be saved. Any unmappable fields (like `output`) should appear as YAML comments in the saved file.
result: issue
reported: "Error: parsing Pet file: parsing pet TOML: toml: cannot decode TOML string into struct field importer.PetSnippet.Tag of type []string"
severity: blocker

### 10. Import from Warp YAML
expected: Create a simple Warp-format YAML file with a workflow (name, command, arguments). Run `wf import warp <file>`. You should see a preview table, workflows saved, and any unmappable fields (author, shells) preserved as YAML comments.
result: pass

### 11. Import Conflict Resolution
expected: Import a file that contains a workflow with the same name as an existing one. You should be prompted with options to skip, rename, or overwrite the conflicting workflow.
result: pass

### 12. Import with --folder Flag
expected: Run `wf import warp test-warp.yaml --folder imported`. The imported workflows should be saved with the folder prefix in their name (e.g., `imported/deploy-service`) and the YAML files should be in a corresponding subdirectory.
result: pass

## Summary

total: 12
passed: 10
issues: 2
pending: 0
skipped: 0

## Gaps

- truth: "wf register captures the most recently run shell command"
  status: failed
  reason: "User reported: I ran 'echo hello world', but it registered 'npm run dev' which has been run previously"
  severity: major
  test: 6
  root_cause: "Shells (bash/zsh) only flush history to disk on session exit. wf register reads $HISTFILE which contains only commands from prior sessions, not the current in-memory shell history."
  artifacts:
    - path: "internal/history/detect.go"
      issue: "NewReader() reads $HISTFILE at construction time with no flush mechanism"
    - path: "cmd/wf/register.go"
      issue: "lastFromHistory() calls history.NewReader() + Last() with no pre-flush or user warning"
  missing:
    - "Extend shell integration scripts (zsh/bash) to write last command to a sidecar file (~/.local/share/wf/last_cmd) via precmd/PROMPT_COMMAND hook"
    - "wf register reads sidecar file first (immune to flush timing) and falls back to $HISTFILE"
    - "Short-term: emit a warning when running in no-arg mode on bash/zsh that history may not be flushed"
  debug_session: ""

- truth: "wf import pet <file> parses a Pet TOML file with a single-string tag field and imports workflows"
  status: failed
  reason: "User reported: Error: parsing Pet file: parsing pet TOML: toml: cannot decode TOML string into struct field importer.PetSnippet.Tag of type []string"
  severity: blocker
  test: 9
  root_cause: "The PetSnippet.Tag struct field is correctly typed as []string (matching upstream Pet schema), but the test input file used tag = 'k8s' (bare string) instead of tag = ['k8s'] (array). The struct is correct; we should add a lenient unmarshaler to tolerate both forms since hand-edited Pet files in the wild may use bare strings."
  artifacts:
    - path: "internal/importer/pet.go"
      issue: "PetSnippet.Tag is []string — correct per spec, but no tolerance for bare string input"
  missing:
    - "Add custom UnmarshalTOML or StringSlice type on PetSnippet to accept both string and []string for the tag field"
    - "Add a test covering bare-string tag input to prevent regression"
  debug_session: ""
