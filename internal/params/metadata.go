package params

import (
	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
)

// OverlayMetadata extracts params from a command template, then overlays stored
// workflow arg metadata by matching parameter name.
//
// Inline defaults remain authoritative; stored defaults fill only missing values.
// Stored type metadata is authoritative when present so saved workflow args can
// change runtime behavior without changing inline template syntax.
func OverlayMetadata(command string, args []store.Arg) []template.Param {
	params := template.ExtractParams(command)
	if len(params) == 0 {
		return nil
	}

	argByName := make(map[string]store.Arg, len(args))
	for _, arg := range args {
		argByName[arg.Name] = arg
	}

	for i := range params {
		arg, ok := argByName[params[i].Name]
		if !ok {
			continue
		}

		if params[i].Default == "" && arg.Default != "" {
			params[i].Default = arg.Default
		}

		if arg.Type == "" {
			continue
		}

		params[i].Type = template.ParamTypeFromString(arg.Type)
		params[i].Options = cloneStrings(arg.Options)
		params[i].DynamicCmd = arg.DynamicCmd
		params[i].ListCmd = arg.ListCmd
		params[i].ListDelimiter = arg.ListDelimiter
		params[i].ListFieldIndex = arg.ListFieldIndex
		params[i].ListSkipHeader = arg.ListSkipHeader
	}

	return params
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}
