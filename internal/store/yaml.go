package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// YAMLStore implements Store using YAML files on disk.
// Each workflow is stored as a separate .yaml file under basePath.
// Supports nested directories up to 2 levels for folder organization.
type YAMLStore struct {
	basePath string
}

// NewYAMLStore creates a new YAML file store rooted at basePath.
func NewYAMLStore(basePath string) *YAMLStore {
	return &YAMLStore{basePath: basePath}
}

// Save persists a workflow to disk as a YAML file.
// The filename is derived from the workflow's Name field.
// Creates parent directories if they don't exist.
func (s *YAMLStore) Save(w *Workflow) error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("creating store directory: %w", err)
	}

	data, err := yaml.Marshal(w)
	if err != nil {
		return fmt.Errorf("marshalling workflow: %w", err)
	}

	fpath := s.WorkflowPath(w.Name)

	// Ensure parent directory exists (for nested paths like "infra/deploy")
	dir := filepath.Dir(fpath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating workflow directory: %w", err)
	}

	if err := os.WriteFile(fpath, data, 0644); err != nil {
		return fmt.Errorf("writing workflow file: %w", err)
	}

	return nil
}

// Get retrieves a workflow by name.
// The name can include a path prefix (e.g., "infra/deploy").
func (s *YAMLStore) Get(name string) (*Workflow, error) {
	fpath := s.WorkflowPath(name)

	data, err := os.ReadFile(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workflow %q not found", name)
		}
		return nil, fmt.Errorf("reading workflow file: %w", err)
	}

	var w Workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parsing workflow file: %w", err)
	}

	return &w, nil
}

// List returns all workflows found under basePath, scanning up to 2 levels deep.
func (s *YAMLStore) List() ([]Workflow, error) {
	var workflows []Workflow

	err := filepath.WalkDir(s.basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// If basePath doesn't exist, return empty list
			if os.IsNotExist(err) {
				return filepath.SkipAll
			}
			return err
		}

		// Limit directory depth to 2 levels below basePath
		rel, _ := filepath.Rel(s.basePath, path)
		depth := len(strings.Split(rel, string(filepath.Separator)))
		if d.IsDir() && depth > 3 {
			return filepath.SkipDir
		}

		// Only process .yaml files
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var w Workflow
		if err := yaml.Unmarshal(data, &w); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		workflows = append(workflows, w)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return workflows, nil
}

// Delete removes a workflow file by name.
func (s *YAMLStore) Delete(name string) error {
	fpath := s.WorkflowPath(name)

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return fmt.Errorf("workflow %q not found", name)
	}

	if err := os.Remove(fpath); err != nil {
		return fmt.Errorf("deleting workflow file: %w", err)
	}

	return nil
}

// WorkflowPath resolves the filesystem path for a workflow name.
// Names can include path separators for folder organization (e.g., "infra/deploy").
func (s *YAMLStore) WorkflowPath(name string) string {
	// If name contains a path separator, use it as a subdirectory
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", -1)
		lastIdx := len(parts) - 1
		// Slugify only the last part (the workflow name)
		w := &Workflow{Name: parts[lastIdx]}
		parts[lastIdx] = w.Filename()
		return filepath.Join(s.basePath, filepath.Join(parts...))
	}

	w := &Workflow{Name: name}
	return filepath.Join(s.basePath, w.Filename())
}
