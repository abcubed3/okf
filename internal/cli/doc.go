package cli

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/generator"
	"github.com/abcubed3/okf/pkg/parser"
)

// RunDoc handles the 'doc' subcommand, compiling a bundle into interactive HTML docs.
func RunDoc(args []string) error {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	output := fs.String("output", "docs", "Output directory for generated documentation portal")
	if err := fs.Parse(args); err != nil {
		return err
	}

	localPath, cleanup, err := parser.ResolvePath(*bundlePath)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path: %w", err)
	}
	defer cleanup()

	absBundlePath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for bundle: %w", err)
	}

	absOutputPath, err := filepath.Abs(*output)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for output: %w", err)
	}

	fmt.Printf("Generating OKF documentation from %s to %s...\n", absBundlePath, absOutputPath)

	if err := generator.Generate(absBundlePath, absOutputPath); err != nil {
		return fmt.Errorf("documentation generation failed: %w", err)
	}

	fmt.Println("Documentation generated successfully! 🎉")
	return nil
}
