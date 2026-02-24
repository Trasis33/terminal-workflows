package importer

import (
	"regexp"
	"strings"
)

// petParamRegex matches Pet's <name> and <name=default> syntax.
// Captures: group 1 = name, group 2 = default value (optional).
// The default value is everything after the first = up to > (with trailing spaces trimmed).
var petParamRegex = regexp.MustCompile(`<([^>=]+?)(?:=([^>]*[^\s>]))?\s*>`)

// convertPetParam converts Pet parameter syntax to wf syntax.
// <param> -> {{param}}, <param=default> -> {{param:default}}
func convertPetParam(command string) string {
	return petParamRegex.ReplaceAllStringFunc(command, func(match string) string {
		sub := petParamRegex.FindStringSubmatch(match)
		name := sub[1]
		def := sub[2]
		if def != "" {
			return "{{" + name + ":" + def + "}}"
		}
		return "{{" + name + "}}"
	})
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)
var dashRun = regexp.MustCompile(`-{2,}`)

// slugifyName converts a description string into a URL/filesystem-safe slug.
// Uses the same logic as store.Workflow.Filename() but without the ".yaml" extension.
func slugifyName(description string) string {
	s := strings.ToLower(description)
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRe.ReplaceAllString(s, "-")
	s = dashRun.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "unnamed"
	}
	return s
}
