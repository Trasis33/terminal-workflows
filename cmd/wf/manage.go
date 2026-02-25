package main

import (
	"github.com/fredriklanga/wf/internal/manage"
	"github.com/spf13/cobra"
)

var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Open the full-screen management TUI",
	Long: `Launch an interactive terminal UI for managing workflows.

Browse, create, edit, and delete workflows in a full-screen interface
with folder organization, tag filtering, fuzzy search, and theme customization.

Keyboard shortcuts:
  n        Create new workflow
  e        Edit selected workflow
  d        Delete selected workflow
  /        Search workflows
  tab      Toggle sidebar (folders/tags)
  S        Open settings
  q        Quit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return manage.Run(getMultiStore())
	},
}
