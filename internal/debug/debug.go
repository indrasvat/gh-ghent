// Package debug provides structured debug logging for ghent.
// When enabled (via --debug flag or GH_DEBUG env var), it logs structured
// key-value output to stderr using log/slog. When disabled, logging goes
// to io.Discard with zero overhead.
package debug

import (
	"io"
	"log/slog"
	"os"
)

var enabled bool

// Init configures the default slog logger based on whether debug mode is active.
// When active, logs go to stderr with source file:line info.
// When inactive, logs go to io.Discard.
func Init(active bool) {
	enabled = active
	if active {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
		slog.SetDefault(slog.New(handler))
	} else {
		handler := slog.NewTextHandler(io.Discard, nil)
		slog.SetDefault(slog.New(handler))
	}
}

// Enabled returns whether debug mode is active.
func Enabled() bool {
	return enabled
}
