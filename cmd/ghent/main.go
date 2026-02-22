// Package main is the entry point for ghent.
package main

import (
	"fmt"
	"os"

	"github.com/indrasvat/ghent/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
