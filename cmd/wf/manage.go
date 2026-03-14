package main

import (
	"fmt"
	"os"

	"github.com/fredriklanga/wf/internal/manage"
	"github.com/spf13/cobra"
)

var manageResultFile string

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
	RunE: runManage,
}

func init() {
	manageCmd.Flags().StringVar(&manageResultFile, "result-file", "", "write selected manage command to a file instead of stdout")
	if err := manageCmd.Flags().MarkHidden("result-file"); err != nil {
		panic(fmt.Errorf("hide manage result-file flag: %w", err))
	}
}

func runManage(cmd *cobra.Command, args []string) error {
	result, err := manage.Run(getMultiStore())
	if err != nil {
		return err
	}

	return emitManageResult(cmd, result, manageResultFile)
}

func emitManageResult(cmd *cobra.Command, result, resultFile string) error {
	if result == "" {
		return nil
	}

	if resultFile != "" {
		if err := os.WriteFile(resultFile, []byte(result), 0o600); err != nil {
			return fmt.Errorf("write manage result file %q: %w", resultFile, err)
		}
		return nil
	}

	_, err := fmt.Fprintln(cmd.OutOrStdout(), result)
	return err
}
