package importer

import (
	"fmt"
	"io"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	toml "github.com/pelletier/go-toml/v2"
)

// PetFile represents the top-level structure of a Pet TOML snippet file.
type PetFile struct {
	Snippets []PetSnippet `toml:"snippets"`
}

// PetSnippet represents a single snippet entry in a Pet TOML file.
type PetSnippet struct {
	Description string      `toml:"description"`
	Command     string      `toml:"command"`
	Tag         interface{} `toml:"tag"` // accepts both string and []interface{}
	Output      string      `toml:"output"`
}

// normalizeTag converts the raw TOML tag value (which may be a bare string or
// an array) into a []string. Returns nil for absent or empty values.
func normalizeTag(v interface{}) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if str, ok := s.(string); ok && str != "" {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

// PetImporter converts Pet TOML snippet files into wf Workflows.
type PetImporter struct{}

// Import reads Pet TOML from the reader and converts to wf Workflows.
func (p *PetImporter) Import(reader io.Reader) (*ImportResult, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading pet file: %w", err)
	}

	var petFile PetFile
	if err := toml.Unmarshal(data, &petFile); err != nil {
		return nil, fmt.Errorf("parsing pet TOML: %w", err)
	}

	result := &ImportResult{}

	for i, snippet := range petFile.Snippets {
		if snippet.Command == "" {
			result.Errors = append(result.Errors, fmt.Errorf("pet snippet #%d: empty command field", i+1))
			continue
		}

		// Convert Pet parameter syntax to wf syntax
		convertedCmd := convertPetParam(snippet.Command)

		// Generate name from description via slugification
		name := slugifyName(snippet.Description)

		// Extract args from the converted command
		params := template.ExtractParams(convertedCmd)
		var args []store.Arg
		for _, param := range params {
			args = append(args, store.Arg{
				Name:    param.Name,
				Default: param.Default,
			})
		}

		wf := store.Workflow{
			Name:        name,
			Command:     convertedCmd,
			Description: snippet.Description,
			Tags:        normalizeTag(snippet.Tag),
			Args:        args,
		}

		result.Workflows = append(result.Workflows, wf)

		// Preserve unmappable fields as warnings
		if snippet.Output != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("pet snippet %q: output: %s", snippet.Description, snippet.Output))
		}
	}

	return result, nil
}
