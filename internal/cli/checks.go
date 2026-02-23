package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
	ghub "github.com/indrasvat/gh-ghent/internal/github"
	"github.com/indrasvat/gh-ghent/internal/tui"
)

func newChecksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checks",
		Short: "Show CI check status",
		Long: `Show CI check runs, their status, and annotations for a pull request.

In TTY mode, launches an interactive TUI with check list, annotation
details, and log viewer. In pipe mode, outputs structured data with
check names, statuses, and annotations.

Use --logs to include failing job log excerpts in pipe output.
Use --watch to poll until all checks complete (fail-fast on failure).

Exit codes: 0 = all pass, 1 = failure, 3 = pending.`,
		Example: `  # Interactive TUI
  gh ghent checks --pr 42

  # JSON status for agents
  gh ghent checks --pr 42 --format json --no-tui

  # Include error logs for failed checks
  gh ghent checks --pr 42 --format json --logs

  # Wait for CI to finish (fail-fast)
  gh ghent checks --pr 42 --watch

  # Check overall status
  gh ghent checks --pr 42 --format json | jq '.overall_status'`,
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

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			// Watch mode: poll until terminal status.
			watch, _ := cmd.Flags().GetBool("watch")
			if watch {
				finalStatus, watchErr := client.WatchChecks(
					ctx, os.Stdout, f,
					owner, repo, Flags.PR,
					ghub.DefaultPollInterval, nil,
				)
				if watchErr != nil {
					return fmt.Errorf("watch checks: %w", watchErr)
				}
				switch finalStatus {
				case domain.StatusFail:
					os.Exit(1)
				case domain.StatusPending:
					os.Exit(3)
				}
				return nil
			}

			result, err := client.FetchChecks(ctx, owner, repo, Flags.PR)
			if err != nil {
				return fmt.Errorf("fetch checks: %w", err)
			}

			// TTY → launch TUI; non-TTY / --no-tui → pipe mode.
			if Flags.IsTTY {
				// Pre-fetch logs for failed checks for the TUI log viewer.
				for i := range result.Checks {
					ch := &result.Checks[i]
					if ch.Conclusion != "failure" {
						continue
					}
					logText, logErr := client.FetchJobLog(ctx, owner, repo, ch.ID)
					if logErr != nil {
						continue // graceful degradation
					}
					ch.LogExcerpt = ghub.ExtractErrorLines(logText)
				}
				repoStr := owner + "/" + repo
				return launchTUI(tui.ViewChecksList,
					withRepo(repoStr), withPR(Flags.PR),
					withChecks(result),
				)
			}

			// Fetch logs for failed checks when --logs is set
			withLogs, _ := cmd.Flags().GetBool("logs")
			if withLogs {
				for i := range result.Checks {
					ch := &result.Checks[i]
					if ch.Conclusion != "failure" {
						continue
					}
					logText, logErr := client.FetchJobLog(ctx, owner, repo, ch.ID)
					if logErr != nil {
						continue // graceful degradation
					}
					ch.LogExcerpt = ghub.ExtractErrorLines(logText)
				}
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

	cmd.Flags().Bool("logs", false, "include failing job log excerpts in output")
	cmd.Flags().Bool("watch", false, "poll until all checks complete, fail-fast on failure")

	return cmd
}
