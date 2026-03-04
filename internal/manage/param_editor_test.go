package manage

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParamEditor_Rename(t *testing.T) {
	args := []store.Arg{
		{Name: "old_name", Type: "text"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Expand the param.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.params[0].expanded)

	// Tab to start editing name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.True(t, m.editing)
	assert.Equal(t, subFieldName, m.params[0].focusedField)

	// Clear existing text and type new name.
	m.params[0].nameInput.SetValue("new_name")
	m.params[0].name = "new_name"

	// Commit with enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.editing)

	// Verify ToArgs returns the new name.
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "new_name", resultArgs[0].Name)
}

func TestParamEditor_RenameDuplicate(t *testing.T) {
	args := []store.Arg{
		{Name: "name1", Type: "text"},
		{Name: "name2", Type: "text"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Move cursor to second param and expand it.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.params[1].expanded)

	// Tab to start editing name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.True(t, m.editing)

	// Simulate typing "name1" — duplicate of the first param.
	m.params[1].nameInput.SetValue("name1")
	// Trigger duplicate check.
	m.params[1].nameErr = m.checkDuplicateName(1, "name1")

	// Verify duplicate error is set.
	assert.Equal(t, "Parameter name already exists", m.params[1].nameErr)
}

func TestParamEditor_TypeChange(t *testing.T) {
	args := []store.Arg{
		{Name: "param1", Type: "text"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Expand param.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Tab to name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to type selector.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldType, m.params[0].focusedField)

	// Press right to change from "text" to "enum".
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, "enum", m.params[0].paramType)

	// Commit and verify.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "enum", resultArgs[0].Type)
}

func TestParamEditor_TypeChangeSoftStaging(t *testing.T) {
	args := []store.Arg{
		{Name: "env", Type: "enum", Options: []string{"staging", "prod"}},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)
	assert.Equal(t, []string{"staging", "prod"}, m.params[0].options)

	// Expand param.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Tab to name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to type selector.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Change type to text.
	// enum is at index 1, text is at index 0, so press left.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, "text", m.params[0].paramType)
	// Options should STILL be preserved (soft staging).
	assert.Equal(t, []string{"staging", "prod"}, m.params[0].options)

	// Change back to enum.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, "enum", m.params[0].paramType)
	// Options should still be there.
	assert.Equal(t, []string{"staging", "prod"}, m.params[0].options)

	// Verify ToArgs with enum type includes options.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // exit editing
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "enum", resultArgs[0].Type)
	assert.Equal(t, []string{"staging", "prod"}, resultArgs[0].Options)
}

func TestParamEditor_EditDefault(t *testing.T) {
	args := []store.Arg{
		{Name: "param1", Type: "text"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Expand param.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Tab to name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to type selector.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to default field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldDefault, m.params[0].focusedField)
	assert.True(t, m.editing)

	// Set default value directly on the input widget.
	m.params[0].defaultInput.SetValue("myval")
	m.params[0].defaultVal = "myval"

	// Commit with enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify ToArgs returns the default.
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "myval", resultArgs[0].Default)
}

func TestParamEditor_EditEnumOptions(t *testing.T) {
	args := []store.Arg{
		{Name: "env", Type: "enum"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Expand param.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Tab to name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to type selector.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to default field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to options field (visible because type is enum).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldOptions, m.params[0].focusedField)
	assert.True(t, m.editing)

	// Set options with * default marker.
	m.params[0].optionsInput.SetValue("opt1, opt2, *opt3")
	m.params[0].options = parseEnumOptions("opt1, opt2, *opt3")

	// Commit with enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify ToArgs returns correct options (with * stripped).
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, []string{"opt1", "opt2", "opt3"}, resultArgs[0].Options)
}

func TestParamEditor_EditDynamicCmd(t *testing.T) {
	args := []store.Arg{
		{Name: "branch", Type: "dynamic"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Expand param.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Tab to name field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to type selector.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to default field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab to dynamic command field (visible because type is dynamic).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldDynamicCmd, m.params[0].focusedField)
	assert.True(t, m.editing)

	// Set dynamic command directly.
	m.params[0].dynamicCmdInput.SetValue("git branch")
	m.params[0].dynamicCmd = "git branch"

	// Commit with enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify ToArgs.
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "git branch", resultArgs[0].DynamicCmd)
}

func TestParamEditor_AddAndEdit(t *testing.T) {
	m := NewParamEditor(nil, DefaultTheme(), 80)
	assert.Equal(t, 0, m.ParamCount())
	assert.True(t, m.onAddButton)

	// Add a new param (enter on add button or ctrl+n).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, 1, m.ParamCount())
	assert.True(t, m.params[0].expanded)
	assert.True(t, m.editing)

	// Rename it.
	m.params[0].nameInput.SetValue("my_param")
	m.params[0].name = "my_param"

	// Tab to type selector.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldType, m.params[0].focusedField)

	// Change type to enum.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, "enum", m.params[0].paramType)

	// Tab to default.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldDefault, m.params[0].focusedField)

	// Tab to options field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, subFieldOptions, m.params[0].focusedField)

	// Set options.
	m.params[0].optionsInput.SetValue("dev, staging, prod")
	m.params[0].options = parseEnumOptions("dev, staging, prod")

	// Commit.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify full round-trip.
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "my_param", resultArgs[0].Name)
	assert.Equal(t, "enum", resultArgs[0].Type)
	assert.Equal(t, []string{"dev", "staging", "prod"}, resultArgs[0].Options)
}

func TestParamEditor_ToArgsStripsIncompatibleMetadata(t *testing.T) {
	// Create an enum param with options, then change type to text.
	// ToArgs should NOT include options for text type.
	args := []store.Arg{
		{Name: "env", Type: "enum", Options: []string{"dev", "prod"}},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	// Verify options are loaded.
	assert.Equal(t, []string{"dev", "prod"}, m.params[0].options)

	// Change type to text (soft staging keeps options in memory).
	m.params[0].paramType = "text"

	// ToArgs should strip options.
	resultArgs := m.ToArgs()
	require.Len(t, resultArgs, 1)
	assert.Equal(t, "text", resultArgs[0].Type)
	assert.Nil(t, resultArgs[0].Options)
	assert.Empty(t, resultArgs[0].DynamicCmd)
}

func TestParamEditor_ValidateForSave(t *testing.T) {
	args := []store.Arg{
		{Name: "p1", Type: "text", Options: []string{"a"}}, // text with options = warning
		{Name: "p2", Type: "enum"},                         // enum with no options = warning
		{Name: "p3", Type: "dynamic"},                      // dynamic with no cmd = warning
		{Name: "p4", Type: "enum", Options: []string{"x"}}, // valid enum
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	warnings := m.ValidateForSave()
	assert.Len(t, warnings, 3)
	assert.Contains(t, warnings[0], "p1")
	assert.Contains(t, warnings[1], "p2")
	assert.Contains(t, warnings[2], "p3")
}

func TestParamEditor_HasDuplicateNames(t *testing.T) {
	args := []store.Arg{
		{Name: "same", Type: "text"},
		{Name: "same", Type: "text"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	assert.True(t, m.HasDuplicateNames())
}

func TestParamEditor_NoDuplicateNames(t *testing.T) {
	args := []store.Arg{
		{Name: "alpha", Type: "text"},
		{Name: "beta", Type: "text"},
	}
	m := NewParamEditor(args, DefaultTheme(), 80)

	assert.False(t, m.HasDuplicateNames())
}

func TestParamEditor_ParseEnumOptions(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"   ", nil},
		{"opt1", []string{"opt1"}},
		{"opt1, opt2, opt3", []string{"opt1", "opt2", "opt3"}},
		{"*opt1, opt2", []string{"opt1", "opt2"}},
		{" opt1 , *opt2 , ", []string{"opt1", "opt2"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseEnumOptions(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParamEditor_ParseEnumOptionsWithDefault(t *testing.T) {
	opts, def := parseEnumOptionsWithDefault("opt1, *opt2, opt3")
	assert.Equal(t, []string{"opt1", "opt2", "opt3"}, opts)
	assert.Equal(t, "opt2", def)

	opts, def = parseEnumOptionsWithDefault("opt1, opt2")
	assert.Equal(t, []string{"opt1", "opt2"}, opts)
	assert.Equal(t, "", def)

	opts, def = parseEnumOptionsWithDefault("")
	assert.Nil(t, opts)
	assert.Equal(t, "", def)
}

func TestFormModel_CommandTemplateRename(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:    "test-wf",
		Command: "deploy {{env}} --force={{env:true}}",
		Args:    []store.Arg{{Name: "env", Type: "text", Default: "prod"}},
	}

	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())
	m.SetDimensions(80, 24)

	// Simulate paramRenamedMsg.
	m.updateCommandTemplateOnRename("env", "environment")

	assert.Equal(t, "deploy {{environment}} --force={{environment:true}}", m.cmdInput.Value())
	assert.Equal(t, "deploy {{environment}} --force={{environment:true}}", m.vals.command)
}

func TestFormModel_CommandTemplateRenameEnumSyntax(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:    "test-wf",
		Command: "deploy {{env|staging|prod}}",
		Args:    []store.Arg{{Name: "env", Type: "enum", Options: []string{"staging", "prod"}}},
	}

	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())
	m.SetDimensions(80, 24)

	m.updateCommandTemplateOnRename("env", "target")

	assert.Equal(t, "deploy {{target|staging|prod}}", m.cmdInput.Value())
}

func TestFormModel_CommandTemplateRenameDynamicSyntax(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:    "test-wf",
		Command: "checkout {{branch!git branch --list}}",
		Args:    []store.Arg{{Name: "branch", Type: "dynamic", DynamicCmd: "git branch --list"}},
	}

	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())
	m.SetDimensions(80, 24)

	m.updateCommandTemplateOnRename("branch", "br")

	assert.Equal(t, "checkout {{br!git branch --list}}", m.cmdInput.Value())
}

func TestParamEditor_VisibleSubFields(t *testing.T) {
	tests := []struct {
		paramType string
		expected  []int
	}{
		{"text", []int{subFieldName, subFieldType, subFieldDefault}},
		{"enum", []int{subFieldName, subFieldType, subFieldDefault, subFieldOptions}},
		{"dynamic", []int{subFieldName, subFieldType, subFieldDefault, subFieldDynamicCmd}},
		{"list", []int{subFieldName, subFieldType, subFieldDefault}},
	}

	for _, tt := range tests {
		t.Run(tt.paramType, func(t *testing.T) {
			m := NewParamEditor([]store.Arg{{Name: "p", Type: tt.paramType}}, DefaultTheme(), 80)
			m.cursor = 0
			assert.Equal(t, tt.expected, m.visibleSubFields())
		})
	}
}
