package params

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadListSource_SkipsHeadersBeforeSelectableRows(t *testing.T) {
	source, err := LoadListSource("printf 'NAME\nalpha\nbeta\n'", 1)
	require.NoError(t, err)
	require.Len(t, source.Rows, 2)
	assert.Equal(t, "alpha", source.Rows[0].Raw)
	assert.Equal(t, "beta", source.Rows[1].Raw)
	assert.False(t, source.EmptyAfterSkip)
}

func TestLoadListSource_EmptyAfterSkip(t *testing.T) {
	source, err := LoadListSource("printf 'HEADER\n'", 1)
	require.NoError(t, err)
	assert.Empty(t, source.Rows)
	assert.True(t, source.EmptyAfterSkip)
}

func TestLoadListSource_CommandFailureIncludesShortAndDetail(t *testing.T) {
	_, err := LoadListSource("printf 'stderr detail\n' >&2; exit 7", 0)
	require.Error(t, err)

	var sourceErr *ListSourceError
	require.ErrorAs(t, err, &sourceErr)
	assert.Contains(t, sourceErr.Short, "list command failed")
	assert.Contains(t, sourceErr.Detail, "stderr detail")
}

func TestScanListRows_ReportsScannerErrorForLongRows(t *testing.T) {
	longRow := bytes.Repeat([]byte("x"), listSourceMaxTokenSize+1)
	_, err := scanListRows(longRow)
	require.Error(t, err)

	var sourceErr *ListSourceError
	require.ErrorAs(t, err, &sourceErr)
	assert.Equal(t, "reading list output failed", sourceErr.Short)
	assert.NotEmpty(t, sourceErr.Detail)
}
