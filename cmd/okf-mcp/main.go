// Package main is the entrypoint for the standalone OKF MCP (Model Context Protocol) server binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/server/mcp"
)

func main() {
	bundlePath := flag.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	transport := flag.String("transport", "stdio", "Transport mode: 'stdio' or 'sse'")
	port := flag.Int("port", 8080, "Port for SSE transport server")
	flag.Parse()

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve bundle path: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve absolute path for bundle: %v\n", err)
		os.Exit(1)
	}

	srv, err := mcp.NewMCPServer(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize MCP server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if *transport == "sse" {
		addr := fmt.Sprintf(":%d", *port)
		fmt.Fprintf(os.Stderr, "Starting OKF MCP Server over SSE at http://localhost%s...\n", addr)
		if err := srv.StartSSE(addr); err != nil {
			fmt.Fprintf(os.Stderr, "Error: server failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Starting OKF MCP Server over Stdio...")
		if err := srv.StartStdio(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: server failed: %v\n", err)
			os.Exit(1)
		}
	}
}
