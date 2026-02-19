# Technology Stack

**Project:** wf (terminal workflow manager)
**Researched:** 2026-02-19
**Overall Confidence:** HIGH (core stack), MEDIUM (Copilot SDK)

---

## Recommended Stack

### Core Framework

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Go | 1.23+ | Language | Fast compilation, single binary output, excellent concurrency, stdlib covers HTTP/JSON/embed/os. No runtime dependencies. | HIGH |
| Bubble Tea v1 | v1.3.10 | TUI framework | The Elm Architecture for terminals. Battle-tested, massive ecosystem (Bubbles, Lip Gloss), sub-100ms startup achievable. v2 is RC2 but not stable yet — start on v1, migrate later. | HIGH |
| Lip Gloss v1 | v1.1.0 | TUI styling | Terminal CSS. Composable styles, adaptive color profiles, pairs with Bubble Tea. v2 (beta) not ready. | HIGH |
| Bubbles v1 | v1.0.0 | TUI components | Pre-built list, textinput, viewport, spinner, progress components. Saves months of custom widget work. | HIGH |
| Cobra | v1.10.2 | CLI framework | Industry standard (kubectl, docker, gh use it). Auto-generates shell completions for bash/zsh/fish/powershell. Subcommand routing, flag parsing, help generation — all free. | HIGH |

### Data Storage

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| goccy/go-yaml | latest (v0.17+) | YAML parsing | Modern from-scratch YAML implementation. Better error messages with line/column info, faster than gopkg.in/yaml.v3, strict mode support. The old `gopkg.in/yaml.v3` is **unmaintained since April 2025** — do not use it. | HIGH |

**Alternative considered:** `go.yaml.in/yaml/v3` (official fork of the unmaintained gopkg.in/yaml.v3). It's a safe drop-in replacement, but `goccy/go-yaml` has better DX with detailed error positions and validation. Either works — goccy is the forward-looking choice.

### Search & Filtering

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| sahilm/fuzzy | latest | Fuzzy matching | Zero dependencies, optimized for filename/symbol matching. Already used internally by Bubbles' list component. Proven in the Charm ecosystem. | HIGH |

### Shell Integration

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Cobra (completions) | (via Cobra) | Shell completions | Cobra auto-generates completion scripts for bash/zsh/fish/powershell. Zero extra code. | HIGH |
| Go embed | stdlib | Shell init scripts | Embed shell integration scripts (zsh/bash/fish) into the binary. Output via `wf init zsh`. Single binary, no external files needed. | HIGH |

### Configuration & XDG

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| adrg/xdg | latest | XDG directories | Full XDG Base Directory spec compliance. Resolves config/data/cache/state dirs correctly across Linux/macOS. Lightweight. | HIGH |
| Go embed | stdlib | Default configs | Embed default YAML templates, shell scripts, and example workflows into the binary. | HIGH |

### AI Integration (Deferred)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Copilot CLI SDK (Go) | technical preview | AI-powered workflow generation | `github.com/github/copilot-cli-sdk-go` released Jan 2026. JSON-RPC architecture, multi-turn conversations, custom tool definitions. **BUT**: requires bundling Copilot CLI binary — conflicts with single-binary constraint. | MEDIUM |

> **⚠️ Copilot SDK Strategy:** The SDK is in technical preview and requires an external Copilot CLI binary. This conflicts with the "single binary, no runtime dependencies" constraint. **Recommendation:** Design an `AIProvider` interface now, implement Copilot integration as an optional plugin that detects if `github-copilot-cli` is installed on the system. Do NOT bundle it. Treat AI features as progressive enhancement — they work if Copilot CLI is available, gracefully degrade if not.

### Build & Distribution

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| GoReleaser | v2 | Build & release | Cross-compilation, checksums, Homebrew formula generation, changelog generation. Standard for Go CLI tools. | HIGH |
| goreleaser/nfpm | (via GoReleaser) | Package formats | .deb, .rpm, .apk packages from GoReleaser config. | HIGH |

### Testing

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| testing (stdlib) | stdlib | Unit tests | Go's built-in testing. No framework needed for most tests. | HIGH |
| charmbracelet/x/exp/teatest | latest | TUI testing | Official Bubble Tea testing library. Programmatic message sending, output assertion, golden file testing for TUI snapshots. | HIGH |
| stretchr/testify | v1.10+ | Test assertions | `assert` and `require` packages for readable test assertions. Industry standard. | HIGH |

### Observability (Development)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| charmbracelet/log | latest | Structured logging | Colorized, leveled logging that works beautifully in terminals. Same ecosystem as Bubble Tea. For dev/debug — production binary should be silent. | MEDIUM |

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| TUI Framework | Bubble Tea v1 | Bubble Tea v2 (RC2) | v2 has breaking API changes, RC stage means possible further changes. v1 is battle-tested. Design for v2 migration. |
| TUI Framework | Bubble Tea | tview | tview is widget-based (not Elm Architecture). Less composable, harder to test, harder to build custom UIs. |
| TUI Framework | Bubble Tea | tcell | Too low-level. Bubble Tea builds on tcell and adds the architecture. |
| CLI Framework | Cobra | urfave/cli v3 | Less ecosystem integration, no auto shell completions for all shells, smaller community. |
| CLI Framework | Cobra | kong | Struct-based config is clever but less flexible for dynamic subcommands. |
| YAML | goccy/go-yaml | gopkg.in/yaml.v3 | **Unmaintained since April 2025.** Do not use for new projects. |
| YAML | goccy/go-yaml | go.yaml.in/yaml/v3 | Valid alternative (official successor). goccy has better error positions and validation. |
| YAML | goccy/go-yaml | JSON/TOML | YAML is specified in project requirements. YAML is also better for human-authored workflow files (comments, multiline strings). |
| Fuzzy Search | sahilm/fuzzy | junegunn/fzf lib | fzf's Go library API is experimental. sahilm/fuzzy is proven in Charm ecosystem. |
| Config | adrg/xdg | os.UserConfigDir() | stdlib only gives config dir. adrg/xdg gives full XDG spec (data, cache, state, runtime). |
| Logging | charmbracelet/log | zerolog/zap | Over-engineered for a CLI tool. charmbracelet/log is simpler and visually matches the TUI. |

---

## Version Pinning Strategy

**Use exact versions in go.mod.** The Charmbracelet ecosystem moves fast — pin to tested versions:

```go
// go.mod
module github.com/you/wf

go 1.23

require (
    github.com/charmbracelet/bubbletea v1.3.10
    github.com/charmbracelet/lipgloss  v1.1.0
    github.com/charmbracelet/bubbles   v1.0.0
    github.com/spf13/cobra             v1.10.2
    github.com/goccy/go-yaml           v0.17.1
    github.com/sahilm/fuzzy            v0.1.1
    github.com/adrg/xdg                v0.5.3
    github.com/stretchr/testify        v1.10.0
)
```

> **Note on versions:** The `goccy/go-yaml`, `sahilm/fuzzy`, and `adrg/xdg` versions above are approximate — verify exact latest at build time. The Charmbracelet and Cobra versions are verified against GitHub releases (HIGH confidence).

---

## Installation

```bash
# Initialize Go module
go mod init github.com/you/wf

# Core dependencies
go get github.com/charmbracelet/bubbletea@v1.3.10
go get github.com/charmbracelet/lipgloss@v1.1.0
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/spf13/cobra@v1.10.2

# Data
go get github.com/goccy/go-yaml

# Search
go get github.com/sahilm/fuzzy

# Config
go get github.com/adrg/xdg

# Testing (dev)
go get github.com/stretchr/testify
go get github.com/charmbracelet/x/exp/teatest

# Logging (dev)
go get github.com/charmbracelet/log

# Build tooling (install separately)
# brew install goreleaser
```

---

## Bubble Tea v1 → v2 Migration Path

v2 is at RC2 (Nov 2025) and will likely stabilize in Q1-Q2 2026. Key breaking changes to design around:

| v1 | v2 | Migration Strategy |
|----|----|--------------------|
| `View() string` | `View() View` (struct) | Wrap view logic in helper functions that return strings. Swap return types later. |
| String concatenation for views | Declarative `View` struct with properties | Keep view logic isolated in dedicated methods per component. |
| `tea.Println` | Removed (use `View` properties) | Don't rely on `tea.Println` for output. |
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` | Import path change — mechanical via `sed` or IDE refactor. |

**Design principle:** Keep Bubble Tea models thin. Business logic in pure Go functions, TUI in models. When v2 stabilizes, only the model layer changes.

---

## Performance Budget

The sub-100ms startup constraint drives several stack choices:

| Concern | Target | How |
|---------|--------|-----|
| Binary startup | <50ms | Go compiles to native code, no runtime init. Bubble Tea initializes fast. |
| YAML parsing | <20ms | goccy/go-yaml is fast. Lazy-load workflow files — only parse what's needed for the picker. |
| Fuzzy search | <10ms | sahilm/fuzzy is optimized for this. Operates on in-memory string slices. |
| TUI first paint | <30ms | Bubble Tea's first render is near-instant. Pre-compute styles at init. |
| Shell completion | <50ms | Cobra completions are generated once, cached by shell. |

**Key optimization:** Don't parse all YAML files at startup. Read directory listing (fast), parse only filenames/metadata for the picker. Full parse happens on selection.

---

## Single Binary Strategy

Everything embedded, nothing external:

```
wf (single binary)
├── Embedded: default config YAML (via go:embed)
├── Embedded: shell init scripts for zsh/bash/fish (via go:embed)
├── Embedded: example workflow templates (via go:embed)
├── Runtime: reads ~/.config/wf/*.yaml (via adrg/xdg)
└── Optional: talks to Copilot CLI if installed (graceful degradation)
```

No database. No runtime dependencies. No network calls (except optional AI features).

---

## Sources

| Source | Type | Confidence |
|--------|------|------------|
| GitHub Releases: charmbracelet/bubbletea | Official releases page | HIGH |
| GitHub Releases: charmbracelet/lipgloss | Official releases page | HIGH |
| GitHub Releases: charmbracelet/bubbles | Official releases page | HIGH |
| GitHub Releases: spf13/cobra | Official releases page | HIGH |
| GitHub: goccy/go-yaml | Official repository | HIGH |
| GitHub: yaml/go-yaml (go.yaml.in) | Official repository | HIGH |
| GitHub: sahilm/fuzzy | Official repository | HIGH |
| GitHub Blog: Copilot CLI SDK announcement (Jan 2026) | Official blog | MEDIUM |
| github.com/github/copilot-cli-sdk-go | Official repository | MEDIUM |
| GoReleaser documentation | Official docs | HIGH |
| WebSearch: "Go TUI framework 2025/2026" | Multiple sources | MEDIUM |
| WebSearch: "gopkg.in/yaml.v3 unmaintained" | Community reports | HIGH (verified via GitHub) |
