package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fredriklanga/wf/internal/config"
	"github.com/fredriklanga/wf/internal/source"
	"github.com/spf13/cobra"
)

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage remote workflow sources",
	Long: `Manage remote git repositories as workflow sources.

Add remote repos to make their workflows available alongside your local ones
in picker, manage, and list commands.`,
}

var sourceNameFlag string

var sourceAddCmd = &cobra.Command{
	Use:   "add <git-url>",
	Short: "Add a remote workflow source",
	Long: `Clone a git repository and register it as a workflow source.

The repository name is used as the alias by default. Use --name to specify
a custom alias. Remote workflows appear with the alias prefix (e.g., "team/deploy").`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureSourcesDir(); err != nil {
			return err
		}
		mgr := source.NewManager(config.SourcesDir())
		url := args[0]
		if err := mgr.Add(cmd.Context(), url, sourceNameFlag); err != nil {
			return err
		}
		// Determine the effective alias for display
		alias := sourceNameFlag
		if alias == "" {
			// Re-read the list to find the alias that was added
			sources := mgr.List()
			if len(sources) > 0 {
				alias = sources[len(sources)-1].Alias
			}
		}
		fmt.Fprintf(os.Stderr, "Added source %q from %s\n", alias, url)
		return nil
	},
}

var sourceRemoveCmd = &cobra.Command{
	Use:   "remove <alias>",
	Short: "Remove a remote workflow source",
	Long:  `Remove a previously added remote source and delete its cloned repository.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		mgr := source.NewManager(config.SourcesDir())
		sources := mgr.List()
		aliases := make([]string, len(sources))
		for i, s := range sources {
			aliases[i] = s.Alias
		}
		return aliases, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureSourcesDir(); err != nil {
			return err
		}
		mgr := source.NewManager(config.SourcesDir())
		if err := mgr.Remove(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed source %q\n", args[0])
		return nil
	},
}

var sourceUpdateCmd = &cobra.Command{
	Use:   "update [alias]",
	Short: "Update remote sources",
	Long: `Pull the latest changes from remote sources.

If an alias is provided, only that source is updated. Otherwise all sources
are updated. A diff summary shows what changed (added, removed, updated workflows).`,
	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		mgr := source.NewManager(config.SourcesDir())
		sources := mgr.List()
		aliases := make([]string, len(sources))
		for i, s := range sources {
			aliases[i] = s.Alias
		}
		return aliases, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureSourcesDir(); err != nil {
			return err
		}
		mgr := source.NewManager(config.SourcesDir())
		ctx := cmd.Context()

		if len(args) == 1 {
			return updateSource(ctx, mgr, args[0])
		}

		// Update all sources
		sources := mgr.List()
		if len(sources) == 0 {
			fmt.Fprintln(os.Stderr, "No remote sources configured. Use 'wf source add <git-url>' to add one.")
			return nil
		}
		for _, s := range sources {
			if err := updateSource(ctx, mgr, s.Alias); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating %q: %v\n", s.Alias, err)
			}
		}
		return nil
	},
}

func updateSource(ctx context.Context, mgr *source.Manager, alias string) error {
	result, err := mgr.Update(ctx, alias)
	if err != nil {
		return err
	}

	total := len(result.Added) + len(result.Removed) + len(result.Updated)
	if total == 0 {
		fmt.Fprintf(os.Stderr, "Source %q: already up to date\n", alias)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Source %q: +%d new, -%d removed, ~%d updated\n",
		alias, len(result.Added), len(result.Removed), len(result.Updated))
	return nil
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured remote sources",
	Long:  `Display all configured remote workflow sources with their alias, URL, and last update time.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureSourcesDir(); err != nil {
			return err
		}
		mgr := source.NewManager(config.SourcesDir())
		sources := mgr.List()

		if len(sources) == 0 {
			fmt.Fprintln(os.Stderr, "No remote sources configured. Use 'wf source add <git-url>' to add one.")
			return nil
		}

		for _, s := range sources {
			updated := formatRelativeTime(s.UpdatedAt)
			fmt.Fprintf(cmd.OutOrStdout(), "%-16s  %s  (%s)\n", s.Alias, s.URL, updated)
		}
		return nil
	},
}

// formatRelativeTime returns a human-friendly relative time string.
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		if days < 30 {
			return fmt.Sprintf("%d days ago", days)
		}
		return t.Format("2006-01-02")
	}
}

func init() {
	sourceAddCmd.Flags().StringVar(&sourceNameFlag, "name", "", "custom alias for the source")
	sourceCmd.AddCommand(sourceAddCmd, sourceRemoveCmd, sourceUpdateCmd, sourceListCmd)
}
