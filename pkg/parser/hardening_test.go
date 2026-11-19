package parser

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserHardening(t *testing.T) {
	dir := t.TempDir()

	// Dot-dir and node_modules must be skipped
	for _, skipDir := range []string{".git", ".okf", ".okf_cache", "node_modules"} {
		p := filepath.Join(dir, skipDir, "ignored.md")
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("---\ntype: Ignored\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// BOM-prefixed valid concept
	bomDoc := "\uFEFF---\ntype: Table\ntitle: Users\n---\n\nUsers body\n"
	if err := os.WriteFile(filepath.Join(dir, "users.md"), []byte(bomDoc), 0644); err != nil {
		t.Fatal(err)
	}

	// Unclosed frontmatter fence on concept
	unclosed := "---\ntype: Table\ntitle: Broken\nNo closing fence\n"
	if err := os.WriteFile(filepath.Join(dir, "broken.md"), []byte(unclosed), 0644); err != nil {
		t.Fatal(err)
	}

	// Unclosed frontmatter fence on index.md
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte("---\nokf_version: \"0.2\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	b, err := ParseBundle(context.Background(), dir)
	if err != nil {
		t.Fatalf("ParseBundle failed: %v", err)
	}

	if _, ok := b.Concepts[".git/ignored"]; ok {
		t.Errorf("expected .git to be skipped")
	}
	if _, ok := b.Concepts["node_modules/ignored"]; ok {
		t.Errorf("expected node_modules to be skipped")
	}
	if u, ok := b.Concepts["users"]; !ok || u.Frontmatter.Type != "Table" || u.ParseError != "" {
		t.Errorf("expected BOM-prefixed concept to parse cleanly, got %+v", u)
	}
	if br, ok := b.Concepts["broken"]; !ok || !strings.Contains(br.ParseError, "unclosed") {
		t.Errorf("expected unclosed fence ParseError on broken concept, got %+v", br)
	}
	if idx, ok := b.IndexFiles["index.md"]; !ok || !strings.Contains(idx.ParseError, "unclosed") {
		t.Errorf("expected unclosed fence ParseError on index.md, got %+v", idx)
	}

	// Context cancellation returns (nil, ctx.Err())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledBundle, err := ParseBundle(ctx, dir)
	if !errors.Is(err, context.Canceled) || canceledBundle != nil {
		t.Errorf("expected (nil, context.Canceled), got (%v, %v)", canceledBundle, err)
	}
}
