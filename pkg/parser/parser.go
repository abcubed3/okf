// Package parser implements parsing logic to read OKF (Open Knowledge Format) concept files
// and organize them into a bundle structure.
package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/abcubed3/okf/pkg/bundle"
	"gopkg.in/yaml.v3"
)

// ParseOptions configures the parser settings.
type ParseOptions struct {
	// MaxFileSize is the maximum allowed file size in bytes. Defaults to 10MB.
	MaxFileSize int64
}

// DefaultParseOptions returns sensible parsing defaults.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		MaxFileSize: 10 * 1024 * 1024, // 10MB
	}
}

// ParseBundle traverses the directory and parses all OKF concepts.
// It scans the workspace for markdown files, skipping reserved documents like index.md and log.md.
func ParseBundle(rootPath string) (*bundle.Bundle, error) {
	return ParseBundleWithOptions(rootPath, DefaultParseOptions())
}

// ParseBundleWithOptions parses all OKF concepts in the rootPath with custom options.
// It parallelizes concept file parsing for scalability and restricts large files.
func ParseBundleWithOptions(rootPath string, opts ParseOptions) (*bundle.Bundle, error) {
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = 10 * 1024 * 1024 // Fallback/default to 10MB
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

		// Skip directories
		if info.IsDir() {
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
		// Skip reserved files
		if filename == "index.md" || filename == "log.md" {
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
		return nil, fmt.Errorf("error walking bundle path: %w", err)
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
			for job := range jobsChan {
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
					concept, parseErr = ParseConceptFile(job.path, job.relPath, job.conceptID)
					if parseErr != nil {
						concept = &bundle.Concept{
							ID:         job.conceptID,
							Path:       job.relPath,
							ParseError: parseErr.Error(),
						}
					}
				}
				resultsChan <- concept
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

	// Collect parsed concepts
	for concept := range resultsChan {
		b.Concepts[concept.ID] = concept
	}

	return b, nil
}

// ParseConceptFile reads a single file from the filesystem and splits it into frontmatter and body.
func ParseConceptFile(fullPath, relPath, conceptID string) (*bundle.Concept, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseConceptReader(file, relPath, conceptID)
}

// ParseConceptReader parses from an io.Reader (useful for unit testing).
// It extracts the YAML frontmatter block enclosed between "---" markers and retrieves the markdown body.
func ParseConceptReader(r io.Reader, relPath, conceptID string) (*bundle.Concept, error) {
	scanner := bufio.NewScanner(r)
	var frontmatterLines []string
	var bodyLines []string

	frontmatterChecked := false
	hasFrontmatter := false

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount == 1 {
			if strings.TrimSpace(line) == "---" {
				hasFrontmatter = true
				continue
			}
			// First line is not "---", so no frontmatter block exists
			frontmatterChecked = true
		}

		if hasFrontmatter && !frontmatterChecked {
			if strings.TrimSpace(line) == "---" {
				frontmatterChecked = true
				continue
			}
			frontmatterLines = append(frontmatterLines, line)
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	body := strings.Join(bodyLines, "\n")
	citations := ParseCitationsFromMarkdown(body)

	var fm bundle.Frontmatter
	if hasFrontmatter {
		fmContent := strings.Join(frontmatterLines, "\n")
		err := yaml.Unmarshal([]byte(fmContent), &fm)
		if err != nil {
			// If YAML parsing fails, return the concept with the error, so validation can catch it
			return &bundle.Concept{
				ID:        conceptID,
				Path:      relPath,
				Body:      body,
				Citations: citations,
			}, fmt.Errorf("failed to parse frontmatter YAML: %w", err)
		}
	}

	return &bundle.Concept{
		ID:          conceptID,
		Path:        relPath,
		Frontmatter: fm,
		Body:        body,
		Citations:   citations,
	}, nil
}

// ParseCitationsFromMarkdown extracts concept citations from under the '# Citations' section.
func ParseCitationsFromMarkdown(body string) []bundle.Citation {
	var citations []bundle.Citation
	lines := strings.Split(body, "\n")
	inCitations := false

	// Regex matching:
	// Group 1: Citation Number
	// Group 2 & 3: Link title and Link URI (if formatted as [Title](URI))
	// Group 4: Raw URI (if formatted as plain URL/path)
	re := regexp.MustCompile(`^\[(\d+)\]\s+(?:\[([^\]]+)\]\(([^)]+)\)|(\S+))`)

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
			matches := re.FindStringSubmatch(line)
			if len(matches) > 0 {
				numVal := 0
				fmt.Sscanf(matches[1], "%d", &numVal)

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
