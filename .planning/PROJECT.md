# wf — Terminal Workflow Manager

## What This Is

A command-line workflow manager that lets users save, search, and execute parameterized command templates directly from the terminal. Built as a Go TUI with a hybrid interface: a fast fuzzy picker for daily use and a full management TUI for creating, editing, and organizing workflows. Supports AI-powered workflow generation via Copilot SDK, Pet/Warp import, and git-based workflow sharing across teams. Think Warp.dev's workflow feature, but terminal-native, local-first, and shareable via git.

## Core Value

Users can find and execute any saved command workflow in under 3 seconds, with arguments filled in inline and the result pasted to their prompt.

## Current State

**Shipped:** v1.0 MVP (2026-02-25)
**Codebase:** 11,736 lines of Go across 158 files
**Tech stack:** Go, Bubble Tea, Lip Gloss, huh, Cobra, goccy/go-yaml, Copilot SDK

v1.0 delivers the complete workflow manager: YAML storage with Norway Problem protection, sub-100ms fuzzy picker with paste-to-prompt, full management TUI, advanced parameters (enum/dynamic), Pet/Warp import, AI generation/autofill, and git-based workflow sharing. Shell integration covers zsh, bash, fish, and PowerShell. Cross-platform binary for macOS, Linux, and Windows.

## Requirements

### Validated

- Save parameterized command templates as YAML files — v1.0
- Fuzzy search and execute workflows via hotkey popup — v1.0
- Full TUI for creating, editing, and managing workflows — v1.0
- Inline argument filling with full command visible (Warp-style) — v1.0
- Paste completed command to user's shell prompt — v1.0
- Tags + folder-based organization — v1.0
- AI-powered workflow generation from natural language descriptions (Copilot SDK) — v1.0
- AI auto-fill for workflow metadata (name, description, argument types) — v1.0
- Shell integration plugin (zsh/bash/fish/PowerShell) for hotkey binding — v1.0
- Shareable workflow libraries via git — v1.0
- Enum parameters with predefined option lists — v1.0
- Dynamic parameters populated by shell command output — v1.0
- Register previous shell command as workflow — v1.0
- Import from Pet TOML and Warp YAML formats — v1.0
- Cross-platform binary (macOS/Linux/Windows) — v1.0

### Active

- [ ] Preserve original values as parameter defaults when saving commands as workflows
- [ ] Syntax highlighting in workflow list rendering
- [ ] Auto-display folder contents in manage without extra keypress
- [ ] Warp terminal Ctrl+G compatibility + fallback keybinding
- [ ] Configurable keybinding via `wf init --key` flag
- [ ] Full execute flow inside wf manage (param fill, paste to prompt)
- [ ] Per-field Generate action for individual variables in manage
- [ ] Fix command preview panel overscroll in manage view
- [ ] List picker dynamic variable type (shell command output, select one, store specific field)
- [ ] Full CRUD on parameters in wf manage (add, remove, rename; change type/defaults/description/enum values)

## Current Milestone: v1.1 Polish & Power

**Goal:** Improve manage UX with execute flow, parameter CRUD, and navigation fixes; add list picker variable type; resolve terminal compatibility issues; add syntax highlighting and smarter default handling.

**Target features:**
- Defaults: preserve original values as param defaults when saving commands
- Manage UX: execute flow, parameter CRUD, per-field AI generate, folder auto-display, overscroll fix
- List picker: general-purpose dynamic variable with column extraction
- Terminal compat: Warp Ctrl+G fix + configurable keybinding
- Display: syntax highlighting in workflow list/preview

### Out of Scope

- Cloud sync — local-first, git handles sharing
- Real-time collaboration — this is a personal/team tool, not a SaaS
- Direct command execution in TUI — paste-to-prompt only, user stays in control
- Mobile or web interface — terminal-native only
- Full shell replacement — users are loyal to their shells, wf integrates with them
- Complex workflow orchestration (DAGs, conditionals) — becomes a CI/CD tool
- Shell history mining — Atuin/fzf handle this, wf is for curated workflows

## Context

Shipped v1.0 MVP with 11,736 LOC Go in 6 days (2026-02-19 to 2026-02-25).

Tech stack: Go, Bubble Tea (TUI), Lip Gloss (styling), huh (forms), Cobra (CLI), goccy/go-yaml, atotto/clipboard, Copilot SDK.

Key architectural patterns:
- Store interface with YAMLStore, RemoteStore, MultiStore for local + remote aggregation
- Template engine with priority-ordered parsing: bang (dynamic) > pipe (enum) > colon (default) > plain
- /dev/tty for picker TUI output (fzf approach) — robust under shell redirections
- Shared formValues struct pattern for huh/bubbletea pointer stability

Known tech debt:
- Help text references ctrl+t for settings (actual binding is S)
- 2 stale test expectations in copilot_test.go
- Copilot CLI SDK is technical preview (v0.1.25) — API may change

## Constraints

- **Tech stack:** Go with Bubble Tea (TUI framework), Lip Gloss (styling), Bubbles (components)
- **Distribution:** Single binary, no runtime dependencies
- **Storage:** YAML files in `~/.config/wf/` (XDG-compliant), no database
- **Performance:** Sub-100ms startup for the quick picker
- **AI dependency:** Copilot SDK for AI features — must degrade gracefully without it
- **Shell support:** zsh, bash, fish, PowerShell (plugin optional, binary works standalone)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go over Rust | Simpler language for agentic coding, mature TUI ecosystem (Bubble Tea), fast compilation, single binary | Good — 11.7K LOC in 6 days |
| YAML over SQLite | Human-readable, version-controllable, shareable via git, sufficient performance for workflow-scale data | Good — enables git sharing |
| Paste-to-prompt over direct execution | User stays in control, can review/edit before running, safer for destructive commands | Good — clean UX model |
| Copilot SDK for AI | GitHub ecosystem integration, available SDK, covers both generation and auto-fill use cases | Good — works, but SDK is preview |
| Hybrid interface (picker + TUI) | Quick picker for speed, full TUI for management — matches different usage contexts | Good — clear separation |
| goccy/go-yaml over gopkg.in/yaml.v3 | gopkg.in/yaml.v3 unmaintained since April 2025, goccy handles Norway Problem better | Good |
| /dev/tty for picker output | ZLE redirects fds before subprocess; /dev/tty matches fzf approach | Good — robust |
| Ctrl+G as default keybinding | Overrides readline's rarely-used "abort" in bash, safe in zsh/fish | Good |
| Shared formValues struct | huh pointer bindings invalidated by bubbletea value-copy cycles | Good — essential fix |
| Typed string fields for YAML | Eliminates Norway Problem without custom MarshalYAML | Good |

---
*Last updated: 2026-02-27 after v1.1 requirements scoped*
