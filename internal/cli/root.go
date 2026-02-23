// Package cli provides the command-line interface for ghent.
package cli

import (
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"

	"github.com/indrasvat/ghent/internal/version"
)

// Flags holds the resolved global flags for the current invocation.
var Flags GlobalFlags

// NewRootCmd creates the root ghent command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ghent",
		Short:   "Agentic PR monitoring for GitHub",
		Long:    "ghent — interactive PR monitoring with TUI for humans and structured output for AI agents.",
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

			// TTY detection via go-gh
			Flags.IsTTY = term.FromEnv().IsTerminalOutput()
			if Flags.NoTUI {
				Flags.IsTTY = false
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
	cmd.PersistentFlags().StringP("repo", "R", "", "repository in OWNER/REPO format")
	cmd.PersistentFlags().StringP("format", "f", "json", "output format: json, md, xml")
	cmd.PersistentFlags().Bool("verbose", false, "verbose output")
	cmd.PersistentFlags().Bool("no-tui", false, "force pipe mode even in TTY")
	cmd.PersistentFlags().Int("pr", 0, "pull request number")

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
