// Package validator checks OKF bundles for schema completeness, broken internal links, and general spec conformance.
package validator

import (
	"fmt"
	"path"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// IssueSeverity indicates whether the finding is a hard error or a soft warning.
type IssueSeverity string

const (
	// SeverityError indicates a conformance error that causes validation failure.
	SeverityError IssueSeverity = "ERROR"
	// SeverityWarning indicates a warning or recommendation violation that does not fail build linting.
	SeverityWarning IssueSeverity = "WARNING"
)

// Issue represents a validation finding.
type Issue struct {
	// ConceptID is the identifier of the concept where the issue was found.
	ConceptID string
	// Path is the file path to the concept document.
	Path string
	// Severity designates the issue as an Error or a Warning.
	Severity IssueSeverity
	// Message is the detailed description of the validation issue.
	Message string
}

// ValidateBundle performs OKF conformance checks on a bundle, verifying frontmatter structures and links.
func ValidateBundle(b *bundle.Bundle) []Issue {
	var issues []Issue

	for id, c := range b.Concepts {
		if c.ParseError != "" {
			issues = append(issues, Issue{
				ConceptID: id,
				Path:      c.Path,
				Severity:  SeverityError,
				Message:   c.ParseError,
			})
			continue
		}

		// 1. Hard Rule: Must have non-empty type
		if strings.TrimSpace(c.Frontmatter.Type) == "" {
			issues = append(issues, Issue{
				ConceptID: id,
				Path:      c.Path,
				Severity:  SeverityError,
				Message:   "missing or empty 'type' field in frontmatter",
			})
		}

		// 2. Soft Rules: Recommended fields
		if strings.TrimSpace(c.Frontmatter.Title) == "" {
			issues = append(issues, Issue{
				ConceptID: id,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   "missing recommended 'title' field",
			})
		}
		if strings.TrimSpace(c.Frontmatter.Desc) == "" {
			issues = append(issues, Issue{
				ConceptID: id,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   "missing recommended 'description' field",
			})
		}

		// 3. Link validation
		linkIssues := validateLinks(c, b)
		issues = append(issues, linkIssues...)
	}

	return issues
}

// validateLinks scans the concept body for broken internal markdown links using a robust Markdown AST parser.
func validateLinks(c *bundle.Concept, b *bundle.Bundle) []Issue {
	var issues []Issue

	// Extract links from AST, ignoring code blocks
	targets := extractLinksFromAST(c.Body)

	for _, target := range targets {
		// Ignore external links
		if isExternalLink(target) {
			continue
		}

		// Ignore links with anchors only in the same file (e.g. #section-name)
		if strings.HasPrefix(target, "#") {
			continue
		}

		// Strip anchor/fragment part of target link (e.g., ../tables/users.md#columns -> ../tables/users.md)
		if idx := strings.Index(target, "#"); idx != -1 {
			target = target[:idx]
		}

		// If target is empty after stripping anchor, skip
		if target == "" {
			continue
		}

		// Resolve the target path relative to the concept file's directory (using logical path)
		currentDir := path.Dir(c.Path)
		resolvedRelPath := path.Clean(path.Join(currentDir, target))

		// Normalize: strip leading slashes or dot-dots that go beyond the root
		if strings.HasPrefix(resolvedRelPath, "..") {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("link '%s' references a path outside the bundle", target),
			})
			continue
		}

		// Convert to Concept ID by removing the extension (.md)
		targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")

		// Check if the concept exists in the bundle
		if _, exists := b.GetConcept(targetConceptID); !exists {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("broken link: target concept '%s' (resolved from '%s') does not exist", targetConceptID, target),
			})
		}
	}

	return issues
}

// extractLinksFromAST parses the markdown body using Goldmark and walks the AST to find all link destinations.
// Links inside inline code or fenced code blocks are automatically bypassed by the parser.
func extractLinksFromAST(body string) []string {
	var targets []string
	markdown := goldmark.New()
	reader := text.NewReader([]byte(body))
	doc := markdown.Parser().Parse(reader)

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if link, ok := n.(*ast.Link); ok {
				targets = append(targets, string(link.Destination))
			}
		}
		return ast.WalkContinue, nil
	})

	return targets
}

// isExternalLink returns true if the URL starts with a protocol scheme.
func isExternalLink(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "ftp://") ||
		strings.HasPrefix(lower, "file://")
}
