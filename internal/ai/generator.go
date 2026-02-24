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

// GenerateResult contains the AI-generated workflow data.
type GenerateResult struct {
	Name        string      `json:"name"`
	Command     string      `json:"command"`
	Description string      `json:"description"`
	Tags        []string    `json:"tags,omitempty"`
	Args        []store.Arg `json:"args,omitempty"`
}

// AutofillRequest describes which fields to auto-fill for an existing workflow.
type AutofillRequest struct {
	Workflow *store.Workflow // Existing workflow to enhance
	Fields   []string        // Which fields to fill: "name", "description", "tags", "args"
}

// AutofillResult contains the AI-generated metadata.
// Pointer fields indicate optional values — nil means the field was not requested or not filled.
type AutofillResult struct {
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Args        []store.Arg `json:"args,omitempty"`
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
