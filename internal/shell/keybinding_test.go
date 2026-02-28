package shell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKeyValid(t *testing.T) {
	tests := []string{"ctrl+g", "Ctrl+G", "alt+f"}
	for _, tt := range tests {
		kb, err := ParseKey(tt)
		require.NoError(t, err)
		require.NotEmpty(t, kb.Modifier)
		require.NotEmpty(t, kb.Letter)
	}
}

func TestParseKeyInvalid(t *testing.T) {
	tests := []string{"ctrl+1", "shift+g", "g", "ctrl+gg"}
	for _, tt := range tests {
		_, err := ParseKey(tt)
		require.Error(t, err)
	}
}

func TestValidateBlocked(t *testing.T) {
	blocked := []string{"ctrl+c", "ctrl+d", "ctrl+z"}
	for _, key := range blocked {
		kb, err := ParseKey(key)
		require.NoError(t, err)
		require.Error(t, kb.Validate())
	}

	ok, err := ParseKey("ctrl+g")
	require.NoError(t, err)
	require.NoError(t, ok.Validate())

	ok, err = ParseKey("ctrl+o")
	require.NoError(t, err)
	require.NoError(t, ok.Validate())
}

func TestShellFormatConversions(t *testing.T) {
	ctrl, _ := ParseKey("ctrl+g")
	alt, _ := ParseKey("alt+f")

	require.Equal(t, `\C-g`, ctrl.ForZsh())
	require.Equal(t, `\ef`, alt.ForZsh())
	require.Equal(t, `\cg`, ctrl.ForFish())
	require.Equal(t, "Ctrl+G", ctrl.ForPowerShell())
}
