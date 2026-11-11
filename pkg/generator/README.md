# `generator`

## Overview
Package `generator` compiles an Open Knowledge Format (OKF) bundle directory into a standalone interactive static web application (`viz.html`), per-concept JSON payloads (`concepts/<id>.json`), and mirrored bundle asset files. It combines a server-side bundle compiler and JSON-LD/index serializer (`generator.go`) with an embedded single-page Cytoscape.js and `marked.js` HTML/CSS/JS catalog template (`template.go`).

## Interface
- `[generator.go:Generate]` — `func Generate(bundlePath, outputPath string) error`: Parses an OKF bundle at `bundlePath` via `parser.ParseBundle` and writes `viz.html`, `concepts/<id>.json` files, and non-markdown bundle assets into `outputPath`.
- `[generator.go:BundleJSONData]` — `type BundleJSONData struct`: Bundle-level JSON index serialized into `#okf-bundle-data` (`Title`, `Concepts map[string]ConceptIndexEntry`, `Types []string`, `Tags []string`, `Index string`, `Log string`, `Links []GraphEdge`).
- `[generator.go:ConceptIndexEntry]` — `type ConceptIndexEntry struct`: Lightweight concept index metadata excluding full markdown body, sources, and citations (`ID`, `Type`, `Title`, `Description`, `Resource`, `Tags`, `Timestamp`, `Status`, `TrustTier`, `StaleAfter`, `Excerpt`).
- `[generator.go:ConceptJSON]` — `type ConceptJSON struct`: Full per-concept JSON payload written to `concepts/<id>.json` (`ID`, `Path`, `Type`, `Title`, `Description`, `Resource`, `Tags`, `Timestamp`, `Status`, `TrustTier`, `StaleAfter`, `Sources []bundle.Source`, `Generated *bundle.ActorInfo`, `Verified bundle.ActorInfoList`, `Attestation *bundle.Attestation`, `Body`, `Citations []bundle.Citation`).
- `[generator.go:GraphEdge]` — `type GraphEdge struct`: Directed concept relationship edge (`From`, `To` concept IDs).
- `[template.go:HTMLTemplate]` — `const HTMLTemplate`: Untyped string constant containing the single-page HTML, CSS, and JavaScript application template for rendering the interactive graph and documentation catalog.

## Invariants
- `[generator.go:Generate]` & `[generator.go:extractTitleFromMarkdown]` — Bundle `Title` resolves from the first `# ` heading in `<bundlePath>/index.md`, falling back to `filepath.Base(b.Path)`, and finally to `"OKF Knowledge Bundle"` when `Base` is `"."` or `""`.
- `[generator.go:Generate]` — Concepts with a non-empty `ParseError` are excluded from `BundleJSONData.Concepts` and from `concepts/<id>.json` emission.
- `[generator.go:Generate]` — `ConceptIndexEntry.Excerpt` truncates `c.Body` to 297 runes plus `"..."` when `len(c.Body) > 300` and `len([]rune(c.Body)) > 300`; full concept bodies are omitted from `BundleJSONData` and written only to `<outputPath>/concepts/<id>.json` mirroring slash-separated concept IDs via `filepath.FromSlash(id) + ".json"`.
- `[generator.go:Generate]` & `[template.go:HTMLTemplate]` — Template execution binds `templateData` fields `{{.Title}}`, `{{.JSONLD}}` (`export.GetJSONLDString(b)` wrapped in `<script type="application/ld+json">` as `template.HTML`), and `{{.BundleJSON}}` (serialized `BundleJSONData` injected into `<script id="okf-bundle-data" type="application/json">` as `template.JS`).
- `[generator.go:copyAssets]` — Asset copying skips hidden files and directories (names starting with `.` other than root `"."`), all `.md` files, and files named `"go.mod"`, `"go.sum"`, or `"okf"`.
- `[template.go:HTMLTemplate]` — Client-side `loadDetail(id)` fetches relative `'concepts/' + id + '.json'` and falls back to `bundle.concepts[id]` on fetch failure; `trustTier` defaults to `'unverified'` when unset.
- `[template.go:HTMLTemplate]` — Hash routing recognizes `#/concept/<urlencoded-id>` (for IDs present in `bundle.concepts`) and `#/log`, and rewrites relative markdown links (excluding `http`, `mailto:`, and `#`) to in-app concept navigation when `resolveConceptId` matches an entry in `bundle.concepts`.
- `[template.go:HTMLTemplate]` — Cytoscape node dimensions are bounded to `20–60px` (`20 + Math.min(degree * 4, 40)`), zoom to `[0.1, 5]`, and `marked@4.3.0` is pinned with SRI hash `sha256-xoB1Zy2Xbkd3OQVguqESGUhVvUQEsTZH2khVquH5Ngw=`.

## Side Effects
- `[generator.go:Generate]` — Reads bundle files from `bundlePath` via `parser.ParseBundle` and attempts `os.ReadFile` on `<bundlePath>/index.md` and `<bundlePath>/log.md` (ignoring read errors on those two optional files).
- `[generator.go:Generate]` — Creates `<outputPath>`, `<outputPath>/concepts`, and nested concept subdirectories on disk (`0755`), spawns concurrent goroutines via `errgroup.Group` to write `<outputPath>/concepts/<id>.json` (`0644`), and creates/truncates `<outputPath>/viz.html`.
- `[generator.go:copyAssets]` & `[generator.go:copyFile]` — Walks `bundlePath`, creates mirrored non-hidden subdirectories in `outputPath` preserving source `FileInfo.Mode()`, and copies non-skipped asset files (truncating target files and calling `Sync()`).
- `[template.go:HTMLTemplate]` — When loaded in a browser, fetches external fonts/stylesheets from `https://fonts.googleapis.com` and `https://fonts.gstatic.com`, scripts from `https://cdn.jsdelivr.net` (`cytoscape@3.28.1`, `marked@4.3.0`), issues relative `fetch('concepts/' + id + '.json')` HTTP GET requests, and mutates DOM state, `document.title`, and URL hash via `history.replaceState`.

## Verification
- Run: `go test ./...`
