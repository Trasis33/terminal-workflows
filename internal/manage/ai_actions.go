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
