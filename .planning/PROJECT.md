# wf — Terminal Workflow Manager

## What This Is

A command-line workflow manager that lets users save, search, and execute parameterized command templates directly from the terminal. Built as a Go TUI with a hybrid interface: a fast fuzzy picker for daily use and a full management TUI for creating, editing, and organizing workflows. Think Warp.dev's workflow feature, but terminal-native, local-first, and shareable via git.

## Core Value

Users can find and execute any saved command workflow in under 3 seconds, with arguments filled in inline and the result pasted to their prompt.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Save parameterized command templates as YAML files
- [ ] Fuzzy search and execute workflows via hotkey popup
- [ ] Full TUI for creating, editing, and managing workflows
- [ ] Inline argument filling with full command visible (Warp-style)
- [ ] Paste completed command to user's shell prompt
- [ ] Tags + folder-based organization
- [ ] AI-powered workflow generation from natural language descriptions (Copilot SDK)
- [ ] AI auto-fill for workflow metadata (name, description, argument types)
- [ ] Shell integration plugin (zsh/bash/fish) for hotkey binding
- [ ] Shareable workflow libraries via git

### Out of Scope

- Cloud sync — local-first, git handles sharing
- Real-time collaboration — this is a personal/team tool, not a SaaS
- Direct command execution in TUI — paste-to-prompt only, user stays in control
- Python implementation — dependency management friction
- Mobile or web interface — terminal-native only

## Context

Inspired by Warp.dev's workflow system (parameterized commands with names, descriptions, arguments, defaults, and enum types). Warp's approach is cloud-first and GUI-embedded. This project extracts the core value — saved command templates with parameters — and makes it work in any terminal via a standalone TUI.

Key Warp workflow concepts to adopt:
- **Arguments:** Named parameters in `{{double_braces}}` syntax with types (text, enum), defaults, and descriptions
- **Enum arguments:** Predefined values or dynamic values from shell command output
- **Searchability:** Workflows indexed by name, description, tags, and command content
- **Organization:** Folders + tags, personal + shared workspaces

Key differentiators from Warp:
- Works in any terminal (not locked to Warp)
- Local-first YAML storage (version-controllable)
- AI features via Copilot SDK for workflow creation
- Single binary distribution (Go)

This is a greenfield project. No existing codebase.

## Constraints

- **Tech stack:** Go with Bubble Tea (TUI framework), Lip Gloss (styling), Bubbles (components)
- **Distribution:** Single binary, no runtime dependencies
- **Storage:** YAML files in `~/.config/wf/` (XDG-compliant), no database
- **Performance:** Sub-100ms startup for the quick picker
- **AI dependency:** Copilot SDK for AI features — must degrade gracefully without it
- **Shell support:** zsh, bash, fish (plugin optional, binary works standalone)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go over Rust | Simpler language for agentic coding, mature TUI ecosystem (Bubble Tea), fast compilation, single binary | — Pending |
| YAML over SQLite | Human-readable, version-controllable, shareable via git, sufficient performance for workflow-scale data | — Pending |
| Paste-to-prompt over direct execution | User stays in control, can review/edit before running, safer for destructive commands | — Pending |
| Copilot SDK for AI | GitHub ecosystem integration, available SDK, covers both generation and auto-fill use cases | — Pending |
| Hybrid interface (picker + TUI) | Quick picker for speed, full TUI for management — matches different usage contexts | — Pending |

---
*Last updated: 2026-02-19 after initialization*
