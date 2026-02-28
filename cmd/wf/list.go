package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	folderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	nameStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	for _, wf := range workflows {
		name := wf.Name
		folder := ""
		if idx := strings.LastIndex(wf.Name, "/"); idx >= 0 {
			folder = wf.Name[:idx]
			name = wf.Name[idx+1:]
		}

		prefix := ""
		if folder != "" {
			prefix = folderStyle.Render(folder + "/")
		}

		desc := ""
		if wf.Description != "" {
			desc = "  " + descStyle.Render(wf.Description)
		}

		tags := ""
		if len(wf.Tags) > 0 {
			tags = "  " + tagStyle.Render("["+strings.Join(wf.Tags, ", ")+"]")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s%s%s%s\n", prefix, nameStyle.Render(name), desc, tags)
	}

	return nil
}
