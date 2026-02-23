// Command theme-demo renders sample styled elements for visual verification
// of the Tokyo Night theme and Lipgloss styles.
//
// Usage: go run ./cmd/theme-demo
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/tui/components"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

func main() {
	// Set terminal background
	output := styles.SetAppBackground()
	defer styles.ResetAppBackground(output)

	w := 78

	fmt.Println()
	fmt.Println(styles.BadgeBlue.Render("ghent") + "  Theme Demo — Tokyo Night")
	fmt.Println(strings.Repeat("─", w))
	fmt.Println()

	// ── Badges ──
	fmt.Println("  Badges:")
	fmt.Print("    ")
	fmt.Print(styles.BadgeBlue.Render("PR #42"))
	fmt.Print("  ")
	fmt.Print(styles.BadgeGreen.Render("4 passed"))
	fmt.Print("  ")
	fmt.Print(styles.BadgeRed.Render("1 failed"))
	fmt.Print("  ")
	fmt.Print(styles.BadgeYellow.Render("pending"))
	fmt.Print("  ")
	fmt.Print(styles.BadgePurple.Render("PRRT_kwDON1..."))
	fmt.Println()
	fmt.Println()

	// ── Status Icons ──
	fmt.Println("  Check Status:")
	fmt.Println("    " + styles.CheckPass.Render("✓ passed") + "  " +
		styles.CheckFail.Render("✗ failed") + "  " +
		styles.CheckPending.Render("○ pending") + "  " +
		styles.CheckRunning.Render("⠋ running"))
	fmt.Println()

	// ── File Paths ──
	fmt.Println("  File References:")
	fmt.Println("    " + styles.FilePath.Render("internal/api/graphql.go") +
		styles.LineNumber.Render(":47") + "  " +
		styles.Author.Render("@reviewer1") + "  " +
		styles.ThreadID.Render("PRRT_kwDON1..."))
	fmt.Println("    " + styles.FilePath.Render("internal/cli/comments.go") +
		styles.LineNumber.Render(":123") + "  " +
		styles.Author.Render("@reviewer2"))
	fmt.Println()

	// ── Diff Hunk ──
	fmt.Println("  Diff Hunk:")
	fmt.Println("    " + styles.DiffHeader.Render("@@ -44,8 +44,10 @@"))
	fmt.Println("    " + styles.DiffContext.Render(" func (c *Client) FetchThreads(...) {"))
	fmt.Println("    " + styles.DiffDel.Render("-    return nil, err"))
	fmt.Println("    " + styles.DiffAdd.Render("+    return nil, fmt.Errorf(\"fetch threads: %w\", err)"))
	fmt.Println("    " + styles.DiffContext.Render(" }"))
	fmt.Println()

	// ── Resolve Checkboxes ──
	fmt.Println("  Resolve Checkboxes:")
	fmt.Println("    " + styles.CheckboxOn.Render("[✓]") + " " +
		styles.FilePath.Render("internal/api/graphql.go") +
		styles.LineNumber.Render(":47"))
	fmt.Println("    " + styles.CheckboxOff.Render("[ ]") + " " +
		styles.FilePath.Render("internal/format/markdown.go") +
		styles.LineNumber.Render(":89"))
	fmt.Println()

	// ── Bordered Box ──
	fmt.Println("  Bordered Box:")
	boxContent := styles.FilePath.Render("internal/api/graphql.go") +
		styles.LineNumber.Render(":47") + "\n" +
		styles.Author.Render("@reviewer1") + " — " +
		lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Text))).
			Render("This error should be wrapped with context.")
	fmt.Println(styles.Box.Width(w - 4).Render(boxContent))
	fmt.Println()

	fmt.Println("  Focused Box:")
	fmt.Println(styles.BoxFocused.Width(w - 4).Render(boxContent))
	fmt.Println()

	// ── Summary KPIs ──
	fmt.Println("  Summary KPIs:")
	kpi := func(count, label string, color lipgloss.Color) string {
		return styles.SummaryCount.Foreground(color).Render(count) + " " +
			styles.SummaryLabel.Render(label)
	}
	fmt.Println("    " +
		kpi("5", "UNRESOLVED", lipgloss.Color(string(styles.Red))) + "  " +
		kpi("4", "PASSED", lipgloss.Color(string(styles.Green))) + "  " +
		kpi("1", "FAILED", lipgloss.Color(string(styles.Red))) + "  " +
		kpi("1", "APPROVAL", lipgloss.Color(string(styles.Yellow))))
	fmt.Println()

	// ── Own Comment ──
	fmt.Println("  Own Comment:")
	fmt.Println("    " + styles.OwnComment.Render("@you") + " — Good catch, fixed!")
	fmt.Println()

	// ── Help Bar ──
	fmt.Println("  Help Bar:")
	helpItem := func(key, desc string) string {
		return styles.HelpKey.Render(key) + " " + styles.HelpSep.Render(desc)
	}
	fmt.Println("    " + helpItem("j/k", "navigate") + "  " +
		helpItem("enter", "expand") + "  " +
		helpItem("r", "resolve") + "  " +
		helpItem("tab", "checks") + "  " +
		helpItem("q", "quit"))
	fmt.Println()

	// ══════════════════════════════════════════════════════════════
	// SHARED COMPONENTS (Task 4.2)
	// ══════════════════════════════════════════════════════════════
	fmt.Println(strings.Repeat("═", w))
	fmt.Println(styles.BadgeBlue.Render("ghent") + "  Component Demo — Shared TUI Components")
	fmt.Println(strings.Repeat("─", w))
	fmt.Println()

	// ── Status Bar Component ──
	fmt.Println("  Status Bar (comments view):")
	fmt.Println(components.RenderStatusBar(components.StatusBarData{
		Repo:  "indrasvat/my-project",
		PR:    42,
		Right: styles.BadgeRed.Render("5 unresolved") + "  " + styles.StatusBarDim.Render("2 resolved"),
	}, w))
	fmt.Println()

	fmt.Println("  Status Bar (checks view):")
	fmt.Println(components.RenderStatusBar(components.StatusBarData{
		Repo:  "indrasvat/my-project",
		PR:    42,
		Left:  styles.StatusBarDim.Render("HEAD: a1b2c3d"),
		Right: styles.BadgeGreen.Render("4 passed") + "  " + styles.BadgeRed.Render("1 failed"),
	}, w))
	fmt.Println()

	fmt.Println("  Status Bar (summary view):")
	fmt.Println(components.RenderStatusBar(components.StatusBarData{
		Repo:       "indrasvat/my-project",
		PR:         42,
		PRTitle:    "feat: add GraphQL client",
		RightBadge: "NOT READY",
		BadgeColor: lipgloss.Color(string(styles.Red)),
	}, w))
	fmt.Println()

	// ── Help Bar Component ──
	fmt.Println("  Help Bar (comments):")
	fmt.Println(components.RenderHelpBar(components.CommentsListKeys(), w))
	fmt.Println()

	fmt.Println("  Help Bar (checks):")
	fmt.Println(components.RenderHelpBar(components.ChecksListKeys(), w))
	fmt.Println()

	fmt.Println("  Help Bar (resolve):")
	fmt.Println(components.RenderHelpBar(components.ResolveKeys(), w))
	fmt.Println()

	// ── Diff Hunk Component ──
	fmt.Println("  Diff Hunk (full):")
	hunk := "@@ -44,8 +44,10 @@\n func (c *Client) FetchThreads(...) {\n     var query threadQuery\n-    if err != nil {\n-        return nil, err\n+    if err != nil {\n+        return nil, fmt.Errorf(\"fetch threads: %w\", err)\n     }"
	fmt.Println(components.RenderDiffHunk(hunk, w))
	fmt.Println()

	fmt.Println("  Diff Hunk (compact, 3 lines):")
	fmt.Println(components.RenderDiffHunkCompact(hunk, 3))
	fmt.Println()

	// ── Width adaptivity ──
	fmt.Println("  Width Adaptivity (status bar at width 40):")
	fmt.Println(components.RenderStatusBar(components.StatusBarData{
		Repo: "indrasvat/my-project", PR: 42,
		Right: styles.BadgeRed.Render("5 unresolved"),
	}, 40))
	fmt.Println()

	fmt.Println("  Width Adaptivity (help bar at width 40):")
	fmt.Println(components.RenderHelpBar(components.CommentsListKeys(), 40))
	fmt.Println()

	fmt.Println(strings.Repeat("─", w))
	fmt.Println(styles.StatusBarDim.Render("  Theme + component demo complete. All styles rendered."))
	fmt.Println()
}
