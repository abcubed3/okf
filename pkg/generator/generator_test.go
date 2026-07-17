package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	// Locate testdata sample relative to this package directory
	samplePath := filepath.Join("..", "..", "testdata", "bundles", "sample")

	// Ensure the sample path exists before testing
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skipf("Skipping TestGenerate: sample data directory %q not found", samplePath)
	}

	// Create temporary directory for output
	tempOutDir := t.TempDir()

	// Generate documentation
	err := Generate(samplePath, tempOutDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify index.html exists
	indexPath := filepath.Join(tempOutDir, "index.html")
	info, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("index.html was not generated: %v", err)
	}
	if info.IsDir() {
		t.Fatal("expected index.html to be a file, but it is a directory")
	}

	// Read index.html content
	contentBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	content := string(contentBytes)

	// Verify key elements are in index.html
	expectedSubstrings := []string{
		"<title>GA4 Sample Merchandise Store Knowledge Catalog — OKF Documentation</title>",
		`"title":"GA4 Sample Merchandise Store Knowledge Catalog"`,
		`"id":"tables/users"`,
		`"id":"tables/orders"`,
		`"id":"playbooks/cleanup"`,
		`"type":"BigQuery Table"`,
		`"type":"Playbook"`,
		`# GA4 Sample Merchandise Store Knowledge Catalog`, // Should be in the index key
		`<!-- Schema.org JSON-LD Graph Bridge -->`,
		`"Dataset"`,
		`"TechArticle"`,
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(content, expected) {
			t.Errorf("expected generated index.html to contain %q, but it did not", expected)
		}
	}

	// Verify that markdown files are NOT copied to the output directory
	err = filepath.Walk(tempOutDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			t.Errorf("found copied markdown file in output directory: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk output directory: %v", err)
	}
}

func TestExtractTitleFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Standard Title",
			input: `# My Awesome Title
Some description here.`,
			expected: "My Awesome Title",
		},
		{
			name: "Title with leading/trailing spaces",
			input: `  
#   Space Title  
More text`,
			expected: "Space Title",
		},
		{
			name:     "No Title",
			input:    `No title heading here`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitleFromMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func BenchmarkGenerate(b *testing.B) {
	samplePath := filepath.Join("..", "..", "testdata", "bundles", "sample")
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		b.Skipf("sample data directory %q not found", samplePath)
	}

	tempOutDir, err := os.MkdirTemp("", "okf-bench-gen-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempOutDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Generate(samplePath, tempOutDir)
		if err != nil {
			b.Fatalf("Generate() failed: %v", err)
		}
	}
}
