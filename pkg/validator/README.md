# validator

## Overview
Package `validator` (`github.com/abcubed3/okf/pkg/validator`) checks parsed Open Knowledge Format (OKF) bundles and concepts ([`../bundle/README.md`](../bundle/README.md)) for schema completeness, OKF v0.2 provenance/freshness/attestation trust signals, broken local or remote `hub://` Markdown AST links (`github.com/yuin/goldmark`, [`../internal/links/README.md`](../internal/links/README.md)), citation URIs, and bundle-contained frontmatter file paths.

## Interface
- `[validator.go:IssueSeverity]` — `type IssueSeverity string`: Classification level for validation findings.
- `[validator.go:SeverityError]` — `const SeverityError IssueSeverity = "ERROR"`: Hard conformance failure severity.
- `[validator.go:SeverityWarning]` — `const SeverityWarning IssueSeverity = "WARNING"`: Soft recommendation or freshness warning severity.
- `[validator.go:Issue]` — `type Issue struct { ConceptID string; Path string; Severity IssueSeverity; Message string }`: Structured validation finding associated with a concept or index file.
- `[validator.go:Options]` — `type Options struct { Host string; APIKey string }`: Remote Hub configuration used when resolving `hub://` links.
- `[validator.go:ValidateBundle]` — `func ValidateBundle(b *bundle.Bundle, opts Options) []Issue`: Validates all concepts in `b.Concepts` and link/citation targets in `b.IndexFiles`, returning findings sorted deterministically.
- `[validator.go:ValidateConcept]` — `func ValidateConcept(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue`: Validates a single concept against `b` (accepting unexported cache type `[validator.go:remoteCache]`).

## Invariants
- `[validator.go:ValidateBundle]` validates full rules on `b.Concepts` via `[validator.go:ValidateConcept]` and only link/citation rules on `b.IndexFiles` via `[validator.go:validateLinks]` and `[validator.go:validateCitations]`, sorting returned `[]Issue` ascending by `Path` then `Message`.
- `[validator.go:ValidateConcept]` short-circuits immediately with a single `SeverityError` containing `c.ParseError` whenever `c.ParseError != ""`, and initializes a fresh `remoteCache` map when `rcache == nil`.
- `[validator.go:ValidateConcept]` requires a non-empty `c.Frontmatter.Type` (`SeverityError`) and emits `SeverityWarning` when `c.Frontmatter.Title` or `c.Frontmatter.Desc` is empty or whitespace.
- `[validator.go:validateTrustSignals]` accepts `ConceptLifecycleDraft` and `ConceptLifecycleStable` silently, emits `SeverityWarning` for `ConceptLifecycleDeprecated` or unrecognized `c.Frontmatter.Status` values, validates optional `stale_after`, `generated.at`, and `verified[i].at` timestamps against RFC3339 or `YYYY-MM-DD` when non-empty (`SeverityWarning` on invalid format or `time.Now().After(staleTime)`), and warns on blank `generated.by` or `verified[i].by`.
- `[validator.go:validateTrustSignals]` emits `SeverityError` when `type` is `"attestation"` without a non-empty `attestation.query` or `attestation.executor`, or when `type` is `"attested computation"` without `runtime` and without a legacy `attestation` query/executor.
- `[validator.go:extractLinksFromAST]` and `[validator.go:validateLinks]` extract `*ast.Link` destinations via Goldmark (bypassing inline code and fenced code blocks), skip external URLs (`[../internal/links/links.go:IsExternal]`) and `#` anchors, validate `hub://` links of `>= 2` slash-separated segments (taking `parts[len(parts)-1]` as `conceptID` and `strings.Join(parts[:len(parts)-1], "/")` as `bundleID`) only when `opts.Host != ""` (`SeverityError` on missing remote concept ID, `SeverityWarning` on empty/unresolved remote bundle), and emit `SeverityWarning` when local links escape the bundle root or fail `[validator.go:bundleTargetExists]`.
- `[validator.go:validateCitations]` checks non-external, non-anchor `c.Citations` URIs and emits `SeverityWarning` if a citation escapes the bundle root or fails `[validator.go:bundleTargetExists]`.
- `[validator.go:bundleTargetExists]` enforces bundle-root containment both lexically (`filepath.Rel`) and after symlink resolution (`filepath.EvalSymlinks`), accepting matching concepts (`b.GetConcept`), `b.IndexFiles` entries, regular files under `b.Path`, or directories under `b.Path` containing a non-directory `index.md`.
- `[frontmatter_paths.go:validateFrontmatterPaths]` and `[frontmatter_paths.go:validateFrontmatterPath]` inspect `resource` and `sources[i].resource` (`pathRequired=false`, ignoring missing targets without slashes or extensions in `[frontmatter_paths.go:knownPathExtensions]` so opaque asset/table identifiers pass) and, for `"attested computation"` concepts, `computation`, `executor.resource`, and `attester.resource` (`pathRequired=true`), skipping external URLs/schemes via `[frontmatter_paths.go:isExternalResource]` (excluding host OS volume names via `filepath.VolumeName`), short-circuiting matching concept IDs (`b.GetConcept`) before filesystem stat/symlink checks, and emitting `SeverityWarning` via `[frontmatter_paths.go:frontmatterPathIssue]` if a target is missing, is a directory, or escapes `b.Path` (or the working directory when `b.Path` is empty) lexically, by volume name, or via symlinks.

## Side Effects
- `[validator.go:validateLinks]` issues outbound HTTP `GET` requests to `{opts.Host}/api/concepts?bundle={bundleID}` with a 5-second timeout (adding `Authorization: Bearer {opts.APIKey}` when `opts.APIKey != ""`) on `remoteCache` misses for `hub://` links, caching empty sets on network or non-200 failures to prevent repeated calls.
- `[validator.go:bundleTargetExists]` and `[frontmatter_paths.go:validateFrontmatterPath]` stat local filesystem paths and resolve symlinks under `b.Path` via `filepath.Abs`, `os.Stat`, and `filepath.EvalSymlinks`.
- `[validator.go:validateTrustSignals]` reads the local system clock via `time.Now()` to evaluate `stale_after` expiration.

## Verification
- Run: `go test ./...`
