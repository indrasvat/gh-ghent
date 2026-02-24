// Package main is the entry point for ghent.
package main

import (
	"fmt"
	"os"

	"github.com/indrasvat/gh-ghent/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		msg, exitCode := cli.FormatError(err)
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(exitCode)
	}
}
