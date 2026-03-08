---
name: wf-workflow-troubleshooting
description: Diagnose and fix wf workflow problems, especially broken param behavior, list_cmd issues, shell command output mismatches, and YAML metadata mistakes.
---

Use this skill when a `wf` workflow exists but does not behave as expected.

This skill focuses on diagnosis, not greenfield authoring. For workflow creation and explanation, use `wf-workflows` instead.

## Primary Debug Model

When a workflow is broken, identify which layer is failing:

1. command template layer
2. arg metadata layer
3. source command layer
4. extraction layer
5. runtime UX layer

Always separate these explicitly.

## Fast Triage Questions

Work through these in order:

1. Does the workflow YAML parse and include the intended `args` entry?
2. Does the parameter name in `args[].name` exactly match the `{{param}}` placeholder?
3. Is the parameter type correct: `text`, `enum`, `dynamic`, or `list`?
4. If `dynamic`, does `dynamic_cmd` produce plain values, one per line?
5. If `list`, does `list_cmd` produce the expected rows, one per line?
6. If `list`, does `list_delimiter` literally exist in every selectable row?
7. If `list`, is `list_field_index` valid and 1-based?
8. Is `list_skip_header` removing only header rows instead of all rows?

## Mental Models By Type

### `text`

Check:

- placeholder name
- default value expectations
- rendered command output

### `enum`

Check:

- options are present
- intended default is present
- the param did not silently fall back to text because metadata was missing

### `dynamic`

Check:

- `dynamic_cmd` runs successfully under `sh -c`
- output is one value per line
- no extra formatting is leaking into the value list
- an empty result is not being mistaken for a bug in picker logic

### `list`

Check:

- `list_cmd` runs successfully under `sh -c`
- output rows look exactly like the extraction logic expects
- delimiter is literal, not regex or CSV-aware
- field index is 1-based
- whole-row fallback is intentional when field index is `0` or empty
- header skipping happens before display and selection

## Most Common `list_cmd` Failures

### 1. Wrong mental model

Symptom:

- user expects `{{param}}` to fetch live values on its own

Fix:

- explain that `list_cmd` is the live data source
- explain that `{{param}}` is only the insertion point in `command`

### 2. Fake test data mistaken for production design

Symptom:

- workflow uses `printf` but the user expects real Git, Bitbucket, or Azure values

Fix:

- replace `printf` with a real command in `list_cmd`
- explain what external tools or env vars are required

### 3. Delimiter mismatch

Symptom:

- selection opens, but confirm fails or inserts the whole row unexpectedly

Fix:

- inspect raw output
- confirm the delimiter literally appears in each row
- prefer `|` or tab over ambiguous spacing

### 4. Field index mismatch

Symptom:

- wrong value gets inserted
- extraction error on some rows

Fix:

- count columns from 1, not 0
- ensure every selectable row has enough fields
- use `0` only if whole-row insertion is desired

### 5. Header skip removes everything

Symptom:

- empty list even though the source command returned output

Fix:

- compare actual row count with `list_skip_header`
- reduce the skip count or remove it

### 6. Shell/runtime mismatch

Symptom:

- command works in PowerShell but fails in `wf`

Fix:

- remember current implementation runs via `sh -c`
- rewrite commands in POSIX-shell-friendly syntax
- do not assume PowerShell syntax works unless the runtime changes

## Troubleshooting Procedure

When debugging a workflow, do this in order:

1. show or inspect the workflow YAML
2. identify the exact param being collected
3. explain what runtime behavior should happen for that param type
4. isolate the source command (`dynamic_cmd` or `list_cmd`)
5. inspect its raw line output
6. verify extraction settings against that raw output
7. explain whether the bug is in data source, metadata, or expectations
8. propose the smallest fix

## Debug Explanations To Give Users

Use explanations like:

- "Your list opens correctly, so the picker is working; the problem is the field extraction settings."
- "The command is returning rows, but your delimiter does not appear in them literally."
- "This works in PowerShell, but `wf` currently runs list commands with `sh -c`, so the command needs POSIX syntax."
- "You want row context plus a single inserted value, so `list` is correct; the bug is in the row format, not the feature choice."

Avoid vague explanations like:

- "The workflow is broken somehow."
- "The picker is buggy."
- "Maybe YAML is wrong."

## Known Truths About Current Implementation

Stay consistent with the current code behavior:

- `list_cmd` runs when the picker opens
- each output line becomes one row
- empty lines are ignored
- extraction is literal string splitting, not structured parsing
- `list_field_index` is 1-based
- `list_skip_header` removes rows before display
- command failures can block progression
- parse failures should stay retryable while the list remains open

## What Good Fixes Look Like

Prefer fixes that:

- change only `list_cmd` when the source data is wrong
- change only delimiter/index when extraction is wrong
- change the param type only when the mental model is wrong
- preserve realistic live commands instead of replacing them with fake data

## Escalation Rule

If the workflow goal is simply "pick one line and insert it unchanged," recommend simplifying from `list` to `dynamic`.

If the workflow goal requires contextual rows but a single inserted identifier, keep `list` and fix the row format or extraction metadata.
