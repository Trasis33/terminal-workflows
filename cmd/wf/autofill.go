package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fredriklanga/wf/internal/ai"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/spf13/cobra"
)

var autofillCmd = &cobra.Command{
	Use:   "autofill [workflow-name]",
	Short: "Auto-fill workflow metadata using AI",
	Long: `Uses AI to generate metadata (name, description, tags, argument types) for an existing workflow.

Specify which fields to fill with flags, or run without flags to choose interactively.

Examples:
  wf autofill my-workflow --description --tags
  wf autofill my-workflow --name --tags --args
  wf autofill my-workflow    (choose fields interactively)
  wf autofill my-workflow --all`,
	Args: cobra.ExactArgs(1),
	RunE: runAutofill,
}

func init() {
	autofillCmd.Flags().Bool("name", false, "fill name field")
	autofillCmd.Flags().Bool("description", false, "fill description field")
	autofillCmd.Flags().Bool("tags", false, "fill tags field")
	autofillCmd.Flags().Bool("args", false, "fill argument descriptions and types")
	autofillCmd.Flags().Bool("all", false, "fill all fields")
}

func runAutofill(cmd *cobra.Command, args []string) error {
	s := getStore()
	wf, err := s.Get(args[0])
	if err != nil {
		return fmt.Errorf("workflow %q not found: %w", args[0], err)
	}

	// Warn if workflow has no command
	if strings.TrimSpace(wf.Command) == "" {
		fmt.Fprintln(os.Stderr, "Warning: this workflow has no command. Consider using 'wf generate' instead.")
		return nil
	}

	gen, err := ai.GetGenerator(cmd.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return nil
	}
	defer gen.Close()

	// Determine which fields to fill
	fields, err := resolveAutofillFields(cmd)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		fmt.Fprintln(os.Stderr, "No fields selected.")
		return nil
	}

	req := ai.AutofillRequest{
		Workflow: wf,
		Fields:   fields,
	}

	fmt.Fprintln(os.Stderr, "Auto-filling metadata...")

	ctx := cmd.Context()
	result, err := gen.Autofill(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error auto-filling metadata: %v\n", err)
		return nil
	}

	// Present results and apply changes
	scanner := bufio.NewScanner(os.Stdin)
	changed := applyAutofillResult(wf, result, scanner)

	if changed == 0 {
		fmt.Println("No changes applied.")
		return nil
	}

	if err := s.Save(wf); err != nil {
		return fmt.Errorf("saving workflow: %w", err)
	}

	fmt.Printf("Updated %s (%d field(s) changed)\n", args[0], changed)
	return nil
}

// resolveAutofillFields determines which fields to fill based on flags or interactive selection.
func resolveAutofillFields(cmd *cobra.Command) ([]string, error) {
	allFlag, _ := cmd.Flags().GetBool("all")
	if allFlag {
		return []string{"name", "description", "tags", "args"}, nil
	}

	nameFlag, _ := cmd.Flags().GetBool("name")
	descFlag, _ := cmd.Flags().GetBool("description")
	tagsFlag, _ := cmd.Flags().GetBool("tags")
	argsFlag, _ := cmd.Flags().GetBool("args")

	var fields []string
	if nameFlag {
		fields = append(fields, "name")
	}
	if descFlag {
		fields = append(fields, "description")
	}
	if tagsFlag {
		fields = append(fields, "tags")
	}
	if argsFlag {
		fields = append(fields, "args")
	}

	// If no flags specified, enter interactive field selection
	if len(fields) == 0 {
		return interactiveFieldSelection()
	}

	return fields, nil
}

// interactiveFieldSelection lets the user choose which fields to fill.
func interactiveFieldSelection() ([]string, error) {
	available := []string{"name", "description", "tags", "args"}

	fmt.Println("Which fields would you like to auto-fill?")
	for i, f := range available {
		fmt.Printf("  %d. %s\n", i+1, f)
	}
	fmt.Print("Select (comma-separated numbers, or 'all'): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no input")
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return nil, nil
	}
	if strings.ToLower(input) == "all" {
		return available, nil
	}

	var fields []string
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(available) {
			continue
		}
		fields = append(fields, available[n-1])
	}

	return fields, nil
}

// applyAutofillResult presents AI suggestions for each field and lets the user
// accept (Enter/y), edit (type new value), or skip (s) each one.
// Returns the number of fields that were changed.
func applyAutofillResult(wf *store.Workflow, result *ai.AutofillResult, scanner *bufio.Scanner) int {
	changed := 0
	fmt.Println()

	// Name
	if result.Name != nil {
		fmt.Printf("Name: %q → %q [accept/edit/skip] (Enter=accept): ", wf.Name, *result.Name)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			switch strings.ToLower(input) {
			case "", "y", "accept":
				wf.Name = *result.Name
				changed++
			case "s", "skip":
				// skip
			default:
				wf.Name = input
				changed++
			}
		}
	}

	// Description
	if result.Description != nil {
		fmt.Printf("Description: %q → %q [accept/edit/skip] (Enter=accept): ", wf.Description, *result.Description)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			switch strings.ToLower(input) {
			case "", "y", "accept":
				wf.Description = *result.Description
				changed++
			case "s", "skip":
				// skip
			default:
				wf.Description = input
				changed++
			}
		}
	}

	// Tags
	if len(result.Tags) > 0 {
		currentTags := strings.Join(wf.Tags, ", ")
		suggestedTags := strings.Join(result.Tags, ", ")
		fmt.Printf("Tags: [%s] → [%s] [accept/edit/skip] (Enter=accept): ", currentTags, suggestedTags)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			switch strings.ToLower(input) {
			case "", "y", "accept":
				wf.Tags = result.Tags
				changed++
			case "s", "skip":
				// skip
			default:
				// Parse comma-separated tags from user input
				var tags []string
				for _, t := range strings.Split(input, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
				wf.Tags = tags
				changed++
			}
		}
	}

	// Args
	if len(result.Args) > 0 {
		fmt.Println("Args:")
		for _, suggested := range result.Args {
			// Find matching existing arg
			found := false
			for i, existing := range wf.Args {
				if existing.Name == suggested.Name {
					found = true
					changes := describeArgChanges(existing, suggested)
					if changes == "" {
						continue
					}
					fmt.Printf("  %s: %s [accept/skip] (Enter=accept): ", existing.Name, changes)
					if scanner.Scan() {
						input := strings.TrimSpace(scanner.Text())
						if input == "" || strings.ToLower(input) == "y" || strings.ToLower(input) == "accept" {
							if suggested.Description != "" {
								wf.Args[i].Description = suggested.Description
							}
							if suggested.Type != "" {
								wf.Args[i].Type = suggested.Type
							}
							if suggested.Default != "" && wf.Args[i].Default == "" {
								wf.Args[i].Default = suggested.Default
							}
							changed++
						}
					}
					break
				}
			}
			if !found {
				// New arg suggested by AI
				fmt.Printf("  %s (new): desc=%q type=%s [accept/skip] (Enter=accept): ",
					suggested.Name, suggested.Description, suggested.Type)
				if scanner.Scan() {
					input := strings.TrimSpace(scanner.Text())
					if input == "" || strings.ToLower(input) == "y" || strings.ToLower(input) == "accept" {
						wf.Args = append(wf.Args, suggested)
						changed++
					}
				}
			}
		}
	}

	return changed
}

// describeArgChanges builds a human-readable description of what would change for an arg.
func describeArgChanges(existing, suggested store.Arg) string {
	var parts []string
	if suggested.Description != "" && existing.Description != suggested.Description {
		parts = append(parts, fmt.Sprintf("desc %q→%q", existing.Description, suggested.Description))
	}
	if suggested.Type != "" && existing.Type != suggested.Type {
		parts = append(parts, fmt.Sprintf("type %q→%q", existing.Type, suggested.Type))
	}
	if suggested.Default != "" && existing.Default == "" {
		parts = append(parts, fmt.Sprintf("default→%q", suggested.Default))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}
