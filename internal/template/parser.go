package template

import (
	"regexp"
	"strings"
)

// ParamType discriminates parameter behavior.
type ParamType int

const (
	ParamText    ParamType = iota // Free text input (default)
	ParamEnum                     // Selection from predefined options
	ParamDynamic                  // Options populated by shell command
	ParamList                     // Selection from shell-command rows with deferred extraction
)

// String returns the author-facing name for a parameter type.
func (t ParamType) String() string {
	switch t {
	case ParamEnum:
		return "enum"
	case ParamDynamic:
		return "dynamic"
	case ParamList:
		return "list"
	default:
		return "text"
	}
}

// ParamTypeFromString converts persisted type strings into runtime ParamType values.
func ParamTypeFromString(s string) ParamType {
	switch strings.TrimSpace(s) {
	case "enum":
		return ParamEnum
	case "dynamic":
		return ParamDynamic
	case "list":
		return ParamList
	default:
		return ParamText
	}
}

// Param represents a named parameter extracted from a command template.
type Param struct {
	Name           string
	Type           ParamType
	Default        string
	Options        []string // For ParamEnum: the option list
	DynamicCmd     string   // For ParamDynamic: shell command to execute
	ListCmd        string   // For ParamList: shell command producing selectable rows
	ListDelimiter  string   // For ParamList: literal field delimiter
	ListFieldIndex int      // For ParamList: 1-based extracted field, 0 = whole row
	ListSkipHeader int      // For ParamList: leading rows removed before selection
}

// paramRegex matches {{content}} patterns where content is one or more non-} characters.
var paramRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ExtractParams extracts unique parameters from a command string.
// Parameters use {{name}}, {{name:default}}, {{name|opt1|opt2|*default}},
// or {{name!command}} syntax.
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
		p := parseInner(inner)

		if idx, exists := seen[p.Name]; exists {
			// Last default wins for duplicates
			if p.Default != "" {
				params[idx].Default = p.Default
			}
		} else {
			seen[p.Name] = len(params)
			params = append(params, p)
		}
	}

	return params
}

// parseInner determines parameter type from the inner content of {{...}}.
// Priority order: bang (dynamic) -> pipe (enum) -> colon (default) -> plain.
//
// Why this order:
//   - Bang first: dynamic commands can contain pipes and colons (e.g., git branch | grep feature)
//   - Pipe second: enum options can contain colons (e.g., dev:3000)
//   - Colon third: existing text-with-default behavior
//   - Plain last: simple name with no special characters
func parseInner(s string) Param {
	// 1. Check for dynamic: name!command
	if idx := strings.IndexByte(s, '!'); idx > 0 {
		return Param{
			Name:       s[:idx],
			Type:       ParamDynamic,
			DynamicCmd: s[idx+1:],
		}
	}

	// 2. Check for enum: name|opt1|opt2|*default_opt
	if idx := strings.IndexByte(s, '|'); idx > 0 {
		parts := strings.Split(s, "|")
		name := parts[0]
		var options []string
		var defaultVal string
		for _, p := range parts[1:] {
			if strings.HasPrefix(p, "*") {
				cleaned := p[1:]
				options = append(options, cleaned)
				defaultVal = cleaned
			} else {
				options = append(options, p)
			}
		}
		return Param{
			Name:    name,
			Type:    ParamEnum,
			Options: options,
			Default: defaultVal,
		}
	}

	// 3. Check for default: name:default (split on first colon only)
	if idx := strings.IndexByte(s, ':'); idx >= 0 {
		return Param{
			Name:    s[:idx],
			Type:    ParamText,
			Default: s[idx+1:],
		}
	}

	// 4. Plain name
	return Param{Name: s, Type: ParamText}
}
