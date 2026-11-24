// Package assembly provides tools to build a relationship graph from OKF concepts and
// assemble context starting from a specific concept, traversing relevant links within budgets.
package assembly

import (
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/internal/links"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var sharedMarkdown = goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))

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

	// 2. Extract outbound links using Goldmark (with GFM tables & footnotes) to avoid parsing links inside code blocks
	for _, node := range g.Nodes {
		c := node.Concept
		if c.ParseError != "" {
			continue
		}

		seen := make(map[string]bool)
		addTarget := func(rawTarget string) {
			if links.IsExternal(rawTarget) || links.IsSchemeURI(rawTarget) || strings.HasPrefix(rawTarget, "#") {
				return
			}
			resolvedRelPath, escaped := links.ResolveBundlePath(b.Path, c.Path, rawTarget, b.OKFVersion)
			if escaped || resolvedRelPath == "" {
				return
			}
			targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")
			if _, exists := b.GetConcept(targetConceptID); exists && !seen[targetConceptID] && targetConceptID != c.ID {
				seen[targetConceptID] = true
				node.OutLinks = append(node.OutLinks, targetConceptID)
			}
		}

		if strings.TrimSpace(c.Frontmatter.SupersededBy) != "" {
			addTarget(c.Frontmatter.SupersededBy)
		}

		reader := text.NewReader([]byte(c.Body))
		doc := sharedMarkdown.Parser().Parse(reader)

		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if link, ok := n.(*ast.Link); ok {
					addTarget(string(link.Destination))
				}
			}
			return ast.WalkContinue, nil
		})
	}

	// 3. Build inbound links
	for id, node := range g.Nodes {
		for _, outID := range node.OutLinks {
			if targetNode, exists := g.Nodes[outID]; exists {
				targetNode.InLinks = append(targetNode.InLinks, id)
			}
		}
	}

	return g
}
