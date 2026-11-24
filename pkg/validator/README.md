# validator

## Overview
Package `validator` (`github.com/abcubed3/okf/pkg/validator`) checks parsed Open Knowledge Format (OKF v0.1/v0.2/v0.3) bundles, reserved `index.md`/`log.md` files, and concepts ([`../bundle/README.md`](../bundle/README.md)) for schema completeness, OKF v0.3 provenance/freshness/lifecycle/attestation trust signals, footnote references (`[^id]`), broken local or remote `hub://` Markdown AST links (`github.com/yuin/goldmark`, [`../internal/links/README.md`](../internal/links/README.md)), citation URIs, and bundle-contained frontmatter file paths.

## Interface
- `[validator.go:IssueSeverity]` — `type IssueSeverity string`: Classification level for validation findings.
- `[validator.go:SeverityError]` — `const SeverityError IssueSeverity = "ERROR"`: Hard conformance failure severity.
- `[validator.go:SeverityWarning]` — `const SeverityWarning IssueSeverity = "WARNING"`: Soft recommendation or freshness warning severity.
- `[validator.go:Issue]` — `type Issue struct { ConceptID string; Path string; Severity IssueSeverity; Message string }`: Structured validation finding associated with a concept or index file.
- `[validator.go:Options]` — `type Options struct { Host string; APIKey string; HTTPClient *http.Client }`: Remote Hub configuration used when resolving `hub://` links.
- `[validator.go:ValidateBundle]` — `func ValidateBundle(b *bundle.Bundle, opts Options) []Issue`: Validates all concepts in `b.Concepts`, reserved `index.md` files in `b.IndexFiles`, and `log.md` files in `b.LogFiles`, returning findings sorted deterministically.
- `[validator.go:ValidateIndexFile]` — `func ValidateIndexFile(relSlash string, idx *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue`: Enforces root `index.md` (`no type`, valid `okf_version`) and nested `index.md` (no frontmatter) rules plus link/citation checks.
- `[validator.go:ValidateLogFile]` — `func ValidateLogFile(relSlash string, lg *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue`: Enforces `log.md` frontmatter prohibition, reverse-chronological `## YYYY-MM-DD` headings, and link/citation checks.
- `[validator.go:ValidateConcept]` — `func ValidateConcept(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue`: Validates a single concept against `b` (accepting unexported cache type `[validator.go:remoteCache]`).

## Invariants
- `[validator.go:ValidateBundle]` resolves `absRoot` and `EvalSymlinks(absRoot)` once per bundle pass (`resolveBundleRoots`), validates `b.Concepts`, `b.IndexFiles`, and `b.LogFiles`, and sorts returned `[]Issue` ascending by `Path` then `Message`.
- `[validator.go:ValidateConcept]` short-circuits immediately with a single `SeverityError` containing `c.ParseError` whenever `c.ParseError != ""`, requires a non-empty `c.Frontmatter.Type` (`SeverityError`), and emits `SeverityWarning` when `c.Frontmatter.Title` or `c.Frontmatter.Desc` is empty or whitespace.
- `[validator.go:validateTrustSignals]` validates lifecycle `status` and `superseded_by` (including multi-hop cycle detection), strict RFC 3339 timestamps with UTC offset (`stale_after`, `generated.at`, `verified[i].at`, `usage_window`, `sources[i].last_modified`), `!time.Now().Before(staleTime)` expiration, duplicate `sources[].id` keys, body `[^id]` footnote provenance references, outdated `verified` records (`verified.at < generated.at`), and `Attested Computation` / silent legacy `Attestation` contracts.
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
