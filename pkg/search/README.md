# search

## Overview
Package `search` queries an Open Knowledge Format (OKF) Hub `/api/bundles` endpoint to discover bundles matching optional text query and tag filters.

## Interface
- `[search.go:BundleResult]` — ``type BundleResult struct { ID string `json:"id"`; URN string `json:"urn,omitempty"`; Name string `json:"name"`; Description string `json:"description"`; Namespace string `json:"namespace"`; IsPrivate bool `json:"is_private"`; Version string `json:"version,omitempty"`; Tags []string `json:"tags,omitempty"` }``
- `[search.go:SearchBundles]` — `func SearchBundles(query, tag, host, apiKey string) ([]BundleResult, error)`
- `[search.go:SearchBundlesWithClient]` — `func SearchBundlesWithClient(query, tag, host, apiKey string, client *http.Client) ([]BundleResult, error)`

## Invariants
- `[search.go:SearchBundlesWithClient]` — Resolves empty `host` from `os.Getenv("OKF_HUB_HOST")`, falling back to `"http://localhost:8080"` when unset, and strips a single trailing `"/"` via `strings.TrimSuffix`.
- `[search.go:SearchBundlesWithClient]` — Includes URL query parameters `"q"` and `"tag"` only when `query != ""` and `tag != ""` respectively on `fmt.Sprintf("%s/api/bundles?%s", host, params.Encode())`, and attaches `Authorization: Bearer <apiKey>` only when `apiKey != ""`.
- `[search.go:SearchBundles]` / `[search.go:SearchBundlesWithClient]` — Uses a default 10-second timeout (`&http.Client{Timeout: 10 * time.Second}`) in `SearchBundles` and whenever `client == nil` in `SearchBundlesWithClient`.
- `[search.go:SearchBundlesWithClient]` — Requires HTTP status `200 OK` (`http.StatusOK`) and returns an error for any non-200 status code prior to decoding JSON into `[]BundleResult`.

## Side Effects
- `[search.go:SearchBundlesWithClient]` — Reads the `OKF_HUB_HOST` environment variable via `os.Getenv` when `host == ""`.
- `[search.go:SearchBundlesWithClient]` — Sends an outbound HTTP `GET` request to `<host>/api/bundles?<params>` via `client.Do` and reads/closes `resp.Body`.

## Verification
- Run: `go test ./...`
