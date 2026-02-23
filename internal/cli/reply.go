package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reply",
		Short: "Reply to a review thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("reply: not implemented")
		},
	}

	cmd.Flags().String("thread", "", "thread ID to reply to")
	cmd.Flags().String("body", "", "reply body text")
	cmd.Flags().String("body-file", "", "read reply body from file")
	_ = cmd.MarkFlagRequired("thread")

	return cmd
}
