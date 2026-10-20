package bundle_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"gopkg.in/yaml.v3"
)

func TestDeriveActorTrustTier(t *testing.T) {
	tests := []struct {
		name     string
		by       string
		expected string
		isHuman  bool
	}{
		{
			name:     "empty by",
			by:       "",
			expected: bundle.TrustTierUnverified,
			isHuman:  false,
		},
		{
			name:     "whitespace only",
			by:       "   ",
			expected: bundle.TrustTierUnverified,
			isHuman:  false,
		},
		{
			name:     "human actor standard",
			by:       "human:ahormati",
			expected: bundle.TrustTierHumanReviewed,
			isHuman:  true,
		},
		{
			name:     "human actor uppercase",
			by:       "HUMAN:alice",
			expected: bundle.TrustTierHumanReviewed,
			isHuman:  true,
		},
		{
			name:     "human actor with spaces",
			by:       "  human:bob@example.com  ",
			expected: bundle.TrustTierHumanReviewed,
			isHuman:  true,
		},
		{
			name:     "process actor",
			by:       "process:finance-nightly",
			expected: bundle.TrustTierMachineConfirmed,
			isHuman:  false,
		},
		{
			name:     "agent tool producer/version",
			by:       "reference_agent/gemini-2.5-pro",
			expected: bundle.TrustTierMachineConfirmed,
			isHuman:  false,
		},
		{
			name:     "okf harvester actor",
			by:       "okf-harvester/v0.2",
			expected: bundle.TrustTierMachineConfirmed,
			isHuman:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actor := bundle.ActorInfo{By: tc.by}
			if got := actor.Tier(); got != tc.expected {
				t.Errorf("actor.Tier() = %q, expected %q", got, tc.expected)
			}
			if got := bundle.DeriveActorTrustTier(tc.by); got != tc.expected {
				t.Errorf("DeriveActorTrustTier(%q) = %q, expected %q", tc.by, got, tc.expected)
			}
			if got := actor.IsHuman(); got != tc.isHuman {
				t.Errorf("actor.IsHuman() = %v, expected %v", got, tc.isHuman)
			}
		})
	}
}

func TestConceptTrustTierPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		verified   bundle.ActorInfoList
		expected   string
		isVerified bool
	}{
		{
			name:       "nil verified",
			verified:   nil,
			expected:   bundle.TrustTierUnverified,
			isVerified: false,
		},
		{
			name:       "empty verified",
			verified:   bundle.ActorInfoList{},
			expected:   bundle.TrustTierUnverified,
			isVerified: false,
		},
		{
			name: "single empty actor in verified",
			verified: bundle.ActorInfoList{
				{By: ""},
			},
			expected:   bundle.TrustTierUnverified,
			isVerified: false,
		},
		{
			name: "single machine-confirmed actor",
			verified: bundle.ActorInfoList{
				{By: "process:finance-nightly", At: "2026-06-25T09:00:00Z"},
			},
			expected:   bundle.TrustTierMachineConfirmed,
			isVerified: true,
		},
		{
			name: "multiple machine-confirmed actors",
			verified: bundle.ActorInfoList{
				{By: "process:finance-nightly"},
				{By: "agent:audit-bot"},
			},
			expected:   bundle.TrustTierMachineConfirmed,
			isVerified: true,
		},
		{
			name: "single human-reviewed actor",
			verified: bundle.ActorInfoList{
				{By: "human:ahormati", At: "2026-06-25T09:00:00Z"},
			},
			expected:   bundle.TrustTierHumanReviewed,
			isVerified: true,
		},
		{
			name: "mixed actors - human review takes highest precedence",
			verified: bundle.ActorInfoList{
				{By: "process:finance-nightly", At: "2026-06-25T09:00:00Z"},
				{By: "human:ahormati", At: "2026-06-26T10:00:00Z"},
				{By: "reference_agent/gemini-2.5-pro", At: "2026-06-27T12:00:00Z"},
			},
			expected:   bundle.TrustTierHumanReviewed,
			isVerified: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &bundle.Concept{
				Frontmatter: bundle.Frontmatter{
					Verified: tc.verified,
				},
			}
			if got := c.TrustTier(); got != tc.expected {
				t.Errorf("c.TrustTier() = %q, expected %q", got, tc.expected)
			}
			if got := c.IsVerified(); got != tc.isVerified {
				t.Errorf("c.IsVerified() = %v, expected %v", got, tc.isVerified)
			}
		})
	}
}

func TestConceptsByTrustTier(t *testing.T) {
	b := bundle.NewBundle("/test")
	b.Concepts["concept-unver"] = &bundle.Concept{
		ID: "concept-unver",
	}
	b.Concepts["concept-machine"] = &bundle.Concept{
		ID: "concept-machine",
		Frontmatter: bundle.Frontmatter{
			Verified: bundle.ActorInfoList{
				{By: "process:validator"},
			},
		},
	}
	b.Concepts["concept-human"] = &bundle.Concept{
		ID: "concept-human",
		Frontmatter: bundle.Frontmatter{
			Verified: bundle.ActorInfoList{
				{By: "human:lead-architect"},
			},
		},
	}

	// Canonical tier tests
	unver := b.ConceptsByTrustTier(bundle.TrustTierUnverified)
	if len(unver) != 1 || unver[0].ID != "concept-unver" {
		t.Errorf("unexpected unverified concepts: %v", unver)
	}

	machine := b.ConceptsByTrustTier(bundle.TrustTierMachineConfirmed)
	if len(machine) != 1 || machine[0].ID != "concept-machine" {
		t.Errorf("unexpected machine-confirmed concepts: %v", machine)
	}

	human := b.ConceptsByTrustTier(bundle.TrustTierHumanReviewed)
	if len(human) != 1 || human[0].ID != "concept-human" {
		t.Errorf("unexpected human-reviewed concepts: %v", human)
	}

	// Backwards compatibility alias tests
	legacyHuman := b.ConceptsByTrustTier("human")
	if len(legacyHuman) != 1 || legacyHuman[0].ID != "concept-human" {
		t.Errorf("legacy 'human' alias failed: %v", legacyHuman)
	}

	legacyAutomated := b.ConceptsByTrustTier("automated")
	if len(legacyAutomated) != 1 || legacyAutomated[0].ID != "concept-machine" {
		t.Errorf("legacy 'automated' alias failed: %v", legacyAutomated)
	}
}

func TestBareMappingVerifiedYAMLUnmarshaling(t *testing.T) {
	yamlInput := `
type: Metric
title: Bare Mapping Verifier
verified: { by: "human:alice", at: "2026-06-25T09:00:00Z" }
`
	var fm bundle.Frontmatter
	err := yaml.Unmarshal([]byte(yamlInput), &fm)
	if err != nil {
		t.Fatalf("failed to unmarshal bare mapping verified: %v", err)
	}

	if len(fm.Verified) != 1 {
		t.Fatalf("expected 1 verified record, got %d", len(fm.Verified))
	}
	if fm.Verified[0].By != "human:alice" {
		t.Errorf("expected by='human:alice', got %q", fm.Verified[0].By)
	}
	if fm.Verified[0].At != "2026-06-25T09:00:00Z" {
		t.Errorf("expected at='2026-06-25T09:00:00Z', got %q", fm.Verified[0].At)
	}

	concept := &bundle.Concept{Frontmatter: fm}
	if concept.TrustTier() != bundle.TrustTierHumanReviewed {
		t.Errorf("expected tier %q, got %q", bundle.TrustTierHumanReviewed, concept.TrustTier())
	}
}

func TestSequenceVerifiedYAMLUnmarshaling(t *testing.T) {
	yamlInput := `
type: Metric
title: Sequence Verifier
verified:
  - by: "process:nightly-cron"
    at: "2026-06-25T00:00:00Z"
  - by: "human:bob"
    at: "2026-06-25T12:00:00Z"
`
	var fm bundle.Frontmatter
	err := yaml.Unmarshal([]byte(yamlInput), &fm)
	if err != nil {
		t.Fatalf("failed to unmarshal sequence verified: %v", err)
	}

	if len(fm.Verified) != 2 {
		t.Fatalf("expected 2 verified records, got %d", len(fm.Verified))
	}
	if fm.Verified[0].By != "process:nightly-cron" {
		t.Errorf("expected by='process:nightly-cron', got %q", fm.Verified[0].By)
	}
	if fm.Verified[1].By != "human:bob" {
		t.Errorf("expected by='human:bob', got %q", fm.Verified[1].By)
	}

	concept := &bundle.Concept{Frontmatter: fm}
	if concept.TrustTier() != bundle.TrustTierHumanReviewed {
		t.Errorf("expected tier %q, got %q", bundle.TrustTierHumanReviewed, concept.TrustTier())
	}
}

func TestConceptIsStale(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	staleConcept := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{StaleAfter: past},
	}
	if !staleConcept.IsStale(time.Now()) {
		t.Error("expected concept to be stale")
	}

	freshConcept := &bundle.Concept{
		Frontmatter: bundle.Frontmatter{StaleAfter: future},
	}
	if freshConcept.IsStale(time.Now()) {
		t.Error("expected concept to be fresh")
	}

	emptyConcept := &bundle.Concept{}
	if emptyConcept.IsStale(time.Now()) {
		t.Error("expected concept with no stale_after to not be stale")
	}
}

func TestSourceBackwardCompatibility(t *testing.T) {
	t.Run("YAML with legacy uri field", func(t *testing.T) {
		input := `
sources:
  - uri: "https://stripe.com/docs/billing"
    title: "Stripe Docs"
    author: "Stripe Team"
`
		var fm bundle.Frontmatter
		if err := yaml.Unmarshal([]byte(input), &fm); err != nil {
			t.Fatalf("failed to unmarshal legacy uri: %v", err)
		}
		if len(fm.Sources) != 1 {
			t.Fatalf("expected 1 source, got %d", len(fm.Sources))
		}
		src := fm.Sources[0]
		if src.Resource != "https://stripe.com/docs/billing" {
			t.Errorf("expected Resource to be populated from uri, got %q", src.Resource)
		}
		if src.URI() != "https://stripe.com/docs/billing" {
			t.Errorf("expected URI() to return %q, got %q", "https://stripe.com/docs/billing", src.URI())
		}
		if src.Title != "Stripe Docs" || src.Author != "Stripe Team" {
			t.Errorf("expected title and author to be preserved, got %+v", src)
		}
	})

	t.Run("YAML with spec resource field", func(t *testing.T) {
		input := `
sources:
  - resource: "https://stripe.com/docs/v0_2"
    title: "Stripe Spec Docs"
`
		var fm bundle.Frontmatter
		if err := yaml.Unmarshal([]byte(input), &fm); err != nil {
			t.Fatalf("failed to unmarshal resource: %v", err)
		}
		if len(fm.Sources) != 1 {
			t.Fatalf("expected 1 source, got %d", len(fm.Sources))
		}
		src := fm.Sources[0]
		if src.Resource != "https://stripe.com/docs/v0_2" {
			t.Errorf("expected Resource %q, got %q", "https://stripe.com/docs/v0_2", src.Resource)
		}
		if src.URI() != "https://stripe.com/docs/v0_2" {
			t.Errorf("expected URI() %q, got %q", "https://stripe.com/docs/v0_2", src.URI())
		}
	})

	t.Run("YAML resource takes precedence over uri", func(t *testing.T) {
		input := `
sources:
  - resource: "https://canonical.com/spec"
    uri: "https://legacy.com/old"
`
		var fm bundle.Frontmatter
		if err := yaml.Unmarshal([]byte(input), &fm); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		src := fm.Sources[0]
		if src.Resource != "https://canonical.com/spec" {
			t.Errorf("expected Resource %q to take precedence, got %q", "https://canonical.com/spec", src.Resource)
		}
	})

	t.Run("YAML marshal emits resource and not uri", func(t *testing.T) {
		src := bundle.Source{
			Resource: "https://example.com/asset",
			Title:    "Example Asset",
		}
		out, err := yaml.Marshal(src)
		if err != nil {
			t.Fatalf("failed to marshal Source: %v", err)
		}
		strOut := string(out)
		if !strings.Contains(strOut, "resource: https://example.com/asset") {
			t.Errorf("expected marshal output to contain 'resource: ...', got:\n%s", strOut)
		}
		if strings.Contains(strOut, "uri:") {
			t.Errorf("expected marshal output to NOT contain 'uri:', got:\n%s", strOut)
		}
	})

	t.Run("JSON unmarshal supports legacy uri and resource", func(t *testing.T) {
		jsonLegacy := `{"uri":"https://stripe.com/json","title":"Stripe JSON"}`
		var src1 bundle.Source
		if err := json.Unmarshal([]byte(jsonLegacy), &src1); err != nil {
			t.Fatalf("failed to unmarshal JSON with uri: %v", err)
		}
		if src1.Resource != "https://stripe.com/json" || src1.URI() != "https://stripe.com/json" {
			t.Errorf("expected Resource to be populated from JSON uri, got %q", src1.Resource)
		}

		jsonSpec := `{"resource":"https://stripe.com/spec","title":"Stripe Spec"}`
		var src2 bundle.Source
		if err := json.Unmarshal([]byte(jsonSpec), &src2); err != nil {
			t.Fatalf("failed to unmarshal JSON with resource: %v", err)
		}
		if src2.Resource != "https://stripe.com/spec" || src2.URI() != "https://stripe.com/spec" {
			t.Errorf("expected Resource %q, got %q", "https://stripe.com/spec", src2.Resource)
		}
	})
}

