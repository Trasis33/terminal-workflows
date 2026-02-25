package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	Long: `List all saved workflows.

Displays each workflow's name, description, and tags.
Workflows in subfolders are included automatically.`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	s := getMultiStore()
	workflows, err := s.List()
	if err != nil {
		return fmt.Errorf("listing workflows: %w", err)
	}

	if len(workflows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No workflows found")
		return nil
	}

	for _, wf := range workflows {
		tags := ""
		if len(wf.Tags) > 0 {
			tags = "  [" + strings.Join(wf.Tags, ", ") + "]"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s  %s%s\n", wf.Name, wf.Description, tags)
	}

	return nil
}
