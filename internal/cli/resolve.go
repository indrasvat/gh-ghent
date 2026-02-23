package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve review threads",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("resolve: not implemented")
		},
	}

	cmd.Flags().String("thread", "", "thread ID to resolve")
	cmd.Flags().Bool("all", false, "resolve all threads")
	cmd.Flags().Bool("unresolve", false, "unresolve instead of resolve")

	return cmd
}
