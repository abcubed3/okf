package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/telemetry"
)

// RunAssemble processes a concept ID and assembles its surrounding context based on a bundle graph.
func RunAssemble(args []string) error {
	fs := flag.NewFlagSet("assemble", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	depth := fs.Int("depth", 2, "Maximum depth of link traversal")
	maxChars := fs.Int("max-chars", 16000, "Maximum character budget for assembled context (0 for unlimited)")
	direction := fs.String("direction", "bidirectional", "Link traversal direction: 'outbound', 'inbound', or 'bidirectional'")
	format := fs.String("format", "xml", "Output format: 'xml' or 'markdown'")
	monitor := fs.Bool("monitor", false, "Display visual feedback showing context window consumption")
	remote := fs.Bool("remote", false, "Run assemble in remote tracking mode")
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

	if *remote {
		bundleName := filepath.Base(b.Path)
		// Import will be added manually later if needed, but let's just make sure it's injected.
		// wait, I can just use telemetry package.
		telemetry.SendEvent("assemble_run", bundleName)
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

	res, err := assembly.AssembleContext(g, startID, opts)
	if err != nil {
		return fmt.Errorf("failed to assemble context: %w", err)
	}

	if *monitor {
		renderTokenMonitor(opts.MaxTokens, res)
	}

	fmt.Println(res.Context)
	return nil
}

func renderTokenMonitor(maxTokens int, res *assembly.AssemblyResult) {
	fmt.Fprintf(os.Stderr, "Token Budget Monitor (Max: %d)\n", maxTokens)
	fmt.Fprintf(os.Stderr, "------------------------------------------------\n")
	
	const barWidth = 20
	
	for _, node := range res.Nodes {
		pct := 0.0
		if maxTokens > 0 {
			pct = float64(node.Tokens) / float64(maxTokens)
		}
		
		filled := int(pct * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		empty := barWidth - filled
		
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
		fmt.Fprintf(os.Stderr, "%-20s [%s] %d tokens (%.1f%%)\n", node.ID, bar, node.Tokens, pct*100)
	}
	
	fmt.Fprintf(os.Stderr, "------------------------------------------------\n")
	
	totalPct := 0.0
	if maxTokens > 0 {
		totalPct = float64(res.TotalTokens) / float64(maxTokens)
	}
	totalFilled := int(totalPct * float64(barWidth))
	if totalFilled > barWidth {
		totalFilled = barWidth
	}
	totalEmpty := barWidth - totalFilled
	
	totalBar := strings.Repeat("█", totalFilled) + strings.Repeat("░", totalEmpty)
	fmt.Fprintf(os.Stderr, "%-20s [%s] %d tokens (%.1f%%)\n\n", "Total", totalBar, res.TotalTokens, totalPct*100)
}
