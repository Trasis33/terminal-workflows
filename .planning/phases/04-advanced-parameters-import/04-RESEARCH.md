# Phase 4: Advanced Parameters & Import - Research

**Researched:** 2026-02-23
**Domain:** Template parsing, TOML/YAML import, shell history parsing
**Confidence:** HIGH (verified via official repos, source code, official docs)

## Summary

This phase extends the wf parameter system with enum and dynamic parameter types, adds a `wf register` command to capture previous shell commands, and adds `wf import` to convert Pet TOML and Warp YAML collections into wf workflows.

The template parser (`internal/template/parser.go`) currently uses a single regex `\{\{([^}]+)\}\}` and splits on `:` for defaults. This regex already captures everything inside `{{...}}`, so extending it to support pipe-delimited enums (`{{env|dev|staging|*prod}}`) and bang-syntax dynamics (`{{branch!git branch --list}}`) requires only changes to `parseInner()` — the regex itself does not need modification. The `Param` struct needs new fields for `Type`, `Options`, and `DynamicCmd`.

For import, Pet uses a simple TOML format with 4 fields per snippet (`command`, `description`, `tag`, `output`). Warp uses YAML with 8 fields per workflow. Both map cleanly to wf's `Workflow` struct with minor translation (Pet's `<param=default>` → `{{param:default}}`, Warp's `{{arg}}` maps directly).

Shell history parsing is straightforward for bash (newline-separated) and fish (pseudo-YAML), but zsh requires handling the "metafied" binary encoding for non-ASCII characters.

**Primary recommendation:** Extend `parseInner()` with ordered checks (bang → pipe → colon → plain name). Use `pelletier/go-toml/v2` for Pet import. Implement `unmetafy()` for zsh history.

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `goccy/go-yaml` | v1.19.2 | YAML parsing | Already used for workflow storage |
| `bubbletea` | v1.3.10 | TUI framework | Already used for picker |
| `bubbles` | v1.0.0 | TUI components | Already used for textinput |
| `cobra` | v1.10.2 | CLI framework | Already used for commands |
| `huh` | v0.8.0 | Form/prompt library | Already available, use for interactive register prompts |

### New Dependencies
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pelletier/go-toml/v2` | v2.x | Parse Pet TOML snippet files | Pet import only |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `pelletier/go-toml/v2` | `BurntSushi/toml` | BurntSushi is simpler but less actively maintained; pelletier has better error messages, strict mode, and is the same lib Pet itself uses internally |
| Custom fish history parser | Standard YAML parser | Fish history is pseudo-YAML ("ad-hoc broken pseudo-YAML"), standard parsers may choke — custom line-by-line parser is safer |

**Installation:**
```bash
go get github.com/pelletier/go-toml/v2@latest
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── template/
│   ├── parser.go       # ExtractParams, parseInner (EXTEND)
│   ├── parser_test.go  # Tests for new param types
│   └── renderer.go     # Render (EXTEND for enum/dynamic)
├── store/
│   ├── workflow.go     # Workflow, Arg structs (EXTEND)
│   └── store.go        # Store interface (unchanged)
├── picker/
│   ├── paramfill.go    # Parameter fill UI (EXTEND for list select + dynamic)
│   └── paramfill_test.go
├── history/            # NEW package
│   ├── history.go      # HistoryReader interface + factory
│   ├── zsh.go          # Zsh history parser (with unmetafy)
│   ├── bash.go         # Bash history parser
│   ├── fish.go         # Fish history parser
│   └── detect.go       # Auto-detect current shell
├── importer/           # NEW package
│   ├── importer.go     # ImportResult, common types
│   ├── pet.go          # Pet TOML → Workflow converter
│   ├── warp.go         # Warp YAML → Workflow converter
│   └── paramconv.go    # Parameter syntax translation
└── register/           # NEW package (or inline in cmd)
    └── detect.go       # Auto-detect parameters in command strings
cmd/wf/
├── register.go         # NEW: wf register command
└── import.go           # NEW: wf import command
```

### Pattern 1: Parameter Type Discrimination in parseInner

**What:** Extend `parseInner()` to detect parameter type by scanning for `!` (dynamic), `|` (enum), or `:` (default), in that priority order.

**When to use:** Every time a `{{...}}` token is extracted.

**Example:**
```go
// parseInner determines parameter type from the inner content of {{...}}.
// Priority: bang (dynamic) → pipe (enum) → colon (default) → plain
func parseInner(s string) Param {
    // 1. Check for dynamic: name!command
    if idx := strings.IndexByte(s, '!'); idx > 0 {
        return Param{
            Name:       s[:idx],
            Type:       ParamDynamic,
            DynamicCmd: s[idx+1:],
        }
    }

    // 2. Check for enum: name|opt1|opt2|*default_opt
    if idx := strings.IndexByte(s, '|'); idx > 0 {
        parts := strings.Split(s, "|")
        name := parts[0]
        var options []string
        var defaultVal string
        for _, p := range parts[1:] {
            if strings.HasPrefix(p, "*") {
                cleaned := p[1:]
                options = append(options, cleaned)
                defaultVal = cleaned
            } else {
                options = append(options, p)
            }
        }
        return Param{
            Name:    name,
            Type:    ParamEnum,
            Options: options,
            Default: defaultVal,
        }
    }

    // 3. Check for default: name:default
    if idx := strings.IndexByte(s, ':'); idx >= 0 {
        return Param{
            Name:    s[:idx],
            Type:    ParamText,
            Default: s[idx+1:],
        }
    }

    // 4. Plain name
    return Param{Name: s, Type: ParamText}
}
```

### Pattern 2: Import Pipeline

**What:** Two-phase import: parse source format → convert to `[]Workflow` → preview/confirm → save.

**When to use:** `wf import` command.

**Example:**
```go
type ImportResult struct {
    Workflows []store.Workflow
    Warnings  []string  // Unmappable features, stored as YAML comments
    Errors    []error   // Parse failures
}

type Importer interface {
    Import(reader io.Reader) (*ImportResult, error)
}
```

### Pattern 3: Shell History Reader

**What:** Interface-based shell history reading with per-shell implementations.

**When to use:** `wf register --pick` and `wf register` (last command).

**Example:**
```go
type HistoryEntry struct {
    Command   string
    Timestamp time.Time // zero value if unavailable
}

type HistoryReader interface {
    // LastN returns the last n commands from history
    LastN(n int) ([]HistoryEntry, error)
    // Last returns the most recent command
    Last() (HistoryEntry, error)
}
```

### Anti-Patterns to Avoid
- **Don't modify the regex:** The existing `\{\{([^}]+)\}\}` regex already captures all content between `{{` and `}}`. All parsing logic belongs in `parseInner()`, not in the regex.
- **Don't block on dynamic param failure:** Always degrade to text input with a visible error message. Never let a failed shell command prevent the user from proceeding.
- **Don't parse fish history with a YAML library:** Fish's format is "ad-hoc broken pseudo-YAML" — use line-by-line custom parsing.
- **Don't silently drop import data:** Unmappable fields (Pet's `output`, Warp's `source_url`, `author`, `author_url`, `shells`) must be preserved as YAML comments.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TOML parsing | Custom TOML parser | `pelletier/go-toml/v2` | TOML has edge cases (multiline strings, escape sequences) |
| List selection UI | Custom list widget | `bubbles/list` or `huh` selection | Already available in the project's deps |
| Shell detection | Manual `$SHELL` parsing | Check `$SHELL` env var + fallback to `/etc/passwd` | Standard approach, but also check `HISTFILE` override |
| Command execution (dynamic params) | Manual `exec.Command` | `os/exec` with `context.WithTimeout` | Need timeout (5s) and proper stdout capture |
| Interactive conflict prompts | Custom prompt loop | `huh` forms | Already a dependency, handles select/confirm patterns |

**Key insight:** The project already has `huh` (charmbracelet forms library) as a dependency. Use it for the interactive import conflict prompts and register metadata collection rather than building raw stdin prompts.

## Common Pitfalls

### Pitfall 1: Zsh Metafied History Encoding
**What goes wrong:** Reading `~/.zsh_history` as UTF-8 text produces garbled output for entries containing non-ASCII characters.
**Why it happens:** Zsh uses a custom "metafied" encoding where byte `0x83` is a meta-character. The byte following `0x83` must be XORed with `0x20` to decode the original byte.
**How to avoid:** Implement an `unmetafy([]byte) []byte` function that processes the raw bytes before treating the content as text. Read the file as `[]byte`, not as `string`.
**Warning signs:** Tests pass with ASCII-only commands but fail with commands containing UTF-8 characters, paths with special chars, or commands with non-English text.

### Pitfall 2: Zsh Extended History Format Variants
**What goes wrong:** Parser expects `EXTENDED_HISTORY` format (`: timestamp:duration;command`) but the file has plain format (one command per line).
**Why it happens:** Not all zsh users enable `EXTENDED_HISTORY`. Some have `INC_APPEND_HISTORY`, `SHARE_HISTORY`, or other options that affect the format.
**How to avoid:** Auto-detect the format: if a line starts with `: ` followed by digits, parse as extended; otherwise treat as plain. Support both formats in the same file.
**Warning signs:** Parser crashes or returns empty results on some users' history files.

### Pitfall 3: Multiline Commands in History
**What goes wrong:** A multiline command (e.g., `for` loop) spans multiple lines in the history file, causing the parser to treat each line as a separate command.
**Why it happens:** Zsh uses `\` + newline continuation. Bash with `cmdhist`/`lithist` may embed real newlines.
**How to avoid:** For zsh extended history, entries start with `: timestamp:...;` — any line NOT starting with that pattern is a continuation of the previous entry. For bash with timestamps, lines between `#timestamp` markers are one entry.
**Warning signs:** Multi-line commands appear truncated or split into fragments.

### Pitfall 4: Pet Parameter Syntax Ambiguity
**What goes wrong:** Pet's `<param=default>` syntax where default values can contain `=` signs and spaces gets mis-parsed.
**Why it happens:** Pet's parameter format is: `<name=default_value>` where the default value is everything after the first `=` up to (but not including) trailing spaces before `>`.
**How to avoid:** Use a proper parser: find `<`, find `>`, split inner on first `=` only, trim trailing spaces from the default value. Use regex: `<([^>=]+)(?:=([^>]*[^\s>]))?\s*>`.
**Warning signs:** Parameters with `=` in default values get truncated.

### Pitfall 5: Template Syntax Collision Between Import Formats
**What goes wrong:** Warp already uses `{{arg}}` syntax which collides with wf's `{{name}}` syntax, but `{{arg}}` in Warp has no default/type metadata — it's just a name placeholder.
**Why it happens:** Both tools independently chose the same `{{}}` delimiters.
**How to avoid:** Warp's `{{arg}}` maps 1:1 to wf's `{{arg}}`. The `arguments` YAML section provides `default_value` and `description` which map to wf's `Arg` struct. This is a non-issue but document it clearly.
**Warning signs:** None — this is actually fortunate convergence.

### Pitfall 6: Dynamic Parameter Security
**What goes wrong:** A dynamic parameter's command (`{{branch!rm -rf /}}`) executes arbitrary shell commands.
**Why it happens:** The user defines the dynamic command, which is executed by `os/exec`.
**How to avoid:** Dynamic commands are defined BY the user FOR the user — this is equivalent to them typing it in their shell. No sandboxing needed. However, imported workflows with dynamic params should show a preview/warning. Use `context.WithTimeout` (5s) to prevent hangs.
**Warning signs:** None for self-defined workflows. For imported workflows, display the dynamic commands during import preview.

## Code Examples

### Extended Param Struct

```go
// ParamType discriminates parameter behavior.
type ParamType int

const (
    ParamText    ParamType = iota // Free text input (default)
    ParamEnum                     // Selection from predefined options
    ParamDynamic                  // Options populated by shell command
)

// Param represents a named parameter extracted from a command template.
type Param struct {
    Name       string
    Type       ParamType
    Default    string
    Options    []string  // For ParamEnum: the option list
    DynamicCmd string    // For ParamDynamic: shell command to execute
}
```

### Extended Arg Struct (store/workflow.go)

```go
// Arg defines a named parameter for a workflow command.
type Arg struct {
    Name        string   `yaml:"name"`
    Default     string   `yaml:"default,omitempty"`
    Description string   `yaml:"description,omitempty"`
    Type        string   `yaml:"type,omitempty"`        // "text", "enum", "dynamic"
    Options     []string `yaml:"options,omitempty"`      // For enum type
    DynamicCmd  string   `yaml:"dynamic_cmd,omitempty"`  // For dynamic type
}
```

### Pet TOML Parsing

```go
// Source: https://github.com/knqyf263/pet/blob/master/snippet/snippet.go
// Pet's SnippetInfo struct for reference:
//   Description string
//   Command     string `toml:"command,multiline"`
//   Tag         []string
//   Output      string

type PetFile struct {
    Snippets []PetSnippet `toml:"snippets"`
}

type PetSnippet struct {
    Description string   `toml:"description"`
    Command     string   `toml:"command"`
    Tag         []string `toml:"tag"`
    Output      string   `toml:"output"`
}

// petParamRegex matches Pet's <name> and <name=default> syntax.
var petParamRegex = regexp.MustCompile(`<([^>=]+?)(?:=([^>]*[^\s>]))?\s*>`)

func convertPetParam(command string) string {
    return petParamRegex.ReplaceAllStringFunc(command, func(match string) string {
        sub := petParamRegex.FindStringSubmatch(match)
        name := sub[1]
        def := sub[2]
        if def != "" {
            return "{{" + name + ":" + def + "}}"
        }
        return "{{" + name + "}}"
    })
}
```

### Warp YAML Parsing

```go
// Source: https://github.com/warpdotdev/workflows/blob/main/FORMAT.md
type WarpWorkflow struct {
    Name        string         `yaml:"name"`
    Command     string         `yaml:"command"`
    Tags        []string       `yaml:"tags"`
    Description string         `yaml:"description"`
    SourceURL   string         `yaml:"source_url"`
    Author      string         `yaml:"author"`
    AuthorURL   string         `yaml:"author_url"`
    Shells      []string       `yaml:"shells"`
    Arguments   []WarpArgument `yaml:"arguments"`
}

type WarpArgument struct {
    Name         string `yaml:"name"`
    Description  string `yaml:"description"`
    DefaultValue string `yaml:"default_value"`
}
```

### Zsh History Unmetafy

```go
// unmetafy decodes zsh's metafied encoding.
// When byte 0x83 is encountered, the following byte is XORed with 0x20.
func unmetafy(data []byte) []byte {
    var out []byte
    for i := 0; i < len(data); i++ {
        if data[i] == 0x83 && i+1 < len(data) {
            i++
            out = append(out, data[i]^0x20)
        } else {
            out = append(out, data[i])
        }
    }
    return out
}

// parseZshExtended parses a line in ": timestamp:duration;command" format.
func parseZshExtended(line string) (cmd string, ts time.Time, ok bool) {
    if !strings.HasPrefix(line, ": ") {
        return "", time.Time{}, false
    }
    rest := line[2:]
    semiIdx := strings.IndexByte(rest, ';')
    if semiIdx < 0 {
        return "", time.Time{}, false
    }
    meta := rest[:semiIdx]
    cmd = rest[semiIdx+1:]

    colonIdx := strings.IndexByte(meta, ':')
    if colonIdx < 0 {
        return "", time.Time{}, false
    }
    epoch, err := strconv.ParseInt(meta[:colonIdx], 10, 64)
    if err != nil {
        return cmd, time.Time{}, true // command ok, timestamp bad
    }
    return cmd, time.Unix(epoch, 0), true
}
```

### Fish History Parsing

```go
// Fish history format (~/.local/share/fish/fish_history):
// - cmd: some command here
//   when: 1678886400
//   paths:
//     - /some/path
//
// Each entry starts with "- cmd: " on a new line.
// The "when:" line has a unix timestamp.
// Parse line-by-line, NOT with a YAML parser.

func parseFishHistory(data []byte) []HistoryEntry {
    var entries []HistoryEntry
    var current *HistoryEntry

    scanner := bufio.NewScanner(bytes.NewReader(data))
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "- cmd: ") {
            if current != nil {
                entries = append(entries, *current)
            }
            current = &HistoryEntry{
                Command: strings.TrimPrefix(line, "- cmd: "),
            }
        } else if current != nil && strings.HasPrefix(strings.TrimSpace(line), "when: ") {
            ts := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "when: "))
            if epoch, err := strconv.ParseInt(ts, 10, 64); err == nil {
                current.Timestamp = time.Unix(epoch, 0)
            }
        }
    }
    if current != nil {
        entries = append(entries, *current)
    }
    return entries
}
```

### Bash History Parsing

```go
// Bash history format (~/.bash_history):
// Without HISTTIMEFORMAT: one command per line
// With HISTTIMEFORMAT: #timestamp\ncommand
//
// Detect by checking if lines start with #<digits>

func parseBashHistory(data []byte) []HistoryEntry {
    var entries []HistoryEntry
    var pendingTS time.Time

    scanner := bufio.NewScanner(bytes.NewReader(data))
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "#") {
            ts := strings.TrimPrefix(line, "#")
            if epoch, err := strconv.ParseInt(ts, 10, 64); err == nil {
                pendingTS = time.Unix(epoch, 0)
                continue
            }
        }
        if line == "" {
            continue
        }
        entry := HistoryEntry{Command: line, Timestamp: pendingTS}
        pendingTS = time.Time{}
        entries = append(entries, entry)
    }
    return entries
}
```

### Dynamic Parameter Execution

```go
func executeDynamicParam(command string, timeout time.Duration) ([]string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "sh", "-c", command)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("dynamic param command failed: %w", err)
    }

    var options []string
    scanner := bufio.NewScanner(bytes.NewReader(output))
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line != "" {
            options = append(options, line)
        }
    }
    return options, nil
}
```

## Format Specifications

### Pet TOML Format (Authoritative — from source code)

**Source:** `github.com/knqyf263/pet/snippet/snippet.go` (verified via WebFetch)

```toml
[[snippets]]
  description = "Deploy to staging environment"
  command = "kubectl apply -f <manifest=deploy.yaml> --namespace <env=staging>"
  tag = ["k8s", "deploy"]
  output = ""

[[snippets]]
  description = "Show SSL certificate expiration"
  command = "echo | openssl s_client -connect <host>:443 2>/dev/null | openssl x509 -dates -noout"
  tag = ["ssl", "security"]
  output = "notBefore=Nov 3 00:00:00 2015 GMT\nnotAfter=Nov 28 12:00:00 2018 GMT"
```

**Go struct (from Pet source):**
```go
type SnippetInfo struct {
    Filename    string   `toml:"-"`
    Description string
    Command     string   `toml:"command,multiline"`
    Tag         []string
    Output      string
}
```

**Field mapping to wf:**
| Pet Field | wf Field | Mapping |
|-----------|----------|---------|
| `description` | `Workflow.Description` | Direct copy |
| `command` | `Workflow.Command` | Convert `<param>` → `{{param}}`, `<param=default>` → `{{param:default}}` |
| `tag` | `Workflow.Tags` | Direct copy (both `[]string`) |
| `output` | — | No equivalent. Preserve as YAML comment: `# output: ...` |
| — | `Workflow.Name` | Auto-generate from `description` (slugify) |
| — | `Workflow.Args` | Auto-extract from converted command template |

**Pet parameter syntax:**
- `<param>` — named parameter, no default
- `<param=default>` — named parameter with default value
- `<param=|_opt1_||_opt2_||_opt3_|>` — named parameter with multiple options (pipe-delimited, underscores around values)
- Default value extends to `>` but trailing spaces before `>` are trimmed

### Warp YAML Format (Authoritative — from FORMAT.md)

**Source:** `github.com/warpdotdev/workflows/blob/main/FORMAT.md` (verified via WebFetch)

```yaml
---
name: Uninstall a Homebrew package and all of its dependencies
command: |-
    brew tap beeftornado/rmtree
    brew rmtree {{package_name}}
tags:
  - homebrew
description: Uses the external command rmtree to remove a Homebrew package and all of its dependencies
arguments:
  - name: package_name
    description: The name of the package that should be removed
    default_value: ~
source_url: "https://stackoverflow.com/questions/7323261"
author: Ory Band
author_url: "https://stackoverflow.com/users/207894"
shells: []
```

**Field mapping to wf:**
| Warp Field | wf Field | Mapping |
|------------|----------|---------|
| `name` | `Workflow.Name` | Direct copy |
| `command` | `Workflow.Command` | Direct copy (same `{{arg}}` syntax) |
| `tags` | `Workflow.Tags` | Direct copy |
| `description` | `Workflow.Description` | Direct copy |
| `arguments[].name` | `Arg.Name` | Direct copy |
| `arguments[].description` | `Arg.Description` | Direct copy |
| `arguments[].default_value` | `Arg.Default` | Direct copy (YAML `~` = empty) |
| `source_url` | — | No equivalent. Preserve as YAML comment |
| `author` | — | No equivalent. Preserve as YAML comment |
| `author_url` | — | No equivalent. Preserve as YAML comment |
| `shells` | — | No equivalent. Preserve as YAML comment |

### Shell History File Formats

#### Zsh (`~/.zsh_history` or `$HISTFILE`)

**Without EXTENDED_HISTORY:**
```
ls -la
git status
docker ps
```
One command per line. No metadata.

**With EXTENDED_HISTORY (common):**
```
: 1471766804:3;git push origin master
: 1471766900:0;ls -la
: 1471767000:15;docker build -t myapp .
```
Format: `: <epoch>:<duration>;<command>`

**Binary encoding:** File uses "metafied" encoding (byte `0x83` = meta-character, next byte XORed with `0x20`). Must read as bytes and unmetafy before text parsing.

**Multiline commands:** Continuation lines do NOT start with `: `. Use this to detect multi-line entries.

#### Bash (`~/.bash_history` or `$HISTFILE`)

**Without HISTTIMEFORMAT (default):**
```
ls -la
git status
docker ps
```
One command per line.

**With HISTTIMEFORMAT:**
```
#1678886400
ls -la
#1678886460
git status
```
Timestamp lines start with `#` followed by epoch. Command follows on next line.

**No binary encoding issues.** Plain text, UTF-8.

#### Fish (`~/.local/share/fish/fish_history` or `$XDG_DATA_HOME/fish/fish_history`)

```
- cmd: git push origin master
  when: 1678886400
- cmd: ls -la
  when: 1678886460
- cmd: docker build -t myapp .
  when: 1678886520
  paths:
    - Dockerfile
```
Pseudo-YAML format. **Do NOT use a YAML parser** — the format is described by fish developers as "ad-hoc broken pseudo-YAML." Parse line-by-line: entries start with `- cmd: `, timestamps on `when: ` lines.

**No binary encoding issues.** Plain text, UTF-8.

## Template Parser Extension Strategy

### Current State
- Regex: `\{\{([^}]+)\}\}` — captures inner content (unchanged)
- `parseInner(s)` splits on first `:` → `(name, default)`
- `Param` has `Name` and `Default` fields only
- `Render()` substitutes by name from values map

### Extension Plan

**Step 1: Add `ParamType` and extend `Param` struct**
- Add `Type ParamType` (Text/Enum/Dynamic)
- Add `Options []string` (for enum)
- Add `DynamicCmd string` (for dynamic)

**Step 2: Modify `parseInner()` to detect type by delimiter priority**
1. Check for `!` → Dynamic (everything before `!` is name, after is command)
2. Check for `|` → Enum (first segment is name, rest are options, `*`-prefixed option is default)
3. Check for `:` → Text with default (existing behavior)
4. No delimiter → Plain text parameter

**Why this order:** `!` takes priority because a dynamic command could contain `|` or `:`. The `|` check comes before `:` because an enum option could theoretically contain `:`. The name portion (before the delimiter) must not contain any of these special chars.

**Step 3: Extend `Render()` to handle enum/dynamic**
- Enum: Render uses the selected option value (same as text — just a string substitution)
- Dynamic: Render uses the selected option value (same as text)
- The `Render` function doesn't need to know about types — it just substitutes strings

**Step 4: Extend `Arg` struct in store**
- Add `Type`, `Options`, `DynamicCmd` fields (YAML-serializable)
- YAML args section can override/extend what's in the inline syntax

**Step 5: Extend paramfill picker**
- For `ParamEnum`: Replace `textinput` with a list selector (bubbles `list` or custom vertical selection)
- For `ParamDynamic`: Execute command on focus, show spinner, populate list selector. On failure, fall back to `textinput`
- For `ParamText`: Unchanged (existing textinput behavior)

### Regex Does NOT Need Changes

The existing regex `\{\{([^}]+)\}\}` already captures everything between `{{` and `}}` including pipes, bangs, colons, spaces, and any other characters. All new parsing logic goes into `parseInner()`.

Proof:
- `{{env|dev|staging|*prod}}` → captured group: `env|dev|staging|*prod` ✓
- `{{branch!git branch --list}}` → captured group: `branch!git branch --list` ✓
- `{{name:default}}` → captured group: `name:default` ✓

### Edge Case: Dynamic Command with Pipes

`{{branch!git branch --list | grep feature}}` — the `|` in the shell command would be misinterpreted as enum delimiter if we check for `|` before `!`.

**Solution:** Check for `!` FIRST. The bang splits `name!command`, and `command` can contain anything (pipes, colons, etc.). This is why the priority order matters.

### Edge Case: Enum Option That Looks Like a Default

`{{env|dev:3000|staging:8080|*prod:443}}` — should `dev:3000` be treated as an option or as `dev` with default `3000`?

**Solution:** Once we detect `|` (enum mode), the `:` is NOT parsed as a default separator. The entire string `dev:3000` is one option. Colons within enum options are literal. This is consistent with the decision to use `*` for enum defaults, not `:`.

## Arg Struct Extension Summary

Current:
```go
type Arg struct {
    Name        string `yaml:"name"`
    Default     string `yaml:"default,omitempty"`
    Description string `yaml:"description,omitempty"`
}
```

Extended:
```go
type Arg struct {
    Name        string   `yaml:"name"`
    Default     string   `yaml:"default,omitempty"`
    Description string   `yaml:"description,omitempty"`
    Type        string   `yaml:"type,omitempty"`         // "text" (default), "enum", "dynamic"
    Options     []string `yaml:"options,omitempty"`       // For "enum": list of valid values
    DynamicCmd  string   `yaml:"dynamic_cmd,omitempty"`   // For "dynamic": shell command
}
```

**Why `string` for Type (not iota):** The Arg struct is YAML-serialized. Using a string ("text", "enum", "dynamic") makes the YAML human-readable and editable. The internal `template.ParamType` can use iota for in-memory efficiency; conversion happens at the boundary.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pet `<param>` syntax | Most tools now use `{{param}}` (Warp, wf) | ~2020+ | Pet's syntax requires translation on import |
| Global shell history access | Per-shell history files | Always | Must support 3 formats (zsh, bash, fish) |
| Plain zsh history | Extended history with timestamps | Widely adopted | Most users have `EXTENDED_HISTORY` but must handle both |

**Deprecated/outdated:**
- Pet's `<param=|_opt1_||_opt2_|>` multi-option syntax is complex — wf's `{{param|opt1|opt2|*default}}` is cleaner. No need to support Pet's multi-option syntax during import (convert to regular default or warn).

## Open Questions

1. **Pet multi-option parameter import**
   - What we know: Pet supports `<param=|_opt1_||_opt2_||_opt3_|>` for multi-value params
   - What's unclear: Should we convert these to wf enum syntax or just take the first option as default?
   - Recommendation: Convert to enum syntax `{{param|opt1|opt2|opt3}}` — direct mapping is possible. Flag as LOW confidence since Pet's exact parsing for multi-option with default selection is not fully documented.

2. **Fish history escaping**
   - What we know: Fish stores commands as-is in pseudo-YAML
   - What's unclear: How does fish handle commands containing `\n` or YAML-special characters in the command field?
   - Recommendation: Fish escapes backslashes in history. Parse carefully: if a `cmd:` value contains `\\n`, it represents a literal newline in the original command. Test with real fish history files.

3. **Register auto-detection patterns**
   - What we know: Should detect IPs, ports, paths, URLs, env-like values
   - What's unclear: What regex patterns work best without false positives?
   - Recommendation: Claude's discretion per CONTEXT.md. Start conservative: detect clearly formatted IPs (`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`), ports (`:(\d{4,5})`), URLs (`https?://\S+`), and ALL_CAPS values. Let users adjust.

## Sources

### Primary (HIGH confidence)
- Pet snippet struct source code — `github.com/knqyf263/pet/blob/master/snippet/snippet.go` (WebFetch verified)
- Pet README — `github.com/knqyf263/pet/blob/master/README.md` (WebFetch verified)
- Warp FORMAT.md — `github.com/warpdotdev/workflows/blob/main/FORMAT.md` (WebFetch verified)
- Warp README — `github.com/warpdotdev/workflows/blob/main/README.md` (WebFetch verified)
- Existing wf codebase — `internal/template/parser.go`, `internal/store/workflow.go` (Read verified)

### Secondary (MEDIUM confidence)
- Zsh EXTENDED_HISTORY format — multiple Stack Overflow sources + Hacker News discussions agree on format
- Zsh metafied encoding — multiple sources agree on `0x83` meta-byte + XOR `0x20` scheme
- Bash history format — well-documented across multiple sources
- Fish history format — confirmed as "pseudo-YAML" by multiple sources including fish shell developers on GitHub

### Tertiary (LOW confidence)
- Pet multi-option syntax (`<param=|_opt1_||_opt2_|>`) — documented in Pet README but parsing edge cases unclear
- Fish history escaping behavior — limited documentation on edge cases

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified from existing go.mod and official library repos
- Architecture: HIGH — based on verified analysis of existing codebase structure
- Import formats: HIGH — verified from official source code and documentation
- Shell history: MEDIUM — format specs verified across multiple sources but metafied encoding edge cases need testing
- Pitfalls: HIGH — identified from verified format specifications and real implementation concerns

**Research date:** 2026-02-23
**Valid until:** 2026-03-23 (30 days — these formats are stable and rarely change)
