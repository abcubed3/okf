// Package generator compiles an OKF bundle into an interactive static web application.
package generator

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/export"
	"github.com/abcubed3/okf/pkg/parser"
	"golang.org/x/sync/errgroup"
)

// ConceptJSON matches the client-side JSON format for a single concept's full data (including body).
type ConceptJSON struct {
	// ID is the unique identifier (e.g. "tables/users").
	ID string `json:"id"`
	// Path is the relative file path within the bundle.
	Path string `json:"path"`
	// Type is the category of the concept.
	Type string `json:"type"`
	// Title is a human-readable title.
	Title string `json:"title"`
	// Description is a short summary description.
	Description string `json:"description"`
	// Resource is the underlying physical resource name.
	Resource string `json:"resource,omitempty"`
	// Tags are label terms for search filtering.
	Tags []string `json:"tags,omitempty"`
	// Timestamp is the RFC3339 formatted generation/update time.
	Timestamp string `json:"timestamp,omitempty"`
	// Body is the raw markdown content of this concept.
	Body string `json:"body"`
	// Citations holds the list of supporting citations.
	Citations []bundle.Citation `json:"citations,omitempty"`
}

// ConceptIndexEntry is a lightweight summary used for the bundle index — no body.
type ConceptIndexEntry struct {
	// ID is the unique identifier (e.g. "tables/users").
	ID string `json:"id"`
	// Type is the category of the concept.
	Type string `json:"type"`
	// Title is a human-readable title.
	Title string `json:"title"`
	// Description is a short summary description.
	Description string `json:"description"`
	// Resource is the underlying physical resource name.
	Resource string `json:"resource,omitempty"`
	// Tags are label terms for search filtering.
	Tags []string `json:"tags,omitempty"`
	// Timestamp is the RFC3339 formatted generation/update time.
	Timestamp string `json:"timestamp,omitempty"`
}

// BundleJSONData matches the client-side JSON format for the entire bundle index.
// Individual concept bodies are NOT included here; they are fetched dynamically from concepts/<id>.json.
type BundleJSONData struct {
	// Title is the overall name of the documentation catalog.
	Title string `json:"title"`
	// Concepts maps each unique Concept ID to its lightweight index entry.
	Concepts map[string]ConceptIndexEntry `json:"concepts"`
	// Types lists all unique concept types present in the bundle.
	Types []string `json:"types"`
	// Tags lists all unique tags present across all concepts.
	Tags []string `json:"tags"`
	// Index holds the content of the index.md file, if available.
	Index string `json:"index,omitempty"`
	// Log holds the content of the log.md update log, if available.
	Log string `json:"log,omitempty"`
}

// templateData is the Go struct passed to the HTML template.
type templateData struct {
	Title      string
	BundleJSON template.JS
	JSONLD     template.HTML
}

// Generate compiles the OKF bundle into an interactive static web application.
// Individual concept bodies are written to separate JSON files under concepts/<id>.json
// to decouple the index bundle from full-body payloads and enable dynamic fetching.
func Generate(bundlePath, outputPath string) error {
	// 1. Verify and parse the bundle
	b, err := parser.ParseBundle(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse bundle: %w", err)
	}

	// 2. Parse reserved files (index.md, log.md)
	indexPath := filepath.Join(bundlePath, "index.md")
	var indexContent string
	if data, err := os.ReadFile(indexPath); err == nil {
		indexContent = string(data)
	}

	logPath := filepath.Join(bundlePath, "log.md")
	var logContent string
	if data, err := os.ReadFile(logPath); err == nil {
		logContent = string(data)
	}

	// Determine bundle title from index.md first heading or fallback to directory name
	title := extractTitleFromMarkdown(indexContent)
	if title == "" {
		title = filepath.Base(b.Path)
		if title == "." || title == "" {
			title = "OKF Knowledge Bundle"
		}
	}

	// 3. Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 4. Ensure concepts sub-directory exists
	conceptsDir := filepath.Join(outputPath, "concepts")
	if err := os.MkdirAll(conceptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create concepts directory: %w", err)
	}

	// 5. Build the lightweight bundle index (no concept bodies)
	jsonData := BundleJSONData{
		Title:    title,
		Concepts: make(map[string]ConceptIndexEntry, len(b.Concepts)),
		Types:    b.ListTypes(),
		Tags:     b.ListTags(),
		Index:    indexContent,
		Log:      logContent,
	}

	for id, c := range b.Concepts {
		if c.ParseError != "" {
			continue
		}
		jsonData.Concepts[id] = ConceptIndexEntry{
			ID:          c.ID,
			Type:        c.Frontmatter.Type,
			Title:       c.Frontmatter.Title,
			Description: c.Frontmatter.Desc,
			Resource:    c.Frontmatter.Resource,
			Tags:        c.Frontmatter.Tags,
			Timestamp:   c.Frontmatter.Timestamp,
		}
	}

	bundleJSONBytes, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to serialize bundle to JSON: %w", err)
	}

	// 6. Write individual concept JSON files concurrently
	var g errgroup.Group
	for id, c := range b.Concepts {
		id, c := id, c
		if c.ParseError != "" {
			continue
		}
		g.Go(func() error {
			cJSON := ConceptJSON{
				ID:          c.ID,
				Path:        c.Path,
				Type:        c.Frontmatter.Type,
				Title:       c.Frontmatter.Title,
				Description: c.Frontmatter.Desc,
				Resource:    c.Frontmatter.Resource,
				Tags:        c.Frontmatter.Tags,
				Timestamp:   c.Frontmatter.Timestamp,
				Body:        c.Body,
				Citations:   c.Citations,
			}
			cJSONBytes, err := json.Marshal(cJSON)
			if err != nil {
				return fmt.Errorf("failed to serialize concept %q: %w", id, err)
			}

			// The concept JSON path mirrors the concept ID hierarchy, e.g. concepts/tables/users.json
			cPath := filepath.Join(conceptsDir, filepath.FromSlash(id)+".json")
			if err := os.MkdirAll(filepath.Dir(cPath), 0755); err != nil {
				return fmt.Errorf("failed to create concept dir for %q: %w", id, err)
			}
			if err := os.WriteFile(cPath, cJSONBytes, 0644); err != nil {
				return fmt.Errorf("failed to write concept JSON %q: %w", id, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to write concept files: %w", err)
	}

	// 7. Copy non-markdown assets from bundle source
	if err := copyAssets(bundlePath, outputPath); err != nil {
		return fmt.Errorf("failed to copy assets: %w", err)
	}

	// 8. Render index.html using html/template (XSS-safe)
	tmpl, err := template.New("okf-doc").Parse(HTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	destHTMLPath := filepath.Join(outputPath, "index.html")
	f, err := os.Create(destHTMLPath)
	if err != nil {
		return fmt.Errorf("failed to create destination HTML file: %w", err)
	}
	defer f.Close()

	jsonLdStr, err := export.GetJSONLDString(b)
	if err != nil {
		return fmt.Errorf("failed to generate JSON-LD: %w", err)
	}

	tmplData := templateData{
		Title:      title,
		BundleJSON: template.JS(bundleJSONBytes),
		JSONLD:     template.HTML("<!-- Schema.org JSON-LD Graph Bridge -->\n<script type=\"application/ld+json\">\n" + jsonLdStr + "\n</script>"),
	}

	if err := tmpl.Execute(f, tmplData); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}

// extractTitleFromMarkdown pulls the first `# ` header from markdown.
func extractTitleFromMarkdown(md string) string {
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

// copyAssets walks the srcDir and copies non-markdown files to destDir.
func copyAssets(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories (like .git)
		if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Skip markdown files (their content is served from concept JSON files)
		ext := filepath.Ext(path)
		if ext == ".md" {
			return nil
		}

		// Skip go module or binary artifacts
		if info.Name() == "go.mod" || info.Name() == "go.sum" || info.Name() == "okf" {
			return nil
		}

		return copyFile(path, destPath)
	})
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
