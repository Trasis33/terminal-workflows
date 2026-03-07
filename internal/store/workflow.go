package store

import (
	"regexp"
	"strings"
)

// Workflow represents a saved command template with metadata.
type Workflow struct {
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags,omitempty"`
	Args        []Arg    `yaml:"args,omitempty"`
}

// Arg defines a named parameter for a workflow command.
type Arg struct {
	Name           string   `yaml:"name"`
	Default        string   `yaml:"default,omitempty"`
	Description    string   `yaml:"description,omitempty"`
	Type           string   `yaml:"type,omitempty"`             // "text" (default/omitted), "enum", "dynamic", "list"
	Options        []string `yaml:"options,omitempty"`          // For enum type
	DynamicCmd     string   `yaml:"dynamic_cmd,omitempty"`      // For dynamic type
	ListCmd        string   `yaml:"list_cmd,omitempty"`         // For list type: shell command producing rows
	ListDelimiter  string   `yaml:"list_delimiter,omitempty"`   // For list type: literal field delimiter
	ListFieldIndex int      `yaml:"list_field_index,omitempty"` // For list type: 1-based extracted field, 0 = whole row
	ListSkipHeader int      `yaml:"list_skip_header,omitempty"` // For list type: number of leading rows to skip
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)
var dashRun = regexp.MustCompile(`-{2,}`)

// Filename returns a filesystem-safe filename derived from the workflow name.
// Converts to lowercase, replaces spaces and special chars with hyphens,
// strips leading/trailing hyphens, and appends .yaml extension.
func (w *Workflow) Filename() string {
	s := strings.ToLower(w.Name)
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRe.ReplaceAllString(s, "-")
	s = dashRun.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "unnamed"
	}
	return s + ".yaml"
}
