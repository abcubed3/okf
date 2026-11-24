// Package parser implements parsing logic to read OKF (Open Knowledge Format) concept files
// and organize them into a bundle structure.
package parser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/abcubed3/okf/pkg/bundle"
	"gopkg.in/yaml.v3"
)

const DefaultMaxFileSize = 10 * 1024 * 1024 // 10MB

var citationLineRe = regexp.MustCompile(`^\[(\d+)\]\s+(?:\[([^\]]+)\]\(([^)]+)\)|(\S+))`)

// ParseOptions configures the parser settings.
type ParseOptions struct {
	// MaxFileSize is the maximum allowed file size in bytes. Defaults to 10MB.
	MaxFileSize int64
}

// DefaultParseOptions returns sensible parsing defaults.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		MaxFileSize: DefaultMaxFileSize, // 10MB
	}
}

// ParseBundle traverses the directory and parses all OKF concepts.
// It scans the workspace for markdown files, skipping reserved documents like index.md and log.md.
func ParseBundle(ctx context.Context, rootPath string) (*bundle.Bundle, error) {
	return ParseBundleWithOptions(ctx, rootPath, DefaultParseOptions())
}

// ParseBundleWithOptions parses all OKF concepts in the rootPath with custom options.
// It parallelizes concept file parsing for scalability and restricts large files.
func ParseBundleWithOptions(ctx context.Context, rootPath string, opts ParseOptions) (*bundle.Bundle, error) {
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = DefaultMaxFileSize // Fallback using default
	}

	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("root path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory")
	}

	b := bundle.NewBundle(absRoot)

	type fileJob struct {
		path      string
		relPath   string
		conceptID string
		size      int64
	}

	var jobs []fileJob

	// 1. Walk directory sequentially to collect all candidate file paths (fast metadata-only check)
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		// Skip dot-directories (.git, .okf, .okf_cache, etc.) and node_modules
		if info.IsDir() {
			name := info.Name()
			if path != absRoot && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a markdown file
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		filename := filepath.Base(relPath)
		// Skip reserved files from being parsed as regular concepts.
		// Record index.md and log.md files for metadata extraction, structural conformance, and link checks.
		if filename == "index.md" || filename == "log.md" {
			relSlash := filepath.ToSlash(relPath)
			id := strings.TrimSuffix(relSlash, ".md")
			var reservedDoc *bundle.Concept
			if info.Size() > opts.MaxFileSize {
				reservedDoc = &bundle.Concept{
					ID:         id,
					Path:       relPath,
					ParseError: fmt.Sprintf("file size %d bytes exceeds the limit of %d bytes", info.Size(), opts.MaxFileSize),
				}
			} else {
				f, openErr := os.Open(path)
				if openErr != nil {
					reservedDoc = &bundle.Concept{
						ID:         id,
						Path:       relPath,
						ParseError: openErr.Error(),
					}
				} else {
					c, parseErr := parseConceptReaderWithLimit(f, relPath, id, opts.MaxFileSize)
					if closeErr := f.Close(); closeErr != nil && parseErr == nil {
						parseErr = closeErr
					}
					if parseErr != nil {
						if c == nil {
							c = &bundle.Concept{ID: id, Path: relPath}
						}
						c.ParseError = parseErr.Error()
					}
					reservedDoc = c
				}
			}
			if reservedDoc != nil {
				if filename == "index.md" {
					if relSlash == "index.md" && reservedDoc.ParseError == "" {
						b.OKFVersion = reservedDoc.Frontmatter.OKFVersion
						b.Title = reservedDoc.Frontmatter.Title
						b.Description = reservedDoc.Frontmatter.Desc
						b.License = reservedDoc.Frontmatter.License
					}
					b.IndexFiles[relSlash] = reservedDoc
				} else {
					b.LogFiles[relSlash] = reservedDoc
				}
			}
			return nil
		}

		// Calculate Concept ID (relative path without .md)
		conceptID := strings.TrimSuffix(filepath.ToSlash(relPath), ".md")
		jobs = append(jobs, fileJob{
			path:      path,
			relPath:   relPath,
			conceptID: conceptID,
			size:      info.Size(),
		})

		return nil
	})

	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("error walking bundle path: %w", err)
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	if len(jobs) == 0 {
		return b, nil
	}

	// 2. Setup parallel parsing workers
	numWorkers := runtime.NumCPU()
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	jobsChan := make(chan fileJob, len(jobs))
	resultsChan := make(chan *bundle.Concept, len(jobs))

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobsChan:
					if !ok {
						return
					}
					var concept *bundle.Concept
					if job.size > opts.MaxFileSize {
						parseErr := fmt.Errorf("file size %d bytes exceeds the limit of %d bytes", job.size, opts.MaxFileSize)
						concept = &bundle.Concept{
							ID:         job.conceptID,
							Path:       job.relPath,
							ParseError: parseErr.Error(),
						}
					} else {
						var parseErr error
						concept, parseErr = parseConceptFileWithLimit(job.path, job.relPath, job.conceptID, opts.MaxFileSize)
						if parseErr != nil {
							concept = &bundle.Concept{
								ID:         job.conceptID,
								Path:       job.relPath,
								ParseError: parseErr.Error(),
							}
						}
					}
					select {
					case <-ctx.Done():
						return
					case resultsChan <- concept:
					}
				}
			}
		}()
	}

	// Push jobs to channel
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	// Wait for workers to finish and close results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect parsed concepts; per OKF v0.3 §4.1 & §11, non-reserved .md files without
	// parseable YAML errors that lack frontmatter or a non-empty 'type' field (such as
	// README.md, AGENTS.md, CONTRIBUTING.md, or SKILL.md) are skipped as non-concepts.
	for concept := range resultsChan {
		if concept.ParseError != "" {
			b.Concepts[concept.ID] = concept
			continue
		}
		if !concept.HasFrontmatter || strings.TrimSpace(concept.Frontmatter.Type) == "" {
			b.SkippedFiles = append(b.SkippedFiles, filepath.ToSlash(concept.Path))
			continue
		}
		b.Concepts[concept.ID] = concept
	}
	slices.Sort(b.SkippedFiles)

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	return b, nil
}

// ParseConceptFile reads a single file from the filesystem and splits it into frontmatter and body.
func ParseConceptFile(fullPath, relPath, conceptID string) (*bundle.Concept, error) {
	return parseConceptFileWithLimit(fullPath, relPath, conceptID, DefaultMaxFileSize)
}

func parseConceptFileWithLimit(fullPath, relPath, conceptID string, maxFileSize int64) (*bundle.Concept, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return parseConceptReaderWithLimit(file, relPath, conceptID, maxFileSize)
}

// ParseConceptReader parses from an io.Reader (useful for unit testing).
// It extracts the YAML frontmatter block enclosed between "---" markers and retrieves the markdown body.
func ParseConceptReader(r io.Reader, relPath, conceptID string) (*bundle.Concept, error) {
	return parseConceptReaderWithLimit(r, relPath, conceptID, DefaultMaxFileSize)
}

type countingReader struct {
	r io.Reader
	n int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

func parseConceptReaderWithLimit(r io.Reader, relPath, conceptID string, maxFileSize int64) (*bundle.Concept, error) {
	if maxFileSize <= 0 {
		maxFileSize = DefaultMaxFileSize
	}
	cr := &countingReader{r: io.LimitReader(r, maxFileSize+1)}
	scanner := bufio.NewScanner(cr)
	maxTokenSize := int(maxFileSize) + 1
	if maxTokenSize < 64*1024 {
		maxTokenSize = 64 * 1024
	}
	scanner.Buffer(make([]byte, 0, 64*1024), maxTokenSize)

	var frontmatterLines []string
	var bodyBuilder strings.Builder

	frontmatterChecked := false
	hasFrontmatter := false

	lineCount := 0
	for scanner.Scan() {
		if cr.n > maxFileSize {
			return nil, fmt.Errorf("file size exceeds the limit of %d bytes", maxFileSize)
		}
		line := scanner.Text()
		lineCount++

		if lineCount == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
			if strings.TrimRight(line, " \t\r") == "---" {
				hasFrontmatter = true
				continue
			}
			// First line is not "---", so no frontmatter block exists
			frontmatterChecked = true
		}

		if hasFrontmatter && !frontmatterChecked {
			if strings.TrimRight(line, " \t\r") == "---" {
				frontmatterChecked = true
				continue
			}
			frontmatterLines = append(frontmatterLines, line)
		} else {
			if bodyBuilder.Len() > 0 {
				bodyBuilder.WriteByte('\n')
			}
			bodyBuilder.WriteString(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if cr.n > maxFileSize {
		return nil, fmt.Errorf("file size exceeds the limit of %d bytes", maxFileSize)
	}

	if hasFrontmatter && !frontmatterChecked {
		return &bundle.Concept{
			ID:             conceptID,
			Path:           relPath,
			HasFrontmatter: true,
		}, fmt.Errorf("unclosed frontmatter '---' fence")
	}

	body := bodyBuilder.String()
	citations := ParseCitationsFromMarkdown(body)

	var fm bundle.Frontmatter
	if hasFrontmatter {
		fmContent := strings.Join(frontmatterLines, "\n")
		err := yaml.Unmarshal([]byte(fmContent), &fm)
		if err != nil {
			// If YAML parsing fails, return the concept with the error, so validation can catch it
			return &bundle.Concept{
				ID:             conceptID,
				Path:           relPath,
				HasFrontmatter: true,
				Body:           body,
				Citations:      citations,
			}, fmt.Errorf("failed to parse frontmatter YAML: %w", err)
		}
	}

	return &bundle.Concept{
		ID:             conceptID,
		Path:           relPath,
		Frontmatter:    fm,
		HasFrontmatter: hasFrontmatter,
		Body:           body,
		Citations:      citations,
	}, nil
}

// ParseCitationsFromMarkdown extracts concept citations from under the '# Citations' section.
func ParseCitationsFromMarkdown(body string) []bundle.Citation {
	var citations []bundle.Citation
	lines := strings.Split(body, "\n")
	inCitations := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect # Citations header
		if strings.HasPrefix(line, "#") {
			headerText := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if strings.ToLower(headerText) == "citations" {
				inCitations = true
				continue
			} else {
				inCitations = false
			}
		}

		if inCitations {
			matches := citationLineRe.FindStringSubmatch(line)
			if len(matches) > 0 {
				var numVal int
				_, _ = fmt.Sscanf(matches[1], "%d", &numVal)

				title := matches[2]
				uri := matches[3]
				if uri == "" {
					uri = matches[4]
					title = uri
				}

				citations = append(citations, bundle.Citation{
					Number: numVal,
					Title:  title,
					URI:    uri,
				})
			}
		}
	}
	return citations
}
