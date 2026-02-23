package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func newTestReplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "reply",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	cmd.Flags().String("body", "", "reply body text")
	cmd.Flags().String("body-file", "", "read reply body from file")
	return cmd
}

func TestResolveBody_InlineBody(t *testing.T) {
	cmd := newTestReplyCmd()
	_ = cmd.Flags().Set("body", "hello world")

	got, err := resolveBody(cmd)
	if err != nil {
		t.Fatalf("resolveBody: %v", err)
	}
	if got != "hello world" {
		t.Errorf("body = %q, want %q", got, "hello world")
	}
}

func TestResolveBody_FromFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "body.txt")
	if err := os.WriteFile(fp, []byte("file body content\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := newTestReplyCmd()
	_ = cmd.Flags().Set("body-file", fp)

	got, err := resolveBody(cmd)
	if err != nil {
		t.Fatalf("resolveBody: %v", err)
	}
	if got != "file body content" {
		t.Errorf("body = %q, want %q", got, "file body content")
	}
}

func TestResolveBody_MutuallyExclusive(t *testing.T) {
	cmd := newTestReplyCmd()
	_ = cmd.Flags().Set("body", "inline")
	_ = cmd.Flags().Set("body-file", "some-file.txt")

	_, err := resolveBody(cmd)
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
	want := "--body and --body-file are mutually exclusive"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestResolveBody_NeitherSet(t *testing.T) {
	cmd := newTestReplyCmd()

	_, err := resolveBody(cmd)
	if err == nil {
		t.Fatal("expected error when neither flag set")
	}
	want := "either --body or --body-file is required"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestResolveBody_FileNotFound(t *testing.T) {
	cmd := newTestReplyCmd()
	_ = cmd.Flags().Set("body-file", "/nonexistent/file.txt")

	_, err := resolveBody(cmd)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestResolveBody_StdinDash(t *testing.T) {
	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.WriteString("stdin body content\n")
		w.Close()
	}()

	cmd := newTestReplyCmd()
	_ = cmd.Flags().Set("body-file", "-")

	got, err := resolveBody(cmd)
	if err != nil {
		t.Fatalf("resolveBody: %v", err)
	}
	if got != "stdin body content" {
		t.Errorf("body = %q, want %q", got, "stdin body content")
	}
}
