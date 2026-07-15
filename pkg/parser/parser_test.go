package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMockConcept writes a mock concept file to a temp directory.
func writeMockConcept(t *testing.T, dir, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", relPath, err)
	}
}

func TestParseConceptReader(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectType string
		expectBody string
		expectErr  bool
	}{
		{
			name: "Valid Frontmatter and Body",
			input: `---
type: BigQuery Table
title: Users Table
description: Store customer demographics.
tags: [ecommerce, users]
---
# Users Table
Demographics information.`,
			expectType: "BigQuery Table",
			expectBody: "# Users Table\nDemographics information.",
			expectErr:  false,
		},
		{
			name: "No Frontmatter Block",
			input: `# Raw Document
Only body here.`,
			expectType: "",
			expectBody: "# Raw Document\nOnly body here.",
			expectErr:  false,
		},
		{
			name: "Invalid YAML Frontmatter",
			input: `---
type: BigQuery Table
title: : invalid:yaml:here
---
Body`,
			expectType: "",
			expectBody: "Body",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			concept, err := ParseConceptReader(r, "test.md", "test")

			if (err != nil) != tt.expectErr {
				t.Fatalf("unexpected error result: got %v, expected err=%v", err, tt.expectErr)
			}

			if concept == nil {
				t.Fatal("concept should not be nil")
			}

			if concept.Frontmatter.Type != tt.expectType {
				t.Errorf("expected type %q, got %q", tt.expectType, concept.Frontmatter.Type)
			}

			if concept.Body != tt.expectBody {
				t.Errorf("expected body %q, got %q", tt.expectBody, concept.Body)
			}
		})
	}
}

func TestDefaultParseOptions(t *testing.T) {
	opts := DefaultParseOptions()
	expectedMaxSize := int64(10 * 1024 * 1024)
	if opts.MaxFileSize != expectedMaxSize {
		t.Errorf("expected MaxFileSize=%d, got %d", expectedMaxSize, opts.MaxFileSize)
	}
}

func TestParseBundle_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	b, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle returned error on empty dir: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil bundle")
	}
	if len(b.Concepts) != 0 {
		t.Errorf("expected 0 concepts for empty dir, got %d", len(b.Concepts))
	}
}

func TestParseBundle_SkipsReservedFiles(t *testing.T) {
	dir := t.TempDir()
	// Write reserved files that should be skipped
	writeMockConcept(t, dir, "index.md", "---\ntype: Index\n---\n# Index")
	writeMockConcept(t, dir, "log.md", "---\ntype: Log\n---\n# Log")
	// Write a valid concept
	writeMockConcept(t, dir, "tables/users.md", "---\ntype: Table\ntitle: Users\n---\n# Users")

	b, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle error: %v", err)
	}
	if len(b.Concepts) != 1 {
		t.Errorf("expected 1 concept (index and log skipped), got %d", len(b.Concepts))
	}
	if _, ok := b.Concepts["tables/users"]; !ok {
		t.Error("expected concept 'tables/users' to be present")
	}
}

func TestParseBundle_SkipsNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	writeMockConcept(t, dir, "data.json", `{"key": "value"}`)
	writeMockConcept(t, dir, "config.yaml", "key: value")
	writeMockConcept(t, dir, "tables/users.md", "---\ntype: Table\n---\n# Users")

	b, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle error: %v", err)
	}
	if len(b.Concepts) != 1 {
		t.Errorf("expected only 1 .md concept, got %d: %v", len(b.Concepts), b.Concepts)
	}
}

func TestParseBundle_MultipleConcepts(t *testing.T) {
	dir := t.TempDir()
	writeMockConcept(t, dir, "tables/users.md", "---\ntype: PostgreSQL Table\ntitle: Users\n---\n# Users")
	writeMockConcept(t, dir, "tables/orders.md", "---\ntype: PostgreSQL Table\ntitle: Orders\n---\n# Orders")
	writeMockConcept(t, dir, "endpoints/get-users.md", "---\ntype: API Endpoint\ntitle: GET /users\n---\n# GET /users")

	b, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle error: %v", err)
	}
	if len(b.Concepts) != 3 {
		t.Errorf("expected 3 concepts, got %d: %v", len(b.Concepts), b.Concepts)
	}

	if _, ok := b.Concepts["tables/users"]; !ok {
		t.Error("missing concept 'tables/users'")
	}
	if _, ok := b.Concepts["tables/orders"]; !ok {
		t.Error("missing concept 'tables/orders'")
	}
	if _, ok := b.Concepts["endpoints/get-users"]; !ok {
		t.Error("missing concept 'endpoints/get-users'")
	}
}

func TestParseBundle_ConceptIDDerivation(t *testing.T) {
	dir := t.TempDir()
	writeMockConcept(t, dir, "deeply/nested/concept.md", "---\ntype: Nested\n---\n# Concept")

	b, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle error: %v", err)
	}
	expectedID := filepath.ToSlash("deeply/nested/concept")
	if _, ok := b.Concepts[expectedID]; !ok {
		t.Errorf("expected concept with ID %q, got concepts: %v", expectedID, b.Concepts)
	}
}

func TestParseBundle_FileSizeLimit(t *testing.T) {
	dir := t.TempDir()
	// Create a concept file that exceeds the custom limit
	largeContent := strings.Repeat("A", 1001)
	writeMockConcept(t, dir, "big.md", largeContent)

	opts := ParseOptions{MaxFileSize: 1000}
	b, err := ParseBundleWithOptions(dir, opts)
	if err != nil {
		t.Fatalf("ParseBundleWithOptions error: %v", err)
	}
	// The concept should exist but have a ParseError
	c, ok := b.Concepts["big"]
	if !ok {
		t.Fatal("expected concept 'big' to exist (with error recorded)")
	}
	if c.ParseError == "" {
		t.Error("expected ParseError to be set for oversized file")
	}
	if !strings.Contains(c.ParseError, "exceeds the limit") {
		t.Errorf("unexpected ParseError: %q", c.ParseError)
	}
}

func TestParseBundle_InvalidPath(t *testing.T) {
	_, err := ParseBundle("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}

func TestParseBundle_FileNotDirectory(t *testing.T) {
	// Create a temp file, not a directory
	f, err := os.CreateTemp("", "okf-*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	_, err = ParseBundle(f.Name())
	if err == nil {
		t.Error("expected error when path is a file, not a directory")
	}
}

func TestParseBundle_BundleRootPath(t *testing.T) {
	dir := t.TempDir()
	b, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle error: %v", err)
	}
	if b.Path != dir {
		t.Errorf("expected bundle path %q, got %q", dir, b.Path)
	}
}

func TestParseConceptFile(t *testing.T) {
	dir := t.TempDir()
	content := "---\ntype: Test Type\ntitle: My Concept\n---\n# My Concept\nBody text."
	fullPath := filepath.Join(dir, "my-concept.md")
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := ParseConceptFile(fullPath, "my-concept.md", "my-concept")
	if err != nil {
		t.Fatalf("ParseConceptFile error: %v", err)
	}
	if c.Frontmatter.Type != "Test Type" {
		t.Errorf("expected type 'Test Type', got %q", c.Frontmatter.Type)
	}
	if c.ID != "my-concept" {
		t.Errorf("expected ID 'my-concept', got %q", c.ID)
	}
}

func TestParseConceptFile_NonExistent(t *testing.T) {
	_, err := ParseConceptFile("/nonexistent/file.md", "file.md", "file")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseBundleWithOptions_DefaultFallback(t *testing.T) {
	dir := t.TempDir()
	// MaxFileSize = 0 should fall back to 10MB default
	opts := ParseOptions{MaxFileSize: 0}
	b, err := ParseBundleWithOptions(dir, opts)
	if err != nil {
		t.Fatalf("ParseBundleWithOptions error: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil bundle")
	}
}

func BenchmarkParseConceptReader(b *testing.B) {
	input := `---
type: BigQuery Table
title: Users Table
description: Store customer demographics.
tags: [ecommerce, users]
---
# Users Table
Demographics information.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(input)
		_, err := ParseConceptReader(r, "test.md", "test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseBundle(b *testing.B) {
	dir := b.TempDir()
	// Set up 50 concept files
	for i := 0; i < 50; i++ {
		path := filepath.Join(dir, fmt.Sprintf("concepts/concept-%d.md", i))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatal(err)
		}
		content := fmt.Sprintf("---\ntype: Test\ntitle: Concept %d\n---\n# Concept %d", i, i)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseBundle(dir)
		if err != nil {
			b.Fatal(err)
		}
	}
}
