---
status: testing
phase: 01-foundation-data-layer
source: [01-01-SUMMARY.md, 01-02-SUMMARY.md, 01-03-SUMMARY.md, 01-04-SUMMARY.md]
started: 2026-02-21T10:00:00Z
updated: 2026-02-21T10:15:00Z
---

## Current Test

number: 7
name: Template parameters are extracted
expected: |
  Create a workflow with `{{named}}` parameters in the command (e.g., `wf add --name "greet" --command "echo Hello {{name:World}}"`). The workflow's args field in the YAML file contains the extracted parameter with its default value.
awaiting: user response

## Tests

### 1. Create a workflow with wf add
expected: Run `wf add` with name and command, it creates a YAML file. `wf list` shows it.
result: issue
reported: "Workflow created and shows in wf list, but cannot find wf/ folder in ~/.config/. Files are actually stored in ~/Library/Application Support/wf/workflows/ on macOS."
severity: minor

### 2. Edit a workflow with wf edit
expected: Run `./wf edit deploy-staging --command "ssh staging 'cd /app && git pull && make build && make deploy'"` and the command field is updated. Running `./wf edit deploy-staging` (no flags) opens the YAML file in your $EDITOR.
result: pass

### 3. List workflows with wf list
expected: Run `./wf list` and see a compact one-line-per-workflow output showing name, description, and tags. Output is greppable.
result: pass

### 4. Delete a workflow with wf rm
expected: Run `./wf rm deploy-staging` and get a confirmation prompt. Typing "y" deletes the workflow. Running `./wf list` no longer shows it. Alternatively, `./wf rm deploy-staging --force` skips confirmation.
result: pass

### 5. Multiline command survives round-trip
expected: Create a workflow with a multiline command, then check the YAML file — the multiline content is preserved intact without corruption.
result: pass

### 6. Norway Problem values survive
expected: Create a workflow where the name, description, or tags contain values like "no", "yes", "on", "off", "true", "false". The values are stored and retrieved as strings, not converted to YAML booleans.
result: pass

### 7. Template parameters are extracted
expected: Create a workflow with `{{named}}` parameters in the command. The workflow's args field in the YAML file contains the extracted parameter with its default value.
result: [pending]

## Summary

total: 7
passed: 5
issues: 1
pending: 1
skipped: 0

## Gaps

- truth: "Workflow files are stored in ~/.config/wf/ as documented"
  status: failed
  reason: "User reported: Workflow created and shows in wf list, but cannot find wf/ folder in ~/.config/. Files are actually stored in ~/Library/Application Support/wf/workflows/ on macOS."
  severity: minor
  test: 1
  root_cause: "adrg/xdg resolves ConfigHome to ~/Library/Application Support/ on macOS when XDG_CONFIG_HOME is unset. Code comment claims ~/.config/ but that's only true with explicit XDG_CONFIG_HOME."
  artifacts:
    - path: "internal/config/config.go"
      issue: "Comment says ~/.config/wf/workflows/ but xdg.ConfigHome resolves to ~/Library/Application Support/ on macOS"
  missing:
    - "Either set XDG_CONFIG_HOME explicitly, or update comments/docs to reflect actual macOS path, or hardcode ~/.config/ if that's the desired behavior"
  debug_session: ""
