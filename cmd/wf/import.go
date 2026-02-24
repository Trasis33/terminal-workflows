package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fredriklanga/wf/internal/importer"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import workflows from external formats",
	Long: `Import workflows from Pet TOML or Warp YAML files.

Supported formats:
  wf import pet <file>   — Import from Pet TOML snippet file
  wf import warp <file>  — Import from Warp YAML workflow file

By default, a dry-run preview is shown before importing.
Use --force to skip the preview and import directly.`,
}

var importPetCmd = &cobra.Command{
	Use:   "pet <file>",
	Short: "Import workflows from a Pet TOML snippet file",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportPet,
}

var importWarpCmd = &cobra.Command{
	Use:   "warp <file>",
	Short: "Import workflows from a Warp YAML workflow file",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportWarp,
}

func init() {
	importCmd.PersistentFlags().Bool("force", false, "skip preview, import directly")
	importCmd.PersistentFlags().String("folder", "", "target folder for imported workflows")
	importCmd.AddCommand(importPetCmd)
	importCmd.AddCommand(importWarpCmd)
}

func runImportPet(cmd *cobra.Command, args []string) error {
	imp := &importer.PetImporter{}
	return runImport(cmd, args[0], imp, "Pet")
}

func runImportWarp(cmd *cobra.Command, args []string) error {
	imp := &importer.WarpImporter{}
	return runImport(cmd, args[0], imp, "Warp")
}

// importWorkflowEntry tracks a workflow through the import pipeline.
type importWorkflowEntry struct {
	workflow store.Workflow
	conflict bool
}

func runImport(cmd *cobra.Command, filePath string, imp importer.Importer, formatName string) error {
	force, _ := cmd.Flags().GetBool("force")
	folder, _ := cmd.Flags().GetString("folder")

	// 1. Open and read the source file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening %s file: %w", formatName, err)
	}
	defer f.Close()

	// 2. Import
	result, err := imp.Import(f)
	if err != nil {
		return fmt.Errorf("parsing %s file: %w", formatName, err)
	}

	// 3. Report parse results
	fmt.Printf("Parsed %d workflows (%d warnings, %d errors)\n",
		len(result.Workflows), len(result.Warnings), len(result.Errors))

	for _, e := range result.Errors {
		fmt.Printf("  Error: %s\n", e)
	}

	if len(result.Workflows) == 0 {
		return fmt.Errorf("no workflows to import")
	}

	// 4. Prepare entries with folder prefix and conflict detection
	s := getStore()
	entries := make([]importWorkflowEntry, 0, len(result.Workflows))

	for _, wf := range result.Workflows {
		if folder != "" {
			wf.Name = strings.Trim(folder, "/") + "/" + wf.Name
		}
		existing, err := s.Get(wf.Name)
		conflict := err == nil && existing != nil
		entries = append(entries, importWorkflowEntry{
			workflow: wf,
			conflict: conflict,
		})
	}

	// 5. Preview and conflict resolution (unless --force)
	if !force {
		scanner := bufio.NewScanner(os.Stdin)

		// Print preview table
		fmt.Println("\nImport preview:")
		for _, entry := range entries {
			status := "new"
			if entry.conflict {
				status = "conflict"
			}
			fmt.Printf("  %-50s %s\n", entry.workflow.Name, status)
		}

		// Print warnings
		if len(result.Warnings) > 0 {
			fmt.Println("\nWarnings (unmappable fields):")
			for _, w := range result.Warnings {
				fmt.Printf("  %s\n", w)
			}
		}

		// Handle conflicts interactively
		var toRemove []int
		for i, entry := range entries {
			if !entry.conflict {
				continue
			}

			fmt.Printf("\nConflict: '%s' already exists. [s]kip / [r]ename / [o]verwrite: ", entry.workflow.Name)
			if !scanner.Scan() {
				return fmt.Errorf("input cancelled")
			}
			choice := strings.TrimSpace(strings.ToLower(scanner.Text()))

			switch {
			case strings.HasPrefix(choice, "s"):
				toRemove = append(toRemove, i)
			case strings.HasPrefix(choice, "r"):
				fmt.Print("New name: ")
				if !scanner.Scan() {
					return fmt.Errorf("input cancelled")
				}
				newName := strings.TrimSpace(scanner.Text())
				if newName == "" {
					return fmt.Errorf("name cannot be empty")
				}
				if folder != "" && !strings.HasPrefix(newName, strings.Trim(folder, "/")+"/") {
					newName = strings.Trim(folder, "/") + "/" + newName
				}
				entries[i].workflow.Name = newName
			case strings.HasPrefix(choice, "o"):
				// Keep in import list — will overwrite
			default:
				return fmt.Errorf("invalid choice: %s", choice)
			}
		}

		// Remove skipped entries (in reverse order to keep indices stable)
		for i := len(toRemove) - 1; i >= 0; i-- {
			idx := toRemove[i]
			entries = append(entries[:idx], entries[idx+1:]...)
		}

		if len(entries) == 0 {
			fmt.Println("No workflows to import after conflict resolution.")
			return nil
		}

		// Confirm
		fmt.Printf("\nProceed with import of %d workflows? (y/n): ", len(entries))
		if !scanner.Scan() {
			return fmt.Errorf("input cancelled")
		}
		confirm := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if !strings.HasPrefix(confirm, "y") {
			fmt.Println("Import cancelled.")
			return nil
		}
	}

	// 6. Save workflows
	// Build a map of warnings keyed by both the original identifier (quoted name
	// in warning string) and the slugified name (for Pet, where warnings use
	// description but workflow Name is slugified).
	warnMap := buildWarningMap(result.Warnings)

	imported := 0
	for _, entry := range entries {
		wf := &entry.workflow
		if err := s.Save(wf); err != nil {
			fmt.Printf("  Error saving '%s': %s\n", wf.Name, err)
			continue
		}

		// Inject unmappable fields as YAML comments.
		// Try matching by workflow name (Warp) or by the original name in
		// the warning string (which buildWarningMap indexes by both raw and
		// slugified keys).
		warns := warnMap[wf.Name]
		if len(warns) == 0 {
			// For Pet: warnings are keyed by description; also try
			// the description field directly.
			warns = warnMap[wf.Description]
		}
		if len(warns) > 0 {
			injectComments(s, wf.Name, formatName, warns)
		}

		imported++
	}

	fmt.Printf("Imported %d workflows\n", imported)
	return nil
}

// buildWarningMap groups warnings by workflow name/description.
// Warning format is: '<format> <type> "<name>": <field>: <value>'
// The quoted name may be either the workflow name (Warp) or the snippet
// description (Pet). We index by both the raw name and its slugified form.
func buildWarningMap(warnings []string) map[string][]string {
	m := make(map[string][]string)
	for _, w := range warnings {
		// Extract the name from the warning string: look for quoted name
		// Format: 'pet snippet "name": ...' or 'warp workflow "name": ...'
		start := strings.Index(w, `"`)
		if start < 0 {
			continue
		}
		end := strings.Index(w[start+1:], `"`)
		if end < 0 {
			continue
		}
		name := w[start+1 : start+1+end]

		// Extract the field:value part after the closing quote + ": "
		rest := w[start+1+end+1:]
		rest = strings.TrimPrefix(rest, ": ")

		m[name] = append(m[name], rest)
	}
	return m
}

// injectComments reads the saved YAML file, prepends unmappable field comments, and writes it back.
func injectComments(s *store.YAMLStore, name string, formatName string, warns []string) {
	fpath := s.WorkflowPath(name)
	data, err := os.ReadFile(fpath)
	if err != nil {
		return // Best effort — don't fail import on comment injection
	}

	var header strings.Builder
	header.WriteString(fmt.Sprintf("# Imported from %s — unmappable fields:\n", formatName))
	for _, w := range warns {
		header.WriteString(fmt.Sprintf("# %s\n", w))
	}

	out := header.String() + string(data)
	_ = os.WriteFile(fpath, []byte(out), 0644) // Best effort
}
