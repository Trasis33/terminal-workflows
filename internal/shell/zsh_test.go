package shell

import (
	"bytes"
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
}
