<p align="center">
  <img src="logo.png" alt="OKF-go Logo" width="200" />
</p>

<h1 align="center">OKF-go</h1>

<p align="center">
  <b>Turn databases, APIs, and docs into verified Markdown knowledge graphs for AI agents and MCP.</b>
</p>

<p align="center">
  <a href="https://github.com/abcubed3/okf/releases"><img src="https://img.shields.io/badge/release-v1.0.9-blue.svg" alt="Release" /></a>
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

**Go:**

```bash
go install github.com/abcubed3/okf/cmd/okf@latest
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
