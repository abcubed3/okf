package bundle_test

import (
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
