package params

import (
	"testing"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlayMetadata_StoredListMetadataOverridesExtractedText(t *testing.T) {
	params := OverlayMetadata("echo {{branch:main}} {{env|dev|*prod}} {{other}}", []store.Arg{
		{
			Name:           "branch",
			Type:           "list",
			Default:        "stored-default-should-not-win",
			ListCmd:        "git branch --format='%(refname:short)'",
			ListDelimiter:  "/",
			ListFieldIndex: 2,
			ListSkipHeader: 1,
		},
		{
			Name:       "env",
			Type:       "dynamic",
			Default:    "stored-env-default",
			DynamicCmd: "printenv ENVIRONMENTS",
		},
		{
			Name:    "other",
			Default: "from-store",
		},
	})

	require.Len(t, params, 3)

	assert.Equal(t, template.ParamList, params[0].Type)
	assert.Equal(t, "main", params[0].Default)
	assert.Equal(t, "git branch --format='%(refname:short)'", params[0].ListCmd)
	assert.Equal(t, "/", params[0].ListDelimiter)
	assert.Equal(t, 2, params[0].ListFieldIndex)
	assert.Equal(t, 1, params[0].ListSkipHeader)

	assert.Equal(t, template.ParamDynamic, params[1].Type)
	assert.Equal(t, "prod", params[1].Default)
	assert.Equal(t, "printenv ENVIRONMENTS", params[1].DynamicCmd)

	assert.Equal(t, template.ParamText, params[2].Type)
	assert.Equal(t, "from-store", params[2].Default)
}

func TestOverlayMetadata_StoredEnumOptionsReplaceInlineOptions(t *testing.T) {
	params := OverlayMetadata("deploy {{env|dev|staging|prod}}", []store.Arg{{
		Name:    "env",
		Type:    "enum",
		Options: []string{"preview", "prod"},
	}})

	require.Len(t, params, 1)
	assert.Equal(t, template.ParamEnum, params[0].Type)
	assert.Equal(t, []string{"preview", "prod"}, params[0].Options)
}

func TestExtractListValue(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		delimiter  string
		fieldIndex int
		want       string
		wantErr    bool
		retryable  bool
	}{
		{name: "whole row when field unset", raw: "a|b|c", delimiter: "|", fieldIndex: 0, want: "a|b|c"},
		{name: "whole row when delimiter empty", raw: "a|b|c", delimiter: "", fieldIndex: 2, want: "a|b|c"},
		{name: "literal delimiter", raw: "api::443::tcp", delimiter: "::", fieldIndex: 2, want: "443"},
		{name: "one based field indexing", raw: "alpha,beta,gamma", delimiter: ",", fieldIndex: 1, want: "alpha"},
		{name: "trimmed selected field", raw: "alpha | beta | gamma ", delimiter: "|", fieldIndex: 2, want: "beta"},
		{name: "missing field returns retryable error", raw: "only|two", delimiter: "|", fieldIndex: 3, wantErr: true, retryable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractListValue(tt.raw, tt.delimiter, tt.fieldIndex)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.retryable, IsRetryable(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
