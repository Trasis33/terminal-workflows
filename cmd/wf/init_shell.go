package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/fredriklanga/wf/internal/shell"
	"github.com/spf13/cobra"
)

var initKeyFlag string

var initCmd = &cobra.Command{
	Use:       "init [shell]",
	Short:     "Output shell integration script",
	Long:      "Output the shell integration script for the specified shell. Use --key to customize the picker binding.",
	Example:   "  eval \"$(wf init zsh)\"\n  eval \"$(wf init bash)\"\n  wf init fish | source\n  wf init powershell | Invoke-Expression\n  wf init zsh --key ctrl+o",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shellName := args[0]
		key, err := resolveKeybinding(initKeyFlag)
		if err != nil {
			return err
		}

		var keyStr string
		tpl, err := templateForShell(shellName)
		if err != nil {
			return err
		}

		switch shellName {
		case "zsh":
			keyStr = key.ForZsh()
		case "bash":
			keyStr = key.ForBash()
		case "fish":
			keyStr = key.ForFish()
		case "powershell":
			keyStr = key.ForPowerShell()
		default:
			return fmt.Errorf("unsupported shell: %s. Supported: zsh, bash, fish, powershell", shellName)
		}

		data := shell.TemplateData{
			Key:     keyStr,
			Comment: fmt.Sprintf("# Keybinding: %s\n# Change with: wf init %s --key ctrl+<letter>", key.String(), shellName),
		}

		var out bytes.Buffer
		if err := tpl.Execute(&out, data); err != nil {
			return fmt.Errorf("rendering shell script: %w", err)
		}
		_, err = os.Stdout.Write(out.Bytes())
		return err
	},
}

func init() {
	initCmd.Flags().StringVar(&initKeyFlag, "key", "", "Custom keybinding (ctrl+g, alt+f)")
}

func templateForShell(shellName string) (*template.Template, error) {
	switch shellName {
	case "zsh":
		return shell.ZshTemplate, nil
	case "bash":
		return shell.BashTemplate, nil
	case "fish":
		return shell.FishTemplate, nil
	case "powershell":
		return shell.PowerShellTemplate, nil
	default:
		return nil, fmt.Errorf("unsupported shell: %s. Supported: zsh, bash, fish, powershell", shellName)
	}
}

func resolveKeybinding(flagValue string) (shell.Keybinding, error) {
	if flagValue != "" {
		key, err := shell.ParseKey(flagValue)
		if err != nil {
			return shell.Keybinding{}, err
		}
		if err := key.Validate(); err != nil {
			return shell.Keybinding{}, err
		}
		return key, nil
	}

	if shell.DetectWarp() {
		fmt.Fprintf(os.Stderr, "wf: Warp terminal detected - binding to %s instead of %s\n", shell.WarpDefaultKey, shell.DefaultKey)
		return shell.WarpDefaultKey, nil
	}

	return shell.DefaultKey, nil
}
