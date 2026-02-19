# Feature Landscape

**Domain:** Terminal workflow / command snippet managers
**Researched:** 2026-02-19
**Confidence:** HIGH (verified against official GitHub READMEs for Pet, Navi, Hoard, SnipKit, nap; Warp docs via search results; multiple ecosystem surveys cross-referenced)

## Competitive Landscape Summary

Before categorizing features, here's what was analyzed:

| Tool | Stars | Language | Last Release | Core Model |
|------|-------|----------|--------------|------------|
| **Navi** | 16.7k | Rust | Jan 2025 | Interactive cheatsheets with dynamic argument suggestions |
| **Pet** | 5.1k | Go | Dec 2024 | Simple snippet CRUD + Gist sync |
| **Hoard** | 643 | Rust | Aug 2023 | Namespaced command organizer + ChatGPT + sync server |
| **Warp Workflows** | N/A (proprietary) | N/A | Ongoing | YAML workflows with `{{arg}}` syntax, Warp Drive sharing |
| **SnipKit** | 115 | Go | Jan 2026 | Meta-manager that loads from other snippet managers + AI assistant |
| **nap** | ~3k | Go | Active | Code snippet TUI with Charmbracelet, folder-based |
| **tldr** | 52k+ | Multi | Active | Community-maintained simplified man pages (read-only) |

---

## Table Stakes

Features users expect. Missing = product feels incomplete or unusable for daily workflow.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Save/create commands** | Core CRUD. Every tool does this. Without it, there's nothing. | Low | Pet: `pet new`, Hoard: `hoard new`, Navi: write `.cheat` files |
| **Fuzzy search** | Users have hundreds of snippets. Linear scanning is unusable. fzf-style search is the universal expectation. | Low | Pet/Navi use fzf/skim. Hoard/SnipKit have built-in fuzzy. This is non-negotiable. |
| **Parameterized commands** | Static snippets are barely better than shell aliases. Parameters with `<name>` or `{{name}}` syntax are standard across all tools. | Med | Pet: `<param=default>`, Warp: `{{arg}}`, Hoard: `#param`, Navi: `<var>` with `$` suggestions. Each tool has a different syntax. |
| **Default values for parameters** | Reduces friction when 80% of the time you use the same value. Pet, Warp, and Hoard all support defaults. | Low | Pet: `<param=default_value>`, Warp: `default_value` in YAML args, Hoard: named params |
| **Descriptions/documentation** | Users forget why they saved a command. Every tool stores a description alongside the command. | Low | Essential metadata: command + description + tags is the minimum triple |
| **Tags/categories** | Users need to filter snippets by domain (docker, git, k8s). All mature tools support tags. | Low | Pet: `tag = ["network", "google"]`, Hoard: namespaces + tags, Warp: `tags: [...]`, Navi: `% tag1, tag2` header |
| **Direct execution** | The whole point: find it and run it. Copy-paste is the fallback; exec is the feature. | Low | Pet: `pet exec`, SnipKit: `snipkit exec`, Hoard: select + Enter |
| **Shell integration (Ctrl-R replacement)** | Must integrate into natural shell workflow. Adding a keybinding (like Ctrl-R) to invoke the picker is standard. | Med | Pet/Navi both provide shell widget scripts for bash/zsh/fish. This is the "RECOMMENDED" usage pattern per Pet's README. |
| **Edit snippets** | Users need to modify saved commands. Must be easy (open in editor or inline). | Low | Pet: `pet edit` opens TOML in editor. Hoard: `hoard edit <name>`. |
| **Human-readable storage format** | Users expect to read/edit/version-control their snippets. TOML, YAML, or plain text files are expected. | Low | Pet: TOML, Warp: YAML, Navi: `.cheat` files, Hoard: `trove.yml`. Binary/database storage is an anti-pattern for this domain. |
| **Cross-platform (macOS + Linux)** | Developer tools must work on both. Windows is nice-to-have. | Med | All surveyed tools support macOS + Linux. Pet/Hoard/SnipKit also installable via Homebrew. |
| **Clipboard copy** | Sometimes you want to copy, not execute (paste into a different context). | Low | Pet: `pet clip`, nap: clipboard integration built-in |

---

## Differentiators

Features that set a product apart. Not universally expected, but create competitive advantage when done well.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Hybrid interface (quick picker + full TUI)** | Two modes: fast fuzzy picker for daily use, full management TUI for CRUD. No existing tool does both well. Pet is CLI-only. nap is TUI-only. Neither combines a fast invocation path with a rich management view. | High | **This is wf's biggest differentiator.** Navi comes closest with its shell widget vs standalone modes, but has no management TUI. |
| **AI-powered workflow creation** | Natural language to parameterized command. Hoard has basic ChatGPT. SnipKit has multi-provider AI assistant (OpenAI, Anthropic, Gemini, Ollama). Warp has deep AI. But no *standalone* open-source CLI tool does this well with Copilot SDK specifically. | High | Hoard's GPT integration is basic (prompt in list view). SnipKit's assistant is more sophisticated (chat interface, save to library). wf using Copilot SDK would be unique positioning. |
| **Inline argument filling (full command visible)** | When filling args, show the complete command with cursor on the parameter being filled. Most tools show a separate prompt per parameter (Pet, Hoard) or a generic fzf overlay (Navi). Seeing the full command while filling is rare and superior UX. | Med | Warp does this in their terminal UI. Pet's `--raw` mode + shell functions approximate it. No standalone CLI tool does this natively. |
| **Enum/select parameters** | Parameters with predefined options (not just free text). Warp supports `enum` type args. SnipKit supports predefined values. Pet does not. | Med | Warp: enum args with static values or shell-command-generated lists. SnipKit: predefined values. Navi: `$ var: command` for dynamic suggestions. Very useful for deploy targets, environments, etc. |
| **Dynamic parameter suggestions** | Values populated from shell command output (e.g., `git branch`, `docker ps`). Navi's killer feature. | Med | Navi: `$ branch: git branch | awk '{print $NF}'`. Warp: enum with shell command. This bridges "snippet" and "interactive tool". |
| **Multi-step workflows (command chaining)** | Chain multiple commands into a single workflow. Beyond single-command snippets. | High | Warp Workflows and Navi aliases get close. Most snippet managers are single-command only. Multi-step with intermediate params is genuinely hard. |
| **Shareable via git** | Snippets stored as plain files in a dotfiles-friendly location. Push to git, clone on another machine. | Low | Pet syncs via Gist. Navi imports from git repos. Hoard has its own sync server. Plain YAML files in `~/.config/wf/` with git is simpler and more flexible than all of these. |
| **Import from other tools** | Migration path from Pet, Navi, or Warp workflows. | Med | SnipKit does this well (loads from Pet, Gist, MassCode, SnippetsLab). Being able to import existing Pet TOML or Warp YAML is a strong adoption accelerator. |
| **Community/shared cheatsheet repos** | Browse and import community-maintained workflow packs (like Navi's featured repos or tldr pages). | Med | Navi: `navi repo browse` for community cheatsheets. tldr: massive community. This creates a network effect. |
| **Folder organization** | Hierarchical organization beyond flat tags. nap uses folders. Warp Drive has team/personal folders. | Low | Tags are flat. Folders add hierarchy. Both is ideal: `~/.config/wf/docker/`, `~/.config/wf/k8s/` + tags within. |
| **Search by tag filter** | Filter search to specific tags before fuzzy matching. | Low | Pet: `pet exec -t google`. Narrows the search space. Simple but valuable with large collections. |
| **Multiple parameter syntaxes (named, repeated)** | Named parameters that auto-fill all occurrences. Pet: `#first` in Hoard fills all `#first` instances. | Low | Hoard: `#first` used multiple times, fill once. Pet: separate `<param>` instances. Named+repeated is better UX. |
| **Password/sensitive parameter masking** | Mask sensitive input during parameter filling. | Low | SnipKit supports `password` parameter type. Useful for secrets that shouldn't appear in terminal history. |
| **Path autocomplete in parameters** | Tab-complete file paths during parameter filling. | Med | SnipKit supports `path` parameter type with autocomplete. Nice DX. |
| **Multiline command support** | Save and execute multi-line commands (heredocs, multi-line scripts). | Low | Pet: `pet new --multiline`. Important for complex commands. |
| **Register previous command** | Quickly save the last command you ran. Reduces friction in the save flow. | Low | Pet: `prev()` shell function grabs last command from history. Very ergonomic. |
| **Auto-sync** | Automatically sync snippets on edit/create. | Med | Pet: `auto_sync = true` in config. Hoard: `hoard sync save/get`. Removes manual sync step. |
| **Tmux integration** | Use workflows from within any tmux pane, including SSH sessions. | Med | Navi: Tmux widget. Extends reach beyond local shell. |
| **Custom themes/styling** | Configurable colors and layout for the TUI. | Low | SnipKit: built-in themes + custom themes. nap: theme customization via config. |
| **Output storage** | Save expected output alongside the command for reference. | Low | Pet: `output` field in TOML. Useful as documentation/verification. |

---

## Anti-Features

Features to explicitly NOT build. Common mistakes in this domain.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Full shell replacement** | Warp went this route and it's a massive scope trap. wf should be a tool *within* any shell, not replace the shell. Users are loyal to their shells (bash/zsh/fish). | Stay a standalone binary that integrates via keybindings. Work inside any terminal emulator and shell. |
| **Cloud-hosted sync service** | Hoard runs its own sync server. This adds infrastructure burden, security concerns, and vendor lock-in for an OSS tool. | Use plain files + git. Let users choose their own sync mechanism (GitHub, GitLab, private git, Syncthing, etc.). |
| **Binary/database storage** | SQLite or custom binary formats prevent users from editing, grep-ing, or version-controlling snippets. intelli-shell uses SQLite and has stalled. | YAML files in `~/.config/wf/`. Human-readable, git-friendly, editor-friendly. |
| **GUI/Electron app** | massCode is Electron. Desktop snippet managers compete in a different space. Terminal users want terminal-native tools. | Pure TUI. Zero web dependencies. Single binary. |
| **Real-time collaboration** | Warp Drive has team sharing, notebooks, live sessions. This is enterprise SaaS scope. | Share via git repos. "Team workflows" = shared git repo. Keep it simple. |
| **Complex workflow orchestration (DAGs, conditionals)** | This turns into a CI/CD system (GitHub Actions, Taskfile). wf is for *interactive* command templates, not automation pipelines. | Keep workflows as single commands or simple command chains. For complex orchestration, point users to `task`, `just`, or `make`. |
| **Automatic shell history mining** | Mining and suggesting from shell history sounds cool but creates noise. Atuin/fzf already do this well. | Focus on *curated* workflows. The value is in intentionally saved, documented, parameterized commands, not raw history. |
| **Plugin system (v1)** | Plugin architectures add API surface, versioning headaches, and complexity before the core is proven. | Build a solid core. Extensions can come later via cheatsheet repos or simple script hooks. |
| **In-app snippet execution sandboxing** | Trying to sandbox or preview command effects adds enormous complexity for marginal safety. | Show the full command before execution. Let the user review and confirm. Trust the user. |

---

## Feature Dependencies

```
Core Storage (YAML files) ──────────────────────────────────────────────┐
  │                                                                     │
  ├── Save/Create commands                                              │
  │     └── Register previous command (shell function)                  │
  │                                                                     │
  ├── Parameterized commands                                            │
  │     ├── Default values                                              │
  │     ├── Enum/select parameters                                      │
  │     ├── Dynamic parameter suggestions (shell command)               │
  │     ├── Named/repeated parameters                                   │
  │     ├── Password masking                                            │
  │     └── Path autocomplete                                           │
  │                                                                     │
  ├── Tags + Folder organization                                        │
  │     └── Search by tag filter                                        │
  │                                                                     │
  ├── Fuzzy search                                                      │
  │     └── Quick fuzzy picker (Ctrl-R style)                           │
  │           └── Shell integration (keybindings)                       │
  │                 └── Paste-to-prompt execution                       │
  │                                                                     │
  ├── Full management TUI                                               │
  │     ├── Create/Edit/Delete workflows in TUI                         │
  │     ├── Tag management                                              │
  │     └── Custom themes/styling                                       │
  │                                                                     │
  ├── Inline argument filling (full command visible)                    │
  │     └── Requires: parameterized commands + TUI rendering            │
  │                                                                     │
  ├── AI-powered creation (Copilot SDK)                                 │
  │     └── Requires: parameterized command format + save flow          │
  │                                                                     │
  ├── Import from other tools                                           │
  │     └── Requires: core storage format defined                       │
  │                                                                     │
  └── Shareable via git                                                 │
        └── Plain YAML files (inherent, no extra work)                  │
```

---

## MVP Recommendation

For MVP, prioritize these in order:

### Must Ship (Table Stakes)
1. **YAML-based workflow storage** in `~/.config/wf/` - Foundation for everything
2. **Save/create workflows** with command, description, tags
3. **Parameterized commands** with `{{name}}` or `<name=default>` syntax and default values
4. **Fuzzy search picker** (the quick-invoke mode) - This IS the daily driver
5. **Direct execution** - Select and run
6. **Shell integration** - Keybinding to invoke picker (Ctrl-R replacement)
7. **Edit workflows** - Open in `$EDITOR` or inline

### Ship for Differentiation (Phase 2)
8. **Inline argument filling** with full command visible - Key UX differentiator
9. **Full management TUI** (Bubble Tea) - Create, edit, delete, organize
10. **Tags + folder organization** with tag filtering

### Ship for Competitive Edge (Phase 3)
11. **AI-powered workflow creation** via Copilot SDK
12. **Register previous command** (shell function)
13. **Enum/select parameters** and **dynamic suggestions**
14. **Import from Pet/Warp YAML** formats

### Defer to Post-MVP
- **Community cheatsheet repos**: Network effect requires critical mass. Build after core is solid.
- **Auto-sync**: Users can `git push` manually. Auto-sync is polish, not core.
- **Tmux integration**: Nice-to-have, not critical for launch.
- **Output storage**: Documentation feature, low priority.
- **Password masking / path autocomplete**: Advanced parameter types. Add when basic params are proven.
- **Custom themes**: Cosmetic. Ship with one good default theme.

---

## Sources

### Official GitHub READMEs (HIGH confidence)
- Pet: https://github.com/knqyf263/pet (5.1k stars, v1.0.1, Dec 2024)
- Navi: https://github.com/denisidoro/navi (16.7k stars, v2.24.0, Jan 2025)
- Hoard: https://github.com/Hyde46/hoard (643 stars, v1.4.2, Aug 2023)
- SnipKit: https://github.com/lemoony/snipkit (115 stars, v1.8.1, Jan 2026)
- nap: https://github.com/maaslalani/nap (Charmbracelet ecosystem)

### Warp Workflows (MEDIUM confidence - from search results, official docs URL 404'd)
- YAML format with `{{arg}}` syntax, `name`/`description`/`tags`/`arguments` structure
- Enum args with static values or shell-command-generated lists
- Warp Drive for team sharing and cloud sync

### Ecosystem Surveys (MEDIUM confidence - multiple sources agreeing)
- Google Search: "terminal command snippet manager tools 2025 2026"
- Google Search: "command line snippet manager UX problems pain points 2025"
- Google Search: "terminal workflow automation tool features table stakes 2025"

### Additional tools surveyed (LOW confidence - from search results only)
- marker: command palette for terminal (fuzzy matcher)
- The Way: Rust snippet manager
- ScriptHaus: script organizer
- intelli-shell: SQLite-backed bookmarks (stalled development)
- boom: text snippet manager
