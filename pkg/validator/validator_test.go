package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/parser"
)

func TestValidateBundle(t *testing.T) {
	b := bundle.NewBundle("/fake/path")

	// 1. A perfectly valid concept
	b.Concepts["valid"] = &bundle.Concept{
		ID:   "valid",
		Path: "valid.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Metric",
			Title: "Active Users",
			Desc:  "Number of unique active users.",
		},
		Body: "Looks good.",
	}

	// 2. Concept with missing type
	b.Concepts["missing-type"] = &bundle.Concept{
		ID:   "missing-type",
		Path: "missing-type.md",
		Frontmatter: bundle.Frontmatter{
			Title: "No Type Table",
			Desc:  "This has no type.",
		},
		Body: "Body text.",
	}

	// 3. Concept with missing recommended fields (warnings)
	b.Concepts["missing-desc"] = &bundle.Concept{
		ID:   "missing-desc",
		Path: "missing-desc.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "BigQuery Table",
			Title: "Titled Table",
		},
		Body: "Body text.",
	}

	// 4. Concept with broken and valid links
	b.Concepts["links-test"] = &bundle.Concept{
		ID:   "links-test",
		Path: "links-test.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Playbook",
			Title: "Test Links",
			Desc:  "Checking link verification.",
		},
		Body: `This is a link to [Valid Concept](valid.md) and a [Broken Link](non-existent.md).
Check out this [External Link](https://google.com) which should be ignored.`,
	}

	// 5. Concept with absolute links (valid and broken)
	b.Concepts["abs-links-test"] = &bundle.Concept{
		ID:   "abs-links-test",
		Path: "sub/folder/abs-links-test.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Playbook",
			Title: "Test Absolute Links",
			Desc:  "Checking absolute link verification.",
		},
		Body: `A [Valid Absolute](/valid.md) and a [Broken Absolute](/non-existent-abs.md).`,
	}

	// 6. Concept with citations (valid and broken)
	b.Concepts["citations-test"] = &bundle.Concept{
		ID:   "citations-test",
		Path: "citations-test.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Playbook",
			Title: "Test Citations",
			Desc:  "Checking citation validation.",
		},
		Citations: []bundle.Citation{
			{Number: 1, Title: "Valid Link", URI: "valid.md"},
			{Number: 2, Title: "Broken Link", URI: "non-existent-cite.md"},
			{Number: 3, Title: "External Link", URI: "https://example.com"},
		},
		Body: "Testing citations.",
	}

	issues := ValidateBundle(b, Options{})

	// We expect:
	// 1. Error on "missing-type" (missing type field)
	// 2. Warning on "missing-desc" (missing description field)
	// 3. Warning on "links-test" (broken link to non-existent.md)
	// 4. Warning on "abs-links-test" (broken absolute link)
	// 5. Warning on "citations-test" (broken citation link)
	expectedErrors := map[string]string{
		"missing-type": "missing or empty 'type' field in frontmatter",
	}
	expectedWarnings := map[string]string{
		"missing-desc":   "missing recommended 'description' field",
		"links-test":     "broken link: target concept 'non-existent' (resolved from 'non-existent.md') does not exist",
		"abs-links-test": "broken link: target concept 'non-existent-abs' (resolved from '/non-existent-abs.md') does not exist",
		"citations-test": "broken citation link: target concept 'non-existent-cite' (resolved from 'non-existent-cite.md') does not exist",
	}

	for _, issue := range issues {
		switch issue.Severity {
		case SeverityError:
			expectedMsg, exists := expectedErrors[issue.ConceptID]
			if !exists {
				t.Errorf("unexpected error on concept %q: %s", issue.ConceptID, issue.Message)
			} else if issue.Message != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, issue.Message)
			}
			delete(expectedErrors, issue.ConceptID)
		case SeverityWarning:
			expectedMsg, exists := expectedWarnings[issue.ConceptID]
			if !exists {
				t.Errorf("unexpected warning on concept %q: %s", issue.ConceptID, issue.Message)
			} else if issue.Message != expectedMsg {
				t.Errorf("expected warning message %q, got %q", expectedMsg, issue.Message)
			}
			delete(expectedWarnings, issue.ConceptID)
		}
	}

	for id := range expectedErrors {
		t.Errorf("missing expected error on concept %q", id)
	}
	for id := range expectedWarnings {
		t.Errorf("missing expected warning on concept %q", id)
	}
}

func TestValidateBundle_SubdirectoryIndexLinks(t *testing.T) {
	root := t.TempDir()
	writeFile := func(relPath, content string) {
		t.Helper()
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", relPath, err)
		}
	}

	writeFile("index.md", "---\nokf_version: \"0.2\"\n---\n# Bundle Index\n- [APIs](apis/index.md)\n- [Playbooks](playbooks/database_cleanup.md)\n")
	writeFile("log.md", "# Change Log\n")
	writeFile("apis/index.md", "# APIs\n- [Create User](create_user.md)\n")
	writeFile("apis/create_user.md", "---\ntype: API\ntitle: Create User\ndescription: Creates a user.\n---\n# Create User\n")
	writeFile("playbooks/database_cleanup.md", `---
type: Playbook
title: Database Cleanup Playbook
description: Cleans up database records.
---

# Database Cleanup Playbook

Refers to [apis](/apis/index.md), [apis relative](../apis/index.md), [apis dir](/apis/), [root index](/index.md), and [changelog](/log.md).
Also refers to [missing index](/missing/index.md).
`)

	b, err := parser.ParseBundle(context.Background(), root)
	if err != nil {
		t.Fatalf("ParseBundle failed: %v", err)
	}

	issues := ValidateBundle(b, Options{})
	if len(issues) != 1 {
		t.Fatalf("expected 1 warning (for /missing/index.md), got %d: %+v", len(issues), issues)
	}
	if issues[0].ConceptID != "playbooks/database_cleanup" || !strings.Contains(issues[0].Message, "/missing/index.md") {
		t.Errorf("unexpected issue: %+v", issues[0])
	}
}

func TestValidateBundle_IndexFileOutboundLinks(t *testing.T) {
	root := t.TempDir()
	writeFile := func(relPath, content string) {
		t.Helper()
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", relPath, err)
		}
	}

	writeFile("apis/index.md", "# APIs\n- [Create User](create_user.md)\n- [Deleted API](deleted_api.md)\n")
	writeFile("apis/create_user.md", "---\ntype: API\ntitle: Create User\ndescription: Creates a user.\n---\n# Create User\n")

	b, err := parser.ParseBundle(context.Background(), root)
	if err != nil {
		t.Fatalf("ParseBundle failed: %v", err)
	}

	issues := ValidateBundle(b, Options{})
	if len(issues) != 1 {
		t.Fatalf("expected 1 broken link warning from apis/index.md, got %d: %+v", len(issues), issues)
	}
	if filepath.ToSlash(issues[0].Path) != "apis/index.md" || !strings.Contains(issues[0].Message, "deleted_api.md") {
		t.Errorf("unexpected issue: %+v", issues[0])
	}
}

func BenchmarkValidateBundle(b *testing.B) {
	bundleObj := bundle.NewBundle("/fake/path")
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("concept-%d", i)
		bundleObj.Concepts[id] = &bundle.Concept{
			ID:   id,
			Path: fmt.Sprintf("concept-%d.md", i),
			Frontmatter: bundle.Frontmatter{
				Type:  "Metric",
				Title: fmt.Sprintf("Metric %d", i),
				Desc:  fmt.Sprintf("Metric description %d", i),
			},
			Body: fmt.Sprintf("This is body %d linking to [concept-%d](concept-%d.md)", i, (i+1)%100, (i+1)%100),
		}
	}

	for b.Loop() {
		_ = ValidateBundle(bundleObj, Options{})
	}
}
