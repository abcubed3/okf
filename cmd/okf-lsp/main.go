// Package main is the entrypoint for the standalone OKF LSP (Language Server Protocol) server binary.
package main

import (
	"github.com/abcubed3/okf/pkg/server/lsp"
)

func main() {
	lsp.Run()
}
