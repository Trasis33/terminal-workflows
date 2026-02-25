package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// Source represents a configured remote workflow source.
type Source struct {
	Alias     string    `yaml:"alias"`
	URL       string    `yaml:"url"`
	UpdatedAt time.Time `yaml:"updated_at,omitempty"`
}

// sourceConfig is the on-disk representation stored in sources.yaml.
type sourceConfig struct {
	Sources []Source `yaml:"sources"`
}

// UpdateResult reports what changed after a source update (git pull).
type UpdateResult struct {
	Added   []string
	Removed []string
	Updated []string
}

// Manager handles CRUD operations for remote workflow sources.
type Manager struct {
	dir string       // base directory for clones (config.SourcesDir())
	cfg sourceConfig // loaded from sources.yaml
}

// NewManager creates a Manager and loads existing config from sources.yaml.
// If the config file doesn't exist, initializes with an empty source list.
func NewManager(dir string) *Manager {
	m := &Manager{dir: dir}
	m.load()
	return m
}

// Add clones a remote repository and registers it as a source.
// If alias is empty, it is auto-derived from the URL.
func (m *Manager) Add(ctx context.Context, url, alias string) error {
	if !gitAvailable() {
		return fmt.Errorf("git is required for remote sources. Install git and try again")
	}

	if alias == "" {
		alias = deriveAlias(url)
	}

	if m.hasAlias(alias) {
		return fmt.Errorf("source %q already exists. Use --name to specify a different alias", alias)
	}

	dest := filepath.Join(m.dir, alias)
	if err := gitClone(ctx, url, dest); err != nil {
		return err
	}

	m.cfg.Sources = append(m.cfg.Sources, Source{
		Alias:     alias,
		URL:       url,
		UpdatedAt: time.Now(),
	})
	return m.save()
}

// Remove deletes a source's clone directory and removes it from config.
func (m *Manager) Remove(alias string) error {
	idx := m.findIndex(alias)
	if idx < 0 {
		return fmt.Errorf("source %q not found", alias)
	}

	if err := os.RemoveAll(filepath.Join(m.dir, alias)); err != nil {
		return fmt.Errorf("remove clone directory: %w", err)
	}

	m.cfg.Sources = append(m.cfg.Sources[:idx], m.cfg.Sources[idx+1:]...)
	return m.save()
}

// Update pulls the latest changes for a source and returns a diff summary.
func (m *Manager) Update(ctx context.Context, alias string) (*UpdateResult, error) {
	idx := m.findIndex(alias)
	if idx < 0 {
		return nil, fmt.Errorf("source %q not found", alias)
	}

	cloneDir := filepath.Join(m.dir, alias)

	// Snapshot before pull
	before, _ := listYAMLFiles(cloneDir)

	if _, err := gitPull(ctx, cloneDir); err != nil {
		return nil, err
	}

	// Snapshot after pull
	after, _ := listYAMLFiles(cloneDir)

	result := diffSnapshots(before, after)

	m.cfg.Sources[idx].UpdatedAt = time.Now()
	if err := m.save(); err != nil {
		return result, err
	}
	return result, nil
}

// List returns a copy of all configured sources.
func (m *Manager) List() []Source {
	out := make([]Source, len(m.cfg.Sources))
	copy(out, m.cfg.Sources)
	return out
}

// SourceDirs returns a map of alias -> clone directory path.
func (m *Manager) SourceDirs() map[string]string {
	dirs := make(map[string]string, len(m.cfg.Sources))
	for _, s := range m.cfg.Sources {
		dirs[s.Alias] = filepath.Join(m.dir, s.Alias)
	}
	return dirs
}

// configPath returns the path to the sources.yaml config file.
func (m *Manager) configPath() string {
	return filepath.Join(m.dir, "sources.yaml")
}

// load reads the sources.yaml config file. Missing file is not an error.
func (m *Manager) load() {
	data, err := os.ReadFile(m.configPath())
	if err != nil {
		m.cfg = sourceConfig{}
		return
	}
	if err := yaml.Unmarshal(data, &m.cfg); err != nil {
		m.cfg = sourceConfig{}
	}
}

// save writes the current config to sources.yaml, creating the directory if needed.
func (m *Manager) save() error {
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return fmt.Errorf("create sources directory: %w", err)
	}
	data, err := yaml.Marshal(&m.cfg)
	if err != nil {
		return fmt.Errorf("marshal sources config: %w", err)
	}
	return os.WriteFile(m.configPath(), data, 0644)
}

// hasAlias returns true if a source with the given alias already exists.
func (m *Manager) hasAlias(alias string) bool {
	return m.findIndex(alias) >= 0
}

// findIndex returns the index of the source with the given alias, or -1.
func (m *Manager) findIndex(alias string) int {
	for i, s := range m.cfg.Sources {
		if s.Alias == alias {
			return i
		}
	}
	return -1
}

// yamlFileInfo holds a filename and its modification time for diff comparison.
type yamlFileInfo struct {
	name    string
	modTime time.Time
}

// listYAMLFiles returns all .yaml files under dir (excluding .git/) with mod times.
func listYAMLFiles(dir string) (map[string]yamlFileInfo, error) {
	files := make(map[string]yamlFileInfo)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		files[rel] = yamlFileInfo{name: rel, modTime: info.ModTime()}
		return nil
	})
	return files, err
}

// diffSnapshots computes added, removed, and updated files between two snapshots.
func diffSnapshots(before, after map[string]yamlFileInfo) *UpdateResult {
	result := &UpdateResult{}
	for name := range after {
		b, existed := before[name]
		if !existed {
			result.Added = append(result.Added, name)
		} else if after[name].modTime.After(b.modTime) {
			result.Updated = append(result.Updated, name)
		}
	}
	for name := range before {
		if _, exists := after[name]; !exists {
			result.Removed = append(result.Removed, name)
		}
	}
	return result
}
