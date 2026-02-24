package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fredriklanga/wf/internal/history"
	"github.com/fredriklanga/wf/internal/register"
	"github.com/fredriklanga/wf/internal/store"
	"github.com/fredriklanga/wf/internal/template"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register [command...]",
	Short: "Save a previous command as a workflow",
	Long: `Register a command as a reusable workflow.

Usage patterns:
  wf register                           Capture last command from shell history
  wf register 'docker run -p 8080:80'   Register a specific command directly
  wf register --pick                    Browse recent history entries to select one

Auto-detects potential parameters (IPs, ports, paths, URLs) and lets you
convert them to {{named}} template parameters before saving.`,
	RunE: runRegister,
}

func init() {
	registerCmd.Flags().Bool("pick", false, "browse recent shell history entries")
}

func runRegister(cmd *cobra.Command, args []string) error {
	pick, _ := cmd.Flags().GetBool("pick")
	scanner := bufio.NewScanner(os.Stdin)

	var command string

	switch {
	case pick:
		// Browse history entries
		c, err := pickFromHistory(scanner)
		if err != nil {
			return err
		}
		command = c

	case len(args) > 0:
		// Direct command input
		command = strings.Join(args, " ")

	default:
		// Grab last command from history
		c, err := lastFromHistory()
		if err != nil {
			return err
		}
		command = c
	}

	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("no command to register")
	}

	fmt.Printf("Captured: %s\n", command)

	// Auto-detect parameters
	command = applyDetectedParams(command, scanner)

	// Collect metadata
	name, description, tags, err := collectMetadata(scanner)
	if err != nil {
		return err
	}

	// Build workflow
	wf := &store.Workflow{
		Name:        name,
		Command:     command,
		Description: description,
		Tags:        tags,
	}

	// Auto-extract parameters from command template
	params := template.ExtractParams(command)
	for _, p := range params {
		wf.Args = append(wf.Args, store.Arg{
			Name:    p.Name,
			Default: p.Default,
		})
	}

	// Save workflow
	s := getStore()
	if err := s.Save(wf); err != nil {
		return fmt.Errorf("saving workflow: %w", err)
	}

	fmt.Printf("Created %s\n", name)
	return nil
}

func pickFromHistory(scanner *bufio.Scanner) (string, error) {
	reader, err := history.NewReader()
	if err != nil {
		return "", fmt.Errorf("reading shell history: %w\nTip: use 'wf register <command>' to register a command directly", err)
	}

	entries, err := reader.LastN(15)
	if err != nil {
		return "", fmt.Errorf("reading history entries: %w", err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no history found\nTip: use 'wf register <command>' to register a command directly")
	}

	fmt.Println("Recent commands:")
	for i, e := range entries {
		fmt.Printf("  %2d. %s\n", i+1, e.Command)
	}

	fmt.Print("Select number: ")
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}

	input := strings.TrimSpace(scanner.Text())
	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(entries) {
		return "", fmt.Errorf("invalid selection: %s", input)
	}

	return entries[n-1].Command, nil
}

func lastFromHistory() (string, error) {
	// Priority 1: Read sidecar file (written by shell integration hooks)
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	sidecarPath := filepath.Join(dataHome, "wf", "last_cmd")

	if data, err := os.ReadFile(sidecarPath); err == nil {
		cmd := strings.TrimSpace(string(data))
		if cmd != "" {
			return cmd, nil
		}
	}

	// Priority 2: Fall back to $HISTFILE with warning
	fmt.Fprintln(os.Stderr, "Warning: shell integration not active — reading $HISTFILE which may not contain your most recent command.")
	fmt.Fprintln(os.Stderr, "Tip: run 'eval \"$(wf init zsh)\"' (or bash/fish) in your shell config to enable accurate history capture.")

	reader, err := history.NewReader()
	if err != nil {
		return "", fmt.Errorf("reading shell history: %w\nTip: use 'wf register <command>' to register a command directly", err)
	}

	entry, err := reader.Last()
	if err != nil {
		return "", fmt.Errorf("no history found\nTip: use 'wf register <command>' to register a command directly")
	}

	return entry.Command, nil
}

func applyDetectedParams(command string, scanner *bufio.Scanner) string {
	suggestions := register.DetectParams(command)
	if len(suggestions) == 0 {
		return command
	}

	fmt.Println("Detected potential parameters:")
	for i, s := range suggestions {
		fmt.Printf("  %d. %s → {{%s}}\n", i+1, s.Original, s.ParamName)
	}

	fmt.Print("Apply all? (y/n/select numbers e.g. 1,3): ")
	if !scanner.Scan() {
		return command
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" || strings.ToLower(input) == "n" {
		return command
	}

	var selected []int
	if strings.ToLower(input) == "y" {
		for i := range suggestions {
			selected = append(selected, i)
		}
	} else {
		// Parse comma-separated numbers
		for _, part := range strings.Split(input, ",") {
			part = strings.TrimSpace(part)
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > len(suggestions) {
				continue
			}
			selected = append(selected, n-1)
		}
	}

	if len(selected) == 0 {
		return command
	}

	// Apply substitutions in reverse order (to preserve positions)
	// First, sort selected indices by position descending
	sortedSel := make([]int, len(selected))
	copy(sortedSel, selected)
	for i := 0; i < len(sortedSel)-1; i++ {
		for j := i + 1; j < len(sortedSel); j++ {
			if suggestions[sortedSel[i]].Start < suggestions[sortedSel[j]].Start {
				sortedSel[i], sortedSel[j] = sortedSel[j], sortedSel[i]
			}
		}
	}

	result := command
	for _, idx := range sortedSel {
		s := suggestions[idx]
		replacement := "{{" + s.ParamName + "}}"
		result = result[:s.Start] + replacement + result[s.End:]
	}

	fmt.Printf("Command: %s\n", result)
	return result
}

func collectMetadata(scanner *bufio.Scanner) (name, description string, tags []string, err error) {
	fmt.Print("Workflow name: ")
	if !scanner.Scan() {
		return "", "", nil, fmt.Errorf("no input")
	}
	name = strings.TrimSpace(scanner.Text())
	if name == "" {
		return "", "", nil, fmt.Errorf("workflow name is required")
	}

	fmt.Print("Description (optional): ")
	if scanner.Scan() {
		description = strings.TrimSpace(scanner.Text())
	}

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

	return name, description, tags, nil
}
