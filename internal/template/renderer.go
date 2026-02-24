package template

// Render substitutes all {{name}}, {{name:default}}, {{name|opt1|*default}},
// and {{name!command}} occurrences in command with values from the provided map.
// If a parameter has no value and no default, the placeholder is left as-is.
func Render(command string, values map[string]string) string {
	return paramRegex.ReplaceAllStringFunc(command, func(match string) string {
		// Strip {{ and }}
		inner := match[2 : len(match)-2]
		p := parseInner(inner)

		if values != nil {
			if v, ok := values[p.Name]; ok {
				return v
			}
		}

		if p.Default != "" {
			return p.Default
		}

		// No value, no default — leave placeholder
		return match
	})
}
