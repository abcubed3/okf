# Package `ai` (`pkg/ai`)

## Overview

Package `ai` (`github.com/abcubed3/okf/pkg/ai`) provides the generative AI integration layer for Open Knowledge Format (OKF) bundles using the Google Gemini SDK (`github.com/google/generative-ai-go/genai`). It supports two primary workflows over `github.com/abcubed3/okf/pkg/bundle` and `github.com/abcubed3/okf/pkg/assembly`:

1. **Grounded Question Answering (`AskBundle`)**: Scores bundle concepts against lexical query terms (`FindSeedConcepts`), assembles bounded bidirectional XML graph context around the selected seed concept, collects attested SQL definitions, and synthesizes an answer via Gemini (or returns a deterministic offline Markdown summary when no API key is configured).
2. **Automated Concept Curation (`CurateConcepts`)**: Enriches raw harvested `*bundle.Concept` entries concurrently using schema-constrained JSON generation from Gemini, updating concept descriptions and Markdown bodies in place.

## Interface

- `[ask.go:AskOptions]` — Configuration struct for `AskBundle` (`Model`, `APIKey`, `Depth`, `MaxChars`, `SeedConcept`).
- `[ask.go:AskResult]` — Output struct returned by `AskBundle` (`Query`, `Answer`, `SeedConcept`, `ReferencedConcepts`, `Attestations`, `AssembledContext`).
- `[ask.go:ConceptMatch]` — `type ConceptMatch struct { Concept *bundle.Concept; Score int }` pairing a `*bundle.Concept` with its lexical relevance `Score` (`int`).
- `[ask.go:FindSeedConcepts]` — `func FindSeedConcepts(b *bundle.Bundle, query string) []*bundle.Concept` scoring bundle concepts against whitespace-split query terms (weights: ID `+15`, Title `+12`, exact case-insensitive Tag `+10`, Resource `+8`, Desc `+6`, Type `+4`) and returning matched `*bundle.Concept` pointers sorted by descending score.
- `[ask.go:AskBundle]` — `func AskBundle(ctx context.Context, b *bundle.Bundle, query string, opts AskOptions) (*AskResult, error)` executing graph-grounded Q&A (extracting sanctioned queries via `Concept.SanctionedQuery()`) with online Gemini synthesis or offline context/attestation fallback.
- `[curator.go:CurateOptions]` — Configuration struct for `CurateConcepts` (`Model`, `APIKey`).
- `[curator.go:CurateConcepts]` — `func CurateConcepts(ctx context.Context, concepts []*bundle.Concept, opts CurateOptions) error` concurrently enriching concept `Frontmatter.Desc` and `Body` fields via Gemini structured JSON output.

## Invariants

- `[ask.go:FindSeedConcepts]` — Returns `nil` if `b == nil`, `b.Concepts` is empty, or `query` contains no whitespace tokens; ignores query tokens shorter than 2 bytes (`len(term) < 2`) and excludes concepts with score `<= 0`.
- `[ask.go:AskBundle]` — Rejects `b == nil`, empty/whitespace-only `query`, or bundles where no seed concept ID can be resolved from `opts.SeedConcept`, `FindSeedConcepts`, or fallback iteration over `b.Concepts`.
- `[ask.go:AskBundle]` — Normalizes `Depth <= 0` to `2` and `MaxChars <= 0` to `16000` (`MaxTokens = maxChars / 4`, `Direction = assembly.DirectionBidirectional`, `Format = "xml"`).
- `[ask.go:AskBundle]` — Resolves model selection in precedence order `opts.Model` -> `GEMINI_MODEL` environment variable -> `"gemini-3.8-flash"`.
- `[ask.go:AskBundle]` — When both `opts.APIKey` and `GEMINI_API_KEY` are empty, skips network client creation and returns a non-error offline `AskResult` (including the first matched attestation query if present); when calling Gemini, returns an error if `Candidates` or `Parts` are empty.
- `[curator.go:CurateConcepts]` — Requires a non-empty API key via `opts.APIKey` or `GEMINI_API_KEY` (no offline fallback), defaults empty `opts.Model` to `"gemini-2.5-flash"`, bounds concurrency with `errgroup.SetLimit(5)`, and enforces `application/json` schema output requiring `"description"` and `"body"` string properties.

## Side Effects

- `[ask.go:AskBundle]` — Reads `GEMINI_API_KEY` and `GEMINI_MODEL` from process environment via `os.Getenv` when unset in `AskOptions`.
- `[ask.go:AskBundle]` — Opens a Google Gemini client (`genai.NewClient`) and performs a network `GenerateContent` RPC when an API key is provided.
- `[curator.go:CurateConcepts]` — Reads `GEMINI_API_KEY` from process environment via `os.Getenv` when `opts.APIKey` is empty.
- `[curator.go:CurateConcepts]` — Spawns up to 5 concurrent goroutines issuing network `GenerateContent` RPCs to the Google Gemini API.
- `[curator.go:CurateConcepts]` — Mutates elements of the caller's `concepts` slice in place (`c.Frontmatter.Desc` and `c.Body`).

## Verification

- Run: `go test ./...`
