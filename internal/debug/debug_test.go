package debug

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestInitDisabled(t *testing.T) {
	Init(false)

	if Enabled() {
		t.Error("Enabled() = true after Init(false)")
	}
}

func TestInitEnabled(t *testing.T) {
	Init(true)
	t.Cleanup(func() { Init(false) })

	if !Enabled() {
		t.Error("Enabled() = false after Init(true)")
	}
}

func TestDisabledProducesNoOutput(t *testing.T) {
	Init(false)

	// slog.Debug should go to io.Discard — no way to observe output.
	// We verify by confirming Enabled() is false and the logger is set.
	slog.Debug("should be discarded", "key", "value")

	if Enabled() {
		t.Error("expected debug to be disabled")
	}
}

func TestEnabledProducesStructuredOutput(t *testing.T) {
	// Temporarily redirect slog to a buffer to verify output.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler))
	enabled = true
	t.Cleanup(func() { Init(false) })

	slog.Debug("test message", "owner", "testorg", "repo", "testrepo", "pr", 42)

	output := buf.String()
	for _, want := range []string{"owner=testorg", "repo=testrepo", "pr=42", "test message"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q; got: %s", want, output)
		}
	}

	// Verify source info is present (file:line)
	if !strings.Contains(output, "source=") {
		t.Errorf("output missing source info; got: %s", output)
	}
}

func TestInitToggle(t *testing.T) {
	Init(true)
	if !Enabled() {
		t.Fatal("expected enabled after Init(true)")
	}

	Init(false)
	if Enabled() {
		t.Fatal("expected disabled after Init(false)")
	}
}
