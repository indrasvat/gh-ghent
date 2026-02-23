package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// launchTUI starts the Bubble Tea TUI with the given starting view and data.
// It handles termenv background setup/reset around the program lifecycle.
func launchTUI(startView tui.View, opts ...tuiOption) error {
	cfg := tuiConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	app := tui.NewApp(cfg.repo, cfg.pr, startView)
	if cfg.comments != nil {
		app.SetComments(cfg.comments)
	}
	if cfg.checks != nil {
		app.SetChecks(cfg.checks)
	}
	if cfg.reviews != nil {
		app.SetReviews(cfg.reviews)
	}
	if cfg.resolveFunc != nil {
		app.SetResolver(cfg.resolveFunc)
	}

	// CRITICAL: Set terminal background BEFORE Bubble Tea starts (pitfall 7.1).
	output := styles.SetAppBackground()
	defer styles.ResetAppBackground(output)

	p := tea.NewProgram(app, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	// Extract exit info from the final model.
	if final, ok := finalModel.(tui.App); ok {
		_ = final // Future: extract exit code from model state.
	}

	return nil
}

type tuiConfig struct {
	repo        string
	pr          int
	comments    *domain.CommentsResult
	checks      *domain.ChecksResult
	reviews     []domain.Review
	resolveFunc func(threadID string) error
}

type tuiOption func(*tuiConfig)

func withRepo(repo string) tuiOption {
	return func(c *tuiConfig) { c.repo = repo }
}

func withPR(pr int) tuiOption {
	return func(c *tuiConfig) { c.pr = pr }
}

func withComments(r *domain.CommentsResult) tuiOption {
	return func(c *tuiConfig) { c.comments = r }
}

func withChecks(r *domain.ChecksResult) tuiOption {
	return func(c *tuiConfig) { c.checks = r }
}

func withReviews(r []domain.Review) tuiOption {
	return func(c *tuiConfig) { c.reviews = r }
}

func withResolver(fn func(threadID string) error) tuiOption {
	return func(c *tuiConfig) { c.resolveFunc = fn }
}
