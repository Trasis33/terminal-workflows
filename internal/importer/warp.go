package importer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/goccy/go-yaml"
)

// WarpWorkflow represents the structure of a Warp YAML workflow file.
type WarpWorkflow struct {
	Name        string         `yaml:"name"`
	Command     string         `yaml:"command"`
	Tags        []string       `yaml:"tags"`
	Description string         `yaml:"description"`
	SourceURL   string         `yaml:"source_url"`
	Author      string         `yaml:"author"`
	AuthorURL   string         `yaml:"author_url"`
	Shells      []string       `yaml:"shells"`
	Arguments   []WarpArgument `yaml:"arguments"`
}

// WarpArgument represents an argument definition in a Warp workflow.
type WarpArgument struct {
	Name         string  `yaml:"name"`
	Description  string  `yaml:"description"`
	DefaultValue *string `yaml:"default_value"`
}

// WarpImporter converts Warp YAML workflow files into wf Workflows.
type WarpImporter struct{}

// Import reads Warp YAML from the reader and converts to wf Workflows.
// Supports multi-document YAML files (separated by ---).
func (w *WarpImporter) Import(reader io.Reader) (*ImportResult, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading warp file: %w", err)
	}

	result := &ImportResult{}

	// Split on YAML document separators to handle multi-document files
	docs := splitYAMLDocuments(data)

	for i, doc := range docs {
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		var warpWF WarpWorkflow
		if err := yaml.Unmarshal(doc, &warpWF); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("warp workflow #%d: %w", i+1, err))
			continue
		}

		// Skip empty entries
		if warpWF.Name == "" && warpWF.Command == "" {
			continue
		}

		// Convert arguments
		var args []store.Arg
		for _, warpArg := range warpWF.Arguments {
			arg := store.Arg{
				Name:        warpArg.Name,
				Description: warpArg.Description,
			}
			// YAML null (~) results in nil pointer; treat as empty string
			if warpArg.DefaultValue != nil {
				arg.Default = *warpArg.DefaultValue
			}
			args = append(args, arg)
		}

		wf := store.Workflow{
			Name:        warpWF.Name,
			Command:     warpWF.Command,
			Description: warpWF.Description,
			Tags:        warpWF.Tags,
			Args:        args,
		}

		result.Workflows = append(result.Workflows, wf)

		// Preserve unmappable fields as warnings
		if warpWF.SourceURL != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("warp workflow %q: source_url: %s", warpWF.Name, warpWF.SourceURL))
		}
		if warpWF.Author != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("warp workflow %q: author: %s", warpWF.Name, warpWF.Author))
		}
		if warpWF.AuthorURL != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("warp workflow %q: author_url: %s", warpWF.Name, warpWF.AuthorURL))
		}
		if len(warpWF.Shells) > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("warp workflow %q: shells: %s", warpWF.Name, strings.Join(warpWF.Shells, ", ")))
		}
	}

	return result, nil
}

// splitYAMLDocuments splits a multi-document YAML byte slice on --- separators.
func splitYAMLDocuments(data []byte) [][]byte {
	// Split on lines that are exactly "---" (with optional trailing whitespace)
	var docs [][]byte
	var current []byte

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if bytes.Equal(trimmed, []byte("---")) {
			if len(current) > 0 {
				docs = append(docs, current)
			}
			current = nil
			continue
		}
		current = append(current, line...)
		current = append(current, '\n')
	}
	if len(bytes.TrimSpace(current)) > 0 {
		docs = append(docs, current)
	}

	// If no --- separator was found, treat the whole thing as one document
	if len(docs) == 0 && len(bytes.TrimSpace(data)) > 0 {
		docs = append(docs, data)
	}

	return docs
}
