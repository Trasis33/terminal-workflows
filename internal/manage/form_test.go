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
	assert.Empty(t, m.name)
	assert.Empty(t, m.description)
	assert.Empty(t, m.command)
	assert.Empty(t, m.tagInput)
	assert.Empty(t, m.folder)
	assert.Empty(t, m.originalName)
	assert.NotNil(t, m.form)
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
	assert.Equal(t, "deploy-app", m.name)
	assert.Equal(t, "infra", m.folder)
	assert.Equal(t, "kubectl apply -f deploy.yaml", m.command)
	assert.Equal(t, "Deploy the app to k8s", m.description)
	assert.Equal(t, "k8s, deploy", m.tagInput)
	assert.NotNil(t, m.form)
}

func TestNewFormModelEditNoFolder(t *testing.T) {
	s := &mockStore{}
	wf := &store.Workflow{
		Name:    "simple-wf",
		Command: "echo hello",
	}

	m := NewFormModel("edit", wf, s, nil, nil, DefaultTheme())

	assert.Equal(t, "simple-wf", m.name)
	assert.Empty(t, m.folder)
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
	m.name = "my-wf"
	m.command = "echo hello"
	m.description = "A test workflow"
	m.tagInput = "test, demo"
	m.folder = "examples"

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
	m.name = "simple"
	m.command = "ls -la"

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
	m.name = "new-name"
	m.command = "echo new"

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
	assert.NotNil(t, cmd, "Init should return a command from huh form")
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
