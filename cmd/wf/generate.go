package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fredriklanga/wf/internal/ai"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate [description]",
	Short: "Generate a workflow from a natural language description",
	Long: `Uses AI to create a workflow from your description. Requires GitHub Copilot CLI.

Without arguments, enters interactive mode where you describe what you want
and answer clarifying questions. With a description argument, generates directly.

Examples:
  wf generate "deploy to kubernetes with rollback"
  wf generate "git squash last N commits"
  wf generate    (interactive mode)`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringSliceP("tag", "t", nil, "suggested tags (repeatable)")
	generateCmd.Flags().StringP("folder", "f", "", "target folder")
	generateCmd.Flags().StringP("shell", "s", "", "target shell (bash/zsh/fish)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	gen, err := ai.GetGenerator(cmd.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return nil
	}
	defer gen.Close()

	scanner := bufio.NewScanner(os.Stdin)

	tags, _ := cmd.Flags().GetStringSlice("tag")
	folder, _ := cmd.Flags().GetString("folder")
	shell, _ := cmd.Flags().GetString("shell")

	var description string
	if len(args) > 0 {
		// Single-shot mode: join args as description
		description = strings.Join(args, " ")
	} else {
		// Interactive mode
		description, shell, tags = runInteractiveGenerate(scanner)
		if description == "" {
			fmt.Fprintln(os.Stderr, "Error: description is required")
			return nil
		}
	}

	req := ai.GenerateRequest{
		Description: description,
		Tags:        tags,
		Folder:      folder,
		Shell:       shell,
	}

	fmt.Fprintln(os.Stderr, "Generating workflow...")

	ctx := cmd.Context()
	result, err := gen.Generate(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating workflow: %v\n", err)
		return nil
	}

	return presentAndSave(result, scanner, folder)
}

// runInteractiveGenerate asks the user questions when no args are provided.
func runInteractiveGenerate(scanner *bufio.Scanner) (description, shell string, tags []string) {
	fmt.Print("What would you like to automate? ")
	if scanner.Scan() {
		description = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Target shell (bash/zsh/fish, or press Enter to skip): ")
	if scanner.Scan() {
		shell = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Any tags? (comma-separated, or press Enter to skip): ")
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

	return description, shell, tags
}

// presentAndSave shows AI-generated values as editable defaults and saves the result.
func presentAndSave(result *ai.GenerateResult, scanner *bufio.Scanner, folder string) error {
	fmt.Println()
	fmt.Println("--- Generated workflow ---")

	// Name
	name := promptWithDefault(scanner, "Name", result.Name)

	// Command
	command := promptWithDefault(scanner, "Command", result.Command)
	if command == "multi" {
		fmt.Println("Enter command lines (empty line to finish):")
		var lines []string
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				break
			}
			lines = append(lines, line)
		}
		command = strings.Join(lines, "\n")
	}

	// Description
	description := promptWithDefault(scanner, "Description", result.Description)

	// Tags
	tagStr := strings.Join(result.Tags, ", ")
	tagsRaw := promptWithDefault(scanner, "Tags", tagStr)
	var tags []string
	if tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Build workflow
	wf := &store.Workflow{
		Name:        name,
		Command:     command,
		Description: description,
		Tags:        tags,
	}

	// Extract template parameters from command
	params := template.ExtractParams(command)
	for _, p := range params {
		wf.Args = append(wf.Args, store.Arg{
			Name:    p.Name,
			Default: p.Default,
		})
	}

	// Merge AI-suggested args (descriptions, types) into extracted params
	if len(result.Args) > 0 {
		argMap := make(map[string]store.Arg)
		for _, a := range result.Args {
			argMap[a.Name] = a
		}
		for i, a := range wf.Args {
			if suggested, ok := argMap[a.Name]; ok {
				if a.Description == "" {
					wf.Args[i].Description = suggested.Description
				}
				if a.Type == "" && suggested.Type != "" {
					wf.Args[i].Type = suggested.Type
				}
				if a.Default == "" && suggested.Default != "" {
					wf.Args[i].Default = suggested.Default
				}
				if len(a.Options) == 0 && len(suggested.Options) > 0 {
					wf.Args[i].Options = suggested.Options
				}
			}
		}
	}

	// Prepend folder to name if specified
	storeName := name
	if folder != "" {
		storeName = strings.Trim(folder, "/") + "/" + name
		wf.Name = storeName
	}

	// Save
	s := getStore()
	if err := s.Save(wf); err != nil {
		return fmt.Errorf("saving workflow: %w", err)
	}

	fmt.Printf("Created %s\n", name)
	return nil
}

// promptWithDefault shows a field with a default value and lets the user edit or accept.
// Press Enter to accept the default, or type a new value.
func promptWithDefault(scanner *bufio.Scanner, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			return input
		}
	}
	return defaultVal
}
