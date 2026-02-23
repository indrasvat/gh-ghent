package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/formatter"
)

func newReplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reply",
		Short: "Reply to a review thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			if Flags.PR == 0 {
				return fmt.Errorf("--pr flag is required")
			}

			threadID, err := cmd.Flags().GetString("thread")
			if err != nil {
				return err
			}

			body, err := resolveBody(cmd)
			if err != nil {
				return err
			}

			owner, repo, err := resolveRepo(Flags.Repo)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			client := GitHubClient()

			result, err := client.ReplyToThread(ctx, owner, repo, Flags.PR, threadID, body)
			if err != nil {
				// Exit code 1 for thread not found / can't reply
				if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "cannot reply") {
					fmt.Fprintf(os.Stderr, "Error: %s\n", err)
					os.Exit(1)
				}
				// Exit code 2 for other errors
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(2)
			}

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			if err := f.FormatReply(os.Stdout, result); err != nil {
				return fmt.Errorf("format output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().String("thread", "", "thread ID to reply to")
	cmd.Flags().String("body", "", "reply body text")
	cmd.Flags().String("body-file", "", "read reply body from file (use - for stdin)")
	_ = cmd.MarkFlagRequired("thread")

	return cmd
}

// resolveBody reads the reply body from --body or --body-file flags.
// The two flags are mutually exclusive; at least one must be set.
func resolveBody(cmd *cobra.Command) (string, error) {
	body, err := cmd.Flags().GetString("body")
	if err != nil {
		return "", err
	}
	bodyFile, err := cmd.Flags().GetString("body-file")
	if err != nil {
		return "", err
	}

	if body != "" && bodyFile != "" {
		return "", fmt.Errorf("--body and --body-file are mutually exclusive")
	}
	if body == "" && bodyFile == "" {
		return "", fmt.Errorf("either --body or --body-file is required")
	}

	if body != "" {
		return body, nil
	}

	// Read from file or stdin
	if bodyFile == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}

	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return "", fmt.Errorf("read body file %q: %w", bodyFile, err)
	}
	return strings.TrimRight(string(data), "\n"), nil
}
