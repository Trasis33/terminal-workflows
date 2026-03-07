package picker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPickerStateFiltersAndRenumbersVisibleRows(t *testing.T) {
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

	m := Model{selected: &wf}
	initParamFill(&m)

	require.Len(t, m.paramListStates[0].visibleRows, 2)
	assert.NotContains(t, m.viewParamFill(), "NAME")

	updated, _ := m.updateParamFill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)
	view := m.viewParamFill()
	assert.Contains(t, view, "1. beta")
	assert.NotContains(t, view, "2. beta")
	assert.NotContains(t, view, "alpha")
}

func TestListPickerStateConfirmsExtractedValueAndQuits(t *testing.T) {
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

	m := Model{selected: &wf}
	initParamFill(&m)

	updated, _ := m.updateParamFill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.updateParamFill(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	assert.Contains(t, m.viewParamFill(), "Will insert: beta")

	updated, cmd := m.updateParamFill(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	require.NotNil(t, cmd)
	assert.Equal(t, "echo beta", m.Result)
}

func TestListPickerStateParseErrorAllowsRetry(t *testing.T) {
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

	m := Model{selected: &wf}
	initParamFill(&m)

	updated, _ := m.updateParamFill(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	assert.Contains(t, m.viewParamFill(), "selected row does not contain field 3")

	updated, _ = m.updateParamFill(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.updateParamFill(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	assert.Contains(t, m.viewParamFill(), "Will insert: value")
}

func TestListPickerStateShowsEmptyAfterSkipMessage(t *testing.T) {
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

	m := Model{selected: &wf}
	initParamFill(&m)

	assert.Contains(t, m.viewParamFill(), "No selectable rows remain after skipping headers.")
}

func TestListPickerStateShowsCommandFailureDetailOnDemand(t *testing.T) {
	wf := store.Workflow{
		Name:    "pods",
		Command: "echo {{pod}}",
		Args: []store.Arg{{
			Name:    "pod",
			Type:    "list",
			ListCmd: "printf 'stderr detail\n' >&2; exit 7",
		}},
	}

	m := Model{selected: &wf}
	initParamFill(&m)

	assert.Contains(t, m.viewParamFill(), "list command failed")
	assert.NotContains(t, m.viewParamFill(), "stderr detail")

	updated, _ := m.updateParamFill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	assert.Contains(t, m.viewParamFill(), "stderr detail")
}
