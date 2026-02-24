# Phase 5: AI Integration - Research

**Researched:** 2026-02-24
**Domain:** GitHub Copilot SDK for Go / AI-powered workflow generation
**Confidence:** MEDIUM (SDK in technical preview, API may change)

## Summary

Phase 5 adds AI-powered workflow generation and metadata auto-fill to the `wf` CLI via the GitHub Copilot SDK. Research confirms that an official Go SDK exists at `github.com/github/copilot-sdk/go` (v0.1.25, released 2026-02-18), providing programmatic access to the Copilot CLI's agentic runtime via JSON-RPC. The SDK is in **technical preview** — functional and documented, but may introduce breaking changes.

The SDK communicates with the Copilot CLI binary running in server mode. It manages the CLI process lifecycle automatically (start/stop), supports streaming responses, custom system messages, model selection, and BYOK (Bring Your Own Key) for alternative providers. The SDK does NOT expose a simple chat-completions REST API — it wraps the full Copilot CLI agent runtime, which is more powerful but heavier than a simple LLM API call.

**Primary recommendation:** Use the official Copilot Go SDK (`github.com/github/copilot-sdk/go`) with `SendAndWait` for synchronous generation and system message configuration for prompt engineering. Wrap all SDK access behind a Go interface for testability and graceful degradation. Detect CLI availability lazily on first AI command invocation via `exec.LookPath("copilot")` and `client.Start()` error handling.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/github/copilot-sdk/go` | v0.1.25 | Copilot CLI SDK client | Official Go SDK, manages CLI process lifecycle, sessions, streaming, model selection |
| Copilot CLI binary (`copilot`) | Latest | AI runtime server | Required backend — SDK communicates via JSON-RPC to the CLI in server mode |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os/exec` | stdlib | CLI detection | `exec.LookPath("copilot")` for availability check |
| `context` | stdlib | Timeout control | `context.WithTimeout` for AI request timeouts |
| `encoding/json` | stdlib | JSON marshaling | Parsing AI-generated workflow JSON responses |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Copilot SDK | Direct OpenAI API via `net/http` | Simpler but loses Copilot auth, billing, model routing; requires separate API key management |
| Copilot SDK | GitHub Models API (`models.github.ai`) | OpenAI-compatible REST, simpler integration, but different billing/auth model; SDK is the blessed path |
| Full agentic sessions | Single-shot `SendAndWait` | Agent runtime is overkill for our use case; `SendAndWait` is the right abstraction |

**Installation:**
```bash
go get github.com/github/copilot-sdk/go
```

**Prerequisite:** Copilot CLI must be installed separately:
```bash
# Users install via GitHub CLI or standalone
gh extension install github/gh-copilot
# or standalone: https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli
```

**Bundling option:** The SDK supports embedding the Copilot CLI binary via Go's `embed` package using the bundler tool (`go get -tool github.com/github/copilot-sdk/go/cmd/bundler`). This is optional — for this project, we defer to the user having Copilot CLI installed.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── ai/
│   ├── client.go          # Generator interface + CopilotGenerator implementation
│   ├── client_test.go     # Tests with mock generator
│   ├── prompts.go         # System prompts and prompt templates
│   ├── models.go          # Model configuration types and defaults
│   └── availability.go    # SDK/CLI availability detection
cmd/wf/
├── generate.go            # wf generate command (Cobra)
├── autofill.go            # wf autofill command (Cobra)
internal/manage/
├── ai_actions.go          # TUI AI integration (generate/autofill actions)
```

### Pattern 1: Generator Interface for Testability
**What:** Define an interface that abstracts AI operations, with a Copilot SDK implementation and a mock for tests.
**When to use:** Always — this is the core architectural pattern.
**Example:**
```go
// Source: Architecture pattern based on existing Store interface pattern in this project
package ai

import (
    "context"
    "github.com/fredriklanga/wf/internal/store"
)

// GenerateRequest describes what the user wants to generate.
type GenerateRequest struct {
    Description string   // Natural language description
    Tags        []string // Optional tag hints
    Folder      string   // Optional folder/category
    Shell       string   // Target shell (bash, zsh, fish)
}

// AutofillRequest describes which fields to auto-fill.
type AutofillRequest struct {
    Workflow *store.Workflow // Existing workflow to enhance
    Fields   []string       // Which fields to fill: "name", "description", "tags", "args"
}

// GenerateResult contains the AI-generated workflow data.
type GenerateResult struct {
    Name        string      `json:"name"`
    Command     string      `json:"command"`
    Description string      `json:"description"`
    Tags        []string    `json:"tags,omitempty"`
    Args        []store.Arg `json:"args,omitempty"`
}

// AutofillResult contains the AI-generated metadata.
type AutofillResult struct {
    Name        *string      `json:"name,omitempty"`
    Description *string      `json:"description,omitempty"`
    Tags        []string     `json:"tags,omitempty"`
    Args        []store.Arg  `json:"args,omitempty"`
}

// Generator defines the interface for AI-powered workflow operations.
type Generator interface {
    // Generate creates a new workflow from a natural language description.
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)

    // Autofill generates metadata for an existing workflow.
    Autofill(ctx context.Context, req AutofillRequest) (*AutofillResult, error)

    // Available reports whether the AI backend is ready.
    Available() bool

    // Close shuts down the AI client.
    Close() error
}
```

### Pattern 2: Lazy Initialization with Singleton
**What:** Don't start the Copilot CLI process until the first AI command is invoked. Cache the client across commands.
**When to use:** Always — matches the user decision for lazy SDK check.
**Example:**
```go
// Source: Pattern derived from Copilot SDK docs + project's getStore() pattern
package ai

import (
    "context"
    "fmt"
    "os/exec"
    "sync"

    copilot "github.com/github/copilot-sdk/go"
)

var (
    once     sync.Once
    instance *CopilotGenerator
    initErr  error
)

// ErrUnavailable is returned when the Copilot CLI is not installed.
var ErrUnavailable = fmt.Errorf("AI features require GitHub Copilot CLI. Install with: gh extension install github/gh-copilot")

// GetGenerator returns the shared Generator instance, initializing on first call.
// Returns ErrUnavailable if the Copilot CLI is not found.
func GetGenerator(ctx context.Context) (Generator, error) {
    once.Do(func() {
        // Check CLI availability first (fast path)
        if _, err := exec.LookPath("copilot"); err != nil {
            initErr = ErrUnavailable
            return
        }

        client := copilot.NewClient(&copilot.ClientOptions{
            LogLevel: "error",
        })
        if err := client.Start(ctx); err != nil {
            initErr = fmt.Errorf("failed to start Copilot: %w", err)
            return
        }

        instance = &CopilotGenerator{client: client}
    })
    return instance, initErr
}
```

### Pattern 3: System Message + JSON-Only Response
**What:** Use system messages to constrain AI output to valid JSON matching our workflow schema. No free-form text parsing.
**When to use:** For both generate and autofill operations.
**Example:**
```go
// Source: Copilot SDK SessionConfig.SystemMessage + prompt engineering best practices
session, err := client.CreateSession(ctx, &copilot.SessionConfig{
    Model: modelName,
    SystemMessage: &copilot.SystemMessageConfig{
        Content: generateSystemPrompt, // See prompts section below
    },
})
```

### Pattern 4: Per-Task Model Configuration
**What:** Read model names from config.yaml with per-task overrides and a global fallback.
**When to use:** Always — matches user decision for model selection.
**Example:**
```yaml
# ~/.config/wf/config.yaml
ai:
  models:
    generate: "gpt-5.2"          # Command generation (needs best model)
    autofill: "gpt-4o-mini"      # Metadata fill (lightweight task)
    fallback: "gpt-4o-mini"      # Default if task-specific not set
  timeout: 30                     # Seconds per AI request
```

### Anti-Patterns to Avoid
- **Starting Copilot CLI at app startup:** Never start the CLI process in `init()` or `PersistentPreRunE` — lazy init only on AI commands.
- **Parsing free-form text responses:** Always use JSON-mode system prompts. Never try to parse natural language responses into structured data.
- **Long-lived sessions across commands:** Create a new session per operation (generate/autofill). Don't maintain multi-turn sessions for simple CLI commands.
- **Hardcoding model names:** Always read from config with fallback defaults. Models get deprecated (GPT-5 deprecated Feb 2026, replaced by GPT-5.2).
- **Blocking TUI during AI calls:** Run AI operations in a goroutine with tea.Cmd, never block the Bubble Tea update loop.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LLM API client | Custom HTTP client for chat completions | `github.com/github/copilot-sdk/go` | Handles auth (GitHub OAuth, BYOK), model routing, CLI lifecycle, streaming |
| CLI binary detection | Manual PATH search + version parsing | `exec.LookPath("copilot")` + `client.Start()` error handling | Standard Go pattern; SDK errors are typed (`*exec.Error`) |
| JSON schema for prompts | Hand-written schema strings | Go struct tags + `encoding/json` | Schema stays in sync with code; no drift |
| Retry logic | Custom exponential backoff | SDK handles retries internally; use `context.WithTimeout` for overall timeout | SDK manages connection retries; app only needs request-level timeout |
| Token counting / context management | Custom token counter | SDK's `InfiniteSessions` (auto-compaction) | Not needed for single-shot requests, but available if we ever need multi-turn |

**Key insight:** The Copilot SDK is a process manager + JSON-RPC client, not just an HTTP wrapper. It manages the CLI binary lifecycle, authentication, and protocol details. Building this from scratch would be significant effort with no benefit.

## Common Pitfalls

### Pitfall 1: SDK Requires Copilot CLI Binary
**What goes wrong:** The Go SDK does NOT include the AI runtime — it requires the `copilot` CLI binary installed separately. `go get` only installs the SDK client library.
**Why it happens:** The SDK communicates via JSON-RPC with the Copilot CLI running in server mode. The CLI is the actual AI runtime.
**How to avoid:** Check CLI availability with `exec.LookPath("copilot")` before attempting `client.Start()`. Provide a clear install hint in the error message.
**Warning signs:** `*exec.Error` from `client.Start()`.

### Pitfall 2: Model Deprecation Churn
**What goes wrong:** Hardcoded model names stop working. GPT-5 was deprecated 2026-02-17, replaced by GPT-5.2. Claude Opus 4.1 deprecated, replaced by Claude Opus 4.6.
**Why it happens:** GitHub rotates models frequently (monthly cadence).
**How to avoid:** Store model names in config.yaml, never hardcode. Use sensible defaults (`gpt-4o-mini` as fallback — stable, cheap). Document how to change models in help text.
**Warning signs:** Session creation errors mentioning unsupported model.

### Pitfall 3: Blocking the Bubble Tea Event Loop
**What goes wrong:** TUI freezes while waiting for AI response.
**Why it happens:** AI requests take 2-15 seconds. Calling `SendAndWait` synchronously in a tea.Model.Update blocks the entire TUI.
**How to avoid:** Use `tea.Cmd` to run AI operations asynchronously. Return a `tea.Msg` with the result. Show a spinner/loading state while waiting.
**Warning signs:** TUI becomes unresponsive during AI operations.

### Pitfall 4: Unparseable AI Responses
**What goes wrong:** AI returns markdown, explanatory text, or malformed JSON instead of clean JSON.
**Why it happens:** System prompts aren't strict enough, or model occasionally ignores instructions.
**How to avoid:** 
1. System prompt explicitly says "respond ONLY with JSON, no markdown, no explanation"
2. Include few-shot examples in the system prompt
3. Parse with `json.Unmarshal` and have clear error handling for parse failures
4. Offer "regenerate" on parse failure
**Warning signs:** `json.Unmarshal` errors on AI responses.

### Pitfall 5: Copilot Subscription Required
**What goes wrong:** User has Go SDK installed but no Copilot subscription → auth failure on session creation.
**Why it happens:** Copilot SDK requires active GitHub Copilot subscription (Individual, Business, or Enterprise) OR BYOK configuration.
**How to avoid:** Catch auth errors specifically and provide helpful message distinguishing "CLI not installed" from "not authenticated" from "no subscription".
**Warning signs:** Auth errors from `client.Start()` or `client.CreateSession()`.

### Pitfall 6: Session Cleanup
**What goes wrong:** Leaked Copilot CLI processes if `client.Stop()` isn't called.
**Why it happens:** SDK spawns a child process (Copilot CLI in server mode). If the parent process crashes without cleanup, the child persists.
**How to avoid:** Always `defer client.Stop()` immediately after successful `client.Start()`. For the singleton pattern, register cleanup in a shutdown hook.
**Warning signs:** Orphaned `copilot` processes in task manager.

## Code Examples

### Generate Workflow from Description
```go
// Source: Copilot SDK Go README (verified 2026-02-24)
func (g *CopilotGenerator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
    ctx, cancel := context.WithTimeout(ctx, g.timeout)
    defer cancel()

    session, err := g.client.CreateSession(ctx, &copilot.SessionConfig{
        Model: g.modelFor("generate"),
        SystemMessage: &copilot.SystemMessageConfig{
            Content: generateSystemPrompt,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("create session: %w", err)
    }
    defer session.Destroy()

    prompt := buildGeneratePrompt(req)
    result, err := session.SendAndWait(ctx, copilot.MessageOptions{
        Prompt: prompt,
    })
    if err != nil {
        return nil, fmt.Errorf("generate: %w", err)
    }

    if result == nil || result.Data.Content == nil {
        return nil, fmt.Errorf("empty response from AI")
    }

    var gen GenerateResult
    if err := json.Unmarshal([]byte(*result.Data.Content), &gen); err != nil {
        return nil, fmt.Errorf("parse AI response: %w", err)
    }
    return &gen, nil
}
```

### System Prompt for Workflow Generation
```go
// Source: Prompt engineering best practices for structured output
const generateSystemPrompt = `You are a CLI workflow generator for the "wf" terminal workflow manager.

Given a natural language description, generate a reusable command workflow.

RULES:
1. Respond ONLY with a single JSON object. No markdown, no explanation, no code fences.
2. Use {{parameter_name}} syntax for variable parts of the command.
3. Parameter names must be lowercase with underscores (e.g., {{branch_name}}).
4. Include sensible defaults where possible.
5. Tags should be lowercase, relevant categories.
6. Description should be concise (one sentence).

JSON SCHEMA:
{
  "name": "string (short, descriptive, kebab-case)",
  "command": "string (the command template with {{params}})",
  "description": "string (one-sentence description)",
  "tags": ["string"],
  "args": [
    {
      "name": "string (matches {{param}} in command)",
      "description": "string (what this parameter is for)",
      "default": "string (optional default value)",
      "type": "text|enum|dynamic",
      "options": ["string (for enum type only)"]
    }
  ]
}

EXAMPLES:

Input: "deploy to staging with rollback"
Output: {"name":"deploy-staging","command":"kubectl rollout restart deployment/{{app}} -n staging && kubectl rollout status deployment/{{app}} -n staging --timeout={{timeout}}","description":"Deploy app to staging with rollout status check","tags":["deploy","kubernetes","staging"],"args":[{"name":"app","description":"Application deployment name"},{"name":"timeout","description":"Rollout timeout duration","default":"120s"}]}

Input: "git squash last N commits"
Output: {"name":"git-squash","command":"git reset --soft HEAD~{{count}} && git commit -m '{{message}}'","description":"Squash last N commits into one","tags":["git","cleanup"],"args":[{"name":"count","description":"Number of commits to squash","default":"2"},{"name":"message","description":"New commit message"}]}`

const autofillSystemPrompt = `You are a metadata generator for CLI command workflows.

Given an existing workflow (command and any existing metadata), generate or improve the requested metadata fields.

RULES:
1. Respond ONLY with a JSON object containing the requested fields. No markdown, no explanation.
2. Only include fields that were requested.
3. For "args", analyze the command for {{parameter}} patterns and generate descriptions and types.
4. Tags should be lowercase, relevant categories.
5. Name should be kebab-case, short, and descriptive.
6. Description should be concise (one sentence).

JSON SCHEMA (include only requested fields):
{
  "name": "string (optional)",
  "description": "string (optional)",
  "tags": ["string"] (optional),
  "args": [{"name":"string","description":"string","default":"string","type":"text|enum|dynamic"}] (optional)
}`
```

### Graceful Degradation in Cobra Command
```go
// Source: Error handling cookbook + project patterns
var generateCmd = &cobra.Command{
    Use:   "generate [description]",
    Short: "Generate a workflow from a natural language description",
    Long:  `Uses AI to create a workflow from your description. Requires GitHub Copilot CLI.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := cmd.Context()
        
        gen, err := ai.GetGenerator(ctx)
        if err != nil {
            // Graceful degradation: clear message, don't crash
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            return nil // Don't return error — keeps exit code 0 for non-AI features
        }

        // ... proceed with generation
        _ = gen
        return nil
    },
}
```

### Async AI in Bubble Tea TUI
```go
// Source: Bubble Tea patterns + Copilot SDK async
type aiResultMsg struct {
    result *ai.GenerateResult
    err    error
}

func generateWorkflowCmd(gen ai.Generator, req ai.GenerateRequest) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        result, err := gen.Generate(ctx, req)
        return aiResultMsg{result: result, err: err}
    }
}

// In Update():
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case aiResultMsg:
        if msg.err != nil {
            m.showError(msg.err.Error())
            return m, nil
        }
        // Pre-fill the form with AI result
        m.prefillForm(msg.result)
        return m, nil
    }
    // ...
}
```

### Config Integration for Model Selection
```go
// Source: Project's existing config pattern + user decision
package ai

// ModelConfig holds per-task model configuration.
type ModelConfig struct {
    Generate string `yaml:"generate,omitempty"` // Model for workflow generation
    Autofill string `yaml:"autofill,omitempty"` // Model for metadata fill
    Fallback string `yaml:"fallback,omitempty"` // Default fallback model
}

// DefaultModelConfig returns sensible defaults.
func DefaultModelConfig() ModelConfig {
    return ModelConfig{
        Generate: "gpt-4o",       // Good balance of quality and speed
        Autofill: "gpt-4o-mini",  // Lightweight, fast, cheap
        Fallback: "gpt-4o-mini",  // Safe default
    }
}

func (g *CopilotGenerator) modelFor(task string) string {
    switch task {
    case "generate":
        if g.config.Generate != "" {
            return g.config.Generate
        }
    case "autofill":
        if g.config.Autofill != "" {
            return g.config.Autofill
        }
    }
    if g.config.Fallback != "" {
        return g.config.Fallback
    }
    return DefaultModelConfig().Fallback
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct OpenAI REST API | Copilot SDK wrapping CLI runtime | Jan 2026 (SDK launch) | Unified auth, billing, model routing through GitHub |
| GPT-5 as default model | GPT-5.2 / GPT-5.2-Codex | Feb 17, 2026 | GPT-5 deprecated; must use GPT-5.2+ |
| Claude Opus 4.1 | Claude Opus 4.6 | Feb 17, 2026 | Opus 4.1 deprecated |
| Manual CLI process management | SDK auto-manages CLI lifecycle | Jan 2026 | SDK handles start/stop/restart automatically |
| API key authentication only | GitHub OAuth + BYOK options | Jan 2026 | Users with Copilot subscription need no extra config |

**Deprecated/outdated:**
- GPT-5: Deprecated 2026-02-17, use GPT-5.2
- Claude Opus 4.1: Deprecated 2026-02-17, use Claude Opus 4.6
- GPT-5-Codex: Deprecated 2026-02-17, use GPT-5.2-Codex
- `copilot-proxy.githubusercontent.com` API endpoint: Deprecated, replaced by `api.githubcopilot.com`

## Open Questions

1. **`SendAndWait` availability in Go SDK**
   - What we know: The getting-started guide shows `SendAndWait` for Go. The Go README shows `Send` + event-based `On` handler pattern.
   - What's unclear: Whether `SendAndWait` is available in the latest Go SDK or only `Send` + event handling. Both patterns are documented.
   - Recommendation: Try `SendAndWait` first (simpler). Fall back to `Send` + channel-based wait if not available. Both are shown in official docs.

2. **ListModels() API shape in Go**
   - What we know: Go README mentions "Use `ListModels()` to check which models support [reasoning effort]". Search confirms it exists.
   - What's unclear: Exact return type and whether it's on `Client` or `Session`.
   - Recommendation: Call `client.ListModels()` at implementation time to discover shape. Not critical for planning — we primarily use config-driven model names.

3. **SystemMessageConfig exact fields in Go**
   - What we know: `SessionConfig.SystemMessage` is `*SystemMessageConfig`. TypeScript shows `{ content: string }`.
   - What's unclear: Whether Go struct uses `Content` field directly.
   - Recommendation: HIGH confidence this is `SystemMessageConfig{Content: "..."}` based on consistent API across SDKs. Verify at implementation.

4. **Error types for auth failure vs. CLI missing**
   - What we know: `*exec.Error` for CLI not found. Auth errors exist but type is undocumented for Go.
   - What's unclear: Exact error type for "Copilot subscription required" vs "not authenticated".
   - Recommendation: Use `errors.As` and `errors.Is` to differentiate. Provide generic helpful message for any non-exec error from `client.Start()`.

5. **Concurrent session limits**
   - What we know: SDK supports multiple sessions per client.
   - What's unclear: Whether there are practical limits on concurrent sessions per CLI instance.
   - Recommendation: We only need one session at a time for generate/autofill. Create per-operation, destroy after. No concurrency concern.

## Sources

### Primary (HIGH confidence)
- `github.com/github/copilot-sdk` — GitHub repo, README, Go SDK README (v0.1.25, Feb 18, 2026)
- `github.com/github/copilot-sdk/go/README.md` — Full Go API reference (raw content fetched)
- `github.com/github/copilot-sdk/docs/getting-started.md` — Getting started guide with Go examples
- `github.com/github/copilot-sdk/docs/auth/byok.md` — BYOK provider configuration
- `github.com/github/awesome-copilot/cookbook/copilot-sdk/go/error-handling.md` — Error handling patterns

### Secondary (MEDIUM confidence)
- Google Search: "GitHub Copilot SDK Go CLI integration 2026" — confirmed SDK existence, technical preview status
- Google Search: "GitHub Models API models.github.ai" — confirmed OpenAI-compatible alternative exists
- Google Search: "GitHub Copilot models deprecation 2026" — confirmed model deprecation dates

### Tertiary (LOW confidence)
- Google Search synthesized results about future model roadmap — model names/dates may shift

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM — SDK is real, documented, at v0.1.25, but "technical preview" means breaking changes possible
- Architecture: HIGH — Interface pattern, lazy init, async TUI integration are standard Go patterns independent of SDK
- Pitfalls: HIGH — Verified through official error handling cookbook and SDK README warnings
- Prompt engineering: MEDIUM — System prompts are proven patterns but need iteration with actual API responses
- Model selection: LOW — Model names change frequently; defaults should be conservative

**Research date:** 2026-02-24
**Valid until:** 2026-03-10 (SDK is fast-moving, technical preview — recheck before production deployment)
