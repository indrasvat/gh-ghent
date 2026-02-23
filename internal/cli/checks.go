package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newChecksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checks",
		Short: "Show CI check status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("checks: not implemented")
		},
	}

	cmd.Flags().Bool("logs", false, "show check run logs")
	cmd.Flags().Bool("watch", false, "watch for check status changes")

	return cmd
}
