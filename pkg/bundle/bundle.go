// Package bundle defines the core structures for an OKF (Open Knowledge Format) bundle and its constituent concepts.
package bundle

import "sort"

// Bundle represents a self-contained, hierarchical collection of knowledge documents.
type Bundle struct {
	// Path is the absolute path to the bundle root directory.
	Path string
	// Concepts maps each unique Concept ID (e.g., "tables/users") to its Concept data.
	Concepts map[string]*Concept
}

// NewBundle initializes a new empty Bundle with the given root path.
func NewBundle(path string) *Bundle {
	return &Bundle{
		Path:     path,
		Concepts: make(map[string]*Concept),
	}
}

// GetConcept retrieves a concept by its ID, returning the concept and a boolean indicating success.
func (b *Bundle) GetConcept(id string) (*Concept, bool) {
	c, ok := b.Concepts[id]
	return c, ok
}

// ListTypes returns a sorted, unique list of all concept types present in the bundle.
func (b *Bundle) ListTypes() []string {
	typeMap := make(map[string]bool)
	for _, c := range b.Concepts {
		if c.Frontmatter.Type != "" {
			typeMap[c.Frontmatter.Type] = true
		}
	}

	types := make([]string, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// ListTags returns a sorted, unique list of all tags present across all concepts.
func (b *Bundle) ListTags() []string {
	tagMap := make(map[string]bool)
	for _, c := range b.Concepts {
		for _, tag := range c.Frontmatter.Tags {
			if tag != "" {
				tagMap[tag] = true
			}
		}
	}

	tags := make([]string, 0, len(tagMap))
	for t := range tagMap {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// ConceptsByType returns all concepts that match a specific type, sorted by ID to ensure a deterministic order.
func (b *Bundle) ConceptsByType(t string) []*Concept {
	var results []*Concept
	for _, c := range b.Concepts {
		if c.Frontmatter.Type == t {
			results = append(results, c)
		}
	}
	// Sort by ID to ensure deterministic order
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	return results
}
