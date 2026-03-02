package main

import (
	"testing"

	"github.com/fredriklanga/wf/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestInitCommentIncludesZshManageFallback(t *testing.T) {
	comment := initComment("zsh", shell.DefaultKey)
	require.Contains(t, comment, "Fallback manage command (no Alt/Meta needed): wfm")
}

func TestInitCommentOmitsFallbackForNonZsh(t *testing.T) {
	comment := initComment("bash", shell.DefaultKey)
	require.NotContains(t, comment, "Fallback manage command")
}
