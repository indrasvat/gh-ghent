package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/formatter"
)

func newCommentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Show unresolved review threads",
		Long: `Show unresolved review threads for a pull request.

In TTY mode, launches an interactive TUI with thread navigation,
diff hunks, and expandable comment chains. In pipe mode, outputs
structured data with thread IDs, file paths, and comment bodies.

Exit codes: 0 = no unresolved threads, 1 = has unresolved threads.`,
		Example: `  # Interactive TUI
  gh ghent comments --pr 42

  # JSON for agents (thread IDs, file:line, bodies)
  gh ghent comments --pr 42 --format json --no-tui

  # Count unresolved threads
  gh ghent comments --pr 42 --format json | jq '.unresolved_count'

  # Markdown summary
  gh ghent comments -R owner/repo --pr 42 --format md`,
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
