package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

const defaultTimeout = 30 * time.Second

// CopilotGenerator implements Generator using the GitHub Copilot SDK.
type CopilotGenerator struct {
	client  *copilot.Client
	models  ModelConfig
	timeout time.Duration
}

// Generate creates a new workflow from a natural language description using the Copilot SDK.
func (g *CopilotGenerator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	session, err := g.client.CreateSession(ctx, &copilot.SessionConfig{
		Model: g.models.ForTask("generate"),
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

	content := *result.Data.Content
	var gen GenerateResult
	if err := json.Unmarshal([]byte(content), &gen); err != nil {
		// Fallback: try to extract JSON from markdown code fences
		extracted := extractJSON(content)
		if extracted == "" {
			return nil, fmt.Errorf("parse AI response: %w", err)
		}
		if err := json.Unmarshal([]byte(extracted), &gen); err != nil {
			return nil, fmt.Errorf("parse AI response after extraction: %w", err)
		}
	}
	return &gen, nil
}

// Autofill generates metadata for an existing workflow using the Copilot SDK.
func (g *CopilotGenerator) Autofill(ctx context.Context, req AutofillRequest) (*AutofillResult, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	session, err := g.client.CreateSession(ctx, &copilot.SessionConfig{
		Model: g.models.ForTask("autofill"),
		SystemMessage: &copilot.SystemMessageConfig{
			Content: autofillSystemPrompt,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	defer session.Destroy()

	prompt := buildAutofillPrompt(req)
	result, err := session.SendAndWait(ctx, copilot.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("autofill: %w", err)
	}

	if result == nil || result.Data.Content == nil {
		return nil, fmt.Errorf("empty response from AI")
	}

	content := *result.Data.Content
	var af AutofillResult
	if err := json.Unmarshal([]byte(content), &af); err != nil {
		extracted := extractJSON(content)
		if extracted == "" {
			return nil, fmt.Errorf("parse AI response: %w", err)
		}
		if err := json.Unmarshal([]byte(extracted), &af); err != nil {
			return nil, fmt.Errorf("parse AI response after extraction: %w", err)
		}
	}
	return &af, nil
}

// Available reports whether the Copilot backend is ready.
// If a CopilotGenerator exists, it was successfully initialized.
func (g *CopilotGenerator) Available() bool {
	return true
}

// Close shuts down the Copilot CLI client.
func (g *CopilotGenerator) Close() error {
	if g.client != nil {
		return g.client.Stop()
	}
	return nil
}

// extractJSON attempts to extract a JSON object from a string that may be
// wrapped in markdown code fences (```json ... ``` or ``` ... ```).
func extractJSON(s string) string {
	// Try ```json\n...\n```
	if idx := strings.Index(s, "```json"); idx >= 0 {
		start := idx + len("```json")
		if end := strings.Index(s[start:], "```"); end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// Try ```\n...\n```
	if idx := strings.Index(s, "```"); idx >= 0 {
		start := idx + len("```")
		if end := strings.Index(s[start:], "```"); end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// Try to find raw JSON object
	if idx := strings.Index(s, "{"); idx >= 0 {
		if end := strings.LastIndex(s, "}"); end > idx {
			return s[idx : end+1]
		}
	}
	return ""
}
