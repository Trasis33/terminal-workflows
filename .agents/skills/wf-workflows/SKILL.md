---
name: wf-workflows
description: Create, explain, and modify wf workflow YAML, including shell-backed list_cmd parameters and practical workflow patterns.
---

Use this skill when the user wants to create, edit, explain, or troubleshoot `wf` workflows.

This skill is especially important for `type: list` parameters, because list behavior does not come from the `{{param}}` command placeholder itself. It comes from persisted arg metadata such as `list_cmd`, `list_delimiter`, `list_field_index`, and `list_skip_header`.

## Core Mental Model

A `wf` workflow has two layers:

1. The command template
2. The arg metadata

Example:

```yaml
name: git-switch-branch
command: git switch {{branch}}
description: Switch to a recent branch
args:
  - name: branch
    type: list
    list_cmd: >-
      git for-each-ref --sort=-committerdate
      --format='%(refname:short)|%(committerdate:relative)|%(subject)'
      refs/heads
    list_delimiter: "|"
    list_field_index: 1
    list_skip_header: 0
```

Important:

- `command` says where the final value is inserted
- `args` says how the value is collected
- `{{branch}}` is only the placeholder
- `type: list` turns that placeholder into a shell-backed picker at runtime

## Workflow Schema

Top-level workflow fields:

```yaml
name: string
command: string
description: string
tags:
  - string
args:
  - name: string
    description: string
    default: string
    type: text | enum | dynamic | list
```

Arg metadata by type:

- `text`
  - `default`
- `enum`
  - `options`
- `dynamic`
  - `dynamic_cmd`
- `list`
  - `list_cmd`
  - `list_delimiter`
  - `list_field_index`
  - `list_skip_header`

## Command Placeholder Syntax

The command template supports these inline forms:

- `{{name}}` -> text
- `{{name:default}}` -> text with default
- `{{name|opt1|opt2|*default}}` -> enum
- `{{name!command}}` -> dynamic

Current rule:

- there is no inline `list` syntax in the command string
- `list` must be defined in arg metadata, typically through `wf manage` or direct YAML editing

## How `list_cmd` Actually Works

When the user opens a `type: list` param:

1. `wf` runs `list_cmd`
2. each non-empty output line becomes one row
3. `list_skip_header` removes leading rows before selection
4. the UI filters visible rows as the user types
5. the user picks one row
6. if `list_field_index` is `0` or unset, the whole row is inserted
7. otherwise `wf` splits the row using the literal `list_delimiter` and inserts the selected 1-based field

Example output:

```text
feature/list-picker|2 hours ago|finish list picker
main|1 day ago|merge release prep
fix/session-bug|3 days ago|handle expired token
```

With:

```yaml
list_delimiter: "|"
list_field_index: 1
```

Selecting the first row inserts:

```text
feature/list-picker
```

Not the whole row.

## `dynamic` vs `list`

Use `dynamic` when:

- one output line equals one final value
- no extra row context is needed

Use `list` when:

- the user needs to see contextual columns before choosing
- the final inserted value is only one part of the row

Good `dynamic` fit:

```yaml
command: git checkout {{branch!git branch --format='%(refname:short)'}}
```

Good `list` fit:

```yaml
command: git switch {{branch}}
args:
  - name: branch
    type: list
    list_cmd: >-
      git for-each-ref --sort=-committerdate
      --format='%(refname:short)|%(committerdate:relative)|%(subject)'
      refs/heads
    list_delimiter: "|"
    list_field_index: 1
```

## Current Implementation Limits

Stay accurate to the current code:

- `list_cmd` is executed via `sh -c`
- extraction uses literal `strings.Split`
- `list_field_index` is 1-based
- `0` or empty means whole-row fallback
- header skipping removes rows before selection
- there is no quote-aware CSV parsing
- there is no multi-select

Implication:

- write `list_cmd` using POSIX-shell-friendly syntax
- do not describe PowerShell-native list commands as if they run directly unless the runtime changes

## Authoring Guidance

Prefer these workflows:

1. For simple text, enum, or dynamic workflows, `wf add` is fine.
2. For `type: list`, prefer `wf manage` or direct YAML editing.
3. Keep the command template simple: usually just `{{param}}`.
4. Put collection behavior in arg metadata.
5. If you need multi-column display, format the command output yourself using a literal delimiter such as `|` or tab.

When generating workflows for users:

- explain where the live values come from
- name the real data source command in `list_cmd`
- say what gets displayed vs what gets inserted
- avoid fake `printf` data except for testing examples

## Practical Examples

### 1. Git Branch Picker

```yaml
name: git-switch-branch
command: git switch {{branch}}
description: Switch to a recent local branch with commit context
tags:
  - git
  - branch
args:
  - name: branch
    type: list
    description: Recent local branches with latest commit context
    list_cmd: >-
      git for-each-ref --sort=-committerdate
      --format='%(refname:short)|%(committerdate:relative)|%(subject)'
      refs/heads
    list_delimiter: "|"
    list_field_index: 1
```

What the user sees:

- branch name
- last commit age
- last commit subject

What gets inserted:

- branch name only

### 2. Bitbucket PR Picker

Requires `curl` and `jq`, plus Bitbucket credentials in env vars.

```yaml
name: bitbucket-pr-view
command: bb pr view {{pr_id}}
description: Open a pull request chosen from live Bitbucket data
tags:
  - bitbucket
  - pr
args:
  - name: pr_id
    type: list
    description: Open PR ID
    list_cmd: >-
      curl -s -u "$BB_USER:$BB_APP_PASSWORD"
      "https://api.bitbucket.org/2.0/repositories/$BB_WORKSPACE/$BB_REPO/pullrequests?q=state=\"OPEN\""
      | jq -r '.values[] | "\(.id)|\(.title)|\(.author.display_name)|\(.state)"'
    list_delimiter: "|"
    list_field_index: 1
```

What the user sees:

- PR id
- title
- author
- state

What gets inserted:

- PR id only

### 3. Azure Pipeline Run Picker

Requires Azure CLI and `jq`.

```yaml
name: azure-rerun-pipeline
command: az pipelines runs rerun --id {{run_id}} --org {{org}} --project {{project}}
description: Re-run a recent Azure pipeline run
tags:
  - azure
  - pipelines
args:
  - name: run_id
    type: list
    description: Recent pipeline runs
    list_cmd: >-
      az pipelines runs list --top 20 --output json
      | jq -r '.[] | "\(.id)|\(.definition.name)|\(.sourceBranch)|\(.result // "inProgress")"'
    list_delimiter: "|"
    list_field_index: 1
  - name: org
    default: https://dev.azure.com/your-org
  - name: project
    default: your-project
```

What the user sees:

- run id
- pipeline name
- branch
- result

What gets inserted:

- run id only

## Testing Example

Use fake data only for testing the UI, not as the real workflow design.

```yaml
name: list-picker-test
command: printf 'Selected value: %s\n' '{{item}}'
description: Test list picker behavior
args:
  - name: item
    type: list
    list_cmd: "printf 'NAME|ID|ENV\napi|pod-123|prod\nworker|pod-456|stage\nbrokenrow\n'"
    list_delimiter: "|"
    list_field_index: 2
    list_skip_header: 1
```

This is useful for:

- header-skip testing
- filtering and renumbering checks
- parse-error retry checks

## Agent Behavior

When asked to create a `wf` workflow:

- prefer complete YAML over vague descriptions
- if `list` is involved, explain the data source command explicitly
- choose `dynamic` unless the user truly needs multi-column context
- choose `list` only when the displayed row and inserted value should differ
- keep examples realistic for the user's domain: git, Bitbucket, Azure pipelines
- mention required external tools such as `jq`, `az`, `curl`, or `bb`

When asked to explain an existing workflow:

- explain the command template separately from arg metadata
- say what runs at selection time
- say what the user sees
- say what final value is inserted

When asked to troubleshoot:

- inspect `list_cmd` first
- confirm its raw output line format
- confirm the delimiter matches the output literally
- confirm the chosen field index exists on every row the user might select
- confirm header skipping is not removing all rows

## Good Explanations To Give Users

Say:

- "`list_cmd` is the live data source. It runs when the picker opens."
- "The row is for human context; the extracted field is the inserted value."
- "Use `dynamic` for plain values, `list` for rich rows."

Do not say:

- "The placeholder itself fetches the values."
- "`printf` is the real source of dynamic data."
- "`list` is just a prettier `dynamic` param."
