package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCommentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Show unresolved review threads",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("comments: not implemented")
		},
	}

	return cmd
}
