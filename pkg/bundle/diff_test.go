package bundle

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	// 1. Setup bundle A (source)
	bundleA := NewBundle("/path/to/a")
	bundleA.Concepts["tables/users"] = &Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: Frontmatter{
			Type:  "PostgreSQL Table",
			Title: "Users Table",
			Desc:  "Stores user accounts",
			Tags:  []string{"auth", "profile"},
			Extra: map[string]interface{}{"schema": "public"},
		},
		Body: "Raw content A",
	}
	bundleA.Concepts["playbooks/deploy"] = &Concept{
		ID:   "playbooks/deploy",
		Path: "playbooks/deploy.md",
		Frontmatter: Frontmatter{
			Type:  "Playbook",
			Title: "Deployment Guide",
		},
		Body: "Guide content",
	}

	// 2. Setup bundle B (target/new)
	bundleB := NewBundle("/path/to/b")
	// tables/users is modified (body and frontmatter description and tag changed)
	bundleB.Concepts["tables/users"] = &Concept{
		ID:   "tables/users",
		Path: "tables/users.md",
		Frontmatter: Frontmatter{
			Type:  "PostgreSQL Table",
			Title: "Users Table",
			Desc:  "Stores active user accounts",         // drifted description
			Tags:  []string{"auth", "profile", "active"}, // drifted tags
			Extra: map[string]interface{}{"schema": "public"},
		},
		Body: "Raw content B", // drifted body
	}
	// playbooks/deploy is deleted (missing in B)
	// apis/signup is added
	bundleB.Concepts["apis/signup"] = &Concept{
		ID:   "apis/signup",
		Path: "apis/signup.md",
		Frontmatter: Frontmatter{
			Type:  "API Endpoint",
			Title: "User Signup",
		},
		Body: "POST /signup",
	}

	// 3. Diff
	d, err := Diff(bundleA, bundleB)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(d.Changes) != 3 {
		t.Errorf("Expected 3 changes, got %d", len(d.Changes))
	}

	var hasAdded, hasDeleted, hasModified bool
	for _, c := range d.Changes {
		switch c.Type {
		case DiffAdded:
			if c.ID != "apis/signup" {
				t.Errorf("Expected apis/signup to be added, got %s", c.ID)
			}
			hasAdded = true
		case DiffDeleted:
			if c.ID != "playbooks/deploy" {
				t.Errorf("Expected playbooks/deploy to be deleted, got %s", c.ID)
			}
			hasDeleted = true
		case DiffModified:
			if c.ID != "tables/users" {
				t.Errorf("Expected tables/users to be modified, got %s", c.ID)
			}
			if !c.BodyDrift {
				t.Errorf("Expected body drift for tables/users")
			}
			if len(c.FmDiffs) != 2 {
				t.Errorf("Expected 2 frontmatter diffs (description, tags), got %d", len(c.FmDiffs))
			}
			if _, exists := c.FmDiffs["description"]; !exists {
				t.Errorf("Expected description metadata drift")
			}
			if _, exists := c.FmDiffs["tags"]; !exists {
				t.Errorf("Expected tags metadata drift")
			}
			hasModified = true
		}
	}

	if !hasAdded || !hasDeleted || !hasModified {
		t.Errorf("Diff did not capture all change categories (added: %v, deleted: %v, modified: %v)", hasAdded, hasDeleted, hasModified)
	}

	// 4. Test PrettyPrint format
	var buf bytes.Buffer
	d.PrettyPrint(&buf)
	output := buf.String()

	if !strings.Contains(output, "[+] ADDED    apis/signup") {
		t.Errorf("PrettyPrint output missing ADDED concept")
	}
	if !strings.Contains(output, "[-] DELETED  playbooks/deploy") {
		t.Errorf("PrettyPrint output missing DELETED concept")
	}
	if !strings.Contains(output, "[~] MODIFIED tables/users") {
		t.Errorf("PrettyPrint output missing MODIFIED concept")
	}
	if !strings.Contains(output, "Body text drifted") {
		t.Errorf("PrettyPrint output missing Body text drift message")
	}
}
