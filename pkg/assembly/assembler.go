// Package assembly provides tools to build a relationship graph from OKF concepts and
// assemble context starting from a specific concept, traversing relevant links within budgets.
package assembly

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"gopkg.in/yaml.v3"
)

// Direction defines the direction of graph traversal.
type Direction string

const (
	// DirectionOutbound follows links directed away from the starting concept.
	DirectionOutbound Direction = "outbound"
	// DirectionInbound follows links directed towards the starting concept.
	DirectionInbound Direction = "inbound"
	// DirectionBidirectional follows both inbound and outbound links.
	DirectionBidirectional Direction = "bidirectional"
)

// AssemblyOptions defines parameters for assembling context.
type AssemblyOptions struct {
	// MaxDepth specifies the maximum link distance to traverse from the start concept.
	MaxDepth int
	// MaxTokens is the estimated token budget for the output. If exceeded, traversal stops.
	MaxTokens int
	// MaxCharacters is the character budget for the output. If exceeded, traversal stops.
	MaxCharacters int
	// Direction controls whether to follow outbound, inbound, or bidirectional links.
	Direction Direction
	// Format is the output syntax, either "xml" or "markdown".
	Format string
}

// DefaultOptions returns standard sensible options for context assembly.
func DefaultOptions() AssemblyOptions {
	return AssemblyOptions{
		MaxDepth:      2,
		MaxTokens:     4000,
		MaxCharacters: 16000,
		Direction:     DirectionBidirectional,
		Format:        "xml",
	}
}

// AssembleContext traverses the concept graph from a start ID and builds a formatted context string.
// It uses BFS traversal to follow links up to MaxDepth, checking against MaxCharacters/MaxTokens budgets.
func AssembleContext(g *ConceptGraph, startID string, opts AssemblyOptions) (string, error) {
	_, exists := g.Nodes[startID]
	if !exists {
		return "", fmt.Errorf("starting concept %q not found in graph", startID)
	}

	// Queue for BFS: contains concept ID and current depth
	type queueItem struct {
		id    string
		depth int
	}

	var queue []queueItem
	queue = append(queue, queueItem{id: startID, depth: 0})
	head := 0

	visited := make(map[string]bool)
	visited[startID] = true

	var formattedTexts []string
	charCount := 0
	tokenCount := 0

	for head < len(queue) {
		item := queue[head]
		head++

		node := g.Nodes[item.id]
		concept := node.Concept

		// Format concept content once and cache it to avoid dual marshaling/formatting
		conceptContent := formatSingleConcept(concept, opts.Format)
		conceptLen := len(conceptContent)
		conceptTokens := conceptLen / 4 // Simple heuristic: 1 token ~ 4 characters

		// Check if adding this concept exceeds budgets. If so, terminate early.
		if opts.MaxCharacters > 0 && charCount+conceptLen > opts.MaxCharacters {
			break
		}
		if opts.MaxTokens > 0 && tokenCount+conceptTokens > opts.MaxTokens {
			break
		}

		formattedTexts = append(formattedTexts, conceptContent)
		charCount += conceptLen
		tokenCount += conceptTokens

		// If we haven't reached max depth, enqueue neighbors
		if item.depth < opts.MaxDepth {
			var neighbors []string
			switch opts.Direction {
			case DirectionOutbound:
				neighbors = node.OutLinks
			case DirectionInbound:
				neighbors = node.InLinks
			case DirectionBidirectional:
				// Combine outbound and inbound links
				neighbors = append(neighbors, node.OutLinks...)
				neighbors = append(neighbors, node.InLinks...)
			default:
				neighbors = node.OutLinks
			}

			for _, neighborID := range neighbors {
				if !visited[neighborID] {
					visited[neighborID] = true
					queue = append(queue, queueItem{id: neighborID, depth: item.depth + 1})
				}
			}
		}
	}

	// Assemble pre-formatted texts
	var buf bytes.Buffer
	isMarkdown := strings.ToLower(opts.Format) == "markdown"

	if !isMarkdown {
		buf.WriteString("<context>\n")
	}

	for _, text := range formattedTexts {
		buf.WriteString(text)
	}

	if !isMarkdown {
		buf.WriteString("</context>\n")
	}

	return buf.String(), nil
}

// formatSingleConcept serializes a single Concept into markdown or XML.
func formatSingleConcept(c *bundle.Concept, format string) string {
	fmBytes, err := yaml.Marshal(c.Frontmatter)
	var fmStr string
	if err != nil {
		fmStr = fmt.Sprintf("error marshaling metadata: %v", err)
	} else {
		fmStr = string(fmBytes)
	}

	if strings.ToLower(format) == "markdown" {
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("# Concept: %s\n", c.ID))
		if c.Frontmatter.Type != "" {
			buf.WriteString(fmt.Sprintf("- **Type:** %s\n", c.Frontmatter.Type))
		}
		if c.Frontmatter.Title != "" {
			buf.WriteString(fmt.Sprintf("- **Title:** %s\n", c.Frontmatter.Title))
		}
		if c.Frontmatter.Desc != "" {
			buf.WriteString(fmt.Sprintf("- **Description:** %s\n", c.Frontmatter.Desc))
		}
		buf.WriteString("\n## Metadata\n```yaml\n")
		buf.WriteString(fmStr)
		buf.WriteString("```\n\n## Body\n")
		buf.WriteString(c.Body)
		buf.WriteString("\n\n---\n")
		return buf.String()
	}

	// Default to XML format
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<concept id=%q>\n", c.ID))
	if c.Frontmatter.Type != "" {
		buf.WriteString(fmt.Sprintf("  <type>%s</type>\n", escapeXML(c.Frontmatter.Type)))
	}
	if c.Frontmatter.Title != "" {
		buf.WriteString(fmt.Sprintf("  <title>%s</title>\n", escapeXML(c.Frontmatter.Title)))
	}
	if c.Frontmatter.Desc != "" {
		buf.WriteString(fmt.Sprintf("  <description>%s</description>\n", escapeXML(c.Frontmatter.Desc)))
	}
	buf.WriteString("  <metadata>\n")
	// Indent metadata lines
	for _, line := range strings.Split(strings.TrimSpace(fmStr), "\n") {
		buf.WriteString(fmt.Sprintf("    %s\n", line))
	}
	buf.WriteString("  </metadata>\n")
	buf.WriteString("  <body>\n")
	buf.WriteString(escapeXML(c.Body))
	buf.WriteString("\n  </body>\n")
	buf.WriteString("</concept>\n")
	return buf.String()
}

// escapeXML escapes standard XML special characters.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
