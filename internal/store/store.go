package store

// Store defines the interface for workflow CRUD operations.
// Implementations handle persistence (YAML files, in-memory, etc.).
type Store interface {
	// List returns all workflows from the store.
	List() ([]Workflow, error)

	// Get retrieves a workflow by name.
	// Returns an error if the workflow does not exist.
	Get(name string) (*Workflow, error)

	// Save creates or updates a workflow.
	// The workflow's Name field determines the storage location.
	Save(w *Workflow) error

	// Delete removes a workflow by name.
	// Returns an error if the workflow does not exist.
	Delete(name string) error
}
