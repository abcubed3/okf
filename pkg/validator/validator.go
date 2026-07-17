// Package validator checks OKF bundles for schema completeness, broken internal links, and general spec conformance.
package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/internal/links"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var sharedMarkdown = goldmark.New()

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

// Options configure the validator.
type Options struct {
	Host   string
	APIKey string
}

// remoteCache stores fetched concepts per bundle ID to avoid repeated network calls.
// Using a map keyed by bundle ID, containing a map of concept IDs.
type remoteCache map[string]map[string]bool

// ValidateBundle performs OKF conformance checks on a bundle, verifying frontmatter structures and links.
func ValidateBundle(b *bundle.Bundle, opts Options) []Issue {
	var issues []Issue
	rcache := make(remoteCache)
	for _, c := range b.Concepts {
		issues = append(issues, ValidateConcept(c, b, opts, rcache)...)
	}
	
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Message < issues[j].Message
	})
	
	return issues
}

// ValidateConcept performs OKF conformance checks on a single concept against the bundle.
func ValidateConcept(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue {
	var issues []Issue

	if c.ParseError != "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityError,
			Message:   c.ParseError,
		})
		return issues
	}

	// 1. Hard Rule: Must have non-empty type
	if strings.TrimSpace(c.Frontmatter.Type) == "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityError,
			Message:   "missing or empty 'type' field in frontmatter",
		})
	}

	// 2. Soft Rules: Recommended fields
	if strings.TrimSpace(c.Frontmatter.Title) == "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   "missing recommended 'title' field",
		})
	}
	if strings.TrimSpace(c.Frontmatter.Desc) == "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   "missing recommended 'description' field",
		})
	}

	// 3. Link validation
	linkIssues := validateLinks(c, b, opts, rcache)
	issues = append(issues, linkIssues...)

	// 4. Citation validation
	citationIssues := validateCitations(c, b, opts, rcache)
	issues = append(issues, citationIssues...)

	return issues
}

// validateLinks scans the concept body for broken internal markdown links using a robust Markdown AST parser.
func validateLinks(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue {
	var issues []Issue

	// Extract links from AST, ignoring code blocks
	targets := extractLinksFromAST(c.Body)

	for _, target := range targets {
		// Ignore external links
		if links.IsExternal(target) {
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

		// Handle remote hub:// links
		if strings.HasPrefix(target, "hub://") {
			if opts.Host == "" {
				// Offline mode or no host configured, we cannot validate
				continue
			}
			
			// target is like hub://stripe/api/charge or hub://postgres/ops
			uri := strings.TrimPrefix(target, "hub://")
			parts := strings.Split(uri, "/")
			if len(parts) < 2 {
				// Require at least a bundle ID that has a namespace/name format, wait.
				// For the prototype, our bundle IDs are "stripe/api" which is 2 parts.
				// A concept inside it would be "stripe/api/charge" (3 parts).
				// We'll just assume the last part is the concept ID, and everything before is bundle ID,
				// or if it's just 2 parts, it's just a bundle link.
				continue // skip complex parsing validation if malformed for now
			}
			
			var bundleID, conceptID string
			if len(parts) > 2 {
				conceptID = parts[len(parts)-1]
				bundleID = strings.Join(parts[:len(parts)-1], "/")
			} else {
				// It's a link to a bundle, not a concept. Let's just validate the bundle exists.
				bundleID = strings.Join(parts, "/")
			}
			
			// Check cache
			conceptSet, ok := rcache[bundleID]
			if !ok {
				// Fetch from Hub
				apiURL := fmt.Sprintf("%s/api/concepts?bundle=%s", opts.Host, url.QueryEscape(bundleID))
				req, _ := http.NewRequest("GET", apiURL, nil)
				if opts.APIKey != "" {
					req.Header.Set("Authorization", "Bearer "+opts.APIKey)
				}
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				conceptSet = make(map[string]bool)
				if err == nil && resp.StatusCode == http.StatusOK {
					var concepts []struct {
						ID string `json:"id"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&concepts); err == nil {
						for _, remC := range concepts {
							conceptSet[remC.ID] = true
						}
					}
				}
				if resp != nil {
					resp.Body.Close()
				}
				// Cache even on failure so we don't spam the network
				rcache[bundleID] = conceptSet
			}
			
			if conceptID != "" && !conceptSet[conceptID] {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityError,
					Message:   fmt.Sprintf("broken remote link: concept '%s' not found in remote bundle '%s'", conceptID, bundleID),
				})
			} else if conceptID == "" && len(conceptSet) == 0 {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityWarning,
					Message:   fmt.Sprintf("remote bundle '%s' could not be resolved or is empty", bundleID),
				})
			}

			continue
		}

		// Resolve logical relative or absolute path inside bundle
		var resolvedRelPath string
		if strings.HasPrefix(target, "/") {
			resolvedRelPath = path.Clean(strings.TrimPrefix(target, "/"))
		} else {
			currentDir := path.Dir(c.Path)
			resolvedRelPath = path.Clean(path.Join(currentDir, target))
		}

		// Normalize: check if resolved path goes beyond the root
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
	reader := text.NewReader([]byte(body))
	doc := sharedMarkdown.Parser().Parse(reader)

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



// validateCitations verifies that any local/internal citation links resolve to existing concepts in the bundle.
func validateCitations(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue {
	var issues []Issue

	for _, cite := range c.Citations {
		target := cite.URI

		// Ignore external links
		if links.IsExternal(target) {
			continue
		}

		// Ignore links with anchors only in the same file (e.g. #section-name)
		if strings.HasPrefix(target, "#") {
			continue
		}

		// Strip anchor/fragment part
		if idx := strings.Index(target, "#"); idx != -1 {
			target = target[:idx]
		}

		if target == "" {
			continue
		}

		// Resolve path
		var resolvedRelPath string
		if strings.HasPrefix(target, "/") {
			resolvedRelPath = path.Clean(strings.TrimPrefix(target, "/"))
		} else {
			currentDir := path.Dir(c.Path)
			resolvedRelPath = path.Clean(path.Join(currentDir, target))
		}

		// Check if resolved path goes beyond root
		if strings.HasPrefix(resolvedRelPath, "..") {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("citation link '%s' references a path outside the bundle", cite.URI),
			})
			continue
		}

		// Convert to Concept ID by removing extension
		targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")

		// Check existence
		if _, exists := b.GetConcept(targetConceptID); !exists {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("broken citation link: target concept '%s' (resolved from '%s') does not exist", targetConceptID, cite.URI),
			})
		}
	}

	return issues
}
