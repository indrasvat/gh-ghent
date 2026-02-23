// Package cli provides the command-line interface for ghent.
package cli

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/debug"
	"github.com/indrasvat/gh-ghent/internal/github"
	"github.com/indrasvat/gh-ghent/internal/version"
)

// Flags holds the resolved global flags for the current invocation.
var Flags GlobalFlags

// ghClient is the GitHub API client, initialized in PersistentPreRunE.
var ghClient *github.Client

// GitHubClient returns the initialized GitHub API client.
func GitHubClient() *github.Client {
	return ghClient
}

// NewRootCmd creates the root ghent command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ghent",
		Short: "Agentic PR monitoring for GitHub",
		Long: `ghent is a GitHub CLI extension for agentic PR monitoring.

Interactive Bubble Tea TUI for humans, structured output (json/md/xml)
for AI agents. Works wherever gh is authenticated — zero config.

  TTY detected  → launches interactive TUI (j/k navigation, tabs, views)
  Piped / no-tui → outputs structured data (default: json)`,
		Example: `  # Interactive TUI for PR #42
  gh ghent comments --pr 42

  # Agent: get unresolved threads as JSON
  gh ghent comments --pr 42 --format json --no-tui

  # Quick merge-readiness check
  gh ghent summary --pr 42 --format json | jq '.is_merge_ready'

  # Watch CI until done, fail-fast on failure
  gh ghent checks --pr 42 --watch`,
		Version: version.String(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Root().PersistentFlags()

			var err error
			Flags.Repo, err = f.GetString("repo")
			if err != nil {
				return err
			}
			Flags.Format, err = f.GetString("format")
			if err != nil {
				return err
			}
			Flags.Verbose, err = f.GetBool("verbose")
			if err != nil {
				return err
			}
			Flags.NoTUI, err = f.GetBool("no-tui")
			if err != nil {
				return err
			}
			Flags.PR, err = f.GetInt("pr")
			if err != nil {
				return err
			}
			Flags.Debug, err = f.GetBool("debug")
			if err != nil {
				return err
			}

			sinceStr, err := f.GetString("since")
			if err != nil {
				return err
			}
			if sinceStr != "" {
				Flags.Since, err = ParseSince(sinceStr)
				if err != nil {
					return err
				}
			}

			// Initialize debug logging: --debug flag or GH_DEBUG env var
			debug.Init(Flags.Debug || os.Getenv("GH_DEBUG") != "")

			// TTY detection via go-gh
			Flags.IsTTY = term.FromEnv().IsTerminalOutput()
			if Flags.NoTUI {
				Flags.IsTTY = false
			}

			// Only initialize GitHub client for subcommands (not root help/version)
			if cmd.Name() != "ghent" {
				ghClient, err = github.New()
				if err != nil {
					return fmt.Errorf("github client: %w", err)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global persistent flags
	cmd.PersistentFlags().StringP("repo", "R", "", "repository in OWNER/REPO format (default: current repo)")
	cmd.PersistentFlags().StringP("format", "f", "json", "output format: json, md, xml (pipe mode)")
	cmd.PersistentFlags().Bool("verbose", false, "show additional context (diff hunks, debug info)")
	cmd.PersistentFlags().Bool("no-tui", false, "force pipe mode even in TTY (for agents)")
	cmd.PersistentFlags().Bool("debug", false, "enable debug logging to stderr")
	cmd.PersistentFlags().Int("pr", 0, "pull request number (required by subcommands)")
	cmd.PersistentFlags().String("since", "", "filter by timestamp (ISO 8601 or relative: 1h, 30m, 2d)")

	// Subcommands
	cmd.AddCommand(
		newCommentsCmd(),
		newChecksCmd(),
		newResolveCmd(),
		newReplyCmd(),
		newSummaryCmd(),
	)

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
