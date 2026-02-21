package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new workflow",
	Long: `Create a new workflow from flags or interactively.

If --name or --command are missing, you will be prompted for them.
Parameters in the command string ({{name}} or {{name:default}}) are
automatically extracted and added as arguments.`,
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringP("name", "n", "", "workflow name (required)")
	addCmd.Flags().StringP("command", "c", "", "command template (required)")
	addCmd.Flags().StringP("description", "d", "", "description")
	addCmd.Flags().StringSliceP("tag", "t", nil, "tags (repeatable)")
	addCmd.Flags().StringP("folder", "f", "", "subfolder path under workflows/ (max 2 levels)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	command, _ := cmd.Flags().GetString("command")
	description, _ := cmd.Flags().GetString("description")
	tags, _ := cmd.Flags().GetStringSlice("tag")
	folder, _ := cmd.Flags().GetString("folder")

	// Determine if we need interactive mode (missing required fields)
	interactive := name == "" || command == ""
	var scanner *bufio.Scanner
	if interactive {
		scanner = bufio.NewScanner(os.Stdin)
	}

	// Interactive fallback for required fields
	if name == "" {
		fmt.Print("Workflow name: ")
		if scanner.Scan() {
			name = strings.TrimSpace(scanner.Text())
		}
		if name == "" {
			return fmt.Errorf("workflow name is required")
		}
	}

	if command == "" {
		fmt.Print("Command (or 'multi' for multiline): ")
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "multi" {
				command = readMultiline(scanner)
			} else {
				command = line
			}
		}
		if command == "" {
			return fmt.Errorf("command is required")
		}
	}

	// Only prompt optional fields in interactive mode
	if interactive {
		if !cmd.Flags().Changed("description") && description == "" {
			fmt.Print("Description (optional): ")
			if scanner.Scan() {
				description = strings.TrimSpace(scanner.Text())
			}
		}

		if !cmd.Flags().Changed("tag") && len(tags) == 0 {
			fmt.Print("Tags (comma-separated, optional): ")
			if scanner.Scan() {
				raw := strings.TrimSpace(scanner.Text())
				if raw != "" {
					for _, t := range strings.Split(raw, ",") {
						t = strings.TrimSpace(t)
						if t != "" {
							tags = append(tags, t)
						}
					}
				}
			}
		}
	}

	// Validate folder depth (max 2 levels)
	if folder != "" {
		parts := strings.Split(strings.Trim(folder, "/"), "/")
		if len(parts) > 2 {
			return fmt.Errorf("folder depth exceeds maximum of 2 levels: %s", folder)
		}
	}

	// Build workflow
	wf := &store.Workflow{
		Name:        name,
		Command:     command,
		Description: description,
		Tags:        tags,
	}

	// Auto-extract parameters from command string
	params := template.ExtractParams(command)
	for _, p := range params {
		wf.Args = append(wf.Args, store.Arg{
			Name:    p.Name,
			Default: p.Default,
		})
	}

	// Resolve store name (prepend folder if specified)
	storeName := name
	if folder != "" {
		storeName = strings.Trim(folder, "/") + "/" + name
	}

	// Check for duplicate
	s := getStore()
	existing, err := s.Get(storeName)
	if err == nil && existing != nil {
		return fmt.Errorf("workflow %q already exists", name)
	}

	// If folder is specified, we need to save with the folder path
	if folder != "" {
		wf.Name = storeName
	}

	if err := s.Save(wf); err != nil {
		return fmt.Errorf("saving workflow: %w", err)
	}

	// Restore display name for output
	wf.Name = name

	fmt.Printf("Created %s\n", name)
	return nil
}

// readMultiline reads lines until an empty line is entered.
func readMultiline(scanner *bufio.Scanner) string {
	fmt.Println("Enter command lines (empty line to finish):")
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
