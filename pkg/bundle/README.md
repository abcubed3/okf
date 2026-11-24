# bundle

## Overview
Package `bundle` (`github.com/abcubed3/okf/pkg/bundle`) defines the core in-memory domain model and structural operations for Open Knowledge Format (OKF v0.1/v0.2/v0.3) knowledge bundles, including concept metadata, provenance and current-only actor verification trust-tier derivation, deterministic queries, v0.3 bundle migration (`MigrateBundleToV03`), structural drift comparison, and strategy-driven bundle merging.

## Interface
- `[bundle.go:Bundle]` — `type Bundle struct { Path, OKFVersion, Title, Description, License string; Concepts, IndexFiles, LogFiles map[string]*Concept; SkippedFiles []string }`: Root container mapping concept IDs and reserved `index.md`/`log.md` paths to parsed documents, plus skipped non-concept `.md` paths.
- `[bundle.go:NewBundle]` — `func NewBundle(path string) *Bundle`: Allocates an empty `Bundle` rooted at `path`.
- `[bundle.go:Bundle.GetConcept]` — `func (b *Bundle) GetConcept(id string) (*Concept, bool)`: Looks up a concept by `id`.
- `[bundle.go:Bundle.ListTypes]` — `func (b *Bundle) ListTypes() []string`: Returns unique non-empty concept types sorted lexicographically.
- `[bundle.go:Bundle.ListTags]` — `func (b *Bundle) ListTags() []string`: Returns unique non-empty tags across all concepts sorted lexicographically.
- `[bundle.go:Bundle.ConceptsByType]` — `func (b *Bundle) ConceptsByType(t string) []*Concept`: Filters concepts matching exact `Frontmatter.Type`, sorted ascending by `ID`.
- `[bundle.go:Bundle.ConceptsByStatus]` — `func (b *Bundle) ConceptsByStatus(status string) []*Concept`: Filters concepts matching `Frontmatter.Status` case-insensitively, sorted ascending by `ID`.
- `[bundle.go:Bundle.ConceptsByTrustTier]` — `func (b *Bundle) ConceptsByTrustTier(tier string) []*Concept`: Filters concepts matching derived trust tier (with legacy alias normalization), sorted ascending by `ID`.
- `[concept.go:TrustTierUnverified]`, `[concept.go:TrustTierMachineConfirmed]`, `[concept.go:TrustTierHumanReviewed]` — `const` strings (`"unverified"`, `"machine-confirmed"`, `"human-reviewed"`) representing OKF Spec §5.3 trust tiers.
- `[concept.go:ConceptLifecycleStatus]`, `[concept.go:ConceptLifecycleStable]`, `[concept.go:ConceptLifecycleDraft]`, `[concept.go:ConceptLifecycleDeprecated]` — Lifecycle status type and constants (`"stable"`, `"draft"`, `"deprecated"`).
- `[concept.go:UsageWindow]` — `type UsageWindow struct { From, To string; Extra map[string]any }`: Date-window framing `usage_count` metrics.
- `[concept.go:Source]` — `type Source struct { ID, Resource, Title, Author string; UsageCount int; UsageWindow *UsageWindow; LastModified string; Extra map[string]any }`: Provenance record with deprecated `[concept.go:Source.URI]`, `[concept.go:Source.UnmarshalYAML]`, and `[concept.go:Source.UnmarshalJSON]`.
- `[concept.go:ActorInfo]` — `type ActorInfo struct { By, At string; Extra map[string]any }`: Actor attribution record with `[concept.go:ActorInfo.IsHuman]` and `[concept.go:ActorInfo.Tier]`.
- `[concept.go:DeriveActorTrustTier]` — `func DeriveActorTrustTier(by string) string`: Derives `"unverified"`, `"human-reviewed"`, or `"machine-confirmed"` from an actor identifier.
- `[concept.go:ActorInfoList]` — `type ActorInfoList []ActorInfo`: Slice type with `[concept.go:ActorInfoList.UnmarshalYAML]` supporting both sequence and bare-mapping YAML forms.
- `[concept.go:Attestation]`, `[concept.go:Executor]`, `[concept.go:Attester]`, `[concept.go:Parameter]` — Sanctioned computation and attested computation metadata structs (`Attestation{Query, Executor, Attester}`, `Executor{Resource string; Receipt []string; Extra map[string]any}`, `Attester{Resource string; Extra map[string]any}`, `Parameter{Name, Type string; Required bool; Extra map[string]any}`).
- `[concept.go:Frontmatter]` — `type Frontmatter struct { Type, Title, Desc, Resource, Timestamp, OKFVersion, License string; Tags []string; Sources []Source; UsageWindow *UsageWindow; Runtime string; Parameters []Parameter; Computation string; Executor *Executor; Attester *Attester; Generated *ActorInfo; Verified ActorInfoList; StaleAfter string; Status ConceptLifecycleStatus; SupersededBy string; Attestation *Attestation; Extra map[string]any }`: Parsed YAML metadata block with consumer coercion (`UnmarshalYAML`) and inline `Extra`.
- `[concept.go:Citation]` — `type Citation struct { Number int; Title, URI string }`: Parsed body citation reference.
- `[concept.go:Concept]` — `type Concept struct { ID, Path string; Frontmatter Frontmatter; HasFrontmatter bool; Body, ParseError string; Citations []Citation }`: Single OKF document with `IsStale`, `CurrentVerifications`, `HasOutdatedVerifications`, `IsVerified`, `TrustTier`, `PrimaryVerifier`, `InlineComputation`, `SanctionedQuery`, `ResolveComputation`, and `ResolvedQuery`.
- `[migrate_v03.go:MigrateBundleToV03]` — `func MigrateBundleToV03(b *Bundle, dryRun bool) (*MigrationReport, error)`: Upgrades OKF v0.1/v0.2 bundles to OKF v0.3 conformance.
- `[writer.go:WriteBundle]`, `[writer.go:FormatConceptMarkdown]` — Symlink-safe bundle writer and Markdown/YAML serializer.
- `[diff.go:ConceptDiffType]`, `[diff.go:DiffAdded]`, `[diff.go:DiffDeleted]`, `[diff.go:DiffModified]` — Drift classification type and constants (`"ADDED"`, `"DELETED"`, `"MODIFIED"`).
- `[diff.go:FieldDiff]`, `[diff.go:ConceptDiff]`, `[diff.go:BundleDiff]` — Drift result structures (`FieldDiff{OldValue, NewValue any}`, `ConceptDiff{ID string; Type ConceptDiffType; FmDiffs map[string]FieldDiff; BodyDrift bool}`, `BundleDiff{SourcePath, TargetPath string; Changes []ConceptDiff}`), with `[diff.go:BundleDiff.PrettyPrint]`.
- `[diff.go:Diff]` — `func Diff(oldB, newB *Bundle) (*BundleDiff, error)`: Computes structural additions, deletions, and modifications between two bundles across `Concepts`, `IndexFiles`, and `LogFiles`.
- `[merge.go:MergeStrategy]`, `[merge.go:MergeUnion]`, `[merge.go:MergeOurs]`, `[merge.go:MergeTheirs]` — Reconciliation strategy type and constants (`"union"`, `"ours"`, `"theirs"`).
- `[merge.go:Merge]`, `[merge.go:MergeWithOptions]` — Reconciles `source` and `target` into a new `*Bundle`.

## Invariants
- `[bundle.go:NewBundle]` initializes `Concepts`, `IndexFiles`, and `LogFiles` to non-nil empty maps.
- `[bundle.go:Bundle.ConceptsByTrustTier]` normalizes legacy aliases (`"human"` / `"certified"` -> `"human-reviewed"`, `"automated"` / `"machine"` -> `"machine-confirmed"`) before matching `Concept.TrustTier()` case-insensitively.
- `[concept.go:Source.UnmarshalYAML]` and `[concept.go:Source.UnmarshalJSON]` fall back to legacy `uri` into `Source.Resource` only when `resource` is empty, while preserving `x-*` keys in `Extra`.
- `[concept.go:Concept.CurrentVerifications]` fails closed (`nil`) when `Generated.At` is non-empty and unparseable, and filters `Verified` entries to `at >= generated.at` when `Generated.At` is valid; `Concept.TrustTier` derives `"unverified"`, `"machine-confirmed"`, or `"human-reviewed"` strictly from `CurrentVerifications`.
- `[concept.go:ActorInfoList.UnmarshalYAML]` decodes `!!null` to `nil`, a YAML `MappingNode` to a single-element slice, and a `SequenceNode` to a slice, returning an error on other node kinds.
- `[concept.go:Concept.IsStale]` parses `Frontmatter.StaleAfter` via `ParseOKFTimestamp` (`RFC3339Nano`, `RFC3339`, or `"2006-01-02"`), returning `false` when empty/unparseable and `!now.Before(t)` (`now >= stale_after`) otherwise.
- `[writer.go:ensureSafeWriteTarget]` enforces lexical and `EvalSymlinks` containment and refuses to write through symlinked directories or overwrite symlink targets in both `WriteBundle` and `MigrateBundleToV03`.
- `[diff.go:Diff]` and `[merge.go:MergeWithOptions]` compare and deep-copy all OKF v0.1/v0.2/v0.3 `Frontmatter` fields (including nested `Extra` maps), `Concepts`, `IndexFiles`, and `LogFiles`; `MergeUnion` protects non-stale `human-reviewed` and `machine-confirmed` revisions against lower-trust overrides and compares recency across `Timestamp`, `Generated.At`, and `Verified[].At`.

## Side Effects
- `[diff.go:BundleDiff.PrettyPrint]` writes formatted drift output to the supplied `io.Writer` and mutates receiver slice `bd.Changes` in-place by sorting ascending by `ConceptDiff.ID`.

## Verification
- Run: `go test ./...`
