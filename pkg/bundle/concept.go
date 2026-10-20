// Package bundle defines the core structures for an OKF (Open Knowledge Format) bundle and its constituent concepts.
package bundle

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Trust tier constants as defined in OKF Spec §5.3.
const (
	TrustTierUnverified       = "unverified"
	TrustTierMachineConfirmed = "machine-confirmed"
	TrustTierHumanReviewed    = "human-reviewed"
)

// ConceptLifecycleStatus represents a concept's lifecycle state.
type ConceptLifecycleStatus string

const (
	// ConceptLifecycleStable is the default lifecycle status for OKF v0.2 concepts.
	ConceptLifecycleStable     ConceptLifecycleStatus = "stable"
	ConceptLifecycleDraft      ConceptLifecycleStatus = "draft"
	ConceptLifecycleDeprecated ConceptLifecycleStatus = "deprecated"
)

// Source represents provenance information for an OKF v0.2 concept document.
type Source struct {
	// Resource is the URI or bundle-relative path of the source material.
	Resource string `yaml:"resource,omitempty" json:"resource,omitempty"`
	// Title is the human-readable description of the source material.
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Author is the creator of the source content.
	Author string `yaml:"author,omitempty" json:"author,omitempty"`
	// UsageCount tracks how many times this source has been referenced.
	UsageCount int `yaml:"usage_count,omitempty" json:"usage_count,omitempty"`
	// LastModified is the RFC3339 formatted modification date of the source.
	LastModified string `yaml:"last_modified,omitempty" json:"last_modified,omitempty"`
}

// ActorInfo records details about an actor (human or machine) in concept creation or verification.
// Per OKF Spec §5.2 and §7, frontmatter records objective actor signals ('by' and 'at'),
// while trust tiers are inferred via Tier() rather than stored in frontmatter.
type ActorInfo struct {
	// By identifies the actor username, agent ID, or system name (§7).
	By string `yaml:"by,omitempty" json:"by,omitempty"`
	// At is the RFC3339 timestamp of the action.
	At string `yaml:"at,omitempty" json:"at,omitempty"`
}

// IsHuman reports whether the actor follows the human convention ("human:<id>") per OKF Spec §7.
func (a ActorInfo) IsHuman() bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.By)), "human:")
}

// Tier derives the trust tier for this actor per OKF Spec §5.3:
// - "human-reviewed" for human:<id> actors
// - "machine-confirmed" for non-human, non-empty actors (e.g. process:<id>, agent/<version>)
// - "unverified" if By is empty
func (a ActorInfo) Tier() string {
	return DeriveActorTrustTier(a.By)
}

// DeriveActorTrustTier derives the trust tier from an actor identity string according to OKF Spec §5.3 and §7.
func DeriveActorTrustTier(by string) string {
	by = strings.TrimSpace(by)
	if by == "" {
		return TrustTierUnverified
	}
	if strings.HasPrefix(strings.ToLower(by), "human:") {
		return TrustTierHumanReviewed
	}
	return TrustTierMachineConfirmed
}

// ActorInfoList represents a list of ActorInfo that supports unmarshaling from
// either a sequence (list) or a single bare mapping (as per OKF Spec §5.2).
type ActorInfoList []ActorInfo

// UnmarshalYAML implements custom unmarshaling to support both:
//   verified:
//     - { by: "human:alice", at: "2026-06-25T09:00:00Z" }
// and bare mapping:
//   verified: { by: "human:alice", at: "2026-06-25T09:00:00Z" }
func (al *ActorInfoList) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		var single ActorInfo
		if err := value.Decode(&single); err != nil {
			return err
		}
		*al = []ActorInfo{single}
		return nil
	}
	if value.Kind == yaml.SequenceNode {
		var list []ActorInfo
		if err := value.Decode(&list); err != nil {
			return err
		}
		*al = list
		return nil
	}
	return nil
}

// Attestation defines a sanctioned computation method for calculating concept values in OKF v0.2.
type Attestation struct {
	// Query is the sanitized SQL statement, code snippet, or formula used.
	Query string `yaml:"query,omitempty" json:"query,omitempty"`
	// Executor is the target database engine, execution runtime, or model runner.
	Executor string `yaml:"executor,omitempty" json:"executor,omitempty"`
	// Attester is the entity that attested to the computation correctness.
	Attester string `yaml:"attester,omitempty" json:"attester,omitempty"`
}

// Executor describes how an attested computation is run.
type Executor struct {
	// Resource identifies run instructions or code.
	Resource string `yaml:"resource,omitempty" json:"resource,omitempty"`
	// Receipt lists the evidence returned by a run.
	Receipt []string `yaml:"receipt,omitempty" json:"receipt,omitempty"`
}

// Attester describes the deterministic check for an attested computation.
type Attester struct {
	// Resource identifies the code that checks an execution receipt.
	Resource string `yaml:"resource,omitempty" json:"resource,omitempty"`
}

// Parameter describes a typed input accepted by a computation.
type Parameter struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Required bool   `yaml:"required" json:"required"`
}

// Frontmatter represents the metadata block of an OKF concept document.
type Frontmatter struct {
	// Type specifies the category of the concept (e.g., "PostgreSQL Table", "API Endpoint", "Attestation").
	Type string `yaml:"type"`
	// Title is a human-readable title for the concept.
	Title string `yaml:"title,omitempty"`
	// Desc is a short description of the concept's purpose.
	Desc string `yaml:"description,omitempty"`
	// Resource defines the physical resource name this concept maps to (e.g., table name, endpoint path).
	Resource string `yaml:"resource,omitempty"`
	// Tags are labels for categorization and searching.
	Tags []string `yaml:"tags,omitempty"`
	// Timestamp is the RFC3339 formatted generation/update time.
	Timestamp string `yaml:"timestamp,omitempty"`
	// OKFVersion is the targeted OKF version, only valid in the root index.md frontmatter.
	OKFVersion string `yaml:"okf_version,omitempty"`

	// --- OKF v0.2 Trust Signals ---

	// Sources records the provenance and origin materials for this concept.
	Sources []Source `yaml:"sources,omitempty"`
	// Runtime identifies the environment used to run an attested computation.
	Runtime string `yaml:"runtime,omitempty"`
	// Parameters lists the typed inputs accepted by an attested computation.
	Parameters []Parameter `yaml:"parameters,omitempty"`
	// Computation is an optional path to the computation file.
	Computation string `yaml:"computation,omitempty"`
	// Executor describes how an attested computation is run.
	Executor *Executor `yaml:"executor,omitempty"`
	// Attester describes the deterministic check for an attested computation.
	Attester *Attester `yaml:"attester,omitempty"`
	// Generated records creation metadata (actor and timestamp).
	Generated *ActorInfo `yaml:"generated,omitempty"`
	// Verified lists independent confirmations of accuracy (human or automated).
	// Supports both a YAML sequence or a single bare mapping (§5.2).
	Verified ActorInfoList `yaml:"verified,omitempty"`
	// StaleAfter defines an absolute RFC3339 expiration date after which re-verification is required.
	StaleAfter string `yaml:"stale_after,omitempty"`
	// Status specifies the lifecycle state ("draft", "stable", "deprecated").
	Status ConceptLifecycleStatus `yaml:"status,omitempty"`
	// Attestation defines sanctioned computation properties for Attestation concepts.
	Attestation *Attestation `yaml:"attestation,omitempty"`

	// Extra captures any other custom metadata key-value pairs inline.
	Extra map[string]interface{} `yaml:",inline"`
}

// Citation represents a single supporting source reference.
type Citation struct {
	// Number is the citation index number, e.g. 1 in [1].
	Number int `yaml:"number" json:"number"`
	// Title is the human-readable text description of the citation.
	Title string `yaml:"title" json:"title"`
	// URI is the link to the resource.
	URI string `yaml:"uri" json:"uri"`
}

// Concept represents a single OKF document (concept).
type Concept struct {
	// ID is the unique identifier (relative filepath without .md suffix, e.g., "tables/users").
	ID string
	// Path is the relative path within the bundle (e.g., "tables/users.md").
	Path string
	// Frontmatter holds the parsed metadata block.
	Frontmatter Frontmatter
	// Body is the raw Markdown body excluding the frontmatter block.
	Body string
	// ParseError contains the error description if the file could not be parsed.
	ParseError string
	// Citations holds the list of parsed citations from the concept.
	Citations []Citation
}

// IsStale checks if the concept's StaleAfter timestamp is past the given reference time.
func (c *Concept) IsStale(now time.Time) bool {
	if c.Frontmatter.StaleAfter == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, c.Frontmatter.StaleAfter)
	if err != nil {
		// Try fallback date-only format YYYY-MM-DD
		t, err = time.Parse("2006-01-02", c.Frontmatter.StaleAfter)
		if err != nil {
			return false
		}
	}
	return now.After(t)
}

// IsVerified returns true if the concept has at least one valid verification record (trust tier is not unverified).
func (c *Concept) IsVerified() bool {
	return c.TrustTier() != TrustTierUnverified
}

// TrustTier returns the highest derived verification tier according to OKF Spec §5.3:
// - "unverified" if verified is missing, empty, or has no recognized verifiers
// - "machine-confirmed" if verified by non-human actors only
// - "human-reviewed" if verified by at least one human:<id> actor
func (c *Concept) TrustTier() string {
	if len(c.Frontmatter.Verified) == 0 {
		return TrustTierUnverified
	}
	hasMachine := false
	for _, v := range c.Frontmatter.Verified {
		if v.IsHuman() {
			return TrustTierHumanReviewed
		}
		if strings.TrimSpace(v.By) != "" {
			hasMachine = true
		}
	}
	if hasMachine {
		return TrustTierMachineConfirmed
	}
	return TrustTierUnverified
}
