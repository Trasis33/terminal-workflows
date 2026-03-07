package manage

import (
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormModelCreateMode(t *testing.T) {
	s := &mockStore{}
	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())

	assert.Equal(t, "create", m.mode)
	assert.Empty(t, m.vals.name)
	assert.Empty(t, m.vals.description)
	assert.Empty(t, m.vals.command)
	assert.Empty(t, m.vals.tagInput)
	assert.Empty(t, m.vals.folder)
	assert.Empty(t, m.originalName)
	assert.Equal(t, formFieldIndex(0), m.focused)
	assert.Equal(t, 0, m.paramEditor.ParamCount())
}

func TestNewFormModelEditMode(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:        "infra/deploy-app",
		Command:     "kubectl apply -f deploy.yaml",
		Description: "Deploy the app to k8s",
		Tags:        []string{"k8s", "deploy"},
	}

	m := NewFormModel("edit", wf, s, []string{"k8s", "deploy", "docker"}, []string{"infra"}, DefaultTheme())

	assert.Equal(t, "edit", m.mode)
	assert.Equal(t, "infra/deploy-app", m.originalName)
	assert.Equal(t, "deploy-app", m.vals.name)
	assert.Equal(t, "infra", m.vals.folder)
	assert.Equal(t, "kubectl apply -f deploy.yaml", m.vals.command)
	assert.Equal(t, "Deploy the app to k8s", m.vals.description)
	assert.Equal(t, "k8s, deploy", m.vals.tagInput)
	assert.Equal(t, 0, m.paramEditor.ParamCount())
}

func TestNewFormModelEditModeHydratesListArg(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:    "infra/pick-pod",
		Command: "kubectl logs {{pod}}",
		Args: []store.Arg{{
			Name:           "pod",
			Type:           "list",
			ListCmd:        "kubectl get pods --no-headers",
			ListDelimiter:  " ",
			ListFieldIndex: 1,
			ListSkipHeader: 0,
		}},
	}

	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())
	require.Len(t, m.paramEditor.params, 1)
	assert.Equal(t, "list", m.paramEditor.params[0].paramType)
	assert.Equal(t, "kubectl get pods --no-headers", m.paramEditor.params[0].listCmd)
	assert.Equal(t, " ", m.paramEditor.params[0].listDelimiter)
	assert.Equal(t, 1, m.paramEditor.params[0].listFieldIndex)
	assert.Zero(t, m.paramEditor.params[0].listSkipHeader)
}

func TestNewFormModelEditNoFolder(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:    "simple-wf",
		Command: "echo hello",
	}

	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())

	assert.Equal(t, "simple-wf", m.vals.name)
	assert.Empty(t, m.vals.folder)
}

func TestFormModelViewCreate(t *testing.T) {
	s := &mockStore{}
	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())
	m.SetDimensions(80, 24)

	v := m.View()
	assert.Contains(t, v, "Create Workflow")
	assert.Contains(t, v, "esc cancel")
}

func TestFormModelViewEdit(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{Name: "test", Command: "echo"}
	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())
	m.SetDimensions(80, 24)

	v := m.View()
	assert.Contains(t, v, "Edit Workflow")
}

func TestFormModelViewShowsError(t *testing.T) {
	s := &mockStore{}
	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())
	m.SetDimensions(80, 24)
	m.err = assert.AnError

	v := m.View()
	assert.Contains(t, v, "Error:")
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"   ", nil},
		{"docker", []string{"docker"}},
		{"docker, deploy, infra", []string{"docker", "deploy", "infra"}},
		{" docker , deploy , ", []string{"docker", "deploy"}},
		{",,,", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTags(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormModelSaveWorkflow(t *testing.T) {
	saved := make([]store.Workflow, 0)
	s := &savingMockStore{saved: &saved}

	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())
	m.vals.name = "my-wf"
	m.vals.command = "echo hello"
	m.vals.description = "A test workflow"
	m.vals.tagInput = "test, demo"
	m.vals.folder = "examples"

	cmd := m.saveWorkflow()
	require.NotNil(t, cmd)

	msg := cmd()
	savedMsg, ok := msg.(workflowSavedMsg)
	require.True(t, ok, "expected workflowSavedMsg, got %T", msg)

	assert.Equal(t, "examples/my-wf", savedMsg.workflow.Name)
	assert.Equal(t, "echo hello", savedMsg.workflow.Command)
	assert.Equal(t, "A test workflow", savedMsg.workflow.Description)
	assert.Equal(t, []string{"test", "demo"}, savedMsg.workflow.Tags)
}

func TestFormModelSaveWorkflowNoFolder(t *testing.T) {
	saved := make([]store.Workflow, 0)
	s := &savingMockStore{saved: &saved}

	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())
	m.vals.name = "simple"
	m.vals.command = "ls -la"

	cmd := m.saveWorkflow()
	msg := cmd()
	savedMsg := msg.(workflowSavedMsg)

	assert.Equal(t, "simple", savedMsg.workflow.Name)
}

func TestFormModelSaveWorkflowEditRename(t *testing.T) {
	deleted := make([]string, 0)
	s := &renameMockStore{deleted: &deleted}

	m := NewFormModel("edit", &store.Workflow{
		Name:    "old-name",
		Command: "echo old",
	}, s, nil, nil, DefaultTheme())

	// Change the name.
	m.vals.name = "new-name"
	m.vals.command = "echo new"

	cmd := m.saveWorkflow()
	msg := cmd()
	savedMsg := msg.(workflowSavedMsg)

	assert.Equal(t, "new-name", savedMsg.workflow.Name)
	assert.Contains(t, *s.deleted, "old-name")
}

func TestFormModelInitReturnsCmd(t *testing.T) {
	s := &mockStore{}
	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())

	cmd := m.Init()
	assert.NotNil(t, cmd, "Init should return a blink command for the focused input")
}

func TestFormModelSaveWorkflowWithArgs(t *testing.T) {
	saved := make([]store.Workflow, 0)
	s := &savingMockStore{saved: &saved}

	wf := &store.Workflow{
		Name:    "test-wf",
		Command: "deploy {{.env}}",
		Args: []store.Arg{
			{Name: "env", Type: "enum", Options: []string{"staging", "prod"}},
		},
	}
	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())

	// Verify param editor loaded the args.
	assert.Equal(t, 1, m.paramEditor.ParamCount())

	cmd := m.saveWorkflow()
	msg := cmd()
	savedMsg := msg.(workflowSavedMsg)

	require.Len(t, savedMsg.workflow.Args, 1)
	assert.Equal(t, "env", savedMsg.workflow.Args[0].Name)
	assert.Equal(t, "enum", savedMsg.workflow.Args[0].Type)
	assert.Equal(t, []string{"staging", "prod"}, savedMsg.workflow.Args[0].Options)
}

func TestFormModelSaveWorkflowWithListArgs(t *testing.T) {
	saved := make([]store.Workflow, 0)
	s := &savingMockStore{saved: &saved}

	wf := &store.Workflow{
		Name:    "test-wf",
		Command: "deploy {{target}}",
		Args: []store.Arg{{
			Name:           "target",
			Type:           "list",
			ListCmd:        "printf 'NAME,ID\\napi,42'",
			ListDelimiter:  ",",
			ListFieldIndex: 2,
			ListSkipHeader: 1,
		}},
	}
	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())

	m.paramEditor.params[0].listCmd = "printf 'NAME,ID\\nweb,99'"
	m.paramEditor.params[0].listCmdInput.SetValue("printf 'NAME,ID\\nweb,99'")
	m.paramEditor.params[0].listDelimiter = "|"
	m.paramEditor.params[0].listDelimiterInput.SetValue("|")
	m.paramEditor.params[0].listFieldIndex = 1
	m.paramEditor.params[0].listFieldIndexInput.SetValue("1")
	m.paramEditor.params[0].listSkipHeader = 2
	m.paramEditor.params[0].listSkipHeaderInput.SetValue("2")

	cmd := m.saveWorkflow()
	msg := cmd()
	savedMsg := msg.(workflowSavedMsg)

	require.Len(t, savedMsg.workflow.Args, 1)
	assert.Equal(t, "list", savedMsg.workflow.Args[0].Type)
	assert.Equal(t, "printf 'NAME,ID\\nweb,99'", savedMsg.workflow.Args[0].ListCmd)
	assert.Equal(t, "|", savedMsg.workflow.Args[0].ListDelimiter)
	assert.Equal(t, 1, savedMsg.workflow.Args[0].ListFieldIndex)
	assert.Equal(t, 2, savedMsg.workflow.Args[0].ListSkipHeader)
	assert.Nil(t, savedMsg.workflow.Args[0].Options)
	assert.Empty(t, savedMsg.workflow.Args[0].DynamicCmd)
}

func TestFormModelSaveWorkflowWithListArgsWholeRowFallback(t *testing.T) {
	saved := make([]store.Workflow, 0)
	s := &savingMockStore{saved: &saved}

	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())
	m.vals.name = "pick-row"
	m.vals.command = "echo {{row}}"
	m.paramEditor = NewParamEditor([]store.Arg{{
		Name:           "row",
		Type:           "list",
		ListCmd:        "printf 'A|B'",
		ListDelimiter:  "|",
		ListFieldIndex: 0,
		ListSkipHeader: 1,
	}}, DefaultTheme(), 80)

	cmd := m.saveWorkflow()
	msg := cmd()
	savedMsg := msg.(workflowSavedMsg)

	require.Len(t, savedMsg.workflow.Args, 1)
	assert.Zero(t, savedMsg.workflow.Args[0].ListFieldIndex)
	assert.Equal(t, "|", savedMsg.workflow.Args[0].ListDelimiter)
	assert.Equal(t, 1, savedMsg.workflow.Args[0].ListSkipHeader)
}

func TestFormModelValidation(t *testing.T) {
	s := &mockStore{}
	m := NewFormModel("create", nil, s, nil, nil, DefaultTheme())

	// Empty name should fail.
	err := m.validate()
	assert.ErrorIs(t, err, errNameRequired)

	// Name with slash should fail.
	m.vals.name = "bad/name"
	err = m.validate()
	assert.ErrorIs(t, err, errNameNoSlash)

	// Empty command should fail.
	m.vals.name = "good-name"
	err = m.validate()
	assert.ErrorIs(t, err, errCommandRequired)

	// Valid form should pass.
	m.vals.command = "echo hello"
	err = m.validate()
	assert.NoError(t, err)
}

// savingMockStore records saved workflows.
type savingMockStore struct {
	saved *[]store.Workflow
}

func (s *savingMockStore) List() ([]store.Workflow, error) { return nil, nil }
func (s *savingMockStore) Get(name string) (*store.Workflow, error) {
	return nil, nil
}
func (s *savingMockStore) Save(w *store.Workflow) error {
	*s.saved = append(*s.saved, *w)
	return nil
}
func (s *savingMockStore) Delete(name string) error { return nil }

// renameMockStore tracks deletions for rename testing.
type renameMockStore struct {
	deleted *[]string
}

func (s *renameMockStore) List() ([]store.Workflow, error) { return nil, nil }
func (s *renameMockStore) Get(name string) (*store.Workflow, error) {
	return nil, nil
}
func (s *renameMockStore) Save(w *store.Workflow) error { return nil }
func (s *renameMockStore) Delete(name string) error {
	*s.deleted = append(*s.deleted, name)
	return nil
}
