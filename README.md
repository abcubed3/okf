<p align="center">
  <img src="logo.png" alt="OKF-go Logo" width="200" />
</p>

<h1 align="center">OKF-go</h1>

<p align="center">
  <b>Turn databases, APIs, and docs into verified Markdown knowledge graphs for AI agents and MCP.</b>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/abcubed3/okf"><img src="https://pkg.go.dev/badge/github.com/abcubed3/okf.svg" alt="Go Reference" /></a>
  <a href="https://github.com/abcubed3/okf/releases"><img src="https://img.shields.io/github/v/release/abcubed3/okf" alt="Release" /></a>
  <a href="https://okf.md"><img src="https://img.shields.io/badge/OKF%20Spec-v0.3-green.svg" alt="OKF Spec v0.3" /></a>
  <a href="https://ghcr.io/abcubed3/okf"><img src="https://img.shields.io/badge/container-ghcr.io-blue.svg" alt="Container" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" /></a>
</p>

---

**OKF-go** is the fast, single-binary CLI and Go library for the **[Open Knowledge Format (OKF)](https://okf.md)**—an open specification for storing schemas, metrics, APIs, and runbooks as plain Markdown files with YAML frontmatter.

Instead of scattering context across wikis, SQL catalogs, and ad-hoc prompts, you maintain a Git-friendly directory of Markdown concepts that OKF-go can **harvest**, **validate**, **assemble within token budgets**, and **serve directly to AI agents**.

## Why OKF-go?

- **Plain Markdown, Git-Native:** Human-readable `.md` files that coexist cleanly alongside `README.md`, `AGENTS.md`, and `SKILL.md`.
- **Auto-Harvest Metadata:** Generate concepts directly from PostgreSQL, MySQL, BigQuery, Cloud Spanner, OpenAPI, Protobuf, Git repos, and web docs.
- **Built-in Trust & Governance (OKF v0.3):** Track provenance (`sources`), human/machine verification (`verified`), freshness (`stale_after`), lifecycle (`superseded_by`), and sanctioned SQL/code (`Attested Computation`).
- **Ready for AI Agents:** Serve bundles over **MCP** (`okf mcp`), compile **Agent Skills** (`okf skill`), assemble bounded graph context (`okf assemble`), or ask grounded questions (`okf ask`).
- **Fast & Lightweight:** Zero-CGO Go binary parsing ~97k concepts/sec and traversing 200-node graphs in <40µs.

---

<a id="installation-and-distribution"></a>

## Installation

**macOS & Linux (One-line installer):**

```bash
curl -sSfL https://okfgo.dev/install.sh | sh
```

**Go CLI:**

```bash
go install github.com/abcubed3/okf/cmd/okf@latest
```

**Go SDK (Library):**

```bash
go get github.com/abcubed3/okf@latest
```

**Docker:**

```bash
docker run --rm -v "$(pwd):/bundle:ro" ghcr.io/abcubed3/okf:latest lint /bundle
```

Pre-built binaries for macOS, Linux, and Windows are also available on [GitHub Releases](https://github.com/abcubed3/okf/releases).

---

## Quickstart

**1. Harvest existing schemas (or write Markdown files by hand):**

```bash
# Extract tables and relationships from Postgres into ./knowledge
okf harvest db \
  --driver postgres \
  --conn "postgresql://postgres:password@localhost:5432/app_db?sslmode=disable" \
  --output ./knowledge
```

**2. Validate links, metadata, and trust signals:**

```bash
okf lint ./knowledge --strict
```

**3. Connect your knowledge bundle to AI tools:**

```bash
# Install the official OKF skill for Antigravity, Claude Code, or Cursor
okf skill install

# Or compile your bundle into a domain-specific SKILL.md
okf skill export --bundle ./knowledge --output .agents/skills/domain/SKILL.md --name "domain"

# Or ask grounded questions from the terminal (uses Gemini)
okf ask "Which tables store active customer subscriptions?" --bundle ./knowledge
```

**4. Build an interactive HTML documentation portal:**

```bash
okf doc --bundle ./knowledge --output ./portal
```

---

## What an OKF Bundle Looks Like

An OKF bundle is just a folder of Markdown files linked together by standard relative links:

```text
knowledge/
├── index.md              # Optional bundle overview
├── tables/
│   ├── users.md
│   └── subscriptions.md
└── metrics/
    └── mrr.md
```

Each concept file pairs structured YAML frontmatter with Markdown documentation:

```markdown
---
type: "Metric"
title: "Monthly Recurring Revenue (MRR)"
description: "Normalized monthly subscription revenue from active customers."
resource: "analytics.mrr_daily"
status: "stable"
stale_after: "2027-01-01T00:00:00Z"
tags: ["finance", "kpi"]
sources:
  - id: "billing-spec"
    resource: "https://docs.example.com/billing"
    title: "Billing Lifecycle Spec"
verified:
  - by: "human:data-lead@example.com"
    at: "2026-08-01T12:15:00Z"
---

# Monthly Recurring Revenue (MRR)

Derived from active records in [Subscriptions](../tables/subscriptions.md).[^billing-spec]

[^billing-spec]: Billing Lifecycle Spec
```

![OKF Concept Document Structure](okf_concept_visual.jpg)

---

## Go SDK

All core capabilities of OKF-go are exposed as reusable Go packages under `github.com/abcubed3/okf/pkg/...` (full reference in [`./pkg/README.md`](./pkg/README.md) and on [`pkg.go.dev`](https://pkg.go.dev/github.com/abcubed3/okf)).

```bash
go get github.com/abcubed3/okf@latest
```

### Example: Parse, Validate, and Assemble Bounded Context

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/abcubed3/okf/pkg/assembly"
	"github.com/abcubed3/okf/pkg/parser"
	"github.com/abcubed3/okf/pkg/validator"
)

func main() {
	ctx := context.Background()

	// 1. Parse an OKF Markdown bundle concurrently
	b, err := parser.ParseBundle(ctx, "./knowledge")
	if err != nil {
		log.Fatalf("parse bundle: %v", err)
	}

	// 2. Validate frontmatter, links, and OKF v0.3 trust signals
	for _, issue := range validator.ValidateBundle(b, validator.Options{}) {
		if issue.Severity == validator.SeverityError {
			log.Fatalf("%s: %s", issue.Path, issue.Message)
		}
	}

	// 3. Build the bidirectional concept graph and assemble bounded LLM context
	graph := assembly.BuildGraph(b)
	opts := assembly.DefaultOptions()
	opts.MaxCharacters = 12000
	opts.Format = "markdown"

	res, err := assembly.AssembleContext(graph, "metrics/mrr", opts)
	if err != nil {
		log.Fatalf("assemble context: %v", err)
	}
	fmt.Printf("Assembled %d nodes (%d tokens):\n%s\n", len(res.Nodes), res.TotalTokens, res.Context)
}
```

### Core SDK Packages

| Package | Description |
| :--- | :--- |
| [`pkg/bundle`](./pkg/bundle/README.md) | Core `Bundle` and `Concept` domain models, OKF v0.3 trust tiers, v0.3 migration, diffing, and merging |
| [`pkg/parser`](./pkg/parser/README.md) | Concurrent Markdown + YAML frontmatter bundle parser and shallow Git URL resolver |
| [`pkg/validator`](./pkg/validator/README.md) | OKF v0.1–v0.3 conformance, link integrity, footnote provenance, and trust-signal validator |
| [`pkg/assembly`](./pkg/assembly/README.md) | Bidirectional concept link graph (`BuildGraph`) and token/character-bounded BFS context assembler |
| [`pkg/harvester`](./pkg/harvester/README.md) | Schema & metadata harvesters for Postgres, MySQL, BigQuery, Spanner, OpenAPI, Protobuf, Git, and Web |
| [`pkg/server/mcp`](./pkg/server/mcp/README.md) | Embeddable Model Context Protocol (MCP) server over stdio and HTTP with live `fsnotify` reload |
| [`pkg/server/lsp`](./pkg/server/lsp/README.md) | Embeddable Language Server Protocol (LSP 3.16) diagnostics server |
| [`pkg/skill`](./pkg/skill/README.md) | Agent Skill (`SKILL.md`) compiler and embedded official OKF skill installer |
| [`pkg/export`](./pkg/export/README.md) | Schema.org JSON-LD graph exporter (`Dataset`, `WebAPI`, `TechArticle`) |
| [`pkg/generator`](./pkg/generator/README.md) | Interactive static HTML documentation and graph portal compiler |
| [`pkg/ai`](./pkg/ai/README.md) | Gemini-backed schema curation and graph-grounded bundle Q&A (`AskBundle`) |

See [`./pkg/README.md`](./pkg/README.md) for the complete SDK guide (including `sync`, `publish`, `pull`, `search`, and `config`).

---

## Common Integrations

### MCP Server (Claude Desktop, Cursor, Antigravity)

Expose search, bounded graph context assembly, skill export, and grounded Q&A to any MCP client:

```json
{
  "mcpServers": {
    "okf-knowledge": {
      "command": "okf",
      "args": ["mcp", "--bundle", "/path/to/knowledge"]
    }
  }
}
```

For remote/containerized deployments, run over HTTP with `okf mcp --bundle ./knowledge --transport http --port 8080`.

### CI/CD Quality Gate (GitHub Actions)

Keep documentation and schema graphs free of broken links or stale metadata on every pull request:

```yaml
name: OKF Lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: abcubed3/okf@v1
        with:
          path: "./knowledge"
```

---

## CLI Command Overview

Run `okf <command> --help` for full flag details.

| Command | Description |
| :--- | :--- |
| `okf lint [path]` | Validate frontmatter, links, and OKF v0.3 rules (`--strict`, `--migrate-v03`) |
| `okf harvest <source>` | Extract concepts from `db`, `openapi`, `proto`, `git`, or `web` (`--ai-enrich`) |
| `okf assemble <id>` | Assemble related concepts via BFS within a character budget (`--max-chars`) |
| `okf mcp` | Start the Model Context Protocol server over `stdio` or `http` |
| `okf skill <install\|export>` | Install the official OKF agent skill or compile a bundle into `SKILL.md` |
| `okf ask "<question>"` | Answer questions grounded in verified bundle concepts using Gemini |
| `okf doc` | Compile a bundle into a searchable static HTML portal with link graphs |
| `okf lsp` | Run the Language Server Protocol daemon for real-time IDE diagnostics |
| `okf export <jsonld\|skill>` | Export a bundle to Schema.org JSON-LD or Agent Skill format |
| `okf diff <a> <b>` / `merge` | Compare two bundles for drift or merge them (`union`, `ours`, `theirs`) |
| `okf sync` | Sync local bundles with configured remotes (`okf.yaml`) |
| `okf auth` / `publish` / `push` / `pull` / `search` | Authenticate and share bundles with an OKF Hub registry |

---

## Documentation & Architecture

- **[User Guides (`./docs`)](./docs/README.md):** Cloud authentication, database/API harvesting, MCP & context assembly, and CI marketplace publishing.
- **[Specification (okf.md)](https://okf.md):** Full OKF v0.3 format specification, trust layer, and `Attested Computation` schemas.
- **[Go Packages (`./pkg`)](./pkg/README.md):** Public Go APIs (`parser`, `validator`, `harvester`, `assembly`, `server`, `skill`, `export`) and [CLI internals (`./cmd`)](./cmd/README.md).
- **[Performance (`./PERFORMANCE.md`)](./PERFORMANCE.md) & [Contributing (`./CONTRIBUTING.md`)](./CONTRIBUTING.md):** Benchmarks, local development (`task build`, `go test ./...`), and release workflows.

## Telemetry & License

Local commands do **not** send telemetry by default (`--remote=false`). Anonymous telemetry is sent to `https://okfgo.dev/api/telemetry` only when `--remote` is explicitly passed (on `okf mcp` or `okf assemble`) or when running `install.sh`.

Licensed under the [Apache License 2.0](./LICENSE).
