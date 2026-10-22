package ai_test

import (
	"context"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/ai"
	"github.com/abcubed3/okf/pkg/bundle"
)

func createSampleBundle() *bundle.Bundle {
	b := bundle.NewBundle("/test")

	b.Concepts["metrics/mrr"] = &bundle.Concept{
		ID:   "metrics/mrr",
		Path: "/test/metrics/mrr.md",
		Frontmatter: bundle.Frontmatter{
			Type:       "Metric",
			Title:      "Monthly Recurring Revenue",
			Desc:       "Normalized monthly revenue from active customer subscriptions.",
			Tags:       []string{"finance", "revenue"},
			Resource:   "analytics.mrr_daily",
			Status:     bundle.ConceptLifecycleStable,
			StaleAfter: "2027-01-01T00:00:00Z",
			Verified: []bundle.ActorInfo{
				{By: "human:alice@company.com", At: "2026-08-01T12:00:00Z"},
			},
		},
		Body: "Related: [Sanctioned MRR Query](../attestations/mrr_query.md)",
	}

	b.Concepts["attestations/mrr_query"] = &bundle.Concept{
		ID:   "attestations/mrr_query",
		Path: "/test/attestations/mrr_query.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Attestation",
			Title: "Sanctioned MRR Calculation Query",
			Desc:  "Audited query for MRR calculation.",
			Tags:  []string{"finance", "sql"},
			Attestation: &bundle.Attestation{
				Query:    "SELECT SUM(amount) AS total_mrr FROM subscriptions WHERE status = 'active';",
				Executor: "BigQuery",
				Attester: "finance-team@company.com",
			},
			Verified: []bundle.ActorInfo{
				{By: "human:bob@company.com", At: "2026-08-01T12:00:00Z"},
			},
		},
		Body: "Attested computation details.",
	}

	b.Concepts["tables/users"] = &bundle.Concept{
		ID:   "tables/users",
		Path: "/test/tables/users.md",
		Frontmatter: bundle.Frontmatter{
			Type:     "PostgreSQL Table",
			Title:    "Users Table",
			Desc:     "Store user accounts.",
			Tags:     []string{"core", "auth"},
			Resource: "public.users",
		},
		Body: "| id | email |",
	}

	return b
}

func TestFindSeedConcepts(t *testing.T) {
	b := createSampleBundle()

	// Query matching finance tag & mrr title
	matches := ai.FindSeedConcepts(b, "How do we calculate monthly recurring revenue MRR?")
	if len(matches) == 0 {
		t.Fatalf("Expected to find matches for MRR query")
	}

	top := matches[0]
	if top.ID != "metrics/mrr" && top.ID != "attestations/mrr_query" {
		t.Errorf("Expected top match to be metrics/mrr or attestations/mrr_query, got: %s", top.ID)
	}

	// Query matching users table
	userMatches := ai.FindSeedConcepts(b, "users email accounts")
	if len(userMatches) == 0 {
		t.Fatalf("Expected to find matches for users query")
	}
	if userMatches[0].ID != "tables/users" {
		t.Errorf("Expected top match to be tables/users, got: %s", userMatches[0].ID)
	}
}

func TestAskBundle_OfflineAssembly(t *testing.T) {
	b := createSampleBundle()

	opts := ai.AskOptions{
		// Leaving APIKey empty to test offline assembly and grounding
		APIKey: "",
		Depth:  2,
	}

	res, err := ai.AskBundle(context.Background(), b, "How is MRR computed?", opts)
	if err != nil {
		t.Fatalf("AskBundle failed: %v", err)
	}

	if res.SeedConcept != "metrics/mrr" && res.SeedConcept != "attestations/mrr_query" {
		t.Errorf("Expected SeedConcept to be MRR related, got: %s", res.SeedConcept)
	}

	if len(res.ReferencedConcepts) == 0 {
		t.Errorf("Expected ReferencedConcepts to not be empty")
	}

	if len(res.Attestations) == 0 {
		t.Errorf("Expected attestations to be found in assembled context")
	} else {
		if !strings.Contains(res.Attestations[0], "SELECT SUM(amount) AS total_mrr") {
			t.Errorf("Expected attested SQL in result, got: %s", res.Attestations[0])
		}
	}

	if !strings.Contains(res.Answer, "Sanctioned Computation Query (Attested)") {
		t.Errorf("Expected offline answer to highlight sanctioned computation query")
	}
}
