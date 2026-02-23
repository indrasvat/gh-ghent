package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/indrasvat/ghent/internal/formatter"
)

func newCommentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Show unresolved review threads",
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

			result, err := client.FetchThreads(ctx, owner, repo, Flags.PR)
			if err != nil {
				return fmt.Errorf("fetch threads: %w", err)
			}

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			if err := f.FormatComments(os.Stdout, result); err != nil {
				return fmt.Errorf("format output: %w", err)
			}

			if result.UnresolvedCount > 0 {
				os.Exit(1)
			}

			return nil
		},
	}

	return cmd
}
