# OKF Go SDK (`github.com/abcubed3/okf/pkg`)

[![Go Reference](https://pkg.go.dev/badge/github.com/abcubed3/okf.svg)](https://pkg.go.dev/github.com/abcubed3/okf)
[![OKF Spec v0.3](https://img.shields.io/badge/OKF%20Spec-v0.3-green.svg)](https://okf.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../LICENSE)

The `pkg` directory provides the public Go SDK (`github.com/abcubed3/okf/pkg/...`) for parsing, validating, harvesting, assembling, exporting, and serving **[Open Knowledge Format (OKF)](https://okf.md)** bundles (v0.1, v0.2, and v0.3).

## Installation

Requires Go `1.26` or newer:

```bash
go get github.com/abcubed3/okf@latest
```

---

## Quickstart

### 1. Parse, Inspect Trust Tiers, and Validate a Bundle

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/validator"
)

func main() {
	ctx := context.Background()

	b, err := parser.ParseBundle(ctx, "./knowledge")
	if err != nil {
		log.Fatalf("parse bundle: %v", err)
	}

	// Run OKF conformance and link validation
	issues := validator.ValidateBundle(b, validator.Options{})
	for _, issue := range issues {
		fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
	}

	// Query concepts and inspect OKF v0.3 trust & freshness signals
	now := time.Now().UTC()
	for _, c := range b.ConceptsByType("Metric") {
		fmt.Printf("%s (%s) — tier=%s stale=%v\n",
			c.ID, c.Frontmatter.Title, c.TrustTier(), c.IsStale(now))
	}
}
```

### 2. Build a Concept Graph & Assemble Bounded LLM Context

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/parser"
)

func main() {
	b, err := parser.ParseBundle(context.Background(), "./knowledge")
	if err != nil {
		log.Fatal(err)
	}

	graph := assembly.BuildGraph(b)
	opts := assembly.AssemblyOptions{
		MaxDepth:      2,
		MaxTokens:     4000,
		MaxCharacters: 16000,
		Direction:     assembly.DirectionBidirectional,
		Format:        "xml", // or "markdown"
	}

	result, err := assembly.AssembleContext(graph, "metrics/mrr", opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Context)
}
```

### 3. Harvest Metadata Programmatically

```go
package main

import (
	"context"
	"log"

	"github.com/abcubed3/okf/pkg/harvester"
)

func main() {
	ctx := context.Background()

	h := harvester.NewOpenAPIHarvester("./openapi.yaml")
	concepts, err := h.Harvest(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := harvester.WriteConcepts(concepts, "./knowledge"); err != nil {
		log.Fatal(err)
	}
}
```

### 4. Embed an OKF MCP Server

```go
package main

import (
	"context"
	"log"

	okfmcp "github.com/abcubed3/okf/pkg/server/mcp"
)

func main() {
	ctx := context.Background()

	srv, err := okfmcp.NewMCPServer("./knowledge")
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.StartWatcher(ctx); err != nil {
		log.Fatal(err)
	}
	if err := srv.StartStdio(ctx); err != nil {
		log.Fatal(err)
	}
}
```

---

## SDK Packages & Boundaries

All public packages below follow semantic versioning on `github.com/abcubed3/okf`. (Helpers under `pkg/internal` are module-private and excluded from the public API.)

### Core Model, Parsing, Validation & Context Graph

- **`bundle`** ([`./bundle/README.md`](./bundle/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/bundle)) — Core OKF `Bundle` and `Concept` domain models, OKF v0.3 current-only trust-tier and staleness evaluation, v0.3 bundle migration (`MigrateBundleToV03`), structural diffing (`Diff`), and strategy-based bundle merging (`Merge`).
- **`parser`** ([`./parser/README.md`](./parser/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/parser)) — Concurrent OKF Markdown/YAML bundle parser (`ParseBundle`, `ParseBundleWithOptions`, `ParseConceptReader`, `ParseCitationsFromMarkdown`) and shallow Git URL resolver (`ResolvePath`).
- **`validator`** ([`./validator/README.md`](./validator/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/validator)) — Conformance and OKF v0.3 trust-signal linter (`ValidateBundle`, `ValidateConcept`, `ValidateIndexFile`, `ValidateLogFile`) verifying required fields, reserved files, RFC 3339 timestamps, lifecycle `superseded_by`, footnote provenance, attestations, symlink-safe bundle path containment, and local/`hub://` links.
- **`assembly`** ([`./assembly/README.md`](./assembly/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/assembly)) — In-memory bidirectional concept link graph builder (`BuildGraph`) and token/character-bounded BFS context assembler (`AssembleContext`).

### Harvesting, AI, Exporters & Documentation

- **`harvester`** ([`./harvester/README.md`](./harvester/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/harvester)) — Metadata harvesters (`Harvester`, `WriteConcepts`) for BigQuery, MySQL, PostgreSQL, Cloud Spanner, Git repositories, OpenAPI specs, Protobuf schemas, and web pages with optional Gemini classification.
- **`ai`** ([`./ai/README.md`](./ai/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/ai)) — Gemini-backed concurrent schema curation (`CurateConcepts`) and graph-grounded bundle Q&A (`AskBundle`, `FindSeedConcepts`) with offline fallback.
- **`skill`** ([`./skill/README.md`](./skill/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/skill)) — Agent Skill compiler (`ExportBundleToSkill`) and embedded official OKF skill workspace installer (`InstallOfficialSkill`, `ResolveSkillInstallPath`).
- **`export`** ([`./export/README.md`](./export/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/export)) — Schema.org JSON-LD graph serializer (`ExportBundleToJSONLD`, `GetJSONLDString`) mapping OKF concepts into `Dataset`, `WebAPI`, and `TechArticle` entities.
- **`generator`** ([`./generator/README.md`](./generator/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/generator)) — Static interactive HTML catalog compiler (`Generate`, `HTMLTemplate`) producing `viz.html`, JSON-LD metadata, and per-concept `concepts/<id>.json` payloads.

### Protocol Servers, Sync & OKF Hub Clients

- **`server`** ([`./server/README.md`](./server/README.md)) — Protocol server adapters:
  - **`server/mcp`** ([`./server/mcp/README.md`](./server/mcp/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/server/mcp)) — MCP stdio and streamable HTTP server with `fsnotify` live reload.
  - **`server/lsp`** ([`./server/lsp/README.md`](./server/lsp/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/server/lsp)) — LSP 3.16 stdio diagnostics server for real-time IDE validation.
- **`sync`** ([`./sync/README.md`](./sync/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/sync)) — One-shot and daemonized knowledge sync engine (`Engine`, `Run`, `StateManager`) with SHA-256 change tracking across Git, Google Drive, Notion, Confluence, and Jira connectors.
- **`publish`** ([`./publish/README.md`](./publish/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/publish)) — OKF Hub bundle uploader (`PublishBundle`).
- **`pull`** ([`./pull/README.md`](./pull/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/pull)) — OKF Hub bundle downloader (`PullBundle`) resolving `hub://` URIs into local Markdown concept files.
- **`search`** ([`./search/README.md`](./search/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/search)) — OKF Hub bundle discovery client (`SearchBundles`, `SearchBundlesWithClient`).
- **`config`** ([`./config/README.md`](./config/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/config)) — Local CLI/SDK credential storage (`~/.okf/config.json`) and OKF Hub API key resolution (`GetAPIKey`, `LoadConfig`, `SaveConfig`).
- **`telemetry`** ([`./telemetry/README.md`](./telemetry/README.md) · [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf/pkg/telemetry)) — Opt-in asynchronous anonymous event dispatcher (`SendEvent`, `IsDisabled`).
- **`internal`** ([`./internal/README.md`](./internal/README.md)) — Module-internal helper packages (`./internal/links/README.md`) withheld from external module consumers.

## License

Licensed under the [Apache License 2.0](../LICENSE).
