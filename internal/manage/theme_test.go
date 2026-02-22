package manage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	assert.Equal(t, "49", theme.Colors.Primary)
	assert.Equal(t, "158", theme.Colors.Secondary)
	assert.Equal(t, "73", theme.Colors.Tertiary)
	assert.Equal(t, "250", theme.Colors.Text)
	assert.Equal(t, "242", theme.Colors.Dim)
	assert.Equal(t, "238", theme.Colors.Border)
	assert.Equal(t, "rounded", theme.Borders.Style)
	assert.Equal(t, 24, theme.Layout.SidebarWidth)
	assert.True(t, theme.Layout.ShowPreview)
	assert.Equal(t, "default", theme.Name)
}

func TestPresetThemes(t *testing.T) {
	names := PresetNames()
	assert.Equal(t, []string{"default", "dark", "light", "dracula", "nord"}, names)

	for _, name := range names {
		theme, ok := PresetByName(name)
		assert.True(t, ok, "preset %q should exist", name)
		assert.Equal(t, name, theme.Name)
		assert.NotEmpty(t, theme.Colors.Primary)
	}

	_, ok := PresetByName("nonexistent")
	assert.False(t, ok)
}

func TestThemeYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := DefaultTheme()

	err := SaveTheme(dir, original)
	require.NoError(t, err)

	loaded, err := LoadTheme(dir)
	require.NoError(t, err)

	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Colors, loaded.Colors)
	assert.Equal(t, original.Borders, loaded.Borders)
	assert.Equal(t, original.Layout, loaded.Layout)
}

func TestLoadThemeNotFound(t *testing.T) {
	_, err := LoadTheme(filepath.Join(t.TempDir(), "nonexistent"))
	assert.Error(t, err)
}

func TestSaveThemeCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	err := SaveTheme(dir, DefaultTheme())
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "theme.yaml"))
	assert.NoError(t, err)
}

func TestBorderFromString(t *testing.T) {
	// Just verify no panics and defaults to rounded for unknown.
	cases := []string{"rounded", "normal", "thick", "double", "hidden", "unknown", ""}
	for _, c := range cases {
		_ = borderFromString(c)
	}
}

func TestThemeStyles(t *testing.T) {
	theme := DefaultTheme()
	s := theme.Styles()
	// Verify styles are non-zero (computed).
	assert.NotEqual(t, s.Highlight, s.Dim, "highlight and dim styles should differ")
}
