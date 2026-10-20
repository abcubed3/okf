package cli

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/ai"
	"github.com/abcubed3/okf/pkg/parser"
)

func printAskUsage() {
	fmt.Println("Usage: okf ask \"<question>\" [flags]")
	fmt.Println("\nFlags:")
	fmt.Println("  --bundle <path>      Path to the OKF knowledge bundle directory (default '.')")
	fmt.Println("  --seed <concept-id>  Explicit entry concept ID for graph traversal (optional)")
	fmt.Println("  --depth <n>          Graph BFS traversal depth (default 2)")
	fmt.Println("  --max-chars <n>      Character budget for context assembly (default 16000)")
	fmt.Println("  --model <model>      Gemini model name (default $GEMINI_MODEL or 'gemini-2.5-flash')")
	fmt.Println("  --api-key <key>      Gemini API key (default $GEMINI_API_KEY)")
	fmt.Println("  --raw                Output raw answer text without metadata header")
}

// RunAsk processes a natural language question against the OKF bundle using graph BFS assembly + Gemini.
func RunAsk(args []string) error {
	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	seed := fs.String("seed", "", "Explicit seed concept ID")
	depth := fs.Int("depth", 2, "Graph traversal depth")
	maxChars := fs.Int("max-chars", 16000, "Maximum character budget")
	model := fs.String("model", "", "Gemini model name")
	apiKey := fs.String("api-key", "", "Gemini API key")
	raw := fs.Bool("raw", false, "Output only raw answer without metadata headers")

	if len(args) > 0 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		printAskUsage()
		return nil
	}

	flagArgs, queryParts := partitionFlagsAndPositional(args)

	if err := fs.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			printAskUsage()
			return nil
		}
		return err
	}

	query := strings.TrimSpace(strings.Join(queryParts, " "))
	if query == "" {
		// Fallback to leftover args in fs if any
		query = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if query == "" {
		printAskUsage()
		return fmt.Errorf("question argument is required. Usage: okf ask \"<question>\" [flags]")
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute bundle path: %w", err)
	}

	b, err := parser.ParseBundle(context.Background(), absBundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	opts := ai.AskOptions{
		Model:       *model,
		APIKey:      *apiKey,
		Depth:       *depth,
		MaxChars:    *maxChars,
		SeedConcept: *seed,
	}

	res, err := ai.AskBundle(context.Background(), b, query, opts)
	if err != nil {
		return fmt.Errorf("failed to answer query: %w", err)
	}

	if *raw {
		fmt.Println(res.Answer)
		return nil
	}

	fmt.Printf("🔍 OKF Knowledge Steward Query: %q\n", query)
	fmt.Printf("🌱 Seed Concept: %s\n", res.SeedConcept)
	if len(res.ReferencedConcepts) > 0 {
		fmt.Printf("🔗 Referenced Subgraph Nodes (%d): %s\n", len(res.ReferencedConcepts), strings.Join(res.ReferencedConcepts, ", "))
	}
	if len(res.Attestations) > 0 {
		fmt.Printf("📐 Sanctioned Queries Found: %d\n", len(res.Attestations))
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(res.Answer)
	fmt.Println("--------------------------------------------------------------------------------")

	return nil
}

// partitionFlagsAndPositional separates CLI flags and their values from positional question arguments,
// enabling users to place flags before or after the question string.
func partitionFlagsAndPositional(args []string) ([]string, []string) {
	flagsWithValues := map[string]bool{
		"-bundle": true, "--bundle": true,
		"-seed": true, "--seed": true,
		"-depth": true, "--depth": true,
		"-max-chars": true, "--max-chars": true,
		"-model": true, "--model": true,
		"-api-key": true, "--api-key": true,
	}

	var flags []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "=") {
				flags = append(flags, arg)
				continue
			}
			if flagsWithValues[arg] {
				flags = append(flags, arg)
				if i+1 < len(args) {
					i++
					flags = append(flags, args[i])
				}
				continue
			}
			flags = append(flags, arg)
		} else {
			positional = append(positional, arg)
		}
	}

	return flags, positional
}
