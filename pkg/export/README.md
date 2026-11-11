# export

## Overview
Package `export` serializes Open Knowledge Format (OKF) bundles (`*bundle.Bundle` from `github.com/abcubed3/okf/pkg/bundle`) into `https://schema.org` JSON-LD metadata graphs, mapping table, API/endpoint, and documentation concepts into typed schema.org entities (`Dataset`, `WebAPI`, and `TechArticle`).

## Interface
- `[jsonld.go:Graph]` — ``type Graph struct { Context string `json:"@context"`; Graph []interface{} `json:"@graph"` }``
- `[jsonld.go:PropertyValue]` — ``type PropertyValue struct { Type string `json:"@type"`; Name string `json:"name"`; Description string `json:"description,omitempty"`; ValueReference string `json:"valueReference,omitempty"` }``
- `[jsonld.go:Dataset]` — ``type Dataset struct { Type string `json:"@type"`; ID string `json:"@id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`; Identifier string `json:"identifier,omitempty"`; VariableMeasured []PropertyValue `json:"variableMeasured,omitempty"`; Keywords []string `json:"keywords,omitempty"`; DateModified string `json:"dateModified,omitempty"` }``
- `[jsonld.go:WebAPI]` — ``type WebAPI struct { Type string `json:"@type"`; ID string `json:"@id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`; Documentation string `json:"documentation,omitempty"`; Keywords []string `json:"keywords,omitempty"`; DateModified string `json:"dateModified,omitempty"` }``
- `[jsonld.go:TechArticle]` — ``type TechArticle struct { Type string `json:"@type"`; ID string `json:"@id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`; ArticleBody string `json:"articleBody,omitempty"`; Keywords []string `json:"keywords,omitempty"`; DateModified string `json:"dateModified,omitempty"` }``
- `[jsonld.go:ExportBundleToJSONLD]` — `func ExportBundleToJSONLD(b *bundle.Bundle) ([]byte, error)`
- `[jsonld.go:GetJSONLDString]` — `func GetJSONLDString(b *bundle.Bundle) (string, error)`

## Invariants
- `[jsonld.go:ExportBundleToJSONLD]` — Iterates over `b.Concepts` (`map[string]*bundle.Concept`) in deterministic sorted key order (`slices.Sorted(maps.Keys(b.Concepts))`), ignores `b.IndexFiles`, and skips concepts with a non-empty `ParseError` (`c.ParseError != ""`).
- `[jsonld.go:ExportBundleToJSONLD]` — Top-level `Graph.Context` (`@context`) is always `"https://schema.org"`, every exported concept `@id` is prefixed with `"okf://"` (`"okf://" + c.ID`), and an empty `Frontmatter.Title` falls back to `c.ID` for `Name`.
- `[jsonld.go:ExportBundleToJSONLD]` — Concept `Frontmatter.Type` classification applies case-insensitive substring checks in order: contains `"table"` maps to `Dataset` (`@type: "Dataset"`), else contains `"api"` or `"endpoint"` maps to `WebAPI` (`@type: "WebAPI"`), otherwise maps to `TechArticle` (`@type: "TechArticle"`).
- `[jsonld.go:ExportBundleToJSONLD]` — JSON output is formatted with 2-space indentation (`json.MarshalIndent(g, "", "  ")`), and when `b.Concepts` is empty (or all entries are skipped due to `c.ParseError != ""`), `@graph` serializes as `[]` rather than `null`.
- `[jsonld.go:parseColumnsFromMarkdown]` — Table schema extraction for `Dataset.VariableMeasured` parses only trimmed markdown pipe rows (`|` prefix and suffix, not containing `"---"`, splitting on `"|"` into at least 6 parts) where column 1 (`parts[1]`) is case-insensitively neither `"column"` nor `"field"`, mapping `Name` from `parts[1]`, `ValueReference` from `parts[2]`, and `Description` from `parts[5]`.

## Side Effects
- None.

## Verification
- Run: `go test ./...`
