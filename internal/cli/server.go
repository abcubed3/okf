package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/lsp"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/server"
)

// RunServer starts the MCP (Model Context Protocol) server over either Stdio or SSE transport.
func RunServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	transport := fs.String("transport", "stdio", "Transport mode: 'stdio' or 'sse'")
	port := fs.Int("port", 8080, "Port for SSE transport server")
	if err := fs.Parse(args); err != nil {
		return err
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for bundle: %w", err)
	}

	srv, err := server.NewMCPServer(absPath)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	ctx := context.Background()

	if *transport == "sse" {
		addr := fmt.Sprintf(":%d", *port)
		fmt.Fprintf(os.Stderr, "Starting OKF MCP Server over SSE at http://localhost%s...\n", addr)
		if err := srv.StartSSE(addr); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Starting OKF MCP Server over Stdio...")
		if err := srv.StartStdio(ctx); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
	}
	return nil
}

// RunLsp starts the Language Server Protocol process.
func RunLsp(args []string) error {
	// lsp.Run() blocks and handles os.Stdin/Stdout. Assuming it returns or panics on fatal errors.
	lsp.Run()
	return nil
}
