package assembly

import (
	"fmt"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

func TestBuildGraphAndAssemble(t *testing.T) {
	// Create a mock bundle
	b := bundle.NewBundle("/mock/path")

	// Create three concepts: A, B, and C.
	// A references B: [Concept B](../concepts/b.md)
	// B references C: [Concept C](c.md)
	b.Concepts["concepts/a"] = &bundle.Concept{
		ID:   "concepts/a",
		Path: "concepts/a.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "ConceptType",
			Title: "Concept A",
			Desc:  "First concept",
		},
		Body: "This is A, referencing [Concept B](../concepts/b.md).",
	}

	b.Concepts["concepts/b"] = &bundle.Concept{
		ID:   "concepts/b",
		Path: "concepts/b.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "ConceptType",
			Title: "Concept B",
			Desc:  "Second concept",
		},
		Body: "This is B, referencing [Concept C](c.md).",
	}

	b.Concepts["concepts/c"] = &bundle.Concept{
		ID:   "concepts/c",
		Path: "concepts/c.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "ConceptType",
			Title: "Concept C",
			Desc:  "Third concept",
		},
		Body: "This is C, referencing nothing.",
	}

	b.Concepts["concepts/d"] = &bundle.Concept{
		ID:   "concepts/d",
		Path: "concepts/d.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "ConceptType",
			Title: "Concept D",
			Desc:  "Fourth concept",
		},
		Body: "This is D, referencing absolute [Concept C](/concepts/c.md).",
	}

	g := BuildGraph(b)

	// Check graph edges
	nodeA := g.Nodes["concepts/a"]
	if len(nodeA.OutLinks) != 1 || nodeA.OutLinks[0] != "concepts/b" {
		t.Errorf("expected A to link to B, got %v", nodeA.OutLinks)
	}

	nodeB := g.Nodes["concepts/b"]
	if len(nodeB.OutLinks) != 1 || nodeB.OutLinks[0] != "concepts/c" {
		t.Errorf("expected B to link to C, got %v", nodeB.OutLinks)
	}
	if len(nodeB.InLinks) != 1 || nodeB.InLinks[0] != "concepts/a" {
		t.Errorf("expected B to have inlink from A, got %v", nodeB.InLinks)
	}

	nodeD := g.Nodes["concepts/d"]
	if len(nodeD.OutLinks) != 1 || nodeD.OutLinks[0] != "concepts/c" {
		t.Errorf("expected D to link to C, got %v", nodeD.OutLinks)
	}

	// Test assembly with depth 0
	opts := AssemblyOptions{
		MaxDepth:      0,
		MaxCharacters: 0,
		MaxTokens:     0,
		Direction:     DirectionOutbound,
		Format:        "xml",
	}
	res, err := AssembleContext(g, "concepts/a", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctxStr := res.Context
	if !strings.Contains(ctxStr, `id="concepts/a"`) || strings.Contains(ctxStr, `id="concepts/b"`) {
		t.Errorf("expected depth 0 to only contain concept A, got:\n%s", ctxStr)
	}

	// Test assembly with depth 1
	opts.MaxDepth = 1
	res, err = AssembleContext(g, "concepts/a", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctxStr = res.Context
	if !strings.Contains(ctxStr, `id="concepts/a"`) || !strings.Contains(ctxStr, `id="concepts/b"`) || strings.Contains(ctxStr, `id="concepts/c"`) {
		t.Errorf("expected depth 1 to contain concept A and B, but not C. Got:\n%s", ctxStr)
	}

	// Test assembly with depth 2
	opts.MaxDepth = 2
	res, err = AssembleContext(g, "concepts/a", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctxStr = res.Context
	if !strings.Contains(ctxStr, `id="concepts/a"`) || !strings.Contains(ctxStr, `id="concepts/b"`) || !strings.Contains(ctxStr, `id="concepts/c"`) {
		t.Errorf("expected depth 2 to contain concept A, B, and C. Got:\n%s", ctxStr)
	}

	// Test budget constraints (MaxCharacters)
	opts.MaxDepth = 2
	opts.MaxCharacters = 500 // Too small for all three, A is around ~350 chars
	res, err = AssembleContext(g, "concepts/a", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctxStr = res.Context
	if !strings.Contains(ctxStr, `id="concepts/a"`) || strings.Contains(ctxStr, `id="concepts/b"`) || strings.Contains(ctxStr, `id="concepts/c"`) {
		t.Errorf("expected budget pruning to limit to concept A, got:\n%s", ctxStr)
	}
}

func BenchmarkAssembleContext(b *testing.B) {
	// Create a large mock bundle
	bundleObj := bundle.NewBundle("/mock/path")
	for i := 0; i < 200; i++ {
		id := fmt.Sprintf("concepts/node-%d", i)
		bundleObj.Concepts[id] = &bundle.Concept{
			ID:   id,
			Path: fmt.Sprintf("concepts/node-%d.md", i),
			Frontmatter: bundle.Frontmatter{
				Type:  "ConceptType",
				Title: fmt.Sprintf("Node %d", i),
				Desc:  fmt.Sprintf("Node description %d", i),
			},
			Body: fmt.Sprintf("Link to [node-%d](../concepts/node-%d.md) and [node-%d](../concepts/node-%d.md)", (i*2)%200, (i*2)%200, (i*3)%200, (i*3)%200),
		}
	}

	g := BuildGraph(bundleObj)
	opts := AssemblyOptions{
		MaxDepth:      3,
		MaxCharacters: 16000,
		MaxTokens:     4000,
		Direction:     DirectionBidirectional,
		Format:        "xml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := AssembleContext(g, "concepts/node-0", opts)
		if err != nil {
			b.Fatalf("benchmark failed: %v", err)
		}
	}
}
