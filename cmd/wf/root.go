package main

import (
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
}
