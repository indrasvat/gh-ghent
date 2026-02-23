package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/indrasvat/ghent/internal/domain"
	"github.com/indrasvat/ghent/internal/formatter"
)

func newChecksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checks",
		Short: "Show CI check status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if Flags.PR == 0 {
				return fmt.Errorf("--pr flag is required")
			}

			owner, repo, err := resolveRepo(Flags.Repo)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			client := GitHubClient()

			result, err := client.FetchChecks(ctx, owner, repo, Flags.PR)
			if err != nil {
				return fmt.Errorf("fetch checks: %w", err)
			}

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			if err := f.FormatChecks(os.Stdout, result); err != nil {
				return fmt.Errorf("format output: %w", err)
			}

			// Exit codes per PRD: 0=pass, 1=fail, 3=pending
			switch result.OverallStatus {
			case domain.StatusFail:
				os.Exit(1)
			case domain.StatusPending:
				os.Exit(3)
			}

			return nil
		},
	}

	cmd.Flags().Bool("logs", false, "show check run logs")
	cmd.Flags().Bool("watch", false, "watch for check status changes")

	return cmd
}
