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
	realOutputDir, err := filepath.EvalSymlinks(absOutputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory %q: %w", absOutputDir, err)
	}

	var g errgroup.Group

	writeDoc := func(relPath string, c *Concept) {
		g.Go(func() error {
			if c == nil {
				return nil
			}
			targetPath, err := ensureSafeWriteTarget(realOutputDir, absOutputDir, relPath, true)
			if err != nil {
				return err
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

	for relSlash, lg := range b.LogFiles {
		if lg == nil {
			continue
		}
		relPath := lg.Path
		if relPath == "" {
			relPath = filepath.FromSlash(relSlash)
		}
		writeDoc(relPath, lg)
	}

	return g.Wait()
}

// ensureSafeWriteTarget verifies that relPath resolves strictly within absRoot/realRoot
// without traversing symlinks outside the bundle directory or overwriting a symlink target.
func ensureSafeWriteTarget(realRoot, absRoot, relPath string, createParents bool) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("document path cannot be empty")
	}
	if filepath.VolumeName(relPath) != "" || filepath.IsAbs(relPath) {
		return "", fmt.Errorf("document path %q escapes output directory", relPath)
	}

	targetPath := filepath.Join(absRoot, filepath.FromSlash(relPath))
	rel, err := filepath.Rel(absRoot, targetPath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("document path %q escapes output directory", relPath)
	}

	parentDir := filepath.Dir(targetPath)
	// Verify deepest existing ancestor is inside realRoot before creating directories
	ancestor := parentDir
	for ancestor != absRoot && len(ancestor) > len(absRoot) {
		if linfo, lerr := os.Lstat(ancestor); lerr == nil {
			if linfo.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("refusing to write through symlink directory %q for %q", ancestor, relPath)
			}
			realAnc, evalErr := filepath.EvalSymlinks(ancestor)
			if evalErr != nil {
				return "", evalErr
			}
			ancRel, relErr := filepath.Rel(realRoot, realAnc)
			if relErr != nil || ancRel == ".." || strings.HasPrefix(ancRel, ".."+string(os.PathSeparator)) {
				return "", fmt.Errorf("directory for %q escapes output directory via symlink", relPath)
			}
			break
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			break
		}
		ancestor = next
	}

	if createParents {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory for %q: %w", relPath, err)
		}
	}

	realParent, err := filepath.EvalSymlinks(parentDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory for %q: %w", relPath, err)
	}
	parentRel, err := filepath.Rel(realRoot, realParent)
	if err != nil || parentRel == ".." || strings.HasPrefix(parentRel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("directory for %q escapes output directory via symlink", relPath)
	}

	if linfo, lerr := os.Lstat(targetPath); lerr == nil {
		if linfo.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to overwrite symlink %q", relPath)
		}
		if linfo.IsDir() {
			return "", fmt.Errorf("target path %q is a directory, not a file", relPath)
		}
	} else if !os.IsNotExist(lerr) {
		return "", lerr
	}

	return targetPath, nil
}

// FormatConceptMarkdown serializes a Concept's Frontmatter and Body to standard OKF Markdown.
// For reserved files (index.md, log.md) or un-fenced documents that carry no frontmatter, it writes only the Markdown body.
func FormatConceptMarkdown(c *Concept) (string, error) {
	if c == nil {
		return "", fmt.Errorf("concept cannot be nil")
	}
	baseName := filepath.Base(c.Path)
	isReserved := baseName == "index.md" || baseName == "log.md" || c.ID == "index" || strings.HasSuffix(c.ID, "/index") || c.ID == "log" || strings.HasSuffix(c.ID, "/log")
	if c.Frontmatter.IsZero() && (isReserved || !c.HasFrontmatter) {
		if c.Body == "" {
			return "", nil
		}
		body := strings.TrimPrefix(c.Body, "\n")
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		return body, nil
	}

	if c.Frontmatter.IsZero() {
		var builder strings.Builder
		builder.WriteString("---\n---\n")
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
