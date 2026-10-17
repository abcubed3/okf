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

func validateFrontmatterPaths(c *bundle.Concept, b *bundle.Bundle) []Issue {
	var issues []Issue
	check := func(field, value string, pathRequired bool) {
		if strings.TrimSpace(value) == "" || isExternalResource(value) {
			return
		}
		if !pathRequired && !looksLikeBundlePath(value) {
			return
		}
		issues = append(issues, validateFrontmatterPath(c, b, field, value)...)
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

func validateFrontmatterPath(c *bundle.Concept, b *bundle.Bundle, field, value string) []Issue {
	target := strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
	target = strings.ReplaceAll(target, "\\", "/")
	if target == "" {
		return nil
	}

	var resolved string
	if strings.HasPrefix(target, "/") {
		resolved = path.Clean(strings.TrimPrefix(target, "/"))
	} else {
		resolved = path.Clean(path.Join(path.Dir(filepath.ToSlash(c.Path)), target))
	}
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
	relativePath, err := filepath.Rel(bundleRoot, targetPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return []Issue{frontmatterPathIssue(c, fmt.Sprintf("frontmatter path field '%s' references a path outside the bundle: %q", field, value))}
	}

	info, err := os.Stat(targetPath)
	if err != nil {
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
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") || strings.Contains(value, "/") {
		return true
	}

	switch strings.ToLower(path.Ext(value)) {
	case ".md", ".sql", ".py", ".sh", ".js", ".ts", ".go", ".proto", ".yaml", ".yml", ".json", ".csv", ".txt", ".xml", ".toml":
		return true
	default:
		return false
	}
}

func frontmatterPathIssue(c *bundle.Concept, message string) Issue {
	return Issue{
		ConceptID: c.ID,
		Path:      c.Path,
		Severity:  SeverityWarning,
		Message:   message,
	}
}
