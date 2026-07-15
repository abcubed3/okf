package validator

import (
	"fmt"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
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

	issues := ValidateBundle(b)

	// We expect:
	// 1. Error on "missing-type" (missing type field)
	// 2. Warning on "missing-desc" (missing description field)
	// 3. Warning on "links-test" (broken link to non-existent.md)
	expectedErrors := map[string]string{
		"missing-type": "missing or empty 'type' field in frontmatter",
	}
	expectedWarnings := map[string]string{
		"missing-desc": "missing recommended 'description' field",
		"links-test":   "broken link: target concept 'non-existent' (resolved from 'non-existent.md') does not exist",
	}

	for _, issue := range issues {
		if issue.Severity == SeverityError {
			expectedMsg, exists := expectedErrors[issue.ConceptID]
			if !exists {
				t.Errorf("unexpected error on concept %q: %s", issue.ConceptID, issue.Message)
			} else if issue.Message != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, issue.Message)
			}
			delete(expectedErrors, issue.ConceptID)
		} else if issue.Severity == SeverityWarning {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateBundle(bundleObj)
	}
}
