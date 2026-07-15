package bundle

import (
	"testing"
)

// makeTestBundle creates a populated Bundle for test assertions.
func makeTestBundle() *Bundle {
	b := NewBundle("/tmp/test-bundle")
	b.Concepts["tables/users"] = &Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: Frontmatter{
			Type:  "PostgreSQL Table",
			Title: "Users Table",
			Desc:  "Stores user accounts.",
			Tags:  []string{"database", "table", "postgresql"},
		},
		Body: "# Users Table\n\nStores user accounts.",
	}
	b.Concepts["tables/orders"] = &Concept{
		ID:   "tables/orders",
		Path: "tables/orders.md",
		Frontmatter: Frontmatter{
			Type:  "PostgreSQL Table",
			Title: "Orders Table",
			Tags:  []string{"database", "table", "postgresql"},
		},
		Body: "# Orders Table",
	}
	b.Concepts["endpoints/get-users"] = &Concept{
		ID:   "endpoints/get-users",
		Path: "endpoints/get-users.md",
		Frontmatter: Frontmatter{
			Type:  "API Endpoint",
			Title: "GET /users",
			Tags:  []string{"api", "endpoint"},
		},
		Body: "# GET /users",
	}
	return b
}

func TestNewBundle(t *testing.T) {
	b := NewBundle("/some/path")
	if b == nil {
		t.Fatal("NewBundle returned nil")
	}
	if b.Path != "/some/path" {
		t.Errorf("expected path '/some/path', got %q", b.Path)
	}
	if b.Concepts == nil {
		t.Error("Concepts map should be initialized")
	}
	if len(b.Concepts) != 0 {
		t.Errorf("expected empty Concepts map, got %d entries", len(b.Concepts))
	}
}

func TestGetConcept(t *testing.T) {
	b := makeTestBundle()

	t.Run("existing concept", func(t *testing.T) {
		c, ok := b.GetConcept("tables/users")
		if !ok {
			t.Fatal("expected GetConcept to return true for existing concept")
		}
		if c == nil {
			t.Fatal("expected non-nil concept")
		}
		if c.ID != "tables/users" {
			t.Errorf("expected ID 'tables/users', got %q", c.ID)
		}
	})

	t.Run("missing concept", func(t *testing.T) {
		c, ok := b.GetConcept("nonexistent/thing")
		if ok {
			t.Error("expected GetConcept to return false for missing concept")
		}
		if c != nil {
			t.Error("expected nil concept for missing ID")
		}
	})
}

func TestListTypes(t *testing.T) {
	b := makeTestBundle()
	types := b.ListTypes()

	// Should return sorted, unique types
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d: %v", len(types), types)
	}
	if types[0] != "API Endpoint" {
		t.Errorf("expected first type 'API Endpoint', got %q", types[0])
	}
	if types[1] != "PostgreSQL Table" {
		t.Errorf("expected second type 'PostgreSQL Table', got %q", types[1])
	}
}

func TestListTypes_Empty(t *testing.T) {
	b := NewBundle("/empty")
	types := b.ListTypes()
	if len(types) != 0 {
		t.Errorf("expected 0 types for empty bundle, got %d", len(types))
	}
}

func TestListTypes_NoType(t *testing.T) {
	b := NewBundle("/noType")
	b.Concepts["foo"] = &Concept{ID: "foo", Path: "foo.md"}
	types := b.ListTypes()
	// Concepts with empty type should not be included
	if len(types) != 0 {
		t.Errorf("expected 0 types when all types are empty, got %d", len(types))
	}
}

func TestListTags(t *testing.T) {
	b := makeTestBundle()
	tags := b.ListTags()

	// Should return sorted, unique tags across all concepts
	// Expected: api, database, endpoint, postgresql, table
	expectedTags := []string{"api", "database", "endpoint", "postgresql", "table"}
	if len(tags) != len(expectedTags) {
		t.Fatalf("expected %d tags, got %d: %v", len(expectedTags), len(tags), tags)
	}
	for i, tag := range tags {
		if tag != expectedTags[i] {
			t.Errorf("expected tag[%d]=%q, got %q", i, expectedTags[i], tag)
		}
	}
}

func TestListTags_Deduplication(t *testing.T) {
	b := NewBundle("/dup")
	b.Concepts["a"] = &Concept{ID: "a", Frontmatter: Frontmatter{Tags: []string{"x", "y", "x"}}}
	b.Concepts["b"] = &Concept{ID: "b", Frontmatter: Frontmatter{Tags: []string{"y", "z"}}}

	tags := b.ListTags()
	// Should deduplicate: x, y, z
	if len(tags) != 3 {
		t.Errorf("expected 3 deduplicated tags, got %d: %v", len(tags), tags)
	}
}

func TestConceptsByType(t *testing.T) {
	b := makeTestBundle()

	t.Run("existing type", func(t *testing.T) {
		results := b.ConceptsByType("PostgreSQL Table")
		if len(results) != 2 {
			t.Fatalf("expected 2 concepts of type 'PostgreSQL Table', got %d", len(results))
		}
		// Results should be sorted by ID
		if results[0].ID != "tables/orders" {
			t.Errorf("expected first result ID 'tables/orders', got %q", results[0].ID)
		}
		if results[1].ID != "tables/users" {
			t.Errorf("expected second result ID 'tables/users', got %q", results[1].ID)
		}
	})

	t.Run("type with single match", func(t *testing.T) {
		results := b.ConceptsByType("API Endpoint")
		if len(results) != 1 {
			t.Fatalf("expected 1 concept of type 'API Endpoint', got %d", len(results))
		}
		if results[0].ID != "endpoints/get-users" {
			t.Errorf("expected ID 'endpoints/get-users', got %q", results[0].ID)
		}
	})

	t.Run("type with no matches", func(t *testing.T) {
		results := b.ConceptsByType("Protobuf Message")
		if len(results) != 0 {
			t.Errorf("expected 0 results for nonexistent type, got %d", len(results))
		}
	})
}

func TestConceptParseError(t *testing.T) {
	c := &Concept{
		ID:         "bad/concept",
		Path:       "bad/concept.md",
		ParseError: "yaml: unmarshal error on line 5",
	}
	if c.ParseError == "" {
		t.Error("expected ParseError to be set")
	}
}
