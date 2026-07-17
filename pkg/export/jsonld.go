// Package export provides utilities for exporting OKF bundles to other structured metadata formats.
package export

import (
	"encoding/json"
	"strings"

	"github.com/abcubed3/okf/pkg/bundle"
)

// Graph represents a schema.org JSON-LD Graph container.
type Graph struct {
	Context string        `json:"@context"`
	Graph   []interface{} `json:"@graph"`
}

// PropertyValue represents a schema.org PropertyValue for columns/fields.
type PropertyValue struct {
	Type           string `json:"@type"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	ValueReference string `json:"valueReference,omitempty"` // Data type
}

// Dataset represents a schema.org Dataset for database tables.
type Dataset struct {
	Type             string          `json:"@type"`
	ID               string          `json:"@id"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	Identifier       string          `json:"identifier,omitempty"`
	VariableMeasured []PropertyValue `json:"variableMeasured,omitempty"`
	Keywords         []string        `json:"keywords,omitempty"`
	DateModified     string          `json:"dateModified,omitempty"`
}

// WebAPI represents a schema.org WebAPI for API endpoints.
type WebAPI struct {
	Type          string   `json:"@type"`
	ID            string   `json:"@id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Documentation string   `json:"documentation,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	DateModified  string   `json:"dateModified,omitempty"`
}

// TechArticle represents a schema.org TechArticle for general human documentation or metrics.
type TechArticle struct {
	Type         string   `json:"@type"`
	ID           string   `json:"@id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	ArticleBody  string   `json:"articleBody,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	DateModified string   `json:"dateModified,omitempty"`
}

// ExportBundleToJSONLD serializes all concepts in an OKF Bundle to schema.org JSON-LD.
func ExportBundleToJSONLD(b *bundle.Bundle) ([]byte, error) {
	graphList := make([]interface{}, 0, len(b.Concepts))

	for _, c := range b.Concepts {
		if c.ParseError != "" {
			continue
		}

		cType := c.Frontmatter.Type
		id := c.ID
		title := c.Frontmatter.Title
		if title == "" {
			title = c.ID
		}
		desc := c.Frontmatter.Desc
		tags := c.Frontmatter.Tags
		timestamp := c.Frontmatter.Timestamp

		// Determine the mapping based on Type
		isTable := strings.Contains(strings.ToLower(cType), "table")
		isAPI := strings.Contains(strings.ToLower(cType), "api") || strings.Contains(strings.ToLower(cType), "endpoint")

		if isTable {
			columns := parseColumnsFromMarkdown(c.Body)
			dataset := Dataset{
				Type:             "Dataset",
				ID:               "okf://" + id,
				Name:             title,
				Description:      desc,
				Identifier:       c.Frontmatter.Resource,
				VariableMeasured: columns,
				Keywords:         tags,
				DateModified:     timestamp,
			}
			graphList = append(graphList, dataset)
		} else if isAPI {
			api := WebAPI{
				Type:          "WebAPI",
				ID:            "okf://" + id,
				Name:          title,
				Description:   desc,
				Documentation: c.Frontmatter.Resource,
				Keywords:      tags,
				DateModified:  timestamp,
			}
			graphList = append(graphList, api)
		} else {
			article := TechArticle{
				Type:         "TechArticle",
				ID:           "okf://" + id,
				Name:         title,
				Description:  desc,
				ArticleBody:  c.Body,
				Keywords:     tags,
				DateModified: timestamp,
			}
			graphList = append(graphList, article)
		}
	}

	g := Graph{
		Context: "https://schema.org",
		Graph:   graphList,
	}

	return json.MarshalIndent(g, "", "  ")
}

// parseColumnsFromMarkdown extracts a table's schema fields from the markdown body table.
func parseColumnsFromMarkdown(body string) []PropertyValue {
	var cols []PropertyValue
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			continue
		}
		// Skip header divider rows (e.g. |---|---|)
		if strings.Contains(line, "---") {
			continue
		}
		// Split by pipe
		parts := strings.Split(line, "|")
		// The split will have empty string at parts[0] and parts[len-1] because of leading/trailing pipes.
		if len(parts) < 6 {
			continue
		}
		colName := strings.TrimSpace(parts[1])
		// Skip table header "Column" or "Field"
		if strings.ToLower(colName) == "column" || strings.ToLower(colName) == "field" {
			continue
		}
		colType := strings.TrimSpace(parts[2])
		colDesc := strings.TrimSpace(parts[5])

		cols = append(cols, PropertyValue{
			Type:           "PropertyValue",
			Name:           colName,
			Description:    colDesc,
			ValueReference: colType,
		})
	}
	return cols
}
func GetJSONLDString(b *bundle.Bundle) (string, error) {
	bytes, err := ExportBundleToJSONLD(b)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
