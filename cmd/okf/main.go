// Package main is the entrypoint for the OKF (Open Knowledge Format) CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/abcubed3/okf/internal/cli"
)

var (
	// Version is the current version of the OKF CLI (injected at build time).
	Version = "dev"
	// Commit is the git commit hash at build time (injected).
	Commit = "none"
	// Date is the build date (injected).
	Date = "unknown"
)

func main() {
	if err := cli.Execute(os.Args[1:], Version, Commit, Date); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
