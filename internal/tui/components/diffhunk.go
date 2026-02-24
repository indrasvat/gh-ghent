package components

import (
	"strings"

	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// RenderDiffHunk renders a diff hunk string with syntax coloring.
// Green for additions (+), red for deletions (-), purple for @@ headers,
// dim for context lines.
//
// Uses strings.Repeat for padding (not empty strings — pitfall 7.3).
// Adds explicit ANSI resets between colored lines (pitfall 7.6).
func RenderDiffHunk(hunk string, width int) string {
	if hunk == "" {
		return ""
	}

	lines := strings.Split(hunk, "\n")
	var rendered []string

	for _, line := range lines {
		if line == "" {
			rendered = append(rendered, "")
			continue
		}

		var styled string
		switch {
		case strings.HasPrefix(line, "@@"):
			styled = styles.DiffHeader.Render(line)
		case strings.HasPrefix(line, "+"):
			styled = styles.DiffAdd.Render(line)
		case strings.HasPrefix(line, "-"):
			styled = styles.DiffDel.Render(line)
		default:
			styled = styles.DiffContext.Render(line)
		}

		// Pad to width with spaces (not empty strings — pitfall 7.3)
		if width > 0 {
			styled = PadLine(styled, width)
		}

		// Explicit ANSI reset after each line (pitfall 7.6)
		rendered = append(rendered, styled+styles.ANSIReset)
	}

	return strings.Join(rendered, "\n")
}

// RenderDiffHunkCompact renders a diff hunk in a compact format suitable
// for list views, showing only the first few lines.
func RenderDiffHunkCompact(hunk string, maxLines int) string {
	if hunk == "" {
		return ""
	}
	if maxLines <= 0 {
		maxLines = 4
	}

	lines := strings.Split(hunk, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	var rendered []string
	for _, line := range lines {
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "@@"):
			rendered = append(rendered, styles.DiffHeader.Render(line))
		case strings.HasPrefix(line, "+"):
			rendered = append(rendered, styles.DiffAdd.Render(line))
		case strings.HasPrefix(line, "-"):
			rendered = append(rendered, styles.DiffDel.Render(line))
		default:
			rendered = append(rendered, styles.DiffContext.Render(line))
		}
	}

	result := strings.Join(rendered, "\n")
	if len(strings.Split(hunk, "\n")) > maxLines {
		result += "\n" + styles.StatusBarDim.Render("  ...")
	}

	return result + styles.ANSIReset
}
