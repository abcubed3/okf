// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// Harvester defines the contract for metadata extraction components.
type Harvester interface {
	// Harvest extracts metadata from a source and converts it into OKF Concepts.
	Harvest(ctx context.Context) ([]*bundle.Concept, error)
}

// WriteConcepts writes a slice of concept documents to the target base directory.
// It creates the parent directories if necessary, formats the frontmatter block, and writes each concept to disk concurrently.
func WriteConcepts(concepts []*bundle.Concept, outputDir string) error {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of output directory: %w", err)
	}

	var g errgroup.Group

	for _, c := range concepts {
		c := c // capture range variable
		g.Go(func() error {
			// Output path is relative to the output directory using the concept's relative path (c.Path).
			targetPath := filepath.Join(absOutputDir, c.Path)
			if !strings.HasPrefix(targetPath, absOutputDir+string(os.PathSeparator)) {
				return fmt.Errorf("concept path %q escapes output directory", c.Path)
			}

			// Create target parent directory if it doesn't exist
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", parentDir, err)
			}

			// Marshal frontmatter to YAML
			fmBytes, err := yaml.Marshal(c.Frontmatter)
			if err != nil {
				return fmt.Errorf("failed to marshal frontmatter for concept %q: %w", c.ID, err)
			}

			// Format the concept file
			var builder strings.Builder
			builder.WriteString("---\n")
			builder.Write(fmBytes)
			builder.WriteString("---\n")
			if c.Body != "" {
				// Ensure there's a blank line between frontmatter and body if body doesn't start with newline
				if !strings.HasPrefix(c.Body, "\n") {
					builder.WriteString("\n")
				}
				builder.WriteString(c.Body)
				// Add a trailing newline if it's missing
				if !strings.HasSuffix(c.Body, "\n") {
					builder.WriteString("\n")
				}
			}

			// Write to disk
			if err := os.WriteFile(targetPath, []byte(builder.String()), 0644); err != nil {
				return fmt.Errorf("failed to write concept file %q: %w", targetPath, err)
			}

			return nil
		})
	}

	return g.Wait()
}
