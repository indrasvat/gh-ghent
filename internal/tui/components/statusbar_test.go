package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

func TestRenderStatusBar(t *testing.T) {
	tests := []struct {
		name  string
		data  StatusBarData
		width int
		want  []string // substrings expected in output
	}{
		{
			name: "basic",
			data: StatusBarData{
				Repo: "indrasvat/my-project",
				PR:   42,
			},
			width: 80,
			want:  []string{"ghent", "indrasvat/my-project", "PR #42"},
		},
		{
			name: "with counts",
			data: StatusBarData{
				Repo:  "indrasvat/my-project",
				PR:    42,
				Right: "5 unresolved",
			},
			width: 80,
			want:  []string{"ghent", "PR #42", "5 unresolved"},
		},
		{
			name: "with badge",
			data: StatusBarData{
				Repo:       "owner/repo",
				PR:         1,
				RightBadge: "NOT READY",
				BadgeColor: lipgloss.Color(string(styles.Red)),
			},
			width: 80,
			want:  []string{"NOT READY"},
		},
		{
			name:  "narrow width",
			data:  StatusBarData{Repo: "indrasvat/my-project", PR: 42},
			width: 30,
			want:  []string{"ghent"},
		},
		{
			name:  "zero width",
			data:  StatusBarData{Repo: "owner/repo", PR: 1},
			width: 0,
			want:  nil, // empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderStatusBar(tt.data, tt.width)
			if tt.width == 0 {
				if got != "" {
					t.Errorf("expected empty string for zero width, got %q", got)
				}
				return
			}
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("RenderStatusBar() missing %q in output:\n%s", sub, got)
				}
			}
		})
	}
}

func TestRenderStatusBarWidths(t *testing.T) {
	data := StatusBarData{
		Repo:  "indrasvat/my-project",
		PR:    42,
		Right: "5 unresolved",
	}

	// Should not panic at any width
	for _, w := range []int{10, 20, 40, 80, 120, 200} {
		t.Run(strings.Repeat("x", 0), func(t *testing.T) {
			got := RenderStatusBar(data, w)
			if got == "" {
				t.Errorf("empty output for width %d", w)
			}
		})
	}
}
