// Command shell-demo launches the ghent Bubble Tea app shell with sample data
// for visual verification of view switching, key routing, and status/help bars.
//
// Usage: go run ./cmd/shell-demo
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

func main() {
	output := styles.SetAppBackground()
	defer styles.ResetAppBackground(output)

	app := tui.NewApp("indrasvat/my-project", 42, tui.ViewCommentsList)
	app.SetComments(&domain.CommentsResult{
		PRNumber:        42,
		UnresolvedCount: 5,
		ResolvedCount:   2,
		TotalCount:      7,
	})
	app.SetChecks(&domain.ChecksResult{
		PRNumber:      42,
		HeadSHA:       "a1b2c3d4e5f6789",
		OverallStatus: domain.StatusFail,
		PassCount:     4,
		FailCount:     1,
	})
	app.SetReviews([]domain.Review{
		{Author: "reviewer1", State: domain.ReviewApproved},
	})

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
