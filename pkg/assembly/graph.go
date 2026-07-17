// Package assembly provides tools to build a relationship graph from OKF concepts and
// assemble context starting from a specific concept, traversing relevant links within budgets.
package assembly

import (
	"path"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ConceptNode represents a concept in the graph, with incoming and outgoing links.
type ConceptNode struct {
	// Concept is the underlying data structure of the OKF document.
	Concept *bundle.Concept
	// OutLinks is a list of target Concept IDs that this concept links to.
	OutLinks []string
	// InLinks is a list of source Concept IDs that link to this concept.
	InLinks []string
}

// ConceptGraph represents the entire graph of concepts and their relationships.
type ConceptGraph struct {
	// Nodes is a map of concept IDs to their corresponding ConceptNode pointers.
	Nodes map[string]*ConceptNode
}

// BuildGraph constructs a ConceptGraph from an OKF bundle.
// It parses all markdown links within each concept's body to establish incoming and outgoing edges.
func BuildGraph(b *bundle.Bundle) *ConceptGraph {
	g := &ConceptGraph{
		Nodes: make(map[string]*ConceptNode),
	}

	// 1. Initialize nodes for all parsed concepts
	for id, c := range b.Concepts {
		g.Nodes[id] = &ConceptNode{
			Concept:  c,
			OutLinks: make([]string, 0),
			InLinks:  make([]string, 0),
		}
	}

	// 2. Extract outbound links using Goldmark to avoid parsing links inside code blocks
	markdown := goldmark.New()
	for _, node := range g.Nodes {
		c := node.Concept
		if c.ParseError != "" {
			continue
		}

		reader := text.NewReader([]byte(c.Body))
		doc := markdown.Parser().Parse(reader)
		seen := make(map[string]bool)

		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if link, ok := n.(*ast.Link); ok {
					target := string(link.Destination)

					if isExternalLink(target) || strings.HasPrefix(target, "#") {
						return ast.WalkContinue, nil
					}

					if idx := strings.Index(target, "#"); idx != -1 {
						target = target[:idx]
					}
					if target == "" {
						return ast.WalkContinue, nil
					}

					// Resolve logical relative or absolute path inside bundle
					var resolvedRelPath string
					if strings.HasPrefix(target, "/") {
						resolvedRelPath = path.Clean(strings.TrimPrefix(target, "/"))
					} else {
						currentDir := path.Dir(c.Path)
						resolvedRelPath = path.Clean(path.Join(currentDir, target))
					}

					if strings.HasPrefix(resolvedRelPath, "..") {
						return ast.WalkContinue, nil
					}

					targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")

					if _, exists := b.GetConcept(targetConceptID); exists {
						if !seen[targetConceptID] {
							seen[targetConceptID] = true
							node.OutLinks = append(node.OutLinks, targetConceptID)
						}
					}
				}
			}
			return ast.WalkContinue, nil
		})
	}

	// 3. Build inbound links
	for id, node := range g.Nodes {
		for _, outID := range node.OutLinks {
			if targetNode, exists := g.Nodes[outID]; exists {
				alreadyExists := false
				for _, inID := range targetNode.InLinks {
					if inID == id {
						alreadyExists = true
						break
					}
				}
				if !alreadyExists {
					targetNode.InLinks = append(targetNode.InLinks, id)
				}
			}
		}
	}

	return g
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
