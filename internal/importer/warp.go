package importer

import (
	"io"
)

// WarpImporter converts Warp YAML workflow files into wf Workflows.
type WarpImporter struct{}

// Import reads Warp YAML from the reader and converts to wf Workflows.
func (w *WarpImporter) Import(reader io.Reader) (*ImportResult, error) {
	// STUB: returns empty result — tests will fail
	return &ImportResult{}, nil
}
