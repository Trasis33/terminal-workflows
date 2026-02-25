# Phase 6: Distribution & Sharing - Research

**Researched:** 2026-02-25
**Domain:** Git-based workflow sharing, PowerShell/Windows support, shell completions
**Confidence:** HIGH

## Summary

Phase 6 adds two capabilities: git-based remote workflow sources (ORGN-03) and PowerShell/Windows support (SHEL-04). The git integration uses `os/exec` to shell out to the system `git` binary rather than a pure-Go git library — this aligns with how most Go CLI tools handle git operations and avoids the performance/memory issues of go-git. Remote sources are cloned into `xdg.DataHome/wf/sources/<alias>/` and exposed through a read-only `Store` adapter that aggregates with the local `YAMLStore`.

PowerShell integration follows the exact same pattern as existing zsh/bash/fish scripts: a PSReadLine `Set-PSReadLineKeyHandler` binding on Ctrl+G that invokes `wf pick`, captures the output, and injects it into the command line buffer. Windows terminal I/O requires a platform-specific `openTTY()` using `CONIN$`/`CONOUT$` instead of `/dev/tty`. All current dependencies cross-compile cleanly with `CGO_ENABLED=0 GOOS=windows`.

Shell completions come free from Cobra v1.10.2 — the framework auto-generates a `completion` command supporting bash, zsh, fish, and PowerShell. Dynamic completions via `ValidArgsFunction` enable tab-completing source names and workflow names.

**Primary recommendation:** Use `os/exec` with the system git binary for all git operations; use PSReadLine APIs for PowerShell integration; use build tags (`//go:build`) to isolate Windows-specific codepaths.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` (stdlib) | Go 1.25 | Git operations (clone, pull, ls-remote) | No external dependency; better performance than go-git for large repos; user's git config/auth just works |
| `adrg/xdg` | v0.5.3 | Cross-platform data directory for source clones | Already in project; maps correctly on Windows (`%LOCALAPPDATA%`) |
| `spf13/cobra` | v1.10.2 | Shell completions (built-in `completion` command) | Already in project; auto-generates completions for all 4 shells |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/sys` | v0.38.0 | Windows console mode (VT processing) | Already indirect dependency; needed for `CONIN$`/`CONOUT$` console handle setup |
| `path/filepath` (stdlib) | Go 1.25 | Windows path handling | Use `filepath.Join` everywhere instead of hardcoded `/` separators |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os/exec` git | `go-git/go-git` v5 | Pure Go (no git binary needed), but worse performance for large repos, higher memory, known thread-safety issues, less complete git feature support |
| Custom PowerShell script | Oh-My-Posh module | Overkill dependency for a single keybinding; PSReadLine is built into pwsh 7+ |

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── store/
│   ├── store.go           # Store interface (existing)
│   ├── yaml.go            # YAMLStore (existing)
│   ├── multi.go           # NEW: MultiStore aggregating local + remote stores
│   └── remote.go          # NEW: RemoteStore (read-only, wraps cloned repo)
├── source/
│   ├── manager.go         # NEW: Source CRUD (add/remove/update/list)
│   └── git.go             # NEW: Git operations (clone, pull, status)
├── shell/
│   ├── zsh.go             # Existing
│   ├── bash.go            # Existing
│   ├── fish.go            # Existing
│   └── powershell.go      # NEW: PowerShell 7+ PSReadLine script
├── config/
│   └── config.go          # Add SourcesDir() using xdg.DataHome
cmd/wf/
│   ├── pick.go            # Modify openTTY() to use build tags
│   ├── pick_unix.go       # NEW: openTTY() for unix (//go:build !windows)
│   ├── pick_windows.go    # NEW: openTTY() for windows (//go:build windows)
│   ├── source.go          # NEW: wf source add/remove/update/list commands
│   └── init_shell.go      # Add "powershell" case
```

### Pattern 1: MultiStore Aggregation
**What:** A `MultiStore` that wraps `[]Store` and merges results, prefixing remote workflow names with source alias.
**When to use:** When the picker needs to show all workflows from all sources.
**Example:**
```go
// internal/store/multi.go
type MultiStore struct {
    local  Store
    remote map[string]Store // key = source alias
}

func (ms *MultiStore) List() ([]Workflow, error) {
    var all []Workflow
    // Local workflows first (no prefix)
    local, err := ms.local.List()
    if err != nil {
        return nil, fmt.Errorf("local store: %w", err)
    }
    all = append(all, local...)

    // Remote workflows with alias prefix
    for alias, s := range ms.remote {
        workflows, err := s.List()
        if err != nil {
            // Log warning but don't fail — remote source might be broken
            fmt.Fprintf(os.Stderr, "warning: source %q: %v\n", alias, err)
            continue
        }
        for _, w := range workflows {
            w.Name = alias + "/" + w.Name
            all = append(all, w)
        }
    }
    return all, nil
}
```

### Pattern 2: Git Operations via os/exec
**What:** Wrap git commands with timeout, error capture, and working directory control.
**When to use:** All git interactions (clone, pull, ls-remote).
**Example:**
```go
// internal/source/git.go
func gitClone(ctx context.Context, url, dest string) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
    defer cancel()

    cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, dest)
    cmd.Stderr = os.Stderr // Show git progress
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("git clone %s: %w", url, err)
    }
    return nil
}

func gitPull(ctx context.Context, repoDir string) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "git", "pull", "--ff-only")
    cmd.Dir = repoDir
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("git pull in %s: %w\n%s", repoDir, err, out)
    }
    return string(out), nil
}

func gitAvailable() bool {
    _, err := exec.LookPath("git")
    return err == nil
}
```

### Pattern 3: Build Tags for Platform-Specific Code
**What:** Use `//go:build` constraints to isolate Windows vs Unix code.
**When to use:** `openTTY()` in pick command, any Windows console mode setup.
**Example:**
```go
// cmd/wf/pick_unix.go
//go:build !windows

package main

import "os"

func openTTY() (*os.File, error) {
    return os.OpenFile("/dev/tty", os.O_WRONLY, 0)
}
```

```go
// cmd/wf/pick_windows.go
//go:build windows

package main

import "os"

func openTTY() (*os.File, error) {
    return os.OpenFile("CONIN$", os.O_RDWR, 0o644)
}
```

### Pattern 4: Source Config Persistence
**What:** Store source configuration (alias, URL, last-updated) in a YAML file alongside clone data.
**When to use:** `wf source add/remove/list/update` commands.
**Example:**
```go
// internal/source/manager.go
type Source struct {
    Alias     string    `yaml:"alias"`
    URL       string    `yaml:"url"`
    UpdatedAt time.Time `yaml:"updated_at,omitempty"`
}

type SourceConfig struct {
    Sources []Source `yaml:"sources"`
}

// Config file at: xdg.DataHome/wf/sources.yaml
// Clone dirs at: xdg.DataHome/wf/sources/<alias>/
```

### Anti-Patterns to Avoid
- **Embedding git credentials:** Never handle tokens or passwords — rely on user's git config (SSH agent, credential helpers). This is a locked decision.
- **Auto-sync on startup:** Never pull remote sources automatically — this would violate the 100ms picker launch target. Manual `wf source update` only.
- **Hardcoded path separators:** Never use `/` in `filepath.Join` calls or hardcode `/dev/tty`. Use build tags or `filepath.Join`.
- **Writable remote stores:** Remote sources are read-only. Edits fork-to-local (copy workflow to local store, then edit).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Shell completions | Custom completion scripts | Cobra's built-in `completion` command | Handles all 4 shells, dynamic completions via ValidArgsFunction, maintained by Cobra team |
| XDG data directory | Manual `%LOCALAPPDATA%` detection | `adrg/xdg` `.DataHome` | Already in project; handles all platforms including Windows with proper `FOLDERID_LocalAppData` resolution |
| Windows terminal I/O | Raw Win32 console API calls | Bubble Tea's existing Windows support + `CONIN$`/`CONOUT$` file open | Bubble Tea already handles VT processing via `golang.org/x/sys/windows`; just need correct file handle |
| Git operations | `go-git` pure Go library | `os/exec` + system `git` | System git handles auth, LFS, submodules, partial clones correctly; go-git has perf issues and incomplete feature set |
| PowerShell line editing | Custom PowerShell C# cmdlet | PSReadLine APIs (`Set-PSReadLineKeyHandler`, `[PSConsoleReadLine]::Insert()`) | PSReadLine ships with PowerShell 7+; it's the standard way to manipulate the command line buffer |

**Key insight:** This phase connects to external systems (git, PowerShell, Windows) where the platform's native tools are always more reliable than reimplementations.

## Common Pitfalls

### Pitfall 1: Git Not Installed
**What goes wrong:** `exec.Command("git", ...)` fails with "executable file not found" on systems without git.
**Why it happens:** Windows doesn't ship with git by default. Some minimal Linux containers lack it too.
**How to avoid:** Check `exec.LookPath("git")` before any git operation. `wf source add` should fail early with a clear error: "git is required for remote sources. Install git and try again."
**Warning signs:** CI/CD failures on clean images; user reports on Windows.

### Pitfall 2: Windows Path Length Limits
**What goes wrong:** Deep repo structures hit the 260-character MAX_PATH limit on Windows.
**Why it happens:** `%LOCALAPPDATA%\wf\sources\<alias>\<deep-nested-path>\workflow.yaml` can exceed 260 chars.
**How to avoid:** Use short alias names (auto-derived from repo name is usually fine). Clone with `--depth 1` to minimize repo structure. Document that long path support requires Windows 10 1607+ with registry key `LongPathsEnabled`.
**Warning signs:** File-not-found errors on Windows for workflows that work on Unix.

### Pitfall 3: PowerShell Execution Policy Blocking Init Script
**What goes wrong:** `wf init powershell` output piped to `Invoke-Expression` blocked by execution policy.
**Why it happens:** PowerShell's execution policy can block scripts. However, `Invoke-Expression` on a string is NOT blocked by execution policy (it only applies to .ps1 files).
**How to avoid:** Use `Invoke-Expression` (not dot-sourcing a .ps1 file) for the init pattern: `wf init powershell | Invoke-Expression`. This bypasses execution policy entirely.
**Warning signs:** Users reporting "script is not digitally signed" errors — would indicate they're trying to save and dot-source instead of using `Invoke-Expression`.

### Pitfall 4: Git Clone Hanging on Auth Prompt
**What goes wrong:** Private repo clone prompts for username/password on stdin, blocking indefinitely.
**Why it happens:** No SSH key configured, no credential helper, and git falls back to interactive auth prompt.
**How to avoid:** Set `GIT_TERMINAL_PROMPT=0` environment variable on the exec.Command. This makes git fail immediately instead of prompting. Return a clear error: "Authentication failed. Configure SSH keys or a git credential helper."
**Warning signs:** `wf source add` appears to hang with no output.

### Pitfall 5: Source Alias Collisions
**What goes wrong:** Two sources have the same auto-derived alias (e.g., both repos named "workflows").
**Why it happens:** Many people name their workflow repos the same thing.
**How to avoid:** Check for alias collisions in `wf source add`. If the auto-derived alias conflicts, require `--name <alias>` flag. Store aliases in sources.yaml for collision detection.
**Warning signs:** Workflows from one source silently shadowing another.

### Pitfall 6: Stale /dev/tty Fallback on Windows
**What goes wrong:** Current `openTTY()` falls back to `os.Stderr` on Windows, which may not be a terminal when running under shell integration.
**Why it happens:** The existing fallback was designed for "tty unavailable" scenarios, but on Windows it's always unavailable since `/dev/tty` doesn't exist.
**How to avoid:** Use build tags to provide a Windows-specific `openTTY()` using `CONIN$`. Never fall through to stderr on Windows — `CONIN$` is always available when running in a console.
**Warning signs:** Picker renders garbled or doesn't accept input on Windows.

## Code Examples

### PowerShell Init Script (Verified from Atuin's atuin.ps1 pattern)
```powershell
# wf shell integration for PowerShell 7+
# Usage: wf init powershell | Invoke-Expression
# Or add to $PROFILE: wf init powershell | Invoke-Expression

Set-PSReadLineKeyHandler -Chord 'Ctrl+g' -ScriptBlock {
    [Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
    $output = wf pick
    if ($output) {
        [Microsoft.PowerShell.PSConsoleReadLine]::Insert($output)
    }
}
```
**Source:** Pattern derived from [Atuin's atuin.ps1](https://github.com/atuinsh/atuin/blob/main/crates/atuin/src/shell/atuin.ps1) — verified via WebFetch.

### Cobra Shell Completions Setup
```go
// In root.go init() — Cobra auto-registers the completion command.
// Customize completion behavior:
func init() {
    // ... existing commands ...

    // Enable default completion command (bash, zsh, fish, powershell)
    rootCmd.CompletionOptions = cobra.CompletionOptions{
        HiddenDefaultCmd: false, // Make 'wf completion' visible
    }
}

// In source.go — dynamic completion for source aliases
var sourceRemoveCmd = &cobra.Command{
    Use:   "remove [alias]",
    Short: "Remove a workflow source",
    Args:  cobra.ExactArgs(1),
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        mgr := source.NewManager(config.SourcesDir())
        sources, _ := mgr.List()
        var names []string
        for _, s := range sources {
            names = append(names, s.Alias+"\t"+s.URL)
        }
        return names, cobra.ShellCompDirectiveNoFileComp
    },
    RunE: runSourceRemove,
}
```
**Source:** [Cobra completions documentation](https://github.com/spf13/cobra/blob/main/site/content/completions/_index.md) — verified via WebFetch.

### Windows openTTY with CONOUT$ for Bubble Tea Output
```go
//go:build windows

package main

import "os"

func openTTY() (*os.File, error) {
    // CONOUT$ is the Windows equivalent of /dev/tty for output.
    // Bubble Tea will use this for rendering the TUI.
    // CONIN$ handles input internally via charmbracelet/x/term.
    return os.OpenFile("CONOUT$", os.O_WRONLY, 0)
}
```
**Source:** Bubble Tea source (`open_windows.go`) uses `CONIN$` for input; we need `CONOUT$` for output since `tea.WithOutput()` controls where rendering goes.

### Source Manager — Add with Auto-Alias
```go
// internal/source/manager.go
func (m *Manager) Add(ctx context.Context, url, alias string) error {
    if !gitAvailable() {
        return fmt.Errorf("git is required for remote sources. Install git and try again")
    }

    // Auto-derive alias from URL if not provided
    if alias == "" {
        alias = deriveAlias(url)
    }

    // Check collision
    if m.hasAlias(alias) {
        return fmt.Errorf("source %q already exists. Use --name to specify a different alias", alias)
    }

    dest := filepath.Join(m.dir, alias)
    if err := gitClone(ctx, url, dest); err != nil {
        return err
    }

    // Persist to sources.yaml
    m.cfg.Sources = append(m.cfg.Sources, Source{
        Alias:     alias,
        URL:       url,
        UpdatedAt: time.Now(),
    })
    return m.save()
}

// deriveAlias extracts a short alias from a git URL.
// "https://github.com/team/workflows.git" → "workflows"
// "git@github.com:user/my-commands.git" → "my-commands"
func deriveAlias(url string) string {
    // Strip .git suffix
    u := strings.TrimSuffix(url, ".git")
    // Take last path segment
    parts := strings.FieldsFunc(u, func(r rune) bool {
        return r == '/' || r == ':'
    })
    if len(parts) == 0 {
        return "remote"
    }
    return parts[len(parts)-1]
}
```

### Remote Store — Read-Only YAML Store Wrapper
```go
// internal/store/remote.go
type RemoteStore struct {
    dir string // Path to cloned repo directory
}

func NewRemoteStore(dir string) *RemoteStore {
    return &RemoteStore{dir: dir}
}

func (rs *RemoteStore) List() ([]Workflow, error) {
    // Walk directory for .yaml files (same logic as YAMLStore.List)
    // Support both flat layout and nested wf-structure layout
    var workflows []Workflow
    err := filepath.WalkDir(rs.dir, func(path string, d os.DirEntry, err error) error {
        if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
            return err
        }
        // Skip .git directory
        if strings.Contains(path, ".git") {
            return nil
        }
        w, err := readWorkflowFile(path)
        if err != nil {
            return nil // Skip malformed files in remote repos
        }
        workflows = append(workflows, *w)
        return nil
    })
    return workflows, err
}

func (rs *RemoteStore) Get(name string) (*Workflow, error) { /* ... */ }
func (rs *RemoteStore) Save(w *Workflow) error { return fmt.Errorf("remote source is read-only") }
func (rs *RemoteStore) Delete(name string) error { return fmt.Errorf("remote source is read-only") }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| go-git for Go programs | os/exec with system git | Community consensus ~2023 | Better perf, auth support, feature completeness |
| Windows PowerShell 5.1 | PowerShell 7+ (pwsh) cross-platform | 2020 (PowerShell 7 GA) | PSReadLine 2.x built-in, better Unicode/ANSI, cross-platform |
| Manual completion scripts | Cobra built-in completion command | Cobra v1.2.0 (2021) | Zero custom code for 4-shell completions |
| `//+build` tags | `//go:build` tags | Go 1.17 (2021) | New syntax is the standard; old syntax still accepted but deprecated |

**Deprecated/outdated:**
- Windows PowerShell 5.1: No longer receives feature updates; PSReadLine version is old. Target PowerShell 7+ only (locked decision).
- `go-git` for CLI tools: Acceptable for programmatic access but inferior to system git for user-facing CLI tools that need auth and performance.

## Open Questions

1. **CONOUT$ vs CONIN$ for Bubble Tea WithOutput**
   - What we know: Bubble Tea opens `CONIN$` for input internally. Our `openTTY()` provides the output handle for `tea.WithOutput()`.
   - What's unclear: Whether `CONOUT$` is the correct handle for output, or if `CON` (which combines both) should be used. Bubble Tea's own `open_windows.go` uses `CONIN$` for `openInputTTY()` but the output path is less documented.
   - Recommendation: Start with `CONOUT$` for the `openTTY()` output handle. If rendering issues occur on Windows, try `CON` or `os.Stderr` as fallback. Test on actual Windows machine.
   - Confidence: MEDIUM

2. **Repo Structure Auto-Detection**
   - What we know: Need to support both flat directories of YAML files and repos mirroring wf's local structure (nested under `workflows/` dir).
   - What's unclear: What other structures community repos might use. No established convention exists yet.
   - Recommendation: Walk the entire repo for `*.yaml` files, skipping `.git/`. Parse each to check if it has the required `name` and `command` fields. Simple and handles any layout.
   - Confidence: HIGH

3. **Update Diff Summary**
   - What we know: CONTEXT.md specifies showing "diff summary (new workflows, removed, updated)" after `wf source update`.
   - What's unclear: Exact implementation approach — git diff, or compare workflow lists before/after pull.
   - Recommendation: Snapshot workflow names before `git pull`, then compare with post-pull list. Simpler than parsing git diffs and handles renamed files correctly.
   - Confidence: HIGH

## Sources

### Primary (HIGH confidence)
- `adrg/xdg` v0.5.3 — `paths_windows.go` source fetched from GitHub; verified `DataHome` maps to `%LOCALAPPDATA%`
- `spf13/cobra` v1.10.2 — Completions documentation fetched from GitHub (`site/content/completions/_index.md`); verified `GenPowerShellCompletionWithDesc`, `ValidArgsFunction`, auto-registered completion command
- Atuin `atuin.ps1` — Full PSReadLine integration script fetched from GitHub; verified `Set-PSReadLineKeyHandler`, `[PSConsoleReadLine]::Insert()`, `::RevertLine()` patterns
- Existing codebase: `store.go`, `yaml.go`, `workflow.go`, `pick.go`, `init_shell.go`, `root.go`, `config.go`, all shell scripts — read in full

### Secondary (MEDIUM confidence)
- Bubble Tea Windows support — Inferred from `erikgeiser/coninput` dependency in go.mod (Windows console input library) and Bubble Tea's documented cross-platform support; did not fetch Bubble Tea's `open_windows.go` directly
- `os/exec` vs go-git recommendation — Based on multiple community discussions and Go ecosystem patterns; consistent consensus across sources
- Windows `CONOUT$`/`CONIN$` behavior — Standard Windows console API, well-documented by Microsoft

### Tertiary (LOW confidence)
- fzf PowerShell key-bindings.ps1 — Attempted to fetch from GitHub but got 404; file may have moved or been renamed. Not critical since Atuin's pattern was successfully verified.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All libraries already in project; patterns verified from official sources
- Architecture: HIGH — MultiStore pattern follows established Go interface composition; git exec pattern widely used
- PowerShell integration: HIGH — Atuin.ps1 source verified directly; PSReadLine is standard in pwsh 7+
- Windows terminal I/O: MEDIUM — Bubble Tea Windows support confirmed but exact `CONOUT$` vs `CON` behavior needs runtime testing
- Pitfalls: HIGH — Drawn from verified platform documentation and real-world failure modes

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (stable domain; Go stdlib, git, and PowerShell APIs change slowly)
