package bundle

import (
	"testing"
)

func TestMerge(t *testing.T) {
	// 1. Setup Source Bundle
	source := NewBundle("/source")
	source.Concepts["tables/users"] = &Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: Frontmatter{
			Type:      "PostgreSQL Table",
			Title:     "Users Table",
			Desc:      "Stores accounts",
			Timestamp: "2026-07-15T00:00:00Z",
			Tags:      []string{"auth", "profile"},
			Extra:     map[string]interface{}{"schema": "public"},
		},
		Body: "Source Body",
	}
	source.Concepts["playbooks/old"] = &Concept{
		ID:   "playbooks/old",
		Path: "playbooks/old.md",
		Frontmatter: Frontmatter{
			Type:  "Playbook",
			Title: "Old Playbook",
		},
		Body: "Old Content",
	}

	// 2. Setup Target Bundle
	target := NewBundle("/target")
	// tables/users conflicts (newer timestamp)
	target.Concepts["tables/users"] = &Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: Frontmatter{
			Type:      "PostgreSQL Table",
			Title:     "Users Table (Modified)",
			Timestamp: "2026-07-16T00:00:00Z", // Newer!
			Tags:      []string{"auth", "active"},
			Extra:     map[string]interface{}{"shard": "true"},
		},
		Body: "Target Body",
	}
	target.Concepts["playbooks/new"] = &Concept{
		ID:   "playbooks/new",
		Path: "playbooks/new.md",
		Frontmatter: Frontmatter{
			Type:  "Playbook",
			Title: "New Playbook",
		},
		Body: "New Content",
	}

	// Test Strategy: Ours
	mergedOurs, err := Merge(source, target, MergeOurs)
	if err != nil {
		t.Fatalf("Ours merge failed: %v", err)
	}
	if len(mergedOurs.Concepts) != 3 {
		t.Errorf("Expected 3 concepts, got %d", len(mergedOurs.Concepts))
	}
	if mergedOurs.Concepts["tables/users"].Frontmatter.Title != "Users Table" {
		t.Errorf("Ours strategy did not keep source concept on conflict")
	}

	// Test Strategy: Theirs
	mergedTheirs, err := Merge(source, target, MergeTheirs)
	if err != nil {
		t.Fatalf("Theirs merge failed: %v", err)
	}
	if mergedTheirs.Concepts["tables/users"].Frontmatter.Title != "Users Table (Modified)" {
		t.Errorf("Theirs strategy did not keep target concept on conflict")
	}

	// Test Strategy: Union
	mergedUnion, err := Merge(source, target, MergeUnion)
	if err != nil {
		t.Fatalf("Union merge failed: %v", err)
	}

	usersConcept := mergedUnion.Concepts["tables/users"]
	if usersConcept.Body != "Target Body" {
		t.Errorf("Expected body of newer concept (Target), got %q", usersConcept.Body)
	}
	if usersConcept.Frontmatter.Title != "Users Table (Modified)" {
		t.Errorf("Expected title of newer concept, got %q", usersConcept.Frontmatter.Title)
	}
	// Verify tags merged and deduplicated/sorted
	expectedTags := []string{"active", "auth", "profile"}
	if len(usersConcept.Frontmatter.Tags) != 3 || usersConcept.Frontmatter.Tags[0] != "active" || usersConcept.Frontmatter.Tags[2] != "profile" {
		t.Errorf("Expected merged tags %v, got %v", expectedTags, usersConcept.Frontmatter.Tags)
	}
	// Verify merged extras
	if usersConcept.Frontmatter.Extra["schema"] != "public" || usersConcept.Frontmatter.Extra["shard"] != "true" {
		t.Errorf("Extra fields were not merged correctly: %v", usersConcept.Frontmatter.Extra)
	}
}
