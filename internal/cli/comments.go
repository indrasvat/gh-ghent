package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
	"github.com/indrasvat/gh-ghent/internal/tui"
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

			// Apply --since filter (no-op if not set).
			FilterThreadsBySince(result, Flags.Since)

			// TTY → launch TUI; non-TTY / --no-tui → pipe mode.
			if Flags.IsTTY {
				repoStr := owner + "/" + repo
				return launchTUI(tui.ViewCommentsList,
					withRepo(repoStr), withPR(Flags.PR),
					withComments(result),
				)
			}

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			groupBy, _ := cmd.Flags().GetString("group-by")
			if groupBy != "" {
				grouped, groupErr := groupThreads(result, groupBy)
				if groupErr != nil {
					return groupErr
				}
				if err := f.FormatGroupedComments(os.Stdout, grouped); err != nil {
					return fmt.Errorf("format output: %w", err)
				}
			} else {
				if err := f.FormatComments(os.Stdout, result); err != nil {
					return fmt.Errorf("format output: %w", err)
				}
			}

			if result.UnresolvedCount > 0 {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().String("group-by", "", "group threads by: file, author, status")

	return cmd
}

// groupThreads groups threads from a CommentsResult by the given dimension.
func groupThreads(result *domain.CommentsResult, groupBy string) (*domain.GroupedCommentsResult, error) {
	grouped := &domain.GroupedCommentsResult{
		PRNumber:        result.PRNumber,
		GroupBy:         groupBy,
		TotalCount:      result.TotalCount,
		ResolvedCount:   result.ResolvedCount,
		UnresolvedCount: result.UnresolvedCount,
	}

	keyFunc, err := threadKeyFunc(groupBy)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string][]domain.ReviewThread)
	var keys []string
	for _, t := range result.Threads {
		k := keyFunc(t)
		if _, exists := groupMap[k]; !exists {
			keys = append(keys, k)
		}
		groupMap[k] = append(groupMap[k], t)
	}

	switch groupBy {
	case "file", "author":
		sort.Strings(keys)
	case "status":
		// unresolved first
		sort.Slice(keys, func(i, j int) bool {
			if keys[i] == "unresolved" {
				return true
			}
			if keys[j] == "unresolved" {
				return false
			}
			return keys[i] < keys[j]
		})
	}

	for _, k := range keys {
		grouped.Groups = append(grouped.Groups, domain.CommentGroup{
			Key:     k,
			Threads: groupMap[k],
		})
	}

	return grouped, nil
}

// threadKeyFunc returns a function that extracts a grouping key from a ReviewThread.
func threadKeyFunc(groupBy string) (func(domain.ReviewThread) string, error) {
	switch groupBy {
	case "file":
		return func(t domain.ReviewThread) string { return t.Path }, nil
	case "author":
		return func(t domain.ReviewThread) string {
			if len(t.Comments) > 0 {
				return t.Comments[0].Author
			}
			return "unknown"
		}, nil
	case "status":
		return func(t domain.ReviewThread) string {
			if t.IsResolved {
				return "resolved"
			}
			return "unresolved"
		}, nil
	default:
		return nil, fmt.Errorf("invalid --group-by value %q: must be file, author, or status", groupBy)
	}
}
