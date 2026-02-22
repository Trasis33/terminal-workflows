package manage

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
)

// mockStore implements store.Store for testing.
type mockStore struct {
	workflows []store.Workflow
}

func (m *mockStore) List() ([]store.Workflow, error) { return m.workflows, nil }
func (m *mockStore) Get(name string) (*store.Workflow, error) {
	for i, w := range m.workflows {
		if w.Name == name {
			return &m.workflows[i], nil
		}
	}
	return nil, nil
}
func (m *mockStore) Save(w *store.Workflow) error { return nil }
func (m *mockStore) Delete(name string) error     { return nil }

func TestNewModel(t *testing.T) {
	s := &mockStore{
		workflows: []store.Workflow{
			{Name: "test-wf", Command: "echo hello"},
		},
	}

	m := New(s, s.workflows, DefaultTheme(), "/tmp/test-config")

	assert.Equal(t, viewBrowse, m.state)
	assert.Equal(t, "49", m.theme.Colors.Primary)
	assert.Len(t, m.workflows, 1)
	assert.Equal(t, "test-wf", m.workflows[0].Name)
	assert.Equal(t, "/tmp/test-config", m.configDir)
}

func TestModelViewRenders(t *testing.T) {
	s := &mockStore{}
	m := New(s, nil, DefaultTheme(), "")
	m.width = 80
	m.height = 24
	m.browse.SetDimensions(80, 24)

	v := m.View()
	// Browse view shows "No workflows found" when empty and keybinding hints.
	assert.Contains(t, v, "No workflows found")
	assert.Contains(t, v, "q quit")
}

func TestModelWindowSizeMsg(t *testing.T) {
	s := &mockStore{}
	m := New(s, nil, DefaultTheme(), "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)

	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
}

func TestViewStateTransitions(t *testing.T) {
	s := &mockStore{}
	m := New(s, nil, DefaultTheme(), "")

	// switchToCreateMsg
	updated, _ := m.Update(switchToCreateMsg{})
	model := updated.(Model)
	assert.Equal(t, viewCreate, model.state)
	assert.Equal(t, viewBrowse, model.prevState)

	// switchToEditMsg
	updated, _ = model.Update(switchToEditMsg{workflow: store.Workflow{Name: "test"}})
	model = updated.(Model)
	assert.Equal(t, viewEdit, model.state)

	// switchToBrowseMsg
	updated, _ = model.Update(switchToBrowseMsg{})
	model = updated.(Model)
	assert.Equal(t, viewBrowse, model.state)

	// switchToSettingsMsg
	updated, _ = model.Update(switchToSettingsMsg{})
	model = updated.(Model)
	assert.Equal(t, viewSettings, model.state)
}

func TestDialogOverlay(t *testing.T) {
	s := &mockStore{}
	m := New(s, nil, DefaultTheme(), "")
	m.width = 80
	m.height = 24

	m.dialog = &dialogState{
		title:   "Confirm Delete",
		message: "Are you sure?",
	}

	v := m.View()
	assert.Contains(t, v, "Confirm Delete")
}

func TestWorkflowsLoadedMsg(t *testing.T) {
	s := &mockStore{}
	m := New(s, nil, DefaultTheme(), "")

	wfs := []store.Workflow{
		{Name: "wf1", Command: "echo 1"},
		{Name: "wf2", Command: "echo 2"},
	}

	updated, _ := m.Update(workflowsLoadedMsg{workflows: wfs})
	model := updated.(Model)
	assert.Len(t, model.workflows, 2)
}

func TestRefreshWorkflowsMsg(t *testing.T) {
	s := &mockStore{
		workflows: []store.Workflow{{Name: "loaded"}},
	}
	m := New(s, nil, DefaultTheme(), "")

	_, cmd := m.Update(refreshWorkflowsMsg{})
	assert.NotNil(t, cmd, "refreshWorkflowsMsg should return a command")
}

func TestBrowseModelWiring(t *testing.T) {
	wfs := []store.Workflow{
		{Name: "infra/deploy", Command: "kubectl apply", Tags: []string{"k8s"}},
		{Name: "dev/test", Command: "go test", Tags: []string{"go", "test"}},
	}
	s := &mockStore{workflows: wfs}
	m := New(s, wfs, DefaultTheme(), "/tmp/cfg")
	m.width = 100
	m.height = 30
	m.browse.SetDimensions(100, 30)

	v := m.View()
	// Should show workflow names from browse model.
	assert.Contains(t, v, "infra/deploy")
	assert.Contains(t, v, "dev/test")
}

func TestWorkflowsLoadedUpdatessBrowse(t *testing.T) {
	s := &mockStore{}
	m := New(s, nil, DefaultTheme(), "")
	m.width = 100
	m.height = 30
	m.browse.SetDimensions(100, 30)

	// Initially empty.
	v := m.View()
	assert.Contains(t, v, "No workflows found")

	// Load workflows — should propagate to browse model.
	wfs := []store.Workflow{
		{Name: "hello", Command: "echo hi", Tags: []string{"greeting"}},
	}
	updated, _ := m.Update(workflowsLoadedMsg{workflows: wfs})
	model := updated.(Model)
	model.browse.SetDimensions(100, 30)

	v = model.View()
	assert.Contains(t, v, "hello")
}

func TestExtractFolders(t *testing.T) {
	wfs := []store.Workflow{
		{Name: "infra/deploy/app"},
		{Name: "infra/monitor"},
		{Name: "simple"},
	}
	folders := extractFolders(wfs)
	assert.Contains(t, folders, "infra")
	assert.Contains(t, folders, "infra/deploy")
	assert.Len(t, folders, 2) // "infra" and "infra/deploy"
}

func TestExtractTags(t *testing.T) {
	wfs := []store.Workflow{
		{Name: "a", Tags: []string{"go", "test"}},
		{Name: "b", Tags: []string{"go", "deploy"}},
	}
	tags := extractTags(wfs)
	assert.Equal(t, []string{"deploy", "go", "test"}, tags)
}
