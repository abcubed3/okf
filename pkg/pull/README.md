# pull

## Overview
Package `pull` downloads Open Knowledge Format (OKF) concepts for a `hub://` bundle URI from an OKF Hub `/api/concepts` endpoint and materializes each concept locally as a Markdown file with YAML frontmatter.

## Interface
- `[pull.go:ConceptResponse]` — ``type ConceptResponse struct { ID string `json:"id"`; Type string `json:"type"`; Title string `json:"title"`; Description string `json:"description"`; Resource string `json:"resource"` }``
- `[pull.go:PullBundle]` — `func PullBundle(hubURI, host, apiKey string) error`

## Invariants
- `[pull.go:PullBundle]` — Requires `hubURI` to begin with `"hub://"` and contain a non-empty trimmed `bundleID`; otherwise returns an error immediately without issuing any network or filesystem calls.
- `[pull.go:PullBundle]` — Enforces a 10-second HTTP client timeout (`10 * time.Second`), attaches `Authorization: Bearer <apiKey>` only when `apiKey != ""`, and requires HTTP status `200 OK` (`http.StatusOK`).
- `[pull.go:PullBundle]` — Attempts JSON unmarshaling first as a wrapped object `{"concepts": *[]ConceptResponse}` (handling empty wrapped `{"concepts": []}` via `wrapper.Concepts != nil`), falling back to a flat `[]ConceptResponse` array.
- `[pull.go:PullBundle]` — Returns `nil` without creating directories or writing files when the decoded `concepts` slice is empty (`len(concepts) == 0`).
- `[pull.go:PullBundle]` — Writes concepts relative to the current working directory under `bundleID` (`filepath.Join(absCacheDir, c.ID+".md")`) after enforcing path-containment (`strings.HasPrefix(filePath, absCacheDir+string(os.PathSeparator))`), logging and continuing past per-concept path-containment, YAML marshal, directory creation, or file write errors rather than aborting.

## Side Effects
- `[pull.go:PullBundle]` — Sends an outbound HTTP `GET` request to `fmt.Sprintf("%s/api/concepts?bundle=%s", host, url.QueryEscape(bundleID))`.
- `[pull.go:PullBundle]` — Writes progress, empty-bundle warnings, per-concept error logs, and completion messages to standard output via `fmt.Printf`.
- `[pull.go:PullBundle]` — Creates local directories (`bundleID` and nested subdirectories for slash-delimited `c.ID` values) via `os.MkdirAll` with mode `0755`.
- `[pull.go:PullBundle]` — Creates or overwrites concept Markdown files (`filepath.Join(bundleID, c.ID+".md")`) with YAML-encoded frontmatter (`gopkg.in/yaml.v3`) via `os.WriteFile` with mode `0644`.

## Verification
- Run: `go test ./...`
