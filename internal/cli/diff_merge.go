package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/harvester"
	"github.com/abcubed3/okf/pkg/parser"
)

// RunDiff compares two bundles (local path or git remote) and prints structural changes.
func RunDiff(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("two bundle paths are required. Usage: okf diff <bundle-path-a> <bundle-path-b>")
	}

	pathA := args[0]
	pathB := args[1]

	localPathA, cleanupA, err := parser.ResolvePath(pathA)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path %q: %w", pathA, err)
	}
	defer cleanupA()

	localPathB, cleanupB, err := parser.ResolvePath(pathB)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle path %q: %w", pathB, err)
	}
	defer cleanupB()

	absPathA, err := filepath.Abs(localPathA)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %q: %w", pathA, err)
	}

	absPathB, err := filepath.Abs(localPathB)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %q: %w", pathB, err)
	}

	bundleA, err := parser.ParseBundle(context.Background(), absPathA)
	if err != nil {
		return fmt.Errorf("failed to parse bundle %q: %w", pathA, err)
	}

	bundleB, err := parser.ParseBundle(context.Background(), absPathB)
	if err != nil {
		return fmt.Errorf("failed to parse bundle %q: %w", pathB, err)
	}

	d, err := bundle.Diff(bundleA, bundleB)
	if err != nil {
		return fmt.Errorf("failed to diff bundles: %w", err)
	}

	d.PrettyPrint(os.Stdout)

	if len(d.Changes) > 0 {
		return fmt.Errorf("differences found")
	}
	return nil
}

// RunMerge merges two OKF bundles together based on a specified strategy.
func RunMerge(args []string) error {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	strategy := fs.String("strategy", "union", "Merge strategy: 'union', 'ours', or 'theirs'")
	output := fs.String("output", "", "Output directory for the merged bundle (defaults to source bundle path)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("two bundle paths are required. Usage: okf merge <bundle-path-source> <bundle-path-target> [flags]")
	}

	sourcePath := fs.Arg(0)
	targetPath := fs.Arg(1)

	localSource, cleanupSource, err := parser.ResolvePath(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve source bundle path %q: %w", sourcePath, err)
	}
	defer cleanupSource()

	localTarget, cleanupTarget, err := parser.ResolvePath(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve target bundle path %q: %w", targetPath, err)
	}
	defer cleanupTarget()

	absSource, err := filepath.Abs(localSource)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for source: %w", err)
	}

	absTarget, err := filepath.Abs(localTarget)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for target: %w", err)
	}

	bundleSource, err := parser.ParseBundle(context.Background(), absSource)
	if err != nil {
		return fmt.Errorf("failed to parse source bundle: %w", err)
	}

	bundleTarget, err := parser.ParseBundle(context.Background(), absTarget)
	if err != nil {
		return fmt.Errorf("failed to parse target bundle: %w", err)
	}

	strat := bundle.MergeStrategy(*strategy)
	merged, err := bundle.Merge(bundleSource, bundleTarget, strat)
	if err != nil {
		return fmt.Errorf("error merging bundles: %w", err)
	}

	outPath := *output
	if outPath == "" {
		if parser.IsGitURL(sourcePath) {
			fmt.Println("Warning: The source bundle is a remote Git URL and no --output was specified. Merged output will not be saved locally.")
		}
		outPath = sourcePath
	}

	localOut, cleanupOut, err := parser.ResolvePath(outPath)
	if err != nil {
		return fmt.Errorf("error resolving output path: %w", err)
	}
	defer cleanupOut()

	absOut, err := filepath.Abs(localOut)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute output path: %w", err)
	}

	var concepts []*bundle.Concept
	for _, c := range merged.Concepts {
		concepts = append(concepts, c)
	}

	if err := harvester.WriteConcepts(concepts, absOut); err != nil {
		return fmt.Errorf("failed to write merged concepts: %w", err)
	}

	fmt.Printf("Successfully merged bundles! Written into %q\n", absOut)
	return nil
}
