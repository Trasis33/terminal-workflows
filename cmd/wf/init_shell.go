package main

import (
	"fmt"

	"github.com/fredriklanga/wf/internal/shell"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:       "init [shell]",
	Short:     "Output shell integration script",
	Long:      `Output the shell integration script for the specified shell. Source the output in your shell config to bind Ctrl+G to invoke the wf picker.`,
	Example:   "  eval \"$(wf init zsh)\"\n  eval \"$(wf init bash)\"\n  wf init fish | source",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "zsh":
			fmt.Print(shell.ZshScript)
		case "bash":
			fmt.Print(shell.BashScript)
		case "fish":
			fmt.Print(shell.FishScript)
		default:
			return fmt.Errorf("unsupported shell: %s. Supported: zsh, bash, fish", args[0])
		}
		return nil
	},
}
