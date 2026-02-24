package importer

import (
	"io"

	"github.com/fredriklanga/wf/internal/store"
)

// ImportResult contains the outcome of importing workflows from an external format.
type ImportResult struct {
	Workflows []store.Workflow // Successfully converted workflows
	Warnings  []string         // Unmappable features preserved as notes
	Errors    []error          // Parse failures for individual entries
}

// Importer converts external workflow formats into wf Workflows.
type Importer interface {
	Import(reader io.Reader) (*ImportResult, error)
}
