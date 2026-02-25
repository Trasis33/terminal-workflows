package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// RemoteStore implements Store as a read-only view over a cloned git repository.
// It walks the entire directory tree (skipping .git) and returns all valid
// workflow YAML files found. Save and Delete operations return errors since
// remote sources are read-only.
type RemoteStore struct {
	dir string // path to cloned repo root
}

// NewRemoteStore creates a new RemoteStore rooted at the given directory.
func NewRemoteStore(dir string) *RemoteStore {
	return &RemoteStore{dir: dir}
}

// List walks the cloned repo directory and returns all valid workflows.
// It skips .git directories and silently ignores malformed YAML files
// or files that don't contain valid workflow definitions (missing Name or Command).
func (rs *RemoteStore) List() ([]Workflow, error) {
	var workflows []Workflow

	err := filepath.WalkDir(rs.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipAll
			}
			return err
		}

		// Skip .git directory entirely
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// Only process .yaml and .yml files
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files
		}

		var w Workflow
		if unmarshalErr := yaml.Unmarshal(data, &w); unmarshalErr != nil {
			return nil // skip malformed YAML
		}

		// Skip files that don't have required workflow fields
		if w.Name == "" || w.Command == "" {
			return nil
		}

		workflows = append(workflows, w)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return workflows, nil
}

// Get retrieves a workflow by name from the cloned repo.
// It iterates over all workflows since remote repos don't follow slug-based
// path conventions.
func (rs *RemoteStore) Get(name string) (*Workflow, error) {
	workflows, err := rs.List()
	if err != nil {
		return nil, err
	}

	for i := range workflows {
		if workflows[i].Name == name {
			return &workflows[i], nil
		}
	}

	return nil, fmt.Errorf("workflow %q not found in remote source", name)
}

// Save returns an error because remote sources are read-only.
func (rs *RemoteStore) Save(w *Workflow) error {
	return fmt.Errorf("cannot save to remote source (read-only)")
}

// Delete returns an error because remote sources are read-only.
func (rs *RemoteStore) Delete(name string) error {
	return fmt.Errorf("cannot delete from remote source (read-only)")
}
