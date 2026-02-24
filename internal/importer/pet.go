package importer

import (
	"io"
)

// PetImporter converts Pet TOML snippet files into wf Workflows.
type PetImporter struct{}

// Import reads Pet TOML from the reader and converts to wf Workflows.
func (p *PetImporter) Import(reader io.Reader) (*ImportResult, error) {
	// STUB: returns empty result — tests will fail
	return &ImportResult{}, nil
}
