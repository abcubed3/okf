package sync

import (
	"fmt"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/parser"
	"gopkg.in/yaml.v3"
)

const okfFrontmatterFence = "```yaml okf-frontmatter"

// SerializeConceptEnvelope renders a Concept using the unified top-of-document
// fenced ```yaml okf-frontmatter code block followed by its Markdown body.
func SerializeConceptEnvelope(c *bundle.Concept) (string, error) {
	if c == nil {
		return "", fmt.Errorf("concept is nil")
	}
	fmBytes, err := yaml.Marshal(c.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter for %q: %w", c.ID, err)
	}

	var b strings.Builder
	b.WriteString(okfFrontmatterFence)
	b.WriteByte('\n')
	b.Write(fmBytes)
	if !strings.HasSuffix(string(fmBytes), "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("```\n")
	if c.Body != "" {
		b.WriteByte('\n')
		b.WriteString(strings.TrimLeft(c.Body, "\n"))
		if !strings.HasSuffix(c.Body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

// ParseConceptEnvelope extracts an optional ```yaml okf-frontmatter block and Markdown body
// from a SaaS document. It strips UTF-8 BOMs and normalizes smart quotes inside the YAML block.
// Returns hasFrontmatter=false if no okf-frontmatter block was present (legacy document).
// Returns an error if an okf-frontmatter block is present but fails YAML unmarshaling.
func ParseConceptEnvelope(conceptID, relPath, raw string) (concept *bundle.Concept, hasFrontmatter bool, err error) {
	normalized := strings.TrimPrefix(raw, "\uFEFF")
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, okfFrontmatterFence) {
			startIdx = i
		}
		break
	}

	if startIdx == -1 {
		body := strings.TrimSpace(normalized)
		return &bundle.Concept{
			ID:        conceptID,
			Path:      relPath,
			Body:      body,
			Citations: parser.ParseCitationsFromMarkdown(body),
		}, false, nil
	}

	endIdx := -1
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```" {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return nil, true, fmt.Errorf("unclosed okf-frontmatter code block")
	}

	yamlRaw := strings.Join(lines[startIdx+1:endIdx], "\n")
	yamlRaw = normalizeSmartQuotes(yamlRaw)

	var fm bundle.Frontmatter
	if err := yaml.Unmarshal([]byte(yamlRaw), &fm); err != nil {
		return nil, true, fmt.Errorf("invalid okf-frontmatter YAML: %w", err)
	}

	body := strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
	return &bundle.Concept{
		ID:          conceptID,
		Path:        relPath,
		Frontmatter: fm,
		Body:        body,
		Citations:   parser.ParseCitationsFromMarkdown(body),
	}, true, nil
}

func normalizeSmartQuotes(s string) string {
	r := strings.NewReplacer(
		"\u201c", `"`,
		"\u201d", `"`,
		"\u2018", `'`,
		"\u2019", `'`,
		"\u2014", "--",
	)
	return r.Replace(s)
}
