package manage

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteDialogListLoadsAndFiltersVisibleRows(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}}",
		Args: []store.Arg{{
			Name:           "pod",
			Type:           "list",
			ListCmd:        "printf 'NAME\nalpha\nbeta\n'",
			ListSkipHeader: 1,
		}},
	}

	dlg := NewExecuteDialog(wf, 70, DefaultTheme())
	assert.NotContains(t, dlg.viewParamFill(), "NAME")

	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	view := dlg.viewParamFill()
	assert.Contains(t, view, "1. beta")
	assert.NotContains(t, view, "alpha")
}

func TestExecuteDialogListSelectionConfirmsExtractedValue(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}}",
		Args: []store.Arg{{
			Name:           "pod",
			Type:           "list",
			ListCmd:        "printf 'NAME\nalpha|one\nbeta|two\n'",
			ListDelimiter:  "|",
			ListFieldIndex: 1,
			ListSkipHeader: 1,
		}},
	}

	dlg := NewExecuteDialog(wf, 70, DefaultTheme())
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, dlg.viewParamFill(), "Will insert: beta")

	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, phaseActionMenu, dlg.phase)
	assert.Equal(t, "echo beta", dlg.renderedCommand)
}

func TestExecuteDialogParseErrorAllowsRetry(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}}",
		Args: []store.Arg{{
			Name:           "pod",
			Type:           "list",
			ListCmd:        "printf 'broken|onlytwo\nokay|three|value\n'",
			ListDelimiter:  "|",
			ListFieldIndex: 3,
		}},
	}

	dlg := NewExecuteDialog(wf, 70, DefaultTheme())
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, dlg.viewParamFill(), "selected row does not contain field 3")

	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyDown})
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, dlg.viewParamFill(), "Will insert: value")
}

func TestExecuteDialogListEmptyAfterSkipShowsMessage(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}}",
		Args: []store.Arg{{
			Name:           "pod",
			Type:           "list",
			ListCmd:        "printf 'HEADER\n'",
			ListSkipHeader: 1,
		}},
	}

	dlg := NewExecuteDialog(wf, 70, DefaultTheme())
	assert.Contains(t, dlg.viewParamFill(), "No selectable rows remain after skipping headers.")
}

func TestExecuteDialogListErrorShowsDetailOnDemand(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}}",
		Args: []store.Arg{{
			Name:    "pod",
			Type:    "list",
			ListCmd: "printf 'stderr detail\n' >&2; exit 7",
		}},
	}

	dlg := NewExecuteDialog(wf, 70, DefaultTheme())
	assert.Contains(t, dlg.viewParamFill(), "list command failed")
	assert.NotContains(t, dlg.viewParamFill(), "stderr detail")

	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Contains(t, dlg.viewParamFill(), "stderr detail")
}

func TestExecuteDialogListConfirmationCommandResult(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}} {{env:prod}}",
		Args: []store.Arg{{
			Name:           "pod",
			Type:           "list",
			ListCmd:        "printf 'NAME\nalpha|one\n'",
			ListDelimiter:  "|",
			ListFieldIndex: 1,
			ListSkipHeader: 1,
		}},
	}

	dlg := NewExecuteDialog(wf, 70, DefaultTheme())
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, dlg.viewParamFill(), "Will insert: alpha")

	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, phaseActionMenu, dlg.phase)
	assert.Equal(t, "echo alpha prod", dlg.renderedCommand)
}
