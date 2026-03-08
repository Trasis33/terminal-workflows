# wf-workflow-troubleshooting skill

Load this skill when a `wf` workflow already exists and the task is to diagnose or fix it.

Use it for:

- broken `list_cmd` workflows
- wrong inserted values
- command output not matching the configured delimiter or field index
- confusion between `dynamic` and `list`
- shell/runtime mismatches such as PowerShell syntax vs `sh -c`

Do not use it as the primary skill for greenfield workflow creation.

For creating or explaining workflows, use:

- `../wf-workflows/SKILL.md`

Quick rule:

- building or explaining -> `wf-workflows`
- diagnosing or fixing -> `wf-workflow-troubleshooting`
