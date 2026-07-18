package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/export"
	"github.com/abcubed3/okf/pkg/parser"
)

func printExportUsage() {
	fmt.Println("Usage: okf export <type> [flags]")
	fmt.Println("\nAvailable export types:")
	fmt.Println("  jsonld     Export OKF bundle to Schema.org JSON-LD")
}

// RunExport routes the export subcommand execution to the appropriate format exporter.
func RunExport(args []string) error {
	if len(args) < 1 {
		printExportUsage()
		return fmt.Errorf("missing export type")
	}

	exportType := args[0]
	cmdArgs := args[1:]
	switch exportType {
	case "jsonld":
		return runExportJSONLD(cmdArgs)
	default:
		fmt.Printf("Unknown export type: %s\n\n", exportType)
		printExportUsage()
		return fmt.Errorf("unknown export type: %s", exportType)
	}
}

// runExportJSONLD handles the 'export jsonld' subcommand.
func runExportJSONLD(args []string) error {
	fs := flag.NewFlagSet("export jsonld", flag.ContinueOnError)
	bundlePath := fs.String("bundle", ".", "Path to the OKF knowledge bundle directory")
	output := fs.String("output", "schema.jsonld", "Output path for the generated JSON-LD schema file")
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

	b, err := parser.ParseBundle(context.Background(), absBundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	bytes, err := export.ExportBundleToJSONLD(b)
	if err != nil {
		return fmt.Errorf("failed to export bundle to JSON-LD: %w", err)
	}

	absOutputPath, err := filepath.Abs(*output)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for output: %w", err)
	}

	if err := os.WriteFile(absOutputPath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully exported OKF bundle to Schema.org JSON-LD at %s! 🎉\n", absOutputPath)
	return nil
}
