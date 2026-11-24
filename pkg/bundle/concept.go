// Package bundle defines the core structures for an OKF (Open Knowledge Format) bundle and its constituent concepts.
package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/internal/links"
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

// UsageWindow frames source usage_count over a {from, to} RFC3339 datetime range per OKF Spec §5.1.
type UsageWindow struct {
	From  string         `yaml:"from,omitempty" json:"from,omitempty"`
	To    string         `yaml:"to,omitempty" json:"to,omitempty"`
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`
}

// Source represents provenance information for an OKF v0.2/v0.3 concept document.
type Source struct {
	// ID is an optional local footnote/citation identifier within the concept document (e.g., "rev-policy").
	ID string `yaml:"id,omitempty" json:"id,omitempty"`
	// Resource is the URI or bundle-relative path of the source material.
	Resource string `yaml:"resource,omitempty" json:"resource,omitempty"`
	// Title is the human-readable description of the source material.
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Author is the creator of the source content.
	Author string `yaml:"author,omitempty" json:"author,omitempty"`
	// UsageCount tracks how many times this source has been referenced.
	UsageCount int `yaml:"usage_count,omitempty" json:"usage_count,omitempty"`
	// UsageWindow optionally overrides the concept-level usage_window for this source.
	UsageWindow *UsageWindow `yaml:"usage_window,omitempty" json:"usage_window,omitempty"`
	// LastModified is the RFC3339 formatted modification date of the source.
	LastModified string `yaml:"last_modified,omitempty" json:"last_modified,omitempty"`
	// Extra preserves custom extension keys (e.g. x-<producer>-<name>) per OKF Spec v0.3 §4.1.
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`
}

// URI returns the source resource identifier for backward compatibility with OKF v0.1.
//
// Deprecated: Use Resource instead per OKF Spec v0.2+ §5.1.
func (s Source) URI() string {
	return s.Resource
}

// UnmarshalYAML implements custom unmarshaling to support both 'resource' (v0.2+)
// and legacy 'uri' (v0.1) source field names while preserving x-* extension keys.
func (s *Source) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		*s = Source{}
		return nil
	}
	type sourceAlias Source
	var raw sourceAlias
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*s = Source(raw)
	if uriVal, ok := s.Extra["uri"]; ok {
		if uriStr, ok := uriVal.(string); ok && s.Resource == "" && uriStr != "" {
			s.Resource = uriStr
		}
		delete(s.Extra, "uri")
	}
	if s.UsageWindow != nil && len(s.UsageWindow.Extra) == 0 {
		s.UsageWindow.Extra = nil
	}
	if len(s.Extra) == 0 {
		s.Extra = nil
	}
	return nil
}

// UnmarshalJSON implements custom unmarshaling to support both 'resource' (v0.2+)
// and legacy 'uri' (v0.1) source field names in JSON.
func (s *Source) UnmarshalJSON(data []byte) error {
	type sourceAlias Source
	var raw struct {
		sourceAlias
		URI string `json:"uri,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = Source(raw.sourceAlias)
	if s.Resource == "" && raw.URI != "" {
		s.Resource = raw.URI
	}
	return nil
}

// ActorInfo records details about an actor (human or machine) in concept creation or verification.
// Per OKF Spec §5.2 and §7, frontmatter records objective actor signals ('by' and 'at'),
// while trust tiers are inferred via Tier() rather than stored in frontmatter.
type ActorInfo struct {
	// By identifies the actor username, agent ID, or system name (§7).
	By string `yaml:"by,omitempty" json:"by,omitempty"`
	// At is the RFC3339 timestamp of the action.
	At string `yaml:"at,omitempty" json:"at,omitempty"`
	// Extra preserves custom extension keys (e.g. x-<producer>-<name>) per OKF Spec v0.3 §4.1.
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`
}

// IsHuman reports whether the actor follows the human convention ("human:<id>") per OKF Spec §7.
func (a ActorInfo) IsHuman() bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.By)), "human:")
}

// Tier derives the trust tier for this actor per OKF Spec §5.3:
// - "human-reviewed" for human:<id> actors
// - "machine-confirmed" for non-human, non-empty actors (e.g. team:<id>, process:<id>, agent/<version>)
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
//
//	verified:
//	  - { by: "human:alice", at: "2026-06-25T09:00:00Z" }
//
// and bare mapping:
//
//	verified: { by: "human:alice", at: "2026-06-25T09:00:00Z" }
func (al *ActorInfoList) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		*al = nil
		return nil
	}
	if value.Kind == yaml.MappingNode {
		var single ActorInfo
		if err := value.Decode(&single); err != nil {
			return err
		}
		if len(single.Extra) == 0 {
			single.Extra = nil
		}
		*al = []ActorInfo{single}
		return nil
	}
	if value.Kind == yaml.SequenceNode {
		var list []ActorInfo
		if err := value.Decode(&list); err != nil {
			return err
		}
		for i := range list {
			if len(list[i].Extra) == 0 {
				list[i].Extra = nil
			}
		}
		*al = list
		return nil
	}
	return fmt.Errorf("cannot unmarshal YAML %s into verified actor list (expected mapping or sequence)", value.ShortTag())
}

// Attestation defines a legacy draft sanctioned computation method for backward compatibility.
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
	// Extra preserves custom extension keys (e.g. x-<producer>-<name>) per OKF Spec v0.3 §4.1.
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`
}

// Attester describes the deterministic check for an attested computation.
type Attester struct {
	// Resource identifies the code that checks an execution receipt.
	Resource string `yaml:"resource,omitempty" json:"resource,omitempty"`
	// Extra preserves custom extension keys (e.g. x-<producer>-<name>) per OKF Spec v0.3 §4.1.
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`
}

// Parameter describes a typed input accepted by a computation.
type Parameter struct {
	Name     string         `yaml:"name" json:"name"`
	Type     string         `yaml:"type" json:"type"`
	Required bool           `yaml:"required,omitempty" json:"required"`
	Extra    map[string]any `yaml:",inline" json:"extra,omitempty"`
}

// Frontmatter represents the metadata block of an OKF concept or bundle-root index document.
type Frontmatter struct {
	// Type specifies the category of the concept (e.g., "PostgreSQL Table", "API Endpoint", "Attested Computation").
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// Title is a human-readable title for the concept or bundle.
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Desc is a short description of the concept's or bundle's purpose.
	Desc string `yaml:"description,omitempty" json:"description,omitempty"`
	// Resource defines the physical resource name this concept maps to (e.g., table name, endpoint path).
	Resource string `yaml:"resource,omitempty" json:"resource,omitempty"`
	// Tags are labels for categorization and searching.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	// Timestamp is the legacy v0.1 RFC3339 formatted generation/update time (superseded by generated.at).
	Timestamp string `yaml:"timestamp,omitempty" json:"timestamp,omitempty"`
	// OKFVersion is the targeted OKF version, valid in the bundle-root index.md frontmatter (§8.1, §12).
	OKFVersion string `yaml:"okf_version,omitempty" json:"okf_version,omitempty"`
	// License specifies the usage license for the bundle in bundle-root index.md frontmatter (OKF v0.3 §8.1).
	License string `yaml:"license,omitempty" json:"license,omitempty"`

	// --- OKF v0.2 / v0.3 Provenance, Trust, Lifecycle & Computation Signals ---

	// Sources records the provenance and origin materials for this concept (§5.1).
	Sources []Source `yaml:"sources,omitempty" json:"sources,omitempty"`
	// UsageWindow frames source usage_count values over a {from, to} range (§5.1).
	UsageWindow *UsageWindow `yaml:"usage_window,omitempty" json:"usage_window,omitempty"`
	// Runtime identifies the environment used to run an attested computation (§10.2).
	Runtime string `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	// Parameters lists the typed inputs accepted by an attested computation (§10.2).
	Parameters []Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	// Computation is an optional path to the computation file (§10.2).
	Computation string `yaml:"computation,omitempty" json:"computation,omitempty"`
	// Executor describes how an attested computation is run (§10.2).
	Executor *Executor `yaml:"executor,omitempty" json:"executor,omitempty"`
	// Attester describes the deterministic check for an attested computation (§10.2).
	Attester *Attester `yaml:"attester,omitempty" json:"attester,omitempty"`
	// Generated records creation metadata (actor and timestamp) (§5.2).
	Generated *ActorInfo `yaml:"generated,omitempty" json:"generated,omitempty"`
	// Verified lists independent confirmations of accuracy (human or automated) (§5.2).
	// Supports both a YAML sequence or a single bare mapping.
	Verified ActorInfoList `yaml:"verified,omitempty" json:"verified,omitempty"`
	// StaleAfter defines an absolute RFC3339 expiration date after which re-verification is required (§5.5).
	StaleAfter string `yaml:"stale_after,omitempty" json:"stale_after,omitempty"`
	// Status specifies the lifecycle state ("draft", "stable", "deprecated") (§5.4).
	Status ConceptLifecycleStatus `yaml:"status,omitempty" json:"status,omitempty"`
	// SupersededBy points to a replacement concept or URL when status is "deprecated" (OKF v0.3 §5.4).
	SupersededBy string `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
	// Attestation defines legacy draft sanctioned computation properties for backward compatibility.
	Attestation *Attestation `yaml:"attestation,omitempty" json:"attestation,omitempty"`

	// Extra captures any other custom or x-* extension metadata key-value pairs inline (§4.1).
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`
}

var scalarStringFrontmatterKeys = map[string]bool{
	"type":          true,
	"title":         true,
	"description":   true,
	"resource":      true,
	"timestamp":     true,
	"okf_version":   true,
	"license":       true,
	"runtime":       true,
	"computation":   true,
	"stale_after":   true,
	"status":        true,
	"superseded_by": true,
}

// UnmarshalYAML implements OKF v0.3 consumer tolerance (§11), normalizing scalar/mapping list
// shorthands (e.g. single-string tags, single-mapping sources/parameters, scalar executor.receipt),
// unquoted numeric/timestamp strings (e.g. okf_version: 0.3), and null values while preserving x-* keys.
func (fm *Frontmatter) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		*fm = Frontmatter{}
		return nil
	}
	if value.Kind != yaml.MappingNode {
		type fmAlias Frontmatter
		var raw fmAlias
		if err := value.Decode(&raw); err != nil {
			return err
		}
		*fm = Frontmatter(raw)
		fm.cleanEmptyExtras()
		return nil
	}

	norm := *value
	norm.Content = make([]*yaml.Node, len(value.Content))
	copy(norm.Content, value.Content)

	for i := 0; i+1 < len(norm.Content); i += 2 {
		kNode := norm.Content[i]
		vNode := norm.Content[i+1]
		key := kNode.Value

		if scalarStringFrontmatterKeys[key] && vNode.Kind == yaml.ScalarNode && vNode.Tag != "!!null" {
			cp := *vNode
			cp.Tag = "!!str"
			norm.Content[i+1] = &cp
			continue
		}

		switch key {
		case "tags":
			if vNode.Kind == yaml.ScalarNode && vNode.Tag != "!!null" {
				strNode := *vNode
				strNode.Tag = "!!str"
				if strings.TrimSpace(strNode.Value) != "" {
					norm.Content[i+1] = &yaml.Node{
						Kind:    yaml.SequenceNode,
						Content: []*yaml.Node{&strNode},
					}
				} else {
					norm.Content[i+1] = &yaml.Node{Kind: yaml.SequenceNode}
				}
			}
		case "sources", "parameters":
			if vNode.Kind == yaml.MappingNode {
				norm.Content[i+1] = &yaml.Node{
					Kind:    yaml.SequenceNode,
					Content: []*yaml.Node{vNode},
				}
			}
		case "executor":
			if vNode.Kind == yaml.MappingNode {
				exCopy := *vNode
				exCopy.Content = make([]*yaml.Node, len(vNode.Content))
				copy(exCopy.Content, vNode.Content)
				for j := 0; j+1 < len(exCopy.Content); j += 2 {
					if exCopy.Content[j].Value == "receipt" && exCopy.Content[j+1].Kind == yaml.ScalarNode && exCopy.Content[j+1].Tag != "!!null" {
						rcpt := *exCopy.Content[j+1]
						rcpt.Tag = "!!str"
						exCopy.Content[j+1] = &yaml.Node{
							Kind:    yaml.SequenceNode,
							Content: []*yaml.Node{&rcpt},
						}
					}
				}
				norm.Content[i+1] = &exCopy
			}
		}
	}

	type fmAlias Frontmatter
	var raw fmAlias
	if err := norm.Decode(&raw); err != nil {
		return err
	}
	*fm = Frontmatter(raw)
	fm.cleanEmptyExtras()
	return nil
}

func (fm *Frontmatter) cleanEmptyExtras() {
	if len(fm.Extra) == 0 {
		fm.Extra = nil
	}
	if fm.UsageWindow != nil && len(fm.UsageWindow.Extra) == 0 {
		fm.UsageWindow.Extra = nil
	}
	if fm.Generated != nil && len(fm.Generated.Extra) == 0 {
		fm.Generated.Extra = nil
	}
	for i := range fm.Parameters {
		if len(fm.Parameters[i].Extra) == 0 {
			fm.Parameters[i].Extra = nil
		}
	}
	if fm.Executor != nil && len(fm.Executor.Extra) == 0 {
		fm.Executor.Extra = nil
	}
	if fm.Attester != nil && len(fm.Attester.Extra) == 0 {
		fm.Attester.Extra = nil
	}
}

// IsZero reports whether the Frontmatter contains no populated metadata fields.
func (fm Frontmatter) IsZero() bool {
	return fm.Type == "" &&
		fm.Title == "" &&
		fm.Desc == "" &&
		fm.Resource == "" &&
		len(fm.Tags) == 0 &&
		fm.Timestamp == "" &&
		fm.OKFVersion == "" &&
		fm.License == "" &&
		len(fm.Sources) == 0 &&
		fm.UsageWindow == nil &&
		fm.Runtime == "" &&
		len(fm.Parameters) == 0 &&
		fm.Computation == "" &&
		fm.Executor == nil &&
		fm.Attester == nil &&
		fm.Generated == nil &&
		len(fm.Verified) == 0 &&
		fm.StaleAfter == "" &&
		fm.Status == "" &&
		fm.SupersededBy == "" &&
		fm.Attestation == nil &&
		len(fm.Extra) == 0
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

// Concept represents a single OKF document (concept or reserved file).
type Concept struct {
	// ID is the unique identifier (relative filepath without .md suffix, e.g., "tables/users").
	ID string
	// Path is the relative path within the bundle (e.g., "tables/users.md").
	Path string
	// Frontmatter holds the parsed metadata block.
	Frontmatter Frontmatter
	// HasFrontmatter indicates whether an explicit "---" YAML frontmatter block was present in source.
	HasFrontmatter bool
	// Body is the raw Markdown body excluding the frontmatter block.
	Body string
	// ParseError contains the error description if the file could not be parsed.
	ParseError string
	// Citations holds the list of parsed legacy v0.1 citations from the concept.
	Citations []Citation
}

// IsStrictRFC3339 reports whether raw is a conformant ISO 8601 / RFC 3339 datetime
// with an explicit time and UTC offset (e.g., "2026-06-30T14:00:00Z") per OKF Spec §5.
func IsStrictRFC3339(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// ParseOKFTimestamp parses an OKF timestamp string, accepting strict RFC3339/RFC3339Nano
// first and falling back to date-only "YYYY-MM-DD" for legacy consumer tolerance.
func ParseOKFTimestamp(raw string) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// IsStale checks if the concept's StaleAfter timestamp is at or past the given reference time (now >= stale_after, OKF Spec §5.5).
func (c *Concept) IsStale(now time.Time) bool {
	if c == nil || c.Frontmatter.StaleAfter == "" {
		return false
	}
	t, ok := ParseOKFTimestamp(c.Frontmatter.StaleAfter)
	if !ok {
		return false
	}
	return !now.Before(t)
}

// CurrentVerifications returns the subset of Verified entries that are current per OKF Spec v0.3 §5.3:
//   - When generated or generated.at is absent, all non-empty Verified entries are current.
//   - When generated.at is present but unparseable, CurrentVerifications fails closed and returns nil.
//   - When generated.at is valid, only Verified entries with a valid 'at' timestamp >= generated.at are current.
func (c *Concept) CurrentVerifications() []ActorInfo {
	if c == nil || len(c.Frontmatter.Verified) == 0 {
		return nil
	}

	var genTime time.Time
	hasGenTime := false
	if c.Frontmatter.Generated != nil && strings.TrimSpace(c.Frontmatter.Generated.At) != "" {
		parsed, ok := ParseOKFTimestamp(c.Frontmatter.Generated.At)
		if !ok {
			// Fail closed when generated.at is present on the document but malformed.
			return nil
		}
		genTime = parsed
		hasGenTime = true
	}

	var current []ActorInfo
	for _, v := range c.Frontmatter.Verified {
		if strings.TrimSpace(v.By) == "" {
			continue
		}
		if hasGenTime {
			verTime, ok := ParseOKFTimestamp(v.At)
			if !ok || verTime.Before(genTime) {
				continue
			}
		}
		current = append(current, v)
	}
	return current
}

// HasOutdatedVerifications reports whether the concept contains one or more valid verification
// events whose 'at' timestamp precedes a valid generated.at timestamp (OKF Spec v0.3 §5.3).
func (c *Concept) HasOutdatedVerifications() bool {
	if c == nil || len(c.Frontmatter.Verified) == 0 || c.Frontmatter.Generated == nil {
		return false
	}
	genTime, ok := ParseOKFTimestamp(c.Frontmatter.Generated.At)
	if !ok {
		return false
	}
	for _, v := range c.Frontmatter.Verified {
		if strings.TrimSpace(v.By) == "" {
			continue
		}
		if verTime, verOK := ParseOKFTimestamp(v.At); verOK && verTime.Before(genTime) {
			return true
		}
	}
	return false
}

// IsVerified returns true if the concept has at least one current verification record (trust tier is not unverified).
func (c *Concept) IsVerified() bool {
	return c.TrustTier() != TrustTierUnverified
}

// TrustTier returns the highest derived verification tier from current verifications per OKF Spec v0.3 §5.3:
// - "unverified" if verified is missing, empty, or has no current verifiers (at >= generated.at)
// - "machine-confirmed" if current verifications are by non-human actors only (e.g. team:, process:, agent/)
// - "human-reviewed" if at least one current verification is by a human:<id> actor
func (c *Concept) TrustTier() string {
	current := c.CurrentVerifications()
	if len(current) == 0 {
		return TrustTierUnverified
	}
	hasMachine := false
	for _, v := range current {
		if v.IsHuman() {
			return TrustTierHumanReviewed
		}
		hasMachine = true
	}
	if hasMachine {
		return TrustTierMachineConfirmed
	}
	return TrustTierUnverified
}

// InlineComputation extracts the inline code block under a "# Computation" heading in Body, if present.
func (c *Concept) InlineComputation() string {
	if c == nil {
		return ""
	}
	return extractComputationFromBody(c.Body)
}

// SanctionedQuery extracts the sanctioned computation query or reference for an Attestation
// or OKF v0.2/v0.3 Attested Computation concept. It checks Frontmatter.Attestation.Query first,
// then extracts inline fenced or indented code blocks under a "# Computation" heading in Body,
// and finally falls back to Frontmatter.Computation.
func (c *Concept) SanctionedQuery() string {
	if c == nil {
		return ""
	}
	if c.Frontmatter.Attestation != nil && strings.TrimSpace(c.Frontmatter.Attestation.Query) != "" {
		return strings.TrimSpace(c.Frontmatter.Attestation.Query)
	}
	cType := strings.ToLower(strings.TrimSpace(c.Frontmatter.Type))
	if cType != "attested computation" && cType != "attestation" {
		return ""
	}
	if q := extractComputationFromBody(c.Body); q != "" {
		return q
	}
	return strings.TrimSpace(c.Frontmatter.Computation)
}

// PrimaryVerifier returns the actor identifier of the first current verifier matching the concept's TrustTier.
func (c *Concept) PrimaryVerifier() string {
	tier := c.TrustTier()
	current := c.CurrentVerifications()
	for _, v := range current {
		if tier == TrustTierHumanReviewed && v.IsHuman() {
			return strings.TrimSpace(v.By)
		}
		if tier == TrustTierMachineConfirmed && !v.IsHuman() {
			return strings.TrimSpace(v.By)
		}
	}
	if len(current) > 0 {
		return strings.TrimSpace(current[0].By)
	}
	return ""
}

// ResolveComputation returns the executable computation text for an Attested Computation or Attestation concept.
// When Frontmatter.Computation names a bundle file and bundleRoot is non-empty, it safely resolves and reads
// the file within bundleRoot.
func (c *Concept) ResolveComputation(bundleRoot string) (string, error) {
	if c == nil {
		return "", nil
	}
	if c.Frontmatter.Attestation != nil && strings.TrimSpace(c.Frontmatter.Attestation.Query) != "" {
		return strings.TrimSpace(c.Frontmatter.Attestation.Query), nil
	}
	cType := strings.ToLower(strings.TrimSpace(c.Frontmatter.Type))
	if cType != "attested computation" && cType != "attestation" {
		return "", nil
	}
	if inline := extractComputationFromBody(c.Body); inline != "" {
		return inline, nil
	}
	compPath := strings.TrimSpace(c.Frontmatter.Computation)
	if compPath == "" {
		return "", nil
	}
	if strings.TrimSpace(bundleRoot) == "" {
		return compPath, nil
	}

	absRoot, err := filepath.Abs(bundleRoot)
	if err != nil {
		return "", err
	}

	containingRel := filepath.ToSlash(c.Path)
	if containingRel == "" && c.ID != "" {
		containingRel = c.ID + ".md"
	} else if filepath.IsAbs(c.Path) {
		if rel, relErr := filepath.Rel(absRoot, c.Path); relErr == nil {
			containingRel = filepath.ToSlash(rel)
		}
	}

	// Support both v0.3 resolution and legacy v0.2 root-relative fallback if needed
	relSlash, escaped := links.ResolveBundlePath(absRoot, containingRel, compPath, "0.2")
	if escaped || relSlash == "" {
		return "", fmt.Errorf("computation path %q escapes bundle root", compPath)
	}

	fullTarget := filepath.Join(absRoot, filepath.FromSlash(relSlash))
	relOS, err := filepath.Rel(absRoot, fullTarget)
	if err != nil || relOS == ".." || strings.HasPrefix(relOS, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("computation path %q escapes bundle root", compPath)
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", err
	}
	realTarget, err := filepath.EvalSymlinks(fullTarget)
	if err != nil {
		return "", err
	}
	realRel, err := filepath.Rel(realRoot, realTarget)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("computation symlink %q escapes bundle root", compPath)
	}

	data, err := os.ReadFile(realTarget)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ResolvedQuery returns the executable computation query text if available, resolving external
// Frontmatter.Computation files when bundleRoot is non-empty without leaking unresolved file paths
// as query text on read failures.
func (c *Concept) ResolvedQuery(bundleRoot string) string {
	if c == nil {
		return ""
	}
	if strings.TrimSpace(bundleRoot) != "" {
		if q, err := c.ResolveComputation(bundleRoot); err == nil && q != "" {
			return q
		}
		if c.Frontmatter.Attestation != nil && strings.TrimSpace(c.Frontmatter.Attestation.Query) != "" {
			return strings.TrimSpace(c.Frontmatter.Attestation.Query)
		}
		return c.InlineComputation()
	}
	return c.SanctionedQuery()
}

func extractComputationFromBody(body string) string {
	lines := strings.Split(body, "\n")
	inSection := false
	inFence := false
	var fenceLines []string
	var indentedLines []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)

		if !inFence && strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if strings.EqualFold(heading, "computation") {
				inSection = true
				continue
			}
			if inSection {
				break
			}
		}
		if !inSection {
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			if inFence {
				return strings.TrimSpace(strings.Join(fenceLines, "\n"))
			}
			inFence = true
			continue
		}
		if inFence {
			fenceLines = append(fenceLines, line)
			continue
		}

		if strings.HasPrefix(line, "    ") {
			indentedLines = append(indentedLines, strings.TrimPrefix(line, "    "))
		} else if strings.HasPrefix(line, "\t") {
			indentedLines = append(indentedLines, strings.TrimPrefix(line, "\t"))
		} else if trimmed == "" {
			if len(indentedLines) > 0 {
				indentedLines = append(indentedLines, "")
			}
		} else if len(indentedLines) > 0 {
			break
		}
	}

	return strings.TrimSpace(strings.Join(indentedLines, "\n"))
}
