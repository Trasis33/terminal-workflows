package ai

import (
	"fmt"
	"strings"
)

// generateSystemPrompt constrains the AI to produce valid JSON matching our workflow schema.
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

// autofillSystemPrompt constrains the AI to generate only the requested metadata fields.
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

// buildGeneratePrompt constructs the user message for workflow generation.
func buildGeneratePrompt(req GenerateRequest) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Generate a workflow for: %s", req.Description)
	if req.Shell != "" {
		fmt.Fprintf(&sb, "\nTarget shell: %s", req.Shell)
	}
	if len(req.Tags) > 0 {
		fmt.Fprintf(&sb, "\nSuggested tags: %s", strings.Join(req.Tags, ", "))
	}
	if req.Folder != "" {
		fmt.Fprintf(&sb, "\nFolder/category: %s", req.Folder)
	}
	return sb.String()
}

// buildAutofillPrompt constructs the user message for metadata autofill.
func buildAutofillPrompt(req AutofillRequest) string {
	var sb strings.Builder
	if req.Workflow != nil {
		fmt.Fprintf(&sb, "Workflow command: %s", req.Workflow.Command)
		if req.Workflow.Name != "" {
			fmt.Fprintf(&sb, "\nExisting name: %s", req.Workflow.Name)
		}
		if req.Workflow.Description != "" {
			fmt.Fprintf(&sb, "\nExisting description: %s", req.Workflow.Description)
		}
		if len(req.Workflow.Tags) > 0 {
			fmt.Fprintf(&sb, "\nExisting tags: %s", strings.Join(req.Workflow.Tags, ", "))
		}
	}
	if len(req.Fields) > 0 {
		fmt.Fprintf(&sb, "\nFill these fields: %s", strings.Join(req.Fields, ", "))
	}
	return sb.String()
}
