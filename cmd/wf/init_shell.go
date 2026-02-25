package main

import (
	"fmt"
	"os"

	"github.com/fredriklanga/wf/internal/shell"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:       "init [shell]",
	Short:     "Output shell integration script",
	Long:      `Output the shell integration script for the specified shell. Source the output in your shell config to bind Ctrl+G to invoke the wf picker.`,
	Example:   "  eval \"$(wf init zsh)\"\n  eval \"$(wf init bash)\"\n  wf init fish | source\n  wf init powershell | Invoke-Expression",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var script string
		switch args[0] {
		case "zsh":
			script = shell.ZshScript
		case "bash":
			script = shell.BashScript
		case "fish":
			script = shell.FishScript
		case "powershell":
			script = shell.PowerShellScript
		default:
			return fmt.Errorf("unsupported shell: %s. Supported: zsh, bash, fish, powershell", args[0])
		}
		_, err := os.Stdout.WriteString(script)
		return err
	},
}
