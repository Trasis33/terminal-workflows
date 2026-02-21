package template

import (
	"regexp"
	"strings"
)

// Param represents a named parameter extracted from a command template.
type Param struct {
	Name    string
	Default string
}

// paramRegex matches {{content}} patterns where content is one or more non-} characters.
var paramRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ExtractParams extracts unique parameters from a command string.
// Parameters use {{name}} or {{name:default}} syntax.
// Duplicates are deduplicated by name (last default wins), preserving order of first appearance.
func ExtractParams(command string) []Param {
	matches := paramRegex.FindAllStringSubmatch(command, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]int) // name -> index in result slice
	var params []Param

	for _, match := range matches {
		inner := match[1]
		name, def := parseInner(inner)

		if idx, exists := seen[name]; exists {
			// Last default wins for duplicates
			if def != "" {
				params[idx].Default = def
			}
		} else {
			seen[name] = len(params)
			params = append(params, Param{Name: name, Default: def})
		}
	}

	return params
}

// parseInner splits "name:default" on the first colon.
func parseInner(s string) (name, def string) {
	idx := strings.IndexByte(s, ':')
	if idx < 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}
