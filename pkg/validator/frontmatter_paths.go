package validator

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/internal/links"
)

var knownPathExtensions = map[string]bool{
	".c":        true,
	".cc":       true,
	".cpp":      true,
	".csv":      true,
	".go":       true,
	".gql":      true,
	".graphql":  true,
	".h":        true,
	".hpp":      true,
	".htm":      true,
	".html":     true,
	".java":     true,
	".js":       true,
	".json":     true,
	".markdown": true,
	".md":       true,
	".proto":    true,
	".py":       true,
	".r":        true,
	".rb":       true,
	".rs":       true,
	".scala":    true,
	".sh":       true,
	".sql":      true,
	".toml":     true,
	".ts":       true,
	".tsv":      true,
	".txt":      true,
	".xml":      true,
	".yaml":     true,
	".yml":      true,
}

func isExplicitPath(target string) bool {
	clean := strings.TrimSpace(strings.ReplaceAll(target, "\\", "/"))
	if strings.HasPrefix(clean, "/") ||
		strings.HasPrefix(clean, "./") ||
		strings.HasPrefix(clean, "../") ||
		strings.Contains(clean, "/") {
		return true
	}
	ext := strings.ToLower(path.Ext(clean))
	return knownPathExtensions[ext]
}

func validateFrontmatterPaths(c *bundle.Concept, b *bundle.Bundle) []Issue {
	var issues []Issue
	check := func(field, value string, pathRequired bool) {
		if strings.TrimSpace(value) == "" || isExternalResource(value) {
			return
		}
		if !pathRequired && !looksLikeBundlePath(value) {
			return
		}
		issues = append(issues, validateFrontmatterPath(c, b, field, value, pathRequired)...)
	}

	check("resource", c.Frontmatter.Resource, false)
	for i, source := range c.Frontmatter.Sources {
		check(fmt.Sprintf("sources[%d].resource", i), source.Resource, false)
	}
	if strings.EqualFold(strings.TrimSpace(c.Frontmatter.Type), "attested computation") {
		check("computation", c.Frontmatter.Computation, true)
		if c.Frontmatter.Executor != nil {
			check("executor.resource", c.Frontmatter.Executor.Resource, true)
		}
		if c.Frontmatter.Attester != nil {
			check("attester.resource", c.Frontmatter.Attester.Resource, true)
		}
	}

	return issues
}

// validateFrontmatterPath resolves a frontmatter path against its concept and bundle,
// then warns when the target is missing, is not a file, or escapes the bundle root.
func validateFrontmatterPath(c *bundle.Concept, b *bundle.Bundle, field, value string, pathRequired bool) []Issue {
	target := strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
	target = strings.ReplaceAll(target, "\\", "/")
	if target == "" {
		return nil
	}

	var resolved string
	if strings.HasPrefix(target, "/") {
		// A leading slash anchors the path at the bundle root.
		resolved = path.Clean(strings.TrimPrefix(target, "/"))
	} else {
		// Relative paths, including bare filenames like query.sql, are resolved
		// from the directory containing this concept.
		resolved = path.Clean(path.Join(path.Dir(filepath.ToSlash(c.Path)), target))
	}
	// Reject paths that traverse above the bundle root or specify a volume.
	if resolved == ".." || strings.HasPrefix(resolved, "../") || filepath.VolumeName(filepath.FromSlash(target)) != "" {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("frontmatter path field '%s' references a path outside the bundle: %q", field, value))}
	}

	if strings.EqualFold(path.Ext(resolved), ".md") {
		conceptID := strings.TrimSuffix(resolved, path.Ext(resolved))
		if _, exists := b.GetConcept(conceptID); exists {
			return nil
		}
	}

	bundleRoot, err := filepath.Abs(b.Path)
	if err != nil {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("could not resolve frontmatter path field '%s' (%q): %v", field, value, err))}
	}
	targetPath := filepath.Join(bundleRoot, filepath.FromSlash(resolved))
	// Check containment after joining to ensure the target remains a bundle subpath.
	relativePath, err := filepath.Rel(bundleRoot, targetPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("frontmatter path field '%s' references a path outside the bundle: %q", field, value))}
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		// If the field is not strictly required to be a path (e.g. 'resource' or 'sources[].resource')
		// and the value does not have explicit path indicators (no slashes and not a recognized file extension),
		// treat it as an opaque asset/table identifier (e.g. 'analytics.mrr_daily') rather than a broken file path.
		if !pathRequired && !isExplicitPath(target) {
			return nil
		}
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("broken frontmatter path: field '%s' references %q, which could not be resolved: %v", field, value, err))}
	}
	if info.IsDir() {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("frontmatter path field '%s' references a directory, not a file: %q", field, value))}
	}

	realRoot, err := filepath.EvalSymlinks(bundleRoot)
	if err != nil {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("could not resolve bundle root for frontmatter path field '%s': %v", field, err))}
	}
	realTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("could not resolve frontmatter path field '%s' (%q): %v", field, value, err))}
	}
	// A symlink can escape the bundle even when its lexical path is inside it.
	realRelativePath, err := filepath.Rel(realRoot, realTarget)
	if err != nil || realRelativePath == ".." || strings.HasPrefix(realRelativePath, ".."+string(os.PathSeparator)) {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("frontmatter path field '%s' references a path outside the bundle: %q", field, value))}
	}
	return nil
}

func isExternalResource(value string) bool {
	value = strings.TrimSpace(value)
	windowsPath := filepath.VolumeName(filepath.FromSlash(strings.ReplaceAll(value, "\\", "/"))) != ""
	if !windowsPath && links.IsExternal(value) {
		return true
	}
	if windowsPath {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != ""
}

func looksLikeBundlePath(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	return strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "./") ||
		strings.HasPrefix(value, "../") ||
		strings.Contains(value, "/") ||
		path.Ext(value) != ""
}

func frontmatterPathIssue(c *bundle.Concept, message string) Issue {
	return Issue{
		ConceptID: c.ID,
		Path:      c.Path,
		Severity:  SeverityWarning,
		Message:   message,
	}
}
