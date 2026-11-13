package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/indrasvat/gh-ghent/internal/github"
)

// An agent reading these messages has to know which of three different
// problems it hit: no token, a rejected token, or a token missing a permission.
func TestFormatErrorAuthCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		scheme   string
		want     []string
		dontWant []string
	}{
		{
			name: "no token says auth is needed even for public repos",
			err:  fmt.Errorf("github client: %w", &github.AuthError{NoToken: true}),
			want: []string{"No GitHub token found", "public repos", "GH_TOKEN"},
		},
		{
			name:   "rejected token suggests bearer when not already bearer",
			err:    &github.AuthError{},
			scheme: "",
			want:   []string{"rejected", "GH_GHENT_AUTH_SCHEME=bearer"},
		},
		{
			name:     "rejected bearer token does not suggest bearer again",
			err:      &github.AuthError{},
			scheme:   "bearer",
			want:     []string{"rejected"},
			dontWant: []string{"set GH_GHENT_AUTH_SCHEME=bearer"},
		},
		{
			name: "403 names the permission problem and GitHub's message",
			err:  &github.PermissionError{Message: "Resource not accessible by personal access token"},
			want: []string{"lacks a permission", "Resource not accessible by personal access token"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GH_GHENT_AUTH_SCHEME", tt.scheme)
			msg, code := FormatError(tt.err)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("message %q missing %q", msg, w)
				}
			}
			for _, w := range tt.dontWant {
				if strings.Contains(msg, w) {
					t.Errorf("message %q should not contain %q", msg, w)
				}
			}
		})
	}
}
