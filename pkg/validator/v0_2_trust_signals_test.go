package validator

import (
	"testing"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
)

func TestValidateTrustSignalsV02(t *testing.T) {
	b := bundle.NewBundle("/fake/path")

	// 1. Valid OKF v0.2 concept
	futureTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	b.Concepts["valid-v02"] = &bundle.Concept{
		ID:   "valid-v02",
		Path: "valid-v02.md",
		Frontmatter: bundle.Frontmatter{
			Type:       "Metric",
			Title:      "Monthly Recurring Revenue",
			Desc:       "Sum of recurring subscription revenues.",
			Status:     "stable",
			StaleAfter: futureTime,
			Sources: []bundle.Source{
				{URI: "https://stripe.com", Title: "Stripe Billing Docs", Author: "Stripe Team"},
			},
			Generated: &bundle.ActorInfo{By: "harvest-job", At: time.Now().Format(time.RFC3339)},
			Verified: []bundle.ActorInfo{
				{By: "alice", At: time.Now().Format(time.RFC3339), Tier: "human"},
			},
		},
		Body: "MRR calculation rules.",
	}

	// 2. Deprecated status
	b.Concepts["deprecated-concept"] = &bundle.Concept{
		ID:   "deprecated-concept",
		Path: "deprecated-concept.md",
		Frontmatter: bundle.Frontmatter{
			Type:   "Legacy Metric",
			Title:  "Old MRR",
			Desc:   "Deprecated calculation method.",
			Status: "deprecated",
		},
		Body: "Old body.",
	}

	// 3. Unrecognized status
	b.Concepts["invalid-status"] = &bundle.Concept{
		ID:   "invalid-status",
		Path: "invalid-status.md",
		Frontmatter: bundle.Frontmatter{
			Type:   "Metric",
			Title:  "Status Test",
			Desc:   "Testing status validation.",
			Status: "unknown-state",
		},
		Body: "Body.",
	}

	// 4. Stale concept (stale_after in the past)
	pastTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	b.Concepts["stale-concept"] = &bundle.Concept{
		ID:   "stale-concept",
		Path: "stale-concept.md",
		Frontmatter: bundle.Frontmatter{
			Type:       "Metric",
			Title:      "Stale Metric",
			Desc:       "Expired concept.",
			StaleAfter: pastTime,
		},
		Body: "Stale body.",
	}

	// 5. Attestation concept without attestation query/executor
	b.Concepts["invalid-attestation"] = &bundle.Concept{
		ID:   "invalid-attestation",
		Path: "invalid-attestation.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Attested Computation",
			Title: "Unsanctioned Query",
			Desc:  "Missing query details.",
		},
		Body: "Attestation body.",
	}

	// 6. Valid Attestation concept
	b.Concepts["valid-attestation"] = &bundle.Concept{
		ID:   "valid-attestation",
		Path: "valid-attestation.md",
		Frontmatter: bundle.Frontmatter{
			Type:  "Attestation",
			Title: "Sanctioned Revenue Query",
			Desc:  "Verified SQL for MRR.",
			Attestation: &bundle.Attestation{
				Query:    "SELECT SUM(amount) FROM subscriptions WHERE status = 'active';",
				Executor: "BigQuery",
				Attester: "data-team-lead",
			},
		},
		Body: "Sanctioned computation definition.",
	}

	issues := ValidateBundle(b, Options{})

	expectedErrors := map[string]bool{
		"invalid-attestation": true,
	}
	expectedWarnings := map[string]bool{
		"deprecated-concept": true,
		"invalid-status":     true,
		"stale-concept":       true,
	}

	for _, issue := range issues {
		if issue.Severity == SeverityError {
			if !expectedErrors[issue.ConceptID] {
				t.Errorf("unexpected error on concept %q: %s", issue.ConceptID, issue.Message)
			}
			delete(expectedErrors, issue.ConceptID)
		} else if issue.Severity == SeverityWarning {
			if !expectedWarnings[issue.ConceptID] {
				t.Errorf("unexpected warning on concept %q: %s", issue.ConceptID, issue.Message)
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

func TestConceptTrustHelpers(t *testing.T) {
	pastTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	cStale := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{
			StaleAfter: pastTime.Format(time.RFC3339),
		},
	}
	if !cStale.IsStale(time.Now()) {
		t.Errorf("expected concept to be stale")
	}

	cFresh := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{
			StaleAfter: futureTime.Format(time.RFC3339),
		},
	}
	if cFresh.IsStale(time.Now()) {
		t.Errorf("expected concept to be fresh")
	}

	cUnver := &bundle.Concept{}
	if cUnver.IsVerified() {
		t.Errorf("expected concept to be unverified")
	}
	if cUnver.TrustTier() != "unverified" {
		t.Errorf("expected trust tier 'unverified', got %q", cUnver.TrustTier())
	}

	cVer := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{
			Verified: []bundle.ActorInfo{
				{By: "bot", Tier: "automated"},
				{By: "lead", Tier: "certified"},
			},
		},
	}
	if !cVer.IsVerified() {
		t.Errorf("expected concept to be verified")
	}
	if cVer.TrustTier() != "certified" {
		t.Errorf("expected trust tier 'certified', got %q", cVer.TrustTier())
	}
}
