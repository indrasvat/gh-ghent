package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "PR status dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("summary: not implemented")
		},
	}

	return cmd
}
