package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

func TestValidateFrontmatterPaths(t *testing.T) {
	root := t.TempDir()
	writeBundleFile := func(relPath string) {
		t.Helper()
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("create path for %q: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte("fixture"), 0o644); err != nil {
			t.Fatalf("write %q: %v", relPath, err)
		}
	}
	for _, relPath := range []string{
		"references/resource.md",
		"references/source.md",
		"references/source.custom",
		"metrics/resource.custom",
		"metrics/query.sql",
		"references/runner.md",
		"references/attester.py",
	} {
		writeBundleFile(relPath)
	}

	concept := &bundle.Concept{
		ID:   "metrics/revenue",
		Path: "metrics/revenue.md",
		Frontmatter: bundle.Frontmatter{
			Type:     "Attested Computation",
			Title:    "Revenue",
			Desc:     "Calculate revenue.",
			Resource: "resource.custom",
			Sources: []bundle.Source{
				{Resource: "../references/source.md"},
				{Resource: "../references/source.custom"},
				{Resource: "https://example.com/source"},
				{Resource: "all queries in BigQuery project X"},
			},
			Computation: "query.sql",
			Executor:    &bundle.Executor{Resource: "/references/runner.md", Receipt: []string{"job_id"}},
			Attester:    &bundle.Attester{Resource: "../references/attester.py"},
		},
	}
	b := bundle.NewBundle(root)
	b.Concepts[concept.ID] = concept

	issues := validateFrontmatterPaths(concept, b)
	if len(issues) != 0 {
		t.Fatalf("expected all spec path fields to resolve without issues, got: %+v", issues)
	}
}

func TestValidateSpecAttestedComputation(t *testing.T) {
	root := t.TempDir()
	for _, relPath := range []string{"metrics/query.sql", "references/runner.md", "references/attester.py"} {
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("create path for %q: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte("fixture"), 0o644); err != nil {
			t.Fatalf("write %q: %v", relPath, err)
		}
	}

	concept := &bundle.Concept{
		ID:   "metrics/revenue",
		Path: "metrics/revenue.md",
		Frontmatter: bundle.Frontmatter{
			Type:        "Attested Computation",
			Title:       "Revenue",
			Desc:        "Calculate revenue.",
			Runtime:     "bigquery",
			Computation: "query.sql",
			Executor:    &bundle.Executor{Resource: "../references/runner.md"},
			Attester:    &bundle.Attester{Resource: "../references/attester.py"},
		},
	}
	b := bundle.NewBundle(root)
	b.Concepts[concept.ID] = concept

	if issues := ValidateConcept(concept, b, Options{}, make(remoteCache)); len(issues) != 0 {
		t.Fatalf("expected valid spec attested computation to have no issues, got: %+v", issues)
	}
}

func TestValidateFrontmatterPathsReportsMissingTargets(t *testing.T) {
	root := t.TempDir()
	concept := &bundle.Concept{
		ID:   "metrics/revenue",
		Path: "metrics/revenue.md",
		Frontmatter: bundle.Frontmatter{
			Type:        "Attested Computation",
			Resource:    "../missing/resource.md",
			Sources:     []bundle.Source{{Resource: "references/missing-source.md"}},
			Computation: "../missing/computation.sql",
			Executor:    &bundle.Executor{Resource: "../missing/executor.md"},
			Attester:    &bundle.Attester{Resource: "../missing/attester.py"},
		},
	}
	b := bundle.NewBundle(root)
	b.Concepts[concept.ID] = concept

	issues := validateFrontmatterPaths(concept, b)
	expectedFields := []string{
		"field 'resource'",
		"field 'sources[0].resource'",
		"field 'computation'",
		"field 'executor.resource'",
		"field 'attester.resource'",
	}
	if len(issues) != len(expectedFields) {
		t.Fatalf("expected %d missing-path warnings, got %d: %+v", len(expectedFields), len(issues), issues)
	}
	for i, issue := range issues {
		if issue.Severity != SeverityWarning {
			t.Errorf("expected warning severity, got %q", issue.Severity)
		}
		if !strings.Contains(issue.Message, expectedFields[i]) || !strings.Contains(issue.Message, "could not be resolved") {
			t.Errorf("expected missing-path warning for %s, got %q", expectedFields[i], issue.Message)
		}
	}
}

func TestValidateFrontmatterPathsGatesAttestedComputationFieldsByType(t *testing.T) {
	concept := &bundle.Concept{
		ID:   "metrics/revenue",
		Path: "metrics/revenue.md",
		Frontmatter: bundle.Frontmatter{
			Type:        "Metric",
			Computation: "missing.sql",
			Executor:    &bundle.Executor{Resource: "missing-runner.md"},
			Attester:    &bundle.Attester{Resource: "missing-attester.py"},
		},
	}
	b := bundle.NewBundle(t.TempDir())
	b.Concepts[concept.ID] = concept

	if issues := validateFrontmatterPaths(concept, b); len(issues) != 0 {
		t.Fatalf("expected Attested Computation-only paths to be ignored for Metric concepts, got: %+v", issues)
	}

	concept.Frontmatter.Type = " Attested Computation "
	issues := validateFrontmatterPaths(concept, b)
	if len(issues) != 3 {
		t.Fatalf("expected all three Attested Computation paths to be checked, got: %+v", issues)
	}
}

func TestValidateFrontmatterPathsSkipsOpaqueResourceValues(t *testing.T) {
	concept := &bundle.Concept{
		ID:   "metric",
		Path: "metric.md",
		Frontmatter: bundle.Frontmatter{
			Resource: "analytics.mrr_daily",
			Sources: []bundle.Source{
				{Resource: "all queries in BigQuery project X"},
				{Resource: "https://example.com/source"},
				{Resource: "bigquery.analytics.mrr_query"},
				{Resource: "analytics_mrr_daily"},
			},
		},
	}
	b := bundle.NewBundle(t.TempDir())
	b.Concepts[concept.ID] = concept

	if issues := validateFrontmatterPaths(concept, b); len(issues) != 0 {
		t.Fatalf("expected opaque values and URLs to be skipped, got: %+v", issues)
	}
}

func TestValidateFrontmatterPathsWarnsOnExplicitMissingPaths(t *testing.T) {
	concept := &bundle.Concept{
		ID:   "metric",
		Path: "metric.md",
		Frontmatter: bundle.Frontmatter{
			Resource: "./missing.custom",
			Sources: []bundle.Source{
				{Resource: "missing.sql"},
				{Resource: "../references/missing.md"},
			},
		},
	}
	b := bundle.NewBundle(t.TempDir())
	b.Concepts[concept.ID] = concept

	issues := validateFrontmatterPaths(concept, b)
	if len(issues) != 3 {
		t.Fatalf("expected 3 broken path warnings for explicit missing paths, got %d: %+v", len(issues), issues)
	}
}

func TestValidateFrontmatterPathsResolvesDottedIdentifierAsFile(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "metrics", "analytics.mrr_daily")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("create concept directory: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write dotted-identifier file: %v", err)
	}

	concept := &bundle.Concept{
		ID:   "metrics/revenue",
		Path: "metrics/revenue.md",
		Frontmatter: bundle.Frontmatter{
			Resource: "analytics.mrr_daily",
		},
	}
	b := bundle.NewBundle(root)
	b.Concepts[concept.ID] = concept

	if issues := validateFrontmatterPaths(concept, b); len(issues) != 0 {
		t.Fatalf("expected dotted identifier to resolve as a file beside the concept, got: %+v", issues)
	}
}

func TestValidateFrontmatterPathsRejectsPathsOutsideBundle(t *testing.T) {
	concept := &bundle.Concept{
		ID:   "metrics/revenue",
		Path: "metrics/revenue.md",
		Frontmatter: bundle.Frontmatter{
			Type:        "Attested Computation",
			Computation: "../../../outside.sql",
		},
	}
	b := bundle.NewBundle(t.TempDir())
	b.Concepts[concept.ID] = concept

	issues := validateFrontmatterPaths(concept, b)
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "outside the bundle") {
		t.Fatalf("expected an outside-bundle warning, got: %+v", issues)
	}
}
