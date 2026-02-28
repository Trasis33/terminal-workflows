package highlight

import (
	"regexp"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/stretchr/testify/require"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func TestShellReturnsStyledOutput(t *testing.T) {
	styles := TokenStylesFromColors("49", "158", "73", "242", "250")
	out := Shell(`echo "hello"`, styles)
	require.NotEmpty(t, out)
	require.Contains(t, stripANSI(out), `echo "hello"`)
}

func TestShellReturnsOriginalWhenStylesEmpty(t *testing.T) {
	cmd := `echo "$HOME"`
	require.Equal(t, cmd, Shell(cmd, TokenStyles{}))
}

func TestShellPreservesTemplateParams(t *testing.T) {
	styles := TokenStylesFromColors("49", "158", "73", "242", "250")
	cmd := `echo "{{name}}" {{project:demo}}`
	out := Shell(cmd, styles)
	plain := stripANSI(out)
	require.Contains(t, plain, "{{name}}")
	require.Contains(t, plain, "{{project:demo}}")
}

func TestTokenStylesFromColorsHasExpectedMappings(t *testing.T) {
	styles := TokenStylesFromColors("49", "158", "73", "242", "250")
	require.NotEmpty(t, styles)
	require.Contains(t, styles, chroma.Keyword)
	require.Contains(t, styles, chroma.NameBuiltin)
	require.Contains(t, styles, chroma.LiteralString)
	require.Contains(t, styles, chroma.Comment)
}

func TestShellSupportsMultiline(t *testing.T) {
	styles := TokenStylesFromColors("49", "158", "73", "242", "250")
	cmd := "if true; then\n  echo ok\nfi"
	out := Shell(cmd, styles)
	require.Contains(t, stripANSI(out), "echo ok")
	require.Contains(t, stripANSI(out), "\n")
}

func TestShellSupportsPipes(t *testing.T) {
	styles := TokenStylesFromColors("49", "158", "73", "242", "250")
	cmd := "cat file.txt | grep pattern | sort"
	out := Shell(cmd, styles)
	require.Contains(t, stripANSI(out), cmd)
}
