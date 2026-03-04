package manage

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/ai"
	"github.com/fredriklanga/wf/internal/store"
)

// --- AI message types ---

// aiGenerateResultMsg carries the result of an AI generate operation.
type aiGenerateResultMsg struct {
	result *ai.GenerateResult
	err    error
}

// aiAutofillResultMsg carries the result of an AI autofill operation.
type aiAutofillResultMsg struct {
	result   *ai.AutofillResult
	workflow store.Workflow
	err      error
}

// aiErrorMsg carries an AI error for inline display.
type aiErrorMsg struct {
	err error
}

// aiLoadingMsg indicates an AI operation is in progress.
type aiLoadingMsg struct {
	task string
}

// perFieldAIRequestMsg requests AI generation for a single field.
type perFieldAIRequestMsg struct {
	fieldName string            // e.g. "name", "description", "command", "tags", "param:0:default"
	context   map[string]string // snapshot of all field values for AI context
}

// perFieldAIResultMsg carries the result of per-field AI generation.
type perFieldAIResultMsg struct {
	fieldName string
	value     string
	err       error
}

// suggestParamsResultMsg carries the result of command-to-params AI analysis.
type suggestParamsResultMsg struct {
	args []store.Arg
	err  error
}

// showAIGenerateDialogMsg triggers the AI description input dialog.
type showAIGenerateDialogMsg struct{}

// showAIAutofillDialogMsg triggers AI autofill for the selected workflow.
type showAIAutofillDialogMsg struct {
	workflow store.Workflow
}

// --- AI async tea.Cmd functions ---

// generateWorkflowCmd creates a tea.Cmd that runs AI generation asynchronously.
// All needed data is passed as function parameters — this runs in a goroutine
// via tea.Cmd and must NOT access any Model fields.
func generateWorkflowCmd(description string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gen, err := ai.GetGenerator(ctx)
		if err != nil {
			return aiGenerateResultMsg{err: err}
		}

		result, err := gen.Generate(ctx, ai.GenerateRequest{
			Description: description,
		})
		return aiGenerateResultMsg{result: result, err: err}
	}
}

// autofillWorkflowCmd creates a tea.Cmd that runs AI autofill asynchronously.
// All needed data is passed as function parameters — this runs in a goroutine
// via tea.Cmd and must NOT access any Model fields.
func autofillWorkflowCmd(wf store.Workflow, fields []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gen, err := ai.GetGenerator(ctx)
		if err != nil {
			return aiAutofillResultMsg{workflow: wf, err: err}
		}

		result, err := gen.Autofill(ctx, ai.AutofillRequest{
			Workflow: &wf,
			Fields:   fields,
		})
		return aiAutofillResultMsg{result: result, workflow: wf, err: err}
	}
}

// --- AI Generate Dialog ---

const dialogAIGenerate dialogType = 10 // separate value space from existing dialog types

// NewAIGenerateDialog creates a dialog with a text input for the AI description.
func NewAIGenerateDialog(theme Theme) DialogModel {
	ti := textinput.New()
	ti.Placeholder = "Describe the workflow you want..."
	ti.CharLimit = 256
	ti.Width = 50
	ti.Focus()

	return DialogModel{
		dtype:    dialogAIGenerate,
		title:    "AI Generate Workflow",
		message:  "Describe what you want this workflow to do:",
		input:    ti,
		hasInput: true,
		theme:    theme,
	}
}

// mergeAutofillResult merges AI autofill results into an existing workflow.
// Only non-nil fields from the result are applied.
func mergeAutofillResult(wf store.Workflow, result *ai.AutofillResult) store.Workflow {
	if result.Name != nil && strings.TrimSpace(*result.Name) != "" {
		wf.Name = *result.Name
	}
	if result.Description != nil && strings.TrimSpace(*result.Description) != "" {
		wf.Description = *result.Description
	}
	if len(result.Tags) > 0 {
		wf.Tags = result.Tags
	}
	if len(result.Args) > 0 {
		wf.Args = result.Args
	}
	return wf
}

// perFieldAICmd creates a tea.Cmd that runs per-field AI generation asynchronously.
// It constructs an AutofillRequest from the context snapshot and requests only the target field.
func perFieldAICmd(fieldName string, ctx map[string]string) tea.Cmd {
	return func() tea.Msg {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gen, err := ai.GetGenerator(bgCtx)
		if err != nil {
			return perFieldAIResultMsg{fieldName: fieldName, err: err}
		}

		// Build a workflow from the context snapshot.
		wf := store.Workflow{
			Name:        ctx["name"],
			Command:     ctx["command"],
			Description: ctx["description"],
		}
		if tags := ctx["tags"]; tags != "" {
			for _, t := range strings.Split(tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					wf.Tags = append(wf.Tags, t)
				}
			}
		}

		// Map field name to autofill field.
		var requestField string
		switch {
		case fieldName == "name":
			requestField = "name"
		case fieldName == "description":
			requestField = "description"
		case fieldName == "tags":
			requestField = "tags"
		case fieldName == "command":
			requestField = "description" // Use description field to generate command context
		default:
			// param-level fields: use "args" field
			requestField = "args"
		}

		result, err := gen.Autofill(bgCtx, ai.AutofillRequest{
			Workflow: &wf,
			Fields:   []string{requestField},
		})
		if err != nil {
			return perFieldAIResultMsg{fieldName: fieldName, err: err}
		}

		// Extract the specific field value from the result.
		var value string
		switch fieldName {
		case "name":
			if result.Name != nil {
				value = *result.Name
			}
		case "description":
			if result.Description != nil {
				value = *result.Description
			}
		case "tags":
			if len(result.Tags) > 0 {
				value = strings.Join(result.Tags, ", ")
			}
		case "command":
			if result.Description != nil {
				value = *result.Description
			}
		default:
			// For param-level fields, extract from args.
			if len(result.Args) > 0 {
				// Try to find a matching arg by name from context.
				paramName := ctx["param_name"]
				for _, arg := range result.Args {
					if arg.Name == paramName {
						// Determine which sub-field was requested.
						if strings.HasSuffix(fieldName, ":default") {
							value = arg.Default
						} else if strings.HasSuffix(fieldName, ":options") {
							value = strings.Join(arg.Options, ", ")
						} else if strings.HasSuffix(fieldName, ":dynamic_cmd") {
							value = arg.DynamicCmd
						}
						break
					}
				}
			}
		}

		return perFieldAIResultMsg{fieldName: fieldName, value: value}
	}
}

// suggestParamsCmd creates a tea.Cmd that analyzes a command to suggest parameters.
func suggestParamsCmd(command string) tea.Cmd {
	return func() tea.Msg {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gen, err := ai.GetGenerator(bgCtx)
		if err != nil {
			return suggestParamsResultMsg{err: err}
		}

		wf := store.Workflow{
			Command: command,
		}

		result, err := gen.Autofill(bgCtx, ai.AutofillRequest{
			Workflow: &wf,
			Fields:   []string{"args"},
		})
		if err != nil {
			return suggestParamsResultMsg{err: err}
		}

		return suggestParamsResultMsg{args: result.Args}
	}
}
