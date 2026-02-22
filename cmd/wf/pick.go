package main

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredriklanga/wf/internal/picker"
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Launch fuzzy workflow picker",
	Long: `Launch an interactive fuzzy picker to search, select, and fill workflow
parameters. The completed command is printed to stdout for shell capture.

Use with shell integration (eval "$(wf init zsh)") to paste the selected
command directly onto your shell prompt via Ctrl+G.`,
	RunE: runPick,
}

var pickCopy bool

func init() {
	pickCmd.Flags().BoolVarP(&pickCopy, "copy", "c", false, "copy selected command to clipboard instead of printing to stdout")
}

func runPick(cmd *cobra.Command, args []string) error {
	// Load workflows synchronously before creating tea.Program (PICK-02 performance).
	s := getStore()
	workflows, err := s.List()
	if err != nil {
		return fmt.Errorf("loading workflows: %w", err)
	}

	// Handle empty state gracefully.
	if len(workflows) == 0 {
		fmt.Fprintln(os.Stderr, "No workflows found. Use 'wf add' to create one.")
		return nil
	}

	m := picker.New(workflows)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Clean overlay, restores on exit
		tea.WithOutput(os.Stderr), // TUI renders to stderr (Pitfall 1)
	)

	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("picker: %w", err)
	}

	fm, ok := final.(picker.Model)
	if !ok {
		return nil
	}

	// If user cancelled (Esc/Ctrl+C), exit cleanly with no output.
	if fm.Result == "" {
		return nil
	}

	// --copy flag: write to clipboard instead of stdout.
	if pickCopy {
		if err := clipboard.WriteAll(fm.Result); err != nil {
			return fmt.Errorf("clipboard: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Copied to clipboard")
		return nil
	}

	// Default: write to stdout for shell function capture.
	fmt.Fprintln(os.Stdout, fm.Result)
	return nil
}
