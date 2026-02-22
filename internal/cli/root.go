// Package cli provides the command-line interface for ghent.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/indrasvat/ghent/internal/version"
)

// NewRootCmd creates the root ghent command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ghent",
		Short:   "Agentic PR monitoring for GitHub",
		Long:    "ghent — interactive PR monitoring with TUI for humans and structured output for AI agents.",
		Version: version.String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	cmd.PersistentFlags().Bool("no-tui", false, "force pipe mode even in TTY")
	cmd.PersistentFlags().StringP("format", "f", "json", "output format: json, md, xml")
	cmd.PersistentFlags().Int("pr", 0, "pull request number")

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
