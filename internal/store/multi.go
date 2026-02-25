package store

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// MultiStore aggregates a local Store with zero or more remote Stores.
// Local workflows are returned without prefix. Remote workflows are
// namespaced with their source alias (e.g., "team/deploy-k8s").
type MultiStore struct {
	local  Store
	remote map[string]Store // key = source alias
}

// NewMultiStore creates a MultiStore that merges local and remote stores.
// If remote is nil, it is initialized to an empty map.
func NewMultiStore(local Store, remote map[string]Store) *MultiStore {
	if remote == nil {
		remote = make(map[string]Store)
	}
	return &MultiStore{local: local, remote: remote}
}

// List returns all workflows from local and remote stores.
// Local workflows are listed first without prefix. Remote workflows
// are prefixed with their source alias (e.g., "alias/name").
// Remote stores are iterated in sorted order for deterministic results.
// If a remote store fails, a warning is printed to stderr and that source
// is skipped — other sources and local workflows are still returned.
func (ms *MultiStore) List() ([]Workflow, error) {
	// Local workflows first — local errors are critical
	all, err := ms.local.List()
	if err != nil {
		return nil, err
	}

	// Sort remote aliases for deterministic ordering
	aliases := make([]string, 0, len(ms.remote))
	for alias := range ms.remote {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	for _, alias := range aliases {
		s := ms.remote[alias]
		workflows, listErr := s.List()
		if listErr != nil {
			fmt.Fprintf(os.Stderr, "warning: source %q: %v\n", alias, listErr)
			continue
		}
		for i := range workflows {
			workflows[i].Name = alias + "/" + workflows[i].Name
			all = append(all, workflows[i])
		}
	}

	return all, nil
}

// Get retrieves a workflow by name. If the name contains a "/" and the prefix
// matches a known remote alias, the request is delegated to that remote store
// with the alias stripped. Otherwise, it falls through to the local store.
func (ms *MultiStore) Get(name string) (*Workflow, error) {
	if idx := strings.Index(name, "/"); idx >= 0 {
		alias := name[:idx]
		remainder := name[idx+1:]
		if s, ok := ms.remote[alias]; ok {
			return s.Get(remainder)
		}
	}

	return ms.local.Get(name)
}

// Save persists a workflow. If the name starts with a known remote alias,
// the operation is rejected (remote sources are read-only). Otherwise
// it delegates to the local store.
func (ms *MultiStore) Save(w *Workflow) error {
	if idx := strings.Index(w.Name, "/"); idx >= 0 {
		alias := w.Name[:idx]
		if _, ok := ms.remote[alias]; ok {
			return fmt.Errorf("cannot save to remote source %q (read-only)", alias)
		}
	}

	return ms.local.Save(w)
}

// Delete removes a workflow by name. If the name starts with a known remote
// alias, the operation is rejected. Otherwise it delegates to the local store.
func (ms *MultiStore) Delete(name string) error {
	if idx := strings.Index(name, "/"); idx >= 0 {
		alias := name[:idx]
		if _, ok := ms.remote[alias]; ok {
			return fmt.Errorf("cannot delete from remote source %q (read-only)", alias)
		}
	}

	return ms.local.Delete(name)
}

// HasRemote returns true if any remote stores are configured.
// Useful for UI to decide whether source labels should be shown.
func (ms *MultiStore) HasRemote() bool {
	return len(ms.remote) > 0
}
