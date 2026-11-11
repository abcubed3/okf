# publish

## Overview
Package `publish` resolves, parses, assembles link edges for, and uploads local or remote Open Knowledge Format (OKF) bundles to an OKF Hub server via `/api/bundles`, coordinating with `github.com/abcubed3/okf/pkg/parser` and `github.com/abcubed3/okf/pkg/assembly`.

## Interface
- `[publish.go:PublishPayload]` — ``type PublishPayload struct { Bundle BundlePayload `json:"bundle"` }``
- `[publish.go:BundlePayload]` — ``type BundlePayload struct { ID string `json:"id"`; Name string `json:"name"`; Description string `json:"description"`; Concepts []PublishConcept `json:"concepts"`; Links []GraphEdge `json:"links"` }``
- `[publish.go:GraphEdge]` — ``type GraphEdge struct { From string `json:"from"`; To string `json:"to"` }``
- `[publish.go:PublishConcept]` — ``type PublishConcept struct { ID string `json:"id"`; Type string `json:"type"`; Title string `json:"title"`; Description string `json:"description"`; Resource string `json:"resource"` }``
- `[publish.go:PublishBundle]` — `func PublishBundle(bundlePath, host, apiKey string) error`

## Invariants
- `[publish.go:PublishBundle]` — Derives both `BundlePayload.ID` and `BundlePayload.Name` from `filepath.Base(b.Path)` and sets `BundlePayload.Description` to `"Published OKF Bundle: " + bundleName`.
- `[publish.go:PublishBundle]` — Initializes `BundlePayload.Concepts` and `BundlePayload.Links` as non-nil empty slices so empty collections serialize as JSON `[]` rather than `null`, iterates `b.Concepts` and `graph.Nodes` keys in deterministic sorted order (`slices.Sorted`), skips concepts where `c.ParseError != ""`, and does not run `validator.ValidateBundle` prior to upload.
- `[publish.go:PublishBundle]` — Targets `fmt.Sprintf("%s/api/bundles", host)` without trailing-slash normalization on `host`, always sets `Content-Type: application/json` and `Authorization: Bearer <apiKey>`, and enforces a 30-second HTTP client timeout (`30 * time.Second`).
- `[publish.go:PublishBundle]` — Treats HTTP status codes in `[200, 299]` as success and returns an error for any status code `< 200` or `>= 300`.

## Side Effects
- `[publish.go:PublishBundle]` — Resolves and cleans up local or temporary bundle directories via `parser.ResolvePath(bundlePath)` and deferred `cleanup()`, and reads/parses bundle files from disk via `parser.ParseBundle(context.Background(), absPath)`.
- `[publish.go:PublishBundle]` — Writes progress and completion messages directly to standard output via `fmt.Printf` and `fmt.Println`.
- `[publish.go:PublishBundle]` — Executes an outbound HTTP `POST` request to `<host>/api/bundles` carrying the JSON-serialized `PublishPayload` and Bearer `apiKey` header.

## Verification
- Run: `go test ./...`
