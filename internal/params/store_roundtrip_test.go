package params

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListArgMetadataRoundTripAndYAMLKeys(t *testing.T) {
	dir := t.TempDir()
	yamlStore := store.NewYAMLStore(dir)

	w := &store.Workflow{
		Name:    "List Param",
		Command: "echo {{row}}",
		Args: []store.Arg{{
			Name:           "row",
			Type:           "list",
			ListCmd:        "kubectl get pods",
			ListDelimiter:  "|",
			ListFieldIndex: 2,
			ListSkipHeader: 1,
		}},
	}

	require.NoError(t, yamlStore.Save(w))

	got, err := yamlStore.Get("List Param")
	require.NoError(t, err)
	require.Len(t, got.Args, 1)
	assert.Equal(t, w.Args[0], got.Args[0])

	data, err := os.ReadFile(filepath.Join(dir, "list-param.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "list_cmd:")
	assert.Contains(t, content, "list_delimiter:")
	assert.Contains(t, content, "list_field_index:")
	assert.Contains(t, content, "list_skip_header:")
	assert.True(t, strings.Contains(content, "type: list"))
}
