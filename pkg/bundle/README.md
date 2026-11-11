# bundle

## Overview
Package `bundle` (`github.com/abcubed3/okf/pkg/bundle`) defines the core in-memory domain model and structural operations for Open Knowledge Format (OKF v0.1/v0.2) knowledge bundles, including concept metadata, provenance and actor verification trust-tier derivation, deterministic queries, structural drift comparison, and strategy-driven bundle merging.

## Interface
- `[bundle.go:Bundle]` — `type Bundle struct { Path string; Concepts map[string]*Concept; IndexFiles map[string]*Concept; OKFVersion string }`: Root container mapping concept IDs and reserved `index.md` paths to parsed documents.
- `[bundle.go:NewBundle]` — `func NewBundle(path string) *Bundle`: Allocates an empty `Bundle` rooted at `path`.
- `[bundle.go:Bundle.GetConcept]` — `func (b *Bundle) GetConcept(id string) (*Concept, bool)`: Looks up a concept by `id`.
- `[bundle.go:Bundle.ListTypes]` — `func (b *Bundle) ListTypes() []string`: Returns unique non-empty concept types sorted lexicographically.
- `[bundle.go:Bundle.ListTags]` — `func (b *Bundle) ListTags() []string`: Returns unique non-empty tags across all concepts sorted lexicographically.
- `[bundle.go:Bundle.ConceptsByType]` — `func (b *Bundle) ConceptsByType(t string) []*Concept`: Filters concepts matching exact `Frontmatter.Type`, sorted ascending by `ID`.
- `[bundle.go:Bundle.ConceptsByStatus]` — `func (b *Bundle) ConceptsByStatus(status string) []*Concept`: Filters concepts matching `Frontmatter.Status` case-insensitively, sorted ascending by `ID`.
- `[bundle.go:Bundle.ConceptsByTrustTier]` — `func (b *Bundle) ConceptsByTrustTier(tier string) []*Concept`: Filters concepts matching derived trust tier (with legacy alias normalization), sorted ascending by `ID`.
- `[concept.go:TrustTierUnverified]`, `[concept.go:TrustTierMachineConfirmed]`, `[concept.go:TrustTierHumanReviewed]` — `const` strings (`"unverified"`, `"machine-confirmed"`, `"human-reviewed"`) representing OKF Spec §5.3 trust tiers.
- `[concept.go:ConceptLifecycleStatus]`, `[concept.go:ConceptLifecycleStable]`, `[concept.go:ConceptLifecycleDraft]`, `[concept.go:ConceptLifecycleDeprecated]` — Lifecycle status type and constants (`"stable"`, `"draft"`, `"deprecated"`).
- `[concept.go:Source]` — `type Source struct { ID, Resource, Title, Author string; UsageCount int; LastModified string }`: Provenance record with deprecated `[concept.go:Source.URI]`, `[concept.go:Source.UnmarshalYAML]`, and `[concept.go:Source.UnmarshalJSON]`.
- `[concept.go:ActorInfo]` — `type ActorInfo struct { By, At string }`: Actor attribution record with `[concept.go:ActorInfo.IsHuman]` and `[concept.go:ActorInfo.Tier]`.
- `[concept.go:DeriveActorTrustTier]` — `func DeriveActorTrustTier(by string) string`: Derives `"unverified"`, `"human-reviewed"`, or `"machine-confirmed"` from an actor identifier.
- `[concept.go:ActorInfoList]` — `type ActorInfoList []ActorInfo`: Slice type with `[concept.go:ActorInfoList.UnmarshalYAML]` supporting both sequence and bare-mapping YAML forms.
- `[concept.go:Attestation]`, `[concept.go:Executor]`, `[concept.go:Attester]`, `[concept.go:Parameter]` — Sanctioned computation and attested computation metadata structs (`Attestation{Query, Parameters, Executor, Attester, LastVerified, OutputSchema}`, `Executor{Resource, Runtime, Type}`, `Attester{Resource, Name}`, `Parameter{Name, Type, Default}`).
- `[concept.go:Frontmatter]` — `type Frontmatter struct { OKFVersion, Type, Title, Desc, Resource, Timestamp string; Tags []string; Status ConceptLifecycleStatus; StaleAfter string; Sources []Source; Generated *ActorInfo; Verified ActorInfoList; Runtime string; Parameters []Parameter; Computation string; Executor *Executor; Attester *Attester; Attestation *Attestation; Extra map[string]interface{} }`: Parsed YAML metadata block covering core fields, OKF v0.2 trust/attestation signals, and inline `Extra`.
- `[concept.go:Citation]` — `type Citation struct { Number int; Title, URI string }`: Parsed body citation reference.
- `[concept.go:Concept]` — `type Concept struct { ID, Path string; Frontmatter Frontmatter; Body, ParseError string; Citations []Citation }`: Single OKF document with `[concept.go:Concept.IsStale]`, `[concept.go:Concept.IsVerified]`, `[concept.go:Concept.TrustTier]`, and `[concept.go:Concept.SanctionedQuery]`.
- `[diff.go:ConceptDiffType]`, `[diff.go:DiffAdded]`, `[diff.go:DiffDeleted]`, `[diff.go:DiffModified]` — Drift classification type and constants (`"ADDED"`, `"DELETED"`, `"MODIFIED"`).
- `[diff.go:FieldDiff]`, `[diff.go:ConceptDiff]`, `[diff.go:BundleDiff]` — Drift result structures (`FieldDiff{Field string; OldVal, NewVal interface{}}`, `ConceptDiff{ID string; Type ConceptDiffType; OldConcept, NewConcept *Concept; FieldChanges []FieldDiff}`, `BundleDiff{OldPath, NewPath string; Changes []ConceptDiff}`), with `[diff.go:BundleDiff.PrettyPrint]` (`func (bd *BundleDiff) PrettyPrint(w io.Writer)`).
- `[diff.go:Diff]` — `func Diff(oldB, newB *Bundle) (*BundleDiff, error)`: Computes structural additions, deletions, and modifications between two bundles.
- `[merge.go:MergeStrategy]`, `[merge.go:MergeUnion]`, `[merge.go:MergeOurs]`, `[merge.go:MergeTheirs]` — Reconciliation strategy type and constants (`"union"`, `"ours"`, `"theirs"`).
- `[merge.go:Merge]` — `func Merge(source, target *Bundle, strategy MergeStrategy) (*Bundle, error)`: Reconciles `source` and `target` into a new `*Bundle`.

## Invariants
- `[bundle.go:NewBundle]` initializes both `Concepts` and `IndexFiles` to non-nil empty maps.
- `[bundle.go:Bundle.ConceptsByTrustTier]` normalizes legacy aliases (`"human"` / `"certified"` -> `"human-reviewed"`, `"automated"` / `"machine"` -> `"machine-confirmed"`) before matching `Concept.TrustTier()` case-insensitively.
- `[concept.go:Source.UnmarshalYAML]` and `[concept.go:Source.UnmarshalJSON]` fall back to legacy `uri` into `Source.Resource` only when `resource` is empty.
- `[concept.go:DeriveActorTrustTier]` returns `"unverified"` for empty/whitespace strings, `"human-reviewed"` when trimmed `by` has case-insensitive prefix `"human:"`, and `"machine-confirmed"` for any other non-empty actor string; `[concept.go:Concept.TrustTier]` returns the highest tier across `Frontmatter.Verified`, and `[concept.go:Concept.IsVerified]` is `true` iff `TrustTier() != TrustTierUnverified`.
- `[concept.go:ActorInfoList.UnmarshalYAML]` decodes a YAML `MappingNode` into a single-element slice and a `SequenceNode` into a slice, leaving other YAML node kinds untouched without error.
- `[concept.go:Concept.IsStale]` parses `Frontmatter.StaleAfter` as `time.RFC3339` with fallback to `"2006-01-02"`, returning `false` when empty or unparseable and `now.After(t)` otherwise.
- `[concept.go:Concept.SanctionedQuery]` resolves a concept's sanctioned computation query in precedence order: `Frontmatter.Attestation.Query`, `# Computation` Markdown fenced/indented blocks on `Attested Computation` / `Attestation` concepts, or `Frontmatter.Computation`.
- `[diff.go:Diff]` requires non-nil `oldB` and `newB` pointers, never returns a non-nil error, compares only `Bundle.Concepts` (ignoring `IndexFiles`, `Bundle.OKFVersion`, `Concept.Path`, and `Concept.ParseError`), and `[diff.go:compareFrontmatter]` inspects only `Type`, `Title`, `Desc` (`"description"`), `Resource`, `Timestamp`, order-insensitive `Tags` (`[diff.go:slicesEqual]`), and `Extra` (`"extra.<key>"` via `reflect.DeepEqual`), omitting `Frontmatter.OKFVersion`, OKF v0.2 `Frontmatter` fields, and `Concept.Citations`.
- `[merge.go:Merge]` requires `strategy` to be `MergeUnion`, `MergeOurs`, or `MergeTheirs` (returning an error otherwise), sets `merged.Path` to `source.Path`, and copies concepts via `[merge.go:copyConcept]`, which preserves `ID`, `Path`, `Body`, `ParseError`, `Citations`, and v0.1 `Frontmatter` fields (`Type`, `Title`, `Desc`, `Resource`, `Timestamp`, `Tags`, shallow `Extra` map) while omitting `IndexFiles`, `OKFVersion`, and OKF v0.2 `Frontmatter` fields.
- `[merge.go:reconcileConcepts]` under `MergeUnion` selects base vs. override by `time.RFC3339` `Frontmatter.Timestamp` (preferring valid RFC3339 over invalid; `t1.After(t2)` when both parse so ties favor `target`; string `>=` when both fail so ties favor `source`), backfills empty `Type`/`Title`/`Desc`/`Resource` and missing `Extra` keys from override, unions non-empty `Tags` sorted lexicographically, and deduplicates `Citations` by `URI` (`target` winning ties) sorted ascending by `Citation.Number`.

## Side Effects
- `[diff.go:BundleDiff.PrettyPrint]` writes formatted drift output to the supplied `io.Writer` and mutates receiver slice `bd.Changes` in-place by sorting ascending by `ConceptDiff.ID`.

## Verification
- Run: `go test ./...`
