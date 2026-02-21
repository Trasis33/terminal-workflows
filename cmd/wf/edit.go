package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit a workflow",
	Long: `Edit a workflow by name.

Without flags, opens the workflow YAML file in $EDITOR (or vi).
With flags, updates specific fields without opening an editor.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().StringP("command", "c", "", "replace command template")
	editCmd.Flags().StringP("description", "d", "", "replace description")
	editCmd.Flags().StringSliceP("tag", "t", nil, "replace all tags")
	editCmd.Flags().String("add-tag", "", "add a single tag")
	editCmd.Flags().String("remove-tag", "", "remove a single tag")
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]
	s := getStore()

	// Check if any update flags were provided
	hasFlags := cmd.Flags().Changed("command") ||
		cmd.Flags().Changed("description") ||
		cmd.Flags().Changed("tag") ||
		cmd.Flags().Changed("add-tag") ||
		cmd.Flags().Changed("remove-tag")

	if hasFlags {
		return runQuickEdit(cmd, s, name)
	}

	return runEditorEdit(s, name)
}

// runQuickEdit updates specific fields via flags without opening an editor.
func runQuickEdit(cmd *cobra.Command, s *store.YAMLStore, name string) error {
	wf, err := s.Get(name)
	if err != nil {
		return fmt.Errorf("workflow %q not found", name)
	}

	if cmd.Flags().Changed("command") {
		command, _ := cmd.Flags().GetString("command")
		wf.Command = command

		// Re-extract params when command changes
		params := template.ExtractParams(command)
		wf.Args = nil
		for _, p := range params {
			wf.Args = append(wf.Args, store.Arg{
				Name:    p.Name,
				Default: p.Default,
			})
		}
	}

	if cmd.Flags().Changed("description") {
		description, _ := cmd.Flags().GetString("description")
		wf.Description = description
	}

	if cmd.Flags().Changed("tag") {
		tags, _ := cmd.Flags().GetStringSlice("tag")
		wf.Tags = tags
	}

	if cmd.Flags().Changed("add-tag") {
		tag, _ := cmd.Flags().GetString("add-tag")
		// Only add if not already present
		found := false
		for _, t := range wf.Tags {
			if t == tag {
				found = true
				break
			}
		}
		if !found {
			wf.Tags = append(wf.Tags, tag)
		}
	}

	if cmd.Flags().Changed("remove-tag") {
		tag, _ := cmd.Flags().GetString("remove-tag")
		filtered := make([]string, 0, len(wf.Tags))
		for _, t := range wf.Tags {
			if t != tag {
				filtered = append(filtered, t)
			}
		}
		wf.Tags = filtered
	}

	if err := s.Save(wf); err != nil {
		return fmt.Errorf("saving workflow: %w", err)
	}

	fmt.Printf("Updated %s\n", name)
	return nil
}

// runEditorEdit opens the workflow YAML in $EDITOR for editing.
func runEditorEdit(s *store.YAMLStore, name string) error {
	wf, err := s.Get(name)
	if err != nil {
		return fmt.Errorf("workflow %q not found", name)
	}

	// Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Get file path for the workflow
	fpath := s.WorkflowPath(name)

	// Open editor
	editorCmd := exec.Command(editor, fpath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	// Re-read and validate the file after editing
	data, err := os.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("reading edited file: %w", err)
	}

	var edited store.Workflow
	if err := yaml.Unmarshal(data, &edited); err != nil {
		return fmt.Errorf("invalid YAML after editing: %w\nPlease re-run 'wf edit %s' to fix", err, name)
	}

	// Suppress unused variable warning - validation passed
	_ = wf

	fmt.Printf("Updated %s\n", name)
	return nil
}
