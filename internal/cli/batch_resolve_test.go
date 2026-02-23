package cli

import (
	"testing"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestMatchesFilters(t *testing.T) {
	thread := func(path, author string) domain.ReviewThread {
		return domain.ReviewThread{
			ID:   "PRRT_test",
			Path: path,
			Line: 10,
			Comments: []domain.Comment{
				{Author: author, Body: "test comment"},
			},
		}
	}

	tests := []struct {
		name     string
		thread   domain.ReviewThread
		fileGlob string
		author   string
		want     bool
	}{
		{
			name:     "no filters matches everything",
			thread:   thread("internal/api/handler.go", "alice"),
			fileGlob: "",
			author:   "",
			want:     true,
		},
		{
			name:     "file glob matches",
			thread:   thread("internal/api/handler.go", "alice"),
			fileGlob: "internal/api/*.go",
			author:   "",
			want:     true,
		},
		{
			name:     "file glob does not match",
			thread:   thread("internal/tui/app.go", "alice"),
			fileGlob: "internal/api/*.go",
			author:   "",
			want:     false,
		},
		{
			name:     "exact file match",
			thread:   thread("main.go", "alice"),
			fileGlob: "main.go",
			author:   "",
			want:     true,
		},
		{
			name:     "author matches",
			thread:   thread("main.go", "alice"),
			fileGlob: "",
			author:   "alice",
			want:     true,
		},
		{
			name:     "author does not match",
			thread:   thread("main.go", "bob"),
			fileGlob: "",
			author:   "alice",
			want:     false,
		},
		{
			name:     "both filters match (intersection)",
			thread:   thread("internal/api/handler.go", "alice"),
			fileGlob: "internal/api/*.go",
			author:   "alice",
			want:     true,
		},
		{
			name:     "file matches but author does not",
			thread:   thread("internal/api/handler.go", "bob"),
			fileGlob: "internal/api/*.go",
			author:   "alice",
			want:     false,
		},
		{
			name:     "author matches but file does not",
			thread:   thread("internal/tui/app.go", "alice"),
			fileGlob: "internal/api/*.go",
			author:   "alice",
			want:     false,
		},
		{
			name:     "glob star matches all go files",
			thread:   thread("anything.go", "alice"),
			fileGlob: "*.go",
			author:   "",
			want:     true,
		},
		{
			name:     "glob star does not match nested paths",
			thread:   thread("internal/foo.go", "alice"),
			fileGlob: "*.go",
			author:   "",
			want:     false,
		},
		{
			name:     "thread with no comments and author filter",
			thread:   domain.ReviewThread{ID: "PRRT_empty", Path: "main.go", Line: 1},
			fileGlob: "",
			author:   "alice",
			want:     true,
		},
		{
			name:     "question mark glob",
			thread:   thread("a.go", "alice"),
			fileGlob: "?.go",
			author:   "",
			want:     true,
		},
		{
			name:     "question mark glob no match",
			thread:   thread("ab.go", "alice"),
			fileGlob: "?.go",
			author:   "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilters(tt.thread, tt.fileGlob, tt.author)
			if got != tt.want {
				t.Errorf("matchesFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesFilters_InvalidGlob(t *testing.T) {
	// Bad glob pattern should not match (path.Match returns error).
	thread := domain.ReviewThread{
		ID:   "PRRT_test",
		Path: "main.go",
		Line: 1,
		Comments: []domain.Comment{
			{Author: "alice"},
		},
	}
	got := matchesFilters(thread, "[invalid", "")
	if got {
		t.Error("expected invalid glob to not match")
	}
}
