package cli

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/parser"
)

// RunAssemble processes a concept ID and assembles its surrounding context based on a bundle graph.
func RunAssemble(args []string) error {
	fs := flag.NewFlagSet("assemble", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	depth := fs.Int("depth", 2, "Maximum depth of link traversal")
	maxChars := fs.Int("max-chars", 16000, "Maximum character budget for assembled context (0 for unlimited)")
	direction := fs.String("direction", "bidirectional", "Link traversal direction: 'outbound', 'inbound', or 'bidirectional'")
	format := fs.String("format", "xml", "Output format: 'xml' or 'markdown'")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("start Concept ID is required. Usage: okf assemble <concept-id> [flags]")
	}
	startID := fs.Arg(0)

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for bundle: %w", err)
	}

	b, err := parser.ParseBundle(context.Background(), absPath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	g := assembly.BuildGraph(b)

	dir := assembly.Direction(*direction)
	if dir != assembly.DirectionOutbound && dir != assembly.DirectionInbound && dir != assembly.DirectionBidirectional {
		return fmt.Errorf("invalid direction %q. Must be 'outbound', 'inbound', or 'bidirectional'", *direction)
	}

	opts := assembly.AssemblyOptions{
		MaxDepth:      *depth,
		MaxCharacters: *maxChars,
		MaxTokens:     *maxChars / 4,
		Direction:     dir,
		Format:        *format,
	}

	ctxStr, err := assembly.AssembleContext(g, startID, opts)
	if err != nil {
		return fmt.Errorf("failed to assemble context: %w", err)
	}

	fmt.Println(ctxStr)
	return nil
}
