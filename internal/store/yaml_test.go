package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	w := &Workflow{
		Name:        "Deploy Staging",
		Command:     "kubectl apply -f deploy.yaml --namespace={{ns}}",
		Description: "Deploy to staging environment",
		Tags:        []string{"k8s", "deploy", "staging"},
		Args: []Arg{
			{Name: "ns", Default: "staging", Description: "Target namespace"},
		},
	}

	err := store.Save(w)
	require.NoError(t, err)

	got, err := store.Get("Deploy Staging")
	require.NoError(t, err)

	assert.Equal(t, w.Name, got.Name)
	assert.Equal(t, w.Command, got.Command)
	assert.Equal(t, w.Description, got.Description)
	assert.Equal(t, w.Tags, got.Tags)
	assert.Equal(t, w.Args, got.Args)
}

func TestMultilineCommandRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	multilineCmd := "echo \"step 1\"\necho \"step 2\"\necho \"step 3\""

	w := &Workflow{
		Name:        "Multi Step",
		Command:     multilineCmd,
		Description: "A multiline command",
	}

	err := store.Save(w)
	require.NoError(t, err)

	got, err := store.Get("Multi Step")
	require.NoError(t, err)

	assert.Equal(t, multilineCmd, got.Command, "multiline command should survive YAML round-trip")
}

func TestNorwayProblemRoundTrip(t *testing.T) {
	// The YAML "Norway Problem": bare words like no, yes, on, off, true, false
	// can be interpreted as booleans. With typed string fields in goccy/go-yaml,
	// these must survive round-trip as strings.
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	commands := []struct {
		name    string
		command string
	}{
		{"redis-no", "redis-cli CONFIG SET appendonly no"},
		{"bare-yes", "yes"},
		{"bare-no", "no"},
		{"bare-on", "on"},
		{"bare-off", "off"},
		{"bare-true", "true"},
		{"bare-false", "false"},
		{"bare-NO", "NO"},
		{"bare-Yes", "Yes"},
		{"bare-TRUE", "TRUE"},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workflow{
				Name:    tc.name,
				Command: tc.command,
			}

			err := store.Save(w)
			require.NoError(t, err)

			got, err := store.Get(tc.name)
			require.NoError(t, err)

			assert.Equal(t, tc.command, got.Command,
				"command %q should not be corrupted by YAML boolean coercion", tc.command)
		})
	}
}

func TestNestedQuotesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	commands := []struct {
		name string
		cmd  string
	}{
		{
			"docker-psql",
			`docker exec -it db psql -c "SELECT * FROM users"`,
		},
		{
			"nested-single",
			`echo "it's a \"test\""`,
		},
		{
			"pipe-chain",
			`cat foo.txt | grep "pattern" | awk '{print $2}' > output.txt`,
		},
		{
			"subshell",
			`echo "today is $(date +%Y-%m-%d)"`,
		},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workflow{
				Name:    tc.name,
				Command: tc.cmd,
			}

			err := store.Save(w)
			require.NoError(t, err)

			got, err := store.Get(tc.name)
			require.NoError(t, err)

			assert.Equal(t, tc.cmd, got.Command,
				"command with nested quotes should survive round-trip")
		})
	}
}

func TestListWorkflows(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	// Create flat workflows
	w1 := &Workflow{Name: "alpha", Command: "echo alpha", Tags: []string{"test"}}
	w2 := &Workflow{Name: "beta", Command: "echo beta", Tags: []string{"test"}}

	require.NoError(t, store.Save(w1))
	require.NoError(t, store.Save(w2))

	// List flat-only directory
	workflows, err := store.List()
	require.NoError(t, err)
	assert.Len(t, workflows, 2, "should find flat workflows")

	// Create nested workflow in a subdirectory
	w3 := &Workflow{Name: "deploy", Command: "kubectl apply", Tags: []string{"k8s"}}
	nestedDir := filepath.Join(dir, "infra")
	nestedStore := NewYAMLStore(nestedDir)
	require.NoError(t, nestedStore.Save(w3))

	// List should now walk subdirectories and find all 3
	allWorkflows, err := store.List()
	require.NoError(t, err)
	assert.Len(t, allWorkflows, 3, "should find flat and nested workflows")

	// List from subdirectory should find only the nested workflow
	subWorkflows, err := nestedStore.List()
	require.NoError(t, err)
	assert.Len(t, subWorkflows, 1, "should find only workflows in subdirectory")
	assert.Equal(t, "deploy", subWorkflows[0].Name)
}

func TestDeleteWorkflow(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	w := &Workflow{Name: "to-delete", Command: "echo delete me"}
	require.NoError(t, store.Save(w))

	// Verify file exists
	_, err := store.Get("to-delete")
	require.NoError(t, err)

	// Delete
	err = store.Delete("to-delete")
	require.NoError(t, err)

	// Verify file is gone
	_, err = store.Get("to-delete")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Verify file doesn't exist on disk
	fpath := filepath.Join(dir, "to-delete.yaml")
	_, err = os.Stat(fpath)
	assert.True(t, os.IsNotExist(err))
}

func TestGetNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	_, err := store.Get("does-not-exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	err := store.Delete("does-not-exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFilenameSlugification(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Deploy Staging", "deploy-staging.yaml"},
		{"docker build & push", "docker-build-push.yaml"},
		{"hello world!", "hello-world.yaml"},
		{"UPPERCASE", "uppercase.yaml"},
		{"  spaces  ", "spaces.yaml"},
		{"special@#$chars", "special-chars.yaml"},
		{"already-slug", "already-slug.yaml"},
		{"multiple---dashes", "multiple-dashes.yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workflow{Name: tc.name}
			assert.Equal(t, tc.expected, w.Filename())
		})
	}
}

func TestSaveWithPathPrefix(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	w := &Workflow{
		Name:    "deploy",
		Command: "kubectl apply -f deploy.yaml",
	}

	// Save via the store's Save method (flat)
	require.NoError(t, store.Save(w))

	// Verify the file exists at the expected path
	_, err := os.Stat(filepath.Join(dir, "deploy.yaml"))
	require.NoError(t, err)

	// Test nested directory save by creating subdir manually
	infraDir := filepath.Join(dir, "infra")
	infraStore := NewYAMLStore(infraDir)
	require.NoError(t, infraStore.Save(w))

	_, err = os.Stat(filepath.Join(infraDir, "deploy.yaml"))
	require.NoError(t, err)
}

func TestListEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	store := NewYAMLStore(dir)

	workflows, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, workflows)
}

func TestListNonExistentDirectory(t *testing.T) {
	store := NewYAMLStore("/tmp/nonexistent-wf-test-dir-12345")

	workflows, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, workflows)
}
