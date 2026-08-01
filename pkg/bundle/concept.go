// Package bundle defines the core structures for an OKF (Open Knowledge Format) bundle and its constituent concepts.
package bundle

import (
	"strings"
	"time"
)


// Source represents provenance information for an OKF v0.2 concept document.
type Source struct {
	// URI is the target web link or file path of the source document.
	URI string `yaml:"uri,omitempty" json:"uri,omitempty"`
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
type ActorInfo struct {
	// By identifies the actor username, agent ID, or system name.
	By string `yaml:"by,omitempty" json:"by,omitempty"`
	// At is the RFC3339 timestamp of the action.
	At string `yaml:"at,omitempty" json:"at,omitempty"`
	// Tier specifies the verification confidence level (e.g. "human", "automated", "certified").
	Tier string `yaml:"tier,omitempty" json:"tier,omitempty"`
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
	// Generated records creation metadata (actor and timestamp).
	Generated *ActorInfo `yaml:"generated,omitempty"`
	// Verified lists independent confirmations of accuracy (human or automated).
	Verified []ActorInfo `yaml:"verified,omitempty"`
	// StaleAfter defines an absolute RFC3339 expiration date after which re-verification is required.
	StaleAfter string `yaml:"stale_after,omitempty"`
	// Status specifies the lifecycle state ("draft", "stable", "deprecated").
	Status string `yaml:"status,omitempty"`
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

// IsVerified returns true if the concept has at least one verification record.
func (c *Concept) IsVerified() bool {
	return len(c.Frontmatter.Verified) > 0
}

// TrustTier returns the highest verification tier ("certified", "human", "automated", or "unverified").
func (c *Concept) TrustTier() string {
	if len(c.Frontmatter.Verified) == 0 {
		return "unverified"
	}
	tier := "verified"
	for _, v := range c.Frontmatter.Verified {
		t := strings.ToLower(v.Tier)
		if t == "certified" {
			return "certified"
		}
		if t == "human" {
			tier = "human"
		} else if t == "automated" && tier != "human" {
			tier = "automated"
		}
	}
	return tier
}

