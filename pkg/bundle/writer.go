package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// WriteBundle writes all concepts and index files in b to outputDir without mutating
// any OKF frontmatter fields (unlike harvester.WriteConcepts, which injects generated/status defaults).
func WriteBundle(b *Bundle, outputDir string) error {
	if b == nil {
		return fmt.Errorf("bundle is nil")
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of output directory: %w", err)
	}

	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", absOutputDir, err)
	}

	var g errgroup.Group

	writeDoc := func(relPath string, c *Concept) {
		g.Go(func() error {
			if c == nil {
				return nil
			}
			targetPath := filepath.Join(absOutputDir, filepath.FromSlash(relPath))
			if !strings.HasPrefix(targetPath, absOutputDir+string(os.PathSeparator)) {
				return fmt.Errorf("document path %q escapes output directory", relPath)
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %q: %w", relPath, err)
			}

			content, err := FormatConceptMarkdown(c)
			if err != nil {
				return fmt.Errorf("failed to format concept %q: %w", c.ID, err)
			}

			if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write file %q: %w", targetPath, err)
			}
			return nil
		})
	}

	for id, c := range b.Concepts {
		if c == nil {
			continue
		}
		relPath := c.Path
		if relPath == "" {
			relPath = filepath.FromSlash(id) + ".md"
		}
		writeDoc(relPath, c)
	}

	for relSlash, idx := range b.IndexFiles {
		if idx == nil {
			continue
		}
		relPath := idx.Path
		if relPath == "" {
			relPath = filepath.FromSlash(relSlash)
		}
		writeDoc(relPath, idx)
	}

	return g.Wait()
}

// FormatConceptMarkdown serializes a Concept's Frontmatter and Body to standard OKF Markdown.
func FormatConceptMarkdown(c *Concept) (string, error) {
	fmBytes, err := yaml.Marshal(c.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.Write(fmBytes)
	builder.WriteString("---\n")
	if c.Body != "" {
		if !strings.HasPrefix(c.Body, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString(c.Body)
		if !strings.HasSuffix(c.Body, "\n") {
			builder.WriteString("\n")
		}
	}
	return builder.String(), nil
}
