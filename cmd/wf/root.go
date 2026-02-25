package main

import (
	"github.com/fredriklanga/wf/internal/config"
	"github.com/fredriklanga/wf/internal/source"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "wf",
	Short: "Terminal workflow manager",
	Long: `wf is a terminal workflow manager that lets you save, search, and execute
parameterized command templates directly from the terminal.

Save complex commands as reusable workflows with named parameters,
then find and paste them to your prompt in under 3 seconds.`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureDir()
	},
}

var yamlStore *store.YAMLStore

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(pickCmd)
	rootCmd.AddCommand(manageCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(autofillCmd)
	rootCmd.AddCommand(sourceCmd)
}

// getStore returns the shared YAMLStore instance, creating it if needed.
func getStore() *store.YAMLStore {
	if yamlStore == nil {
		yamlStore = store.NewYAMLStore(config.WorkflowsDir())
	}
	return yamlStore
}

// getMultiStore returns a Store that merges local and remote workflows.
// If no remote sources are configured, it returns the local store directly
// to avoid any overhead.
func getMultiStore() store.Store {
	local := getStore()
	mgr := source.NewManager(config.SourcesDir())
	sources := mgr.SourceDirs()
	if len(sources) == 0 {
		return local
	}
	remote := make(map[string]store.Store, len(sources))
	for alias, dir := range sources {
		remote[alias] = store.NewRemoteStore(dir)
	}
	return store.NewMultiStore(local, remote)
}
