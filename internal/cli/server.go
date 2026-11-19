package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/config"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/server/lsp"
	"github.com/abcubed3/okf/pkg/server/mcp"
	"github.com/abcubed3/okf/pkg/telemetry"
)

// RunServer starts the MCP (Model Context Protocol) server over either Stdio or HTTP transport.
func RunServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory (or <org>/<bundle> with --remote)")
	transport := fs.String("transport", "stdio", "Transport mode: 'stdio' or 'http'")
	port := fs.Int("port", 8080, "Port for HTTP transport server")
	remote := fs.Bool("remote", false, "Proxy requests to the hosted MCP server on the OKF Hub")
	host := fs.String("host", "", "OKF Hub host URL (overrides OKF_HUB_HOST and ~/.okf/config.json)")
	apiKey := fs.String("api-key", "", "OKF Hub API Key (overrides OKF_HUB_API_KEY and ~/.okf/config.json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	portSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portSet = true
		}
	})

	if *remote {
		if *transport != "stdio" || portSet {
			return fmt.Errorf("--remote only supports stdio transport (--transport http and --port cannot be used with --remote)")
		}
		target := *bundlePath
		if target == "." && fs.NArg() > 0 {
			target = fs.Arg(0)
		}
		resolvedHost := config.GetHost(*host)
		resolvedKey := config.GetAPIKey(*apiKey)
		telemetry.SendEvent("mcp_start_remote", target)
		return mcp.RunRemoteBridge(context.Background(), mcp.RemoteBridgeConfig{
			BundleID: target,
			Host:     resolvedHost,
			APIKey:   resolvedKey,
		})
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

	srv, err := mcp.NewMCPServer(absPath)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	ctx := context.Background()

	if *transport == "http" {
		addr := fmt.Sprintf(":%d", *port)
		fmt.Fprintf(os.Stderr, "Starting OKF MCP Server over HTTP at http://localhost%s...\n", addr)
		if err := srv.StartHTTP(addr); err != nil {
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
