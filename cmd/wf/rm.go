package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [name]",
	Short: "Delete a workflow",
	Long: `Delete a workflow by name.

By default, asks for confirmation before deleting.
Use --force to skip the confirmation prompt.`,
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
}

func runRm(cmd *cobra.Command, args []string) error {
	name := args[0]
	force, _ := cmd.Flags().GetBool("force")
	s := getStore()

	// Verify workflow exists before prompting
	if _, err := s.Get(name); err != nil {
		return fmt.Errorf("workflow %q not found", name)
	}

	if !force {
		fmt.Printf("Delete workflow '%s'? [y/N]: ", name)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		} else {
			// No input (e.g., piped stdin with no data) — treat as "no"
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := s.Delete(name); err != nil {
		return fmt.Errorf("deleting workflow: %w", err)
	}

	fmt.Printf("Deleted %s\n", name)
	return nil
}
