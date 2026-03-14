package shell

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZshTemplateIncludesManageFallbackCommand(t *testing.T) {
	var out bytes.Buffer
	err := ZshTemplate.Execute(&out, TemplateData{
		Key:                 `\C-g`,
		ManageKey:           `\em`,
		ManageFallbackUsage: "wfm",
		Comment:             "# test",
	})
	require.NoError(t, err)

	rendered := out.String()
	require.Contains(t, rendered, "wfm() {")
	require.Contains(t, rendered, "print -z -- \"$output\"")
	require.Contains(t, rendered, "Usage: wfm")
	require.Equal(t, 2, strings.Count(rendered, "result_file=$(mktemp)"))
	require.Equal(t, 2, strings.Count(rendered, "wf manage --result-file \"$result_file\""))
	require.Equal(t, 2, strings.Count(rendered, "rm -f -- \"$result_file\""))
	require.Contains(t, rendered, "zle reset-prompt")
	require.Contains(t, rendered, "return $ret")
	require.NotContains(t, rendered, "output=$(wf manage)")
}
