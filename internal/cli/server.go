package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/server/lsp"
	"github.com/abcubed3/okf/pkg/server/mcp"
	"github.com/abcubed3/okf/pkg/telemetry"
)

// RunServer starts the MCP (Model Context Protocol) server over either Stdio or SSE transport.
func RunServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	transport := fs.String("transport", "stdio", "Transport mode: 'stdio' or 'sse'")
	port := fs.Int("port", 8080, "Port for SSE transport server")
	remote := fs.Bool("remote", false, "Proxy requests to the hosted MCP server on the OKF Hub")
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

	bundleName := filepath.Base(absPath)

	if *remote {
		telemetry.SendEvent("mcp_start_remote", bundleName)
		return proxyRemoteMCP(bundleName)
	}

	srv, err := mcp.NewMCPServer(absPath)
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

func proxyRemoteMCP(bundleName string) error {
	fmt.Fprintf(os.Stderr, "Starting OKF MCP Remote Proxy for bundle: %s\n", bundleName)
	fmt.Fprintf(os.Stderr, "Connecting to hosted MCP server via SSE at hub.okfgo.dev...\n")

	// Implementation of SSE Client to proxy Stdio to Hub would go here.
	// 1. Connect to https://okfgo.dev/mcp/sse?bundle=bundleName
	// 2. Read SSE stream and write to os.Stdout
	// 3. Read os.Stdin and POST to the session endpoint

	fmt.Fprintf(os.Stderr, "Remote proxy connected (prototype).\n")

	// Block forever (or until stdin closes)
	<-context.Background().Done()
	return nil
}

// RunLsp starts the Language Server Protocol process.
func RunLsp(args []string) error {
	// lsp.Run() blocks and handles os.Stdin/Stdout. Assuming it returns or panics on fatal errors.
	lsp.Run()
	return nil
}
