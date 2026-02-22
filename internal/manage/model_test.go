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

	v := m.View()
	assert.Contains(t, v, "wf manage")
	assert.Contains(t, v, "0 workflows")
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
