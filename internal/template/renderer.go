package template

// Render substitutes all {{name}} and {{name:default}} occurrences in command
// with values from the provided map. If a parameter has no value and no default,
// the {{name}} placeholder is left as-is.
func Render(command string, values map[string]string) string {
	return paramRegex.ReplaceAllStringFunc(command, func(match string) string {
		// Strip {{ and }}
		inner := match[2 : len(match)-2]
		name, def := parseInner(inner)

		if values != nil {
			if v, ok := values[name]; ok {
				return v
			}
		}

		if def != "" {
			return def
		}

		// No value, no default — leave placeholder
		return match
	})
}
