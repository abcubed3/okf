package skill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/skill"
)

func createTestBundle() *bundle.Bundle {
	b := bundle.NewBundle("/test/bundle")

	// 1. Metric concept
	b.Concepts["metrics/mrr"] = &bundle.Concept{
		ID:   "metrics/mrr",
		Path: "/test/bundle/metrics/mrr.md",
		Frontmatter: bundle.Frontmatter{
			Type:       "Metric",
			Title:      "Monthly Recurring Revenue",
			Desc:       "Normalized monthly revenue from active customer subscriptions.",
			Resource:   "analytics.mrr_daily",
			Tags:       []string{"finance", "revenue"},
			Status:     bundle.ConceptLifecycleStable,
			StaleAfter: "2027-01-01T00:00:00Z",
			Verified: []bundle.ActorInfo{
				{By: "human:alice@company.com", At: "2026-08-01T12:00:00Z"},
			},
		},
		Body: "## Business Logic\nOnly active subscriptions count.",
	}

	// 2. Attestation concept
	b.Concepts["attestations/mrr_query"] = &bundle.Concept{
		ID:   "attestations/mrr_query",
		Path: "/test/bundle/attestations/mrr_query.md",
		Frontmatter: bundle.Frontmatter{
			Type:   "Attestation",
			Title:  "Sanctioned MRR Calculation Query",
			Desc:   "Audited query for MRR calculation.",
			Tags:   []string{"finance", "sql"},
			Status: bundle.ConceptLifecycleStable,
			Attestation: &bundle.Attestation{
				Query:    "SELECT SUM(amount) AS total_mrr FROM subscriptions WHERE status = 'active';",
				Executor: "BigQuery",
				Attester: "finance-team@company.com",
			},
			Verified: []bundle.ActorInfo{
				{By: "human:bob@company.com", At: "2026-08-01T12:00:00Z"},
			},
		},
		Body: "Sanctioned computation query.",
	}

	// 3. Table concept
	b.Concepts["tables/users"] = &bundle.Concept{
		ID:   "tables/users",
		Path: "/test/bundle/tables/users.md",
		Frontmatter: bundle.Frontmatter{
			Type:   "PostgreSQL Table",
			Title:  "Users Table",
			Desc:   "User account entities.",
			Tags:   []string{"core", "users"},
			Status: bundle.ConceptLifecycleStable,
		},
		Body: "| id | email | status |",
	}

	return b
}

func TestExportBundleToSkill(t *testing.T) {
	b := createTestBundle()

	opts := skill.ExportOptions{
		Name:                "finance-knowledge",
		Description:         "Finance metrics and sanctioned queries.",
		IncludeAttestations: true,
	}

	content, err := skill.ExportBundleToSkill(b, opts)
	if err != nil {
		t.Fatalf("ExportBundleToSkill failed: %v", err)
	}

	// Verify frontmatter
	if !strings.Contains(content, "name: finance-knowledge") {
		t.Errorf("Expected skill frontmatter to contain 'name: finance-knowledge', got:\n%s", content)
	}
	if !strings.Contains(content, "description: Finance metrics and sanctioned queries.") {
		t.Errorf("Expected skill frontmatter description, got:\n%s", content)
	}

	// Verify Attested Computations section
	if !strings.Contains(content, "## 📐 Sanctioned Computations & Attested Queries") {
		t.Errorf("Expected sanctioned computations section")
	}
	if !strings.Contains(content, "SELECT SUM(amount) AS total_mrr") {
		t.Errorf("Expected attested SQL query to be included")
	}

	// Verify Concepts Catalog
	if !strings.Contains(content, "## 📚 Domain Concepts & Schema Catalog") {
		t.Errorf("Expected domain concepts catalog section")
	}
	if !strings.Contains(content, "Monthly Recurring Revenue (`metrics/mrr`)") {
		t.Errorf("Expected MRR concept in catalog")
	}
	if !strings.Contains(content, "Users Table (`tables/users`)") {
		t.Errorf("Expected Users Table concept in catalog")
	}
}

func TestExportBundleToSkill_Filtering(t *testing.T) {
	b := createTestBundle()

	// Filter by tag: finance
	opts := skill.ExportOptions{
		Name: "finance-only",
		Tags: []string{"finance"},
	}

	content, err := skill.ExportBundleToSkill(b, opts)
	if err != nil {
		t.Fatalf("ExportBundleToSkill failed: %v", err)
	}

	if !strings.Contains(content, "Monthly Recurring Revenue") {
		t.Errorf("Expected finance metric to be included")
	}
	if strings.Contains(content, "Users Table") {
		t.Errorf("Expected non-finance concept 'Users Table' to be excluded")
	}

	// Filter by type: Metric
	optsType := skill.ExportOptions{
		Name:  "metrics-only",
		Types: []string{"Metric"},
	}

	contentMetric, err := skill.ExportBundleToSkill(b, optsType)
	if err != nil {
		t.Fatalf("ExportBundleToSkill failed: %v", err)
	}

	if !strings.Contains(contentMetric, "Monthly Recurring Revenue") {
		t.Errorf("Expected Metric concept to be included")
	}
	if strings.Contains(contentMetric, "Sanctioned MRR Calculation Query") {
		t.Errorf("Expected Attestation concept to be excluded when filtering by type=Metric")
	}
}

func TestInstallOfficialSkill(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "skills", "okf")

	installedPath, err := skill.InstallOfficialSkill(skill.InstallOptions{
		TargetDir: targetDir,
	})
	if err != nil {
		t.Fatalf("InstallOfficialSkill failed: %v", err)
	}

	if installedPath != targetDir {
		t.Errorf("Expected installed path %s, got %s", targetDir, installedPath)
	}

	skillFile := filepath.Join(targetDir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("Failed to read installed SKILL.md: %v", err)
	}

	if !strings.Contains(string(data), "name: okf") {
		t.Errorf("Expected installed SKILL.md to contain 'name: okf'")
	}

	refFile := filepath.Join(targetDir, "references", "spec-v02.md")
	if _, err := os.Stat(refFile); os.IsNotExist(err) {
		t.Errorf("Expected reference file spec-v02.md to exist")
	}
}
