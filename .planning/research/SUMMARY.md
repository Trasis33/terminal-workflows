# Project Research Summary

**Project:** wf (Terminal Workflow Manager)
**Domain:** Go TUI terminal command snippet/workflow manager
**Researched:** 2026-02-19
**Confidence:** HIGH

## Executive Summary

`wf` is a terminal workflow manager in a well-established domain (command snippet managers) with clear competitive benchmarks: Navi (16.7k stars), Pet (5.1k), Hoard, SnipKit, and nap. The recommended approach is a Go single binary using the Charmbracelet ecosystem (Bubble Tea v1, Lip Gloss, Bubbles) with Cobra for CLI routing and YAML file storage via `goccy/go-yaml`. The stack is battle-tested and HIGH confidence across the board, with the sole exception of the Copilot CLI SDK (technical preview, MEDIUM confidence). The core differentiator — a dual-mode hybrid interface (inline fuzzy picker + full alt-screen management TUI) — is architecturally sound but requires the critical decision to run two separate `tea.Program` instances rather than mode-switching within one program.

The architecture follows a "Dual-Entry Elm Architecture with Shared Core" pattern: both the quick picker and the full TUI share a common domain layer (store, search, template engine, shell bridge) but run as independent Bubble Tea programs with different terminal requirements. This clean separation enables sub-100ms startup for the picker (the critical performance path) while allowing the full TUI to be richer and heavier. The YAML file-per-workflow storage model treats the filesystem as the database — simple, git-friendly, and sufficient for the expected scale (<2000 workflows).

The top risks are concentrated in Phase 1: (1) paste-to-prompt must use shell function wrappers (TIOCSTI is dead on modern Linux), (2) YAML's "Norway Problem" silently corrupts stored commands if not handled with double-quoted style marshalling, and (3) terminal state corruption on panic requires defensive `defer term.Restore()` from day one. All three are well-documented with proven solutions. The AI integration (Copilot SDK) is the largest uncertainty, but by design it's deferred to Phase 3+ and behind an interface abstraction — it cannot block the core product.

## Key Findings

### Recommended Stack

The entire stack is Go-native with zero runtime dependencies, targeting a single-binary distribution. The Charmbracelet ecosystem provides the TUI layer, and all choices are at stable v1 releases.

**Core technologies:**
- **Go 1.23+**: Language — fast compilation, single binary output, excellent stdlib (HTTP/JSON/embed/os)
- **Bubble Tea v1.3.10**: TUI framework — Elm Architecture for terminals, battle-tested, massive ecosystem
- **Lip Gloss v1.1.0 + Bubbles v1.0.0**: TUI styling & components — composable styles, pre-built widgets
- **Cobra v1.10.2**: CLI framework — auto shell completions, subcommand routing (used by kubectl, docker, gh)
- **goccy/go-yaml v0.17+**: YAML parsing — modern implementation with better error messages; `gopkg.in/yaml.v3` is unmaintained since April 2025
- **sahilm/fuzzy**: Fuzzy matching — zero deps, optimized for filename/symbol matching, used internally by Bubbles
- **adrg/xdg**: XDG directories — full spec compliance, resolves `~/.config/` correctly on macOS (not `~/Library/Application Support`)
- **GoReleaser v2**: Build & release — cross-compilation, Homebrew formula, .deb/.rpm packages

**Version note:** Bubble Tea v2 is at RC2 (not stable). Start on v1, design thin models for easy migration. The `goccy/go-yaml` choice over `go.yaml.in/yaml/v3` is forward-looking — either works.

### Expected Features

**Must have (table stakes):**
- YAML-based workflow storage in `~/.config/wf/`
- Save/create commands with description, tags
- Parameterized commands with `{{name}}` syntax and default values
- Fuzzy search picker (the daily driver invocation)
- Direct execution / paste-to-prompt
- Shell integration via keybinding (Ctrl-R replacement)
- Edit workflows (inline or `$EDITOR`)
- Clipboard copy fallback
- Cross-platform (macOS + Linux)

**Should have (differentiators):**
- Hybrid interface: fast picker + full management TUI — no existing tool does both well
- Inline argument filling with full command visible — rare and superior UX
- Enum/select parameters and dynamic suggestions from shell commands
- AI-powered workflow creation via Copilot SDK — unique positioning
- Import from Pet/Warp YAML formats — adoption accelerator
- Register previous command from shell history
- Folder + tag organization with tag-scoped search

**Defer (v2+):**
- Community cheatsheet repos (needs critical mass)
- Auto-sync (users can `git push`)
- Tmux integration, custom themes, output storage
- Password masking, path autocomplete (advanced param types)
- Plugin system

### Architecture Approach

The architecture uses a dual-entry point model with shared core domain packages. Two separate `tea.Program` instances serve different needs: the picker runs inline (no alt-screen) for speed, the manager runs fullscreen with `tea.WithAltScreen()`. Core domain packages (store, search, template, shell) have zero Bubble Tea dependency, enabling independent testing and reuse by non-TUI commands.

**Major components:**
1. **CLI Router** (Cobra) — dispatches to picker or manager mode based on subcommand
2. **Picker Model** — inline fuzzy search, workflow selection, argument fill, stdout output for shell capture
3. **Manager Model** — alt-screen CRUD, tag management, AI creation UI, tree-of-models for screens
4. **Store** — YAML file CRUD, one-file-per-workflow, filesystem-as-database
5. **Search** — fuzzy matching via `sahilm/fuzzy` over name/description/tags/command
6. **Template Engine** — custom `{{param}}` and `{{param|opt1|opt2}}` parser (~50 lines, NOT `text/template`)
7. **Shell Bridge** — generates per-shell plugin scripts, exposed via `wf init zsh|bash|fish`

### Critical Pitfalls

1. **TIOCSTI is dead** — paste-to-prompt via ioctl fails on Linux 6.2+. Use shell function wrappers (`print -z` for zsh, `READLINE_LINE` for bash, `commandline -r` for fish) via `eval "$(wf init zsh)"` pattern. Phase 1 blocker.
2. **YAML Norway Problem** — bare `no`, `yes`, `off`, `on` silently become booleans. Use typed `string` struct fields and `DoubleQuotedStyle` marshalling. Add round-trip tests. Phase 1 blocker.
3. **Terminal state corruption** — Bubble Tea raw mode + panic = unusable terminal. `defer term.Restore()` and panic recovery in `main()` from day one.
4. **State explosion in multi-view TUI** — single-model pattern collapses fast. Use tree-of-models with per-screen `Update()`/`View()` from initial scaffold.
5. **Shell wrapper varies per shell** — zsh/bash/fish have fundamentally different prompt-buffer APIs. Ship per-shell scripts via `go:embed`, test all three.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Foundation & Core Data Layer
**Rationale:** Everything depends on the data model, YAML storage, and config resolution. This phase has zero TUI dependency and validates the storage layer independently — critical because YAML pitfalls (Norway Problem, quote mangling, unknown fields) must be solved first.
**Delivers:** Working YAML workflow CRUD, template parser, config/XDG resolution, comprehensive round-trip tests.
**Addresses:** YAML storage, parameterized commands, config paths, project scaffolding.
**Avoids:** Norway Problem (#3), unknown fields (#16), nested quote mangling (#10), macOS XDG surprise (#7).

### Phase 2: Quick Picker TUI + Shell Integration
**Rationale:** The quick picker is the core value proposition and the simplest TUI to build (inline, single-purpose). Building it second validates the architecture before the more complex manager TUI. Shell integration wraps the picker and must work immediately.
**Delivers:** Working `wf` command with fuzzy search, workflow selection, argument filling, paste-to-prompt via shell wrapper. Daily-driver usable.
**Addresses:** Fuzzy search, direct execution, shell integration (Ctrl-R replacement), inline argument filling.
**Avoids:** TIOCSTI death (#1), shell wrapper variations (#4), terminal state corruption (#5), alt-screen confusion (#8), blocking Update/View (#9).

### Phase 3: Full Management TUI
**Rationale:** With the picker working and architecture validated, build the full-screen management interface. This is a separate `tea.Program` with tree-of-models. Depends on Phase 1 (store) and reuses Phase 2's shared components.
**Delivers:** `wf manage` command with list/create/edit/delete screens, tag management, folder browsing.
**Addresses:** Full TUI management, tags + folder organization, edit workflows, multiline support.
**Avoids:** State explosion (#2), over-nesting models, value vs pointer receivers (#13), terminal width assumptions (#15).

### Phase 4: Advanced Parameters & Import
**Rationale:** With basic parameterized commands working, add advanced parameter types (enum/select, dynamic suggestions) and migration tools. These build on the template engine from Phase 1.
**Delivers:** Enum parameters, dynamic suggestions from shell commands, Pet/Warp YAML import, register-previous-command shell function.
**Addresses:** Enum/select parameters, dynamic suggestions, import from other tools, register previous command.
**Avoids:** Command injection (#6), fuzzy search perf at scale (#11).

### Phase 5: AI Integration & Polish
**Rationale:** AI features are deferred until the Copilot CLI SDK has more stability time (it entered technical preview Jan 2026). Designed behind a `WorkflowGenerator` interface so it's swappable. This phase also includes polish: themes, validation commands, shell compatibility hardening.
**Delivers:** AI-powered workflow creation from natural language, `wf validate` command, improved error messages, multi-terminal testing.
**Addresses:** AI workflow generation, AI metadata auto-fill, custom themes, validation tooling.
**Avoids:** Copilot SDK instability (#12), coupling to a single AI provider.

### Phase 6: Distribution & Community
**Rationale:** Once the product is feature-complete and tested, set up GoReleaser for cross-platform distribution and consider community workflow sharing.
**Delivers:** Homebrew formula, .deb/.rpm packages, `wf completion` command, community cheatsheet repo support.
**Addresses:** Cross-platform distribution, shell completions, shareable workflow libraries.

### Phase Ordering Rationale

- **Phase 1 before everything** because store/template/config are dependencies for all other phases. YAML pitfalls caught here prevent cascading data corruption.
- **Phase 2 before Phase 3** because the picker is simpler (validates architecture) and is the core value prop. A tool with only a great picker is useful; a tool with only a management TUI is not.
- **Phase 3 before Phase 4** because advanced parameters need the TUI for enum selection UI. The management TUI also provides the creation flow that AI features plug into.
- **Phase 5 last-but-one** because the Copilot SDK needs calendar time to stabilize, and AI features require all prior features to be useful (they generate workflows that use the parameter system, storage, etc.).
- **Phase 6 last** because distribution is polish — the tool is usable via `go install` throughout development.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (Shell Integration):** Bash's readline API for paste-to-prompt has edge cases. Test with Oh My Zsh, Prezto, Starship. Bracketed paste mode interference needs empirical validation.
- **Phase 4 (Dynamic Parameters):** Shell command output for dynamic suggestions (Navi's `$ var: command` pattern) needs design research on syntax and security implications.
- **Phase 5 (AI Integration):** Copilot CLI SDK API may have changed by the time this phase starts. Re-research SDK state before planning. Also research alternative providers (Ollama for local, direct OpenAI API).

Phases with standard patterns (skip deep research):
- **Phase 1 (Foundation):** YAML parsing, XDG config, Go project structure — all well-documented with established patterns.
- **Phase 3 (Full TUI):** Bubble Tea tree-of-models, screen routing, Lip Gloss styling — extensively documented in official examples.
- **Phase 6 (Distribution):** GoReleaser is standard tooling with excellent docs.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All core technologies verified against official GitHub releases. Only Copilot SDK is MEDIUM (technical preview). |
| Features | HIGH | Verified against 7 competitor READMEs (Pet, Navi, Hoard, SnipKit, nap, Warp, tldr). Feature landscape is well-mapped. |
| Architecture | HIGH | Dual tea.Program pattern verified from Bubble Tea docs and examples. Shell integration pattern verified from fzf/Atuin implementations. Sub-100ms timing is MEDIUM (needs benchmarking). |
| Pitfalls | HIGH | 17 pitfalls identified from official docs, CVE records, kernel changelogs, and library documentation. Top 5 are well-documented with proven solutions. |

**Overall confidence:** HIGH

### Gaps to Address

- **Sub-100ms startup validation:** Timing estimates are theoretical. Benchmark with real YAML parsing of 100+ workflow files to confirm the lazy-loading strategy works. Address in Phase 2 planning.
- **Bubble Tea v2 migration timeline:** v2 is at RC2 (Nov 2025). If it stabilizes during development, evaluate migration cost. Current design (thin models, business logic in core) minimizes migration effort.
- **Copilot CLI SDK stability:** Technical preview as of Jan 2026. Re-assess before Phase 5 planning. Have fallback plan (direct API calls or alternative providers).
- **Bash paste-to-prompt reliability:** The `READLINE_LINE` approach has edge cases outside `bind -x` context. Needs empirical testing across bash versions (4.x vs 5.x) and common frameworks.
- **goccy/go-yaml exact version pinning:** Version listed as `v0.17+` — verify exact latest stable version at project initialization.

## Sources

### Primary (HIGH confidence)
- Charmbracelet Bubble Tea v1.3.10 — official GitHub releases, README, examples
- Charmbracelet Lip Gloss v1.1.0, Bubbles v1.0.0 — official releases
- Cobra v1.10.2 — official GitHub releases
- goccy/go-yaml — official repository (modern YAML implementation)
- Pet, Navi, Hoard, SnipKit, nap — official GitHub READMEs (feature landscape)
- Linux kernel 6.2 TIOCSTI deprecation — kernel changelog, CVE-2023-28339
- go-yaml.v3 YAML boolean handling — YAML spec, pkg.go.dev docs
- fzf, Atuin, zoxide — shell integration patterns (paste-to-prompt)
- XDG Base Directory spec — freedesktop.org

### Secondary (MEDIUM confidence)
- Copilot CLI SDK (github.com/github/copilot-cli-sdk-go) — technical preview, Jan 2026
- Sub-100ms startup timing estimates — Go compilation benchmarks, general performance characteristics
- Warp Workflows YAML format — from search results (official docs URL 404'd)
- sahilm/fuzzy performance at scale — needs benchmarking validation
- Go project layout conventions — community consensus, no official spec

### Tertiary (LOW confidence)
- Copilot SDK stability trajectory — too new to assess
- Fish shell freeze on specific paste lengths — reported but not widely confirmed
- Bubble Tea v2 stabilization timeline — RC2 as of Nov 2025, no announced stable date

---
*Research completed: 2026-02-19*
*Ready for roadmap: yes*
