<p align="center">
  <img src="logo.png" alt="OKF-go Logo" width="220" />
</p>

<h1 align="center">OKF-go — Open Knowledge Format Tool Suite</h1>

<p align="center">
  <b>A high-performance Open Knowledge Format (OKF) tool suite written in Go for AI Agents, Context Traversal, and Model Context Protocol.</b>
</p>

<p align="center">
  <a href="https://github.com/abcubed3/okf/releases/tag/v1.0.9"><img src="https://img.shields.io/badge/release-v1.0.9-blue.svg" alt="Release v1.0.9" /></a>
  <a href="https://github.com/GoogleCloudPlatform/knowledge-catalog"><img src="https://img.shields.io/badge/OKF%20Spec-v0.2-green.svg" alt="OKF Spec v0.2" /></a>
  <a href="https://ghcr.io/abcubed3/okf"><img src="https://img.shields.io/badge/container-ghcr.io-blue.svg" alt="Docker Container" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" /></a>
</p>

---

> [!TIP]
> **🚀 Highlights & Recent Capabilities (v1.0.x):**
>
> - **Multi-Arch Container Distribution (`ghcr.io/abcubed3/okf:latest`)**: Official distroless images for Linux AMD64 and ARM64 published to GitHub Container Registry, ready for CI linting and containerized MCP servers.
> - **Strict Conformance Linter (`okf lint --strict`)**: Elevate soft validation warnings (missing titles/descriptions, unverified links) to hard errors to enforce zero-defect quality gates in CI/CD pipelines.
> - **Opinionated AI Agent Skill (`okf skill`)**: Compile any knowledge bundle into an active `SKILL.md` or install the official OKF skill (`okf skill install`) with embedded references (`spec-v02.md`, `create-bundle.md`, `cli-cheatsheet.md`) and examples for Antigravity, Claude Code, Cursor, Windsurf, and Codex.
> - **Grounded AI Q&A (`okf ask`)**: Natural language question answering directly from the CLI backed by deterministic BFS context assembly and Gemini generative AI, citing verified concept files and verbatim attested computations.
> - **Attested Computation & Trust Layer (OKF Spec v0.2)**: Native support for provenance (`sources`), freshness (`stale_after`), verification tiers (`generated`/`verified`), and reproducible computations (`type: Attested Computation`).
> - **Unified CLI Architecture**: All toolchains (`lint`, `harvest`, `assemble`, `mcp` / `server`, `skill`, `ask`, `doc`, `lsp`, `sync`, `export`, `diff`, `merge`, `publish`, `push`, `pull`, `search`, `auth`) consolidated into a single, high-performance `okf` executable.

## System Overview

The **Open Knowledge Format (OKF)** is a vendor-neutral, lightweight specification for structuring organizational knowledge (documentation, runbooks, metrics, database schemas, and API definitions) into machine-readable, human-friendly Markdown files.

By representing knowledge as a directory of Markdown files with structured YAML frontmatter, OKF bridges the gap between structured metadata repositories and unstructured text documentation, serving as an ideal substrate for **AI agents**, **Retrieval-Augmented Generation (RAG)** pipelines, and the **Model Context Protocol (MCP)**. **OKF-go** (`github.com/abcubed3/okf`) provides native, full-featured support for **OKF Specification v0.2**, including its Trust Layer and Attested Computation, consolidating CLI subcommands (`internal/cli`) over modular Go library packages (`pkg/*`) and embedded agent skills (`skills`).

## Local Interface & Invariants

- **Direct Symbols & Build Interfaces**:
  - `[go.mod:module]` — Go module `github.com/abcubed3/okf` targeting `go 1.26.2` (`[go.mod:go]`).
  - `[Taskfile.yml:tasks]` — Task runner targets `default`, `build`, `test`, `benchmark`, `lint`, `cross-compile`, `clean`, `release:status`, `tag`, `tag:patch`, `tag:minor`, `tag:major`, and `tag:push`, injecting `Version`, `Commit`, and `Date` via `[Taskfile.yml:vars]` (`LDFLAGS`).
  - `[action.yml:inputs]` — Composite GitHub Action `'OKFgo Lint'` accepting optional inputs `path` (default `'.'`), `version` (default `'latest'`), and `working-directory` (default `'.'`), emitting output `bundle-path` (`[action.yml:outputs.bundle-path]`).
  - `[.goreleaser.yaml:builds]` — GoReleaser v2 build configuration compiling `./cmd/okf` into `tar.gz`/`zip` release archives (`[.goreleaser.yaml:archives]`), `checksums.txt` (`[.goreleaser.yaml:checksum]`), and multi-arch OCI container images `ghcr.io/abcubed3/okf` (`[.goreleaser.yaml:dockers_v2]`).
  - `[Containerfile:ENTRYPOINT]` — Distroless container definition (`WORKDIR /bundle`, `EXPOSE 8080`, `ENTRYPOINT ["/usr/local/bin/okf"]`) parameterized by `ARG TARGETPLATFORM`.
  - `[.gitlab-ci.yml:include]` — GitLab CI pipeline testing component `$CI_SERVER_FQDN/$CI_PROJECT_PATH/lint@$CI_COMMIT_SHA` (`path: 'testdata'`) defined in `[templates/lint.yml]` (`[templates/lint.yml:spec]` inputs `stage`, `path`, `version`) and publishing releases via `[.gitlab-ci.yml:component-release]`.
  - `[okf.yaml:connectors.google_drive]` — Sample workspace connector configuration defining `google_drive` (`folder_id`; `type` and `resolve_linked_resources` keys are informational and ignored by `pkg/sync.GoogleDriveConfig`, while `service_account` or `credentials_file` is required at runtime).
  - `[install.sh:TAG]` — POSIX shell installer accepting optional positional version tag `$1` and optional `INSTALL_DIR` environment variable (`[install.sh:INSTALL_DIR]`).
- **Root-Level Invariants**:
  - `[go.mod:go]` requires Go `1.26.2`; `[.goreleaser.yaml:builds]` enforces `CGO_ENABLED=0` across `linux`/`darwin`/`windows` (`amd64`/`arm64`) while excluding `windows/arm64` (`[.goreleaser.yaml:builds.ignore]`).
  - `[Taskfile.yml:tag]` rejects empty or `"dev"` tags, enforces semver regex `^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$`, requires a clean Git working tree unless `FORCE=true`/`1`, and forbids overwriting existing Git tags.
  - `[Containerfile:FROM]` runs non-root on `gcr.io/distroless/static-debian13:nonroot`; `[.goreleaser.yaml:dockers_v2]` restricts container platforms to `linux/amd64` and `linux/arm64` and applies `latest` only when `not .Prerelease`.
  - `[install.sh:OS]` and `[install.sh:ARCH]` restrict targets to `darwin`/`linux`/`windows` on `amd64`/`arm64`, reject `windows/arm64`, and exit `1` if version lookup fails or neither `curl` nor `wget` is installed.
- **Root-Level Side Effects**:
  - `[Taskfile.yml:build]` and `[Taskfile.yml:cross-compile]` write `./okf` and `dist/bin/*`; `[Taskfile.yml:lint]` runs `go fmt ./...` in place; `[Taskfile.yml:tag]` creates and pushes annotated Git tags (`PUSH=true` by default unless `DRY_RUN` is set).
  - `[install.sh:URL]` fetches GitHub release archives into a `mktemp -d` directory, extracts and installs `okf` (`chmod +x`) into `/usr/local/bin`, `~/.local/bin`, or `./bin`, and sends an asynchronous `POST` `cli_installed` event to `https://okfgo.dev/api/telemetry`; `[action.yml:runs.steps]` invokes `install.sh`, updates `$GITHUB_PATH`/`$GITHUB_OUTPUT`, and runs `okf lint`.

## Subdirectories & System Boundaries

- `cmd`: [./cmd/README.md](./cmd/README.md) - Executable binary entrypoints grouping `./cmd/okf` ([./cmd/okf/README.md](./cmd/okf/README.md)), which defines build-time `Version`, `Commit`, and `Date` metadata and delegates process execution to `internal/cli`.
- `internal`: [./internal/README.md](./internal/README.md) - Module-private Go packages grouping `./internal/cli` ([./internal/cli/README.md](./internal/cli/README.md)), which routes all `okf` CLI subcommands and flags across `pkg/*` services.
- `pkg`: [./pkg/README.md](./pkg/README.md) - Public Go domain and infrastructure packages covering bundle models (`bundle`), concurrent parsing (`parser`), conformance/trust validation (`validator`), database/API/Git/web harvesting (`harvester`), bounded BFS context assembly (`assembly`), Gemini Q&A and curation (`ai`), static HTML portal generation (`generator`), JSON-LD and Agent Skill export (`export`, `skill`), OKF Hub operations (`config`, `publish`, `pull`, `search`), bidirectional sync (`sync`), telemetry (`telemetry`), and LSP/MCP servers (`server`).
- `skills`: [./skills/README.md](./skills/README.md) - Compile-time `embed.FS` wrapper (`ReadSkill`, `CopySkill`) embedding the official Open Knowledge Format agent skill at [./skills/okf/SKILL.md](./skills/okf/SKILL.md).
- `docs`: [./docs/README.md](./docs/README.md) - Supplemental user guides for cloud authentication, metadata harvesting, MCP context assembly, and CI/CD marketplace publishing.

## Global Invariants

- **Module Visibility Boundaries**: Packages under `internal/cli` (`internal/`) and `pkg/internal/links` (`pkg/internal/`) are restricted by Go's `internal` package visibility rules so external consumers depend only on `pkg/*` and `skills`.
- **Bundle Parsing & Reserved Files**: `pkg/parser.ParseBundleWithOptions` inspects only `.md` files (default 10 MB per-file cap), skips `log.md`, stores `index.md` files in `Bundle.IndexFiles` rather than `Bundle.Concepts`, derives forward-slash `Concept.ID` paths relative to bundle root without `.md`, and records per-file parse errors on `Concept.ParseError` so callers (`pkg/validator`, `pkg/assembly`, `pkg/export`, `pkg/generator`, `pkg/sync`) explicitly guard or report malformed concepts.
- **Filesystem Containment**: `pkg/validator` (`bundleTargetExists`, `validateFrontmatterPath`) and `pkg/harvester.WriteConcepts` enforce lexical and symlink-resolved path containment so relative links, frontmatter resource paths, and harvested concept writes cannot escape the target bundle or output root directory.
- **OKF v0.2 Trust & Attestation Semantics**: Actor trust tiers (`pkg/bundle.DeriveActorTrustTier`) resolve deterministically (`"human:"` prefix -> `"human-reviewed"`, other non-empty -> `"machine-confirmed"`, empty -> `"unverified"`), while `pkg/validator` enforces required `runtime` or legacy `attestation` fields on `Attested Computation` / `Attestation` concepts and elevates `WARNING` findings to CLI exit failures only under `okf lint --strict` (`internal/cli.RunLint`).
- **Thread-Safe Live Server State**: Long-running protocol daemons (`pkg/server/mcp.Server`, `pkg/server/lsp.Server`) and sync persistence (`pkg/sync.StateManager`) guard shared in-memory bundle graphs and state maps with `sync.RWMutex`, preserving the last valid bundle on `mcp` hot-reload parse errors and persisting `.okf/state.json` via atomic temp-file rename.

## Global Side Effects

- **Local Filesystem Reads & Mutations**: Parses and writes Markdown/YAML concept bundles, `.okf/state.json` sync state, `~/.okf/config.json` (`0700`/`0600`) credentials, `viz.html` + `concepts/<id>.json` static portals, `schema.jsonld` graphs, workspace `SKILL.md` directories, and temporary shallow Git clones (`os.MkdirTemp`).
- **External Databases, Cloud APIs, & LLMs**: Issues metadata queries and RPCs against PostgreSQL, MySQL, Cloud Spanner, BigQuery (`pkg/harvester`), Google Gemini (`pkg/ai`, `pkg/harvester` via `GEMINI_API_KEY`), Google Drive v3 (`pkg/sync`), remote Git origins (`pkg/parser`, `pkg/harvester`, `pkg/sync`), target web hosts (`colly` crawler in `pkg/harvester`), and OKF Hub REST endpoints (`/api/bundles`, `/api/bundles/push`, `/api/concepts`).
- **Local Network Listeners, Stdio Servers, & Telemetry**: Serves blocking JSON-RPC over process stdio (`okf mcp`, `okf lsp`), binds TCP HTTP listeners for streamable MCP (`:<port>` with localhost `Origin` and optional bearer `Token` checks) and OAuth/auth callbacks (`:8080` in `pkg/sync`, `:8989` in `internal/cli`), and sends asynchronous fire-and-forget anonymous telemetry events to `https://okfgo.dev/api/telemetry` when `--remote` is enabled or via `install.sh`.

## System Verification

- Run: `go test ./...`

## 📄 Knowledge Format Specification (OKF Spec v0.2)

![OKF Concept Document Structure](okf_concept_visual.jpg)

The **Open Knowledge Format (OKF)** is an open, vendor-neutral specification developed by Google Cloud for representing structured metadata as plain Markdown files with YAML frontmatter. For the full format specification, conventions, schema details, and reference examples, see the official [Google Cloud OKF Specification](https://github.com/GoogleCloudPlatform/knowledge-catalog).

The **OKF-go** provides a high-performance suite of utilities to validate bundles, harvest metadata from databases and APIs, assemble context for Large Language Models (LLMs), and run an MCP Server.

### 📜 OKF v0.2 Concept Examples

OKF v0.2 introduces a **trust layer** (`status`, `stale_after`, `sources`, `generated`, `verified`) and **Attested Computation** (`type: Attested Computation`).

#### 1. Metric Concept with OKF v0.2 Trust Signals

```markdown
---
type: "Metric"
title: "Monthly Recurring Revenue (MRR)"
description: "Total normalized monthly subscription revenue generated by active customers."
resource: "analytics.mrr_daily"
tags:
  - "finance"
  - "revenue"
  - "kpi"

# --- OKF v0.2 Trust Signals ---
status: "stable"
stale_after: "2027-01-01T00:00:00Z"

sources:
  - resource: "https://stripe.com/docs/billing/subscriptions"
    title: "Stripe Subscription Lifecycle Specs"
    author: "Billing Architecture Team"
    usage_count: 142
    last_modified: "2026-06-15T10:00:00Z"

generated:
  by: "okf-harvester/v0.2"
  at: "2026-08-01T12:00:00Z"

verified:
  - by: "human:lead-data-architect@company.com"
    at: "2026-08-01T12:15:00Z"
---

# Monthly Recurring Revenue (MRR)

MRR measures the predictable and recurring revenue components of a subscription business.
```

#### 2. OKF v0.2 Attested Computation Concept

```markdown
---
type: "Attested Computation"
title: "Sanctioned MRR Calculation Query"
description: "Audited and attester-approved BigQuery SQL for computing MRR."
resource: "bigquery.analytics.mrr_query"
tags:
  - "finance"
  - "revenue"
  - "kpi"
status: "stable"
runtime: "bigquery"
parameters:
  - name: "year"
    type: "integer"
    required: false
executor:
  resource: "references/skills/run-on-bq.md"
  receipt: ["job_id", "executed_sql", "result"]
attester:
  resource: "references/attesters/sql-equality.py"

generated:
  by: "okf-ai-curator"
  at: "2026-08-01T12:00:00Z"

verified:
  - by: "human:head-of-finance@company.com"
    at: "2026-08-01T12:20:00Z"
stale_after: "2027-01-01T00:00:00Z"
sources:
  - id: "rev-policy"
    resource: "https://wiki.acme/finance/revenue-recognition"
    title: "Revenue Recognition Policy"
---

# Computation

    SELECT
      DATE_TRUNC(created_at, MONTH) AS revenue_month,
      SUM(amount) AS total_mrr
    FROM `my-project.analytics.subscriptions`
    WHERE status = 'active'
    GROUP BY 1

This concept defines the sanctioned, audit-verified computation logic for financial reporting.[^rev-policy]

[^rev-policy]: Revenue Recognition Policy
```

> [!NOTE]
> OKF v0.2 Attested Computations define a `runtime`, typed `parameters`, `executor`, and `attester`, evaluated via deterministic verification recipes. The CLI also provides backward compatibility for legacy `type: Attestation` concepts.

---

- **OKF v0.2 Trust Signals & Attestation:** Native support for the v0.2 Trust Layer, including Provenance (`sources`), Trust Verification (`generated`/`verified`), Freshness Expiration (`stale_after`), Lifecycle Status (`status`), and Attested Computation (`type: Attested Computation`).
- **Conformance Engine & Linter (`okf lint`):** Validates knowledge bundles for YAML frontmatter correctness, required attributes, broken internal links, and OKF v0.2 trust signal compliance. Supports `--strict` to fail on soft warnings in CI/CD pipelines.
- **Metadata Harvesters (`okf harvest`):** Automatically extracts and converts schemas from databases (PostgreSQL, MySQL, Cloud Spanner, BigQuery), OpenAPI specs, Protobuf files, Git repositories, and web pages into OKF concept documents.
- **Context Assembler (`okf assemble`):** Performs graph-based Breadth-First Search (BFS) starting from a core concept, resolving related concepts within a specified character/token budget to build pruned, high-quality prompt context.
- **Agent Skills Engine (`okf skill`):** Compile OKF knowledge bundles into production-grade AI Agent Skills (`SKILL.md`) for Antigravity, Claude, Cursor, and Codex, arming autonomous agents with verified schemas, attested SQL queries, and domain rules. Also provides `okf skill install` for deploying the official OKF skill into workspaces (`.agents/skills/okf`, `.claude/skills/okf`, `.cursor/skills/okf`).
- **Grounded AI Q&A (`okf ask`):** Query verified organizational knowledge using graph BFS context assembly + Gemini generative AI, providing grounded answers with source concept citations and verbatim attested computation queries.
- **Model Context Protocol Server (`okf mcp`):** Exposes your knowledge base directly to MCP-compatible AI clients (e.g. Claude Desktop, Cursor, Antigravity) via Stdio or HTTP transport, with built-in tools for search, context assembly, dynamic skill compilation, and grounded Q&A.
- **Bundle Operations (`okf diff` / `okf merge`):** Compare two OKF bundles for structural and content differences, or merge them to unify distributed knowledge bases.
- **AI Curator (`--ai-enrich` flag):** AI-powered concept enrichment during metadata harvesting (`pkg/ai.CurateConcepts` for `db`, `openapi`, `proto`, and `git`; `pkg/harvester.LLMClassifier` for `web`) to automatically generate business descriptions and categorize concepts using Gemini models.
- **JSON-LD & Skill Export (`okf export`):** Export your OKF graph into Schema.org JSON-LD format (`okf export jsonld`) or directly into Agent Skills (`okf export skill` / `okf export agent-skill`).
- **Interactive HTML Portal Compiler (`okf doc`):** Compiles an OKF bundle into an interactive static web application with search, tagging, and link relationship graphs.
- **Language Server Protocol Daemon (`okf lsp`):** Runs an LSP 3.16 server over Standard I/O to publish diagnostic errors/warnings in real-time inside LSP-capable IDEs (such as VS Code and Cursor).
- **Knowledge Sync Daemon (`okf sync`):** Synchronizes local concepts with remote repositories via the active bidirectional `git` connector, Google Drive push + prototype pull (`google_drive`), and stub `notion`, `confluence`, and `jira` connectors.
- **OKF Hub Operations (`okf auth`, `okf publish`, `okf push`, `okf pull`, `okf search`):** Authenticate with an OKF Hub registry, publish or incrementally push bundles, pull remote `hub://` bundles, and search shared knowledge bundles.
- **Blazing Fast Performance:** Highly optimized Go architecture parsing ~97k concepts/sec, completing full 200-node graph BFS traversals in <40µs, and validating 100-concept bundles in ~125µs.

---

## 📦 Installation and Distribution

This section details how to install and run the pre-built `okf` CLI binaries, how developers can automate releases, and how to structure and bundle OKF repositories.

### 1. Installing the Pre-Built CLI

You can install the `okf` CLI without building it from source using the options below:

#### Option A: One-Line Shell Installer (macOS & Linux)

Run the automated installer script to download, verify, and install the correct binary for your OS and CPU architecture:

```bash
curl -sSfL https://okfgo.dev/install.sh | sh
```

_By default, this installs to `/usr/local/bin` (if writeable), `~/.local/bin`, or `./bin` (fallback)._

#### Option B: GitHub Releases (Manual Download)

Go to the [Releases](https://github.com/abcubed3/okf/releases) page on GitHub and download the appropriate archive for your operating system:

- **macOS (Apple Silicon / M-series):** `okf_<version>_darwin_arm64.tar.gz`
- **macOS (Intel):** `okf_<version>_darwin_amd64.tar.gz`
- **Linux (64-bit AMD64):** `okf_<version>_linux_amd64.tar.gz`
- **Linux (ARM64 / Graviton):** `okf_<version>_linux_arm64.tar.gz`
- **Windows (64-bit):** `okf_<version>_windows_amd64.zip`

Extract the binary and add it to your system's `PATH`.

#### Option C: Go Install (Build from Source)

If you have Go installed, you can compile and install the CLI directly from the GitHub repository:

```bash
go install github.com/abcubed3/okf/cmd/okf@latest
```

Ensure your `GOBIN` directory (typically `$HOME/go/bin` or `$GOPATH/bin`) is in your system's `PATH`.

#### Option D: Docker

GoReleaser publishes a multi-platform image to GitHub Container Registry for
each release, using the same Linux binaries as the downloadable release
archives. The runtime is a minimal, non-root Debian 13 distroless image, and
the entrypoint is `okf`, so Docker arguments are passed to the CLI as
subcommands.

```bash
docker pull ghcr.io/abcubed3/okf:latest
```

Lint a bundle mounted from the current directory:

```bash
docker run --rm -v "$(pwd):/bundle:ro" ghcr.io/abcubed3/okf:latest lint /bundle
```

Run MCP over HTTP with Docker Compose. Save this as `compose.yaml` and change
`my_bundle_location` to the path to your bundle:

```yaml
services:
  okf-kb:
    image: ghcr.io/abcubed3/okf:latest
    command: ["mcp", "--bundle", "/bundle", "--transport", "http", "--port", "8080"]
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./my_bundle_location:/bundle:ro
```

Then run it as a background task

```bash
docker compose up -d
```

---

### 2. Packaging an OKF Knowledge Bundle

An **OKF Knowledge Bundle** is a structured directory of Markdown files. Bundles can be versioned, archived, and distributed easily.

#### Folder Structure Layout

A typical bundle layout looks like this:

```text
my-knowledge-bundle/
├── index.md                  # Optional directory index for progressive disclosure
├── log.md                    # Optional change history log
├── tables/                   # Conceptual namespace for DB tables
│   ├── users.md
│   └── orders.md
├── apis/                     # Conceptual namespace for API routes
│   └── create_user.md
└── playbooks/                # Runbooks and guides
    └── database_cleanup.md
```

#### Compressing the Bundle for Distribution

To distribute a bundle (e.g., to upload to an ingestion pipeline, send to an agent, or attach to a deployment artifact), you can compress the bundle folder:

**Using tar (Gzipped Tarball):**

```bash
# Compress the bundle
tar -czvf my-bundle-v1.0.tar.gz -C my-knowledge-bundle/ .

# Extract the bundle
tar -xzvf my-bundle-v1.0.tar.gz -C /path/to/destination/
```

**Using zip:**

```bash
# Compress the bundle
zip -r my-bundle-v1.0.zip my-knowledge-bundle/

# Extract the bundle
unzip my-bundle-v1.0.zip -d /path/to/destination/
```

> [!TIP]
> **GitOps Distribution:** Because OKF bundles are plain-text Markdown files, the recommended way to distribute, track changes, and review updates to your knowledge graph is via **Git**. You can run `okf lint` as a pre-commit hook or CI/CD workflow step.

---

## 💻 CLI Commands & Detailed Documentation

### 1. Linter & Validator (`lint`)

Ensures that the knowledge bundle conforms to the specification. It verifies YAML syntax, ensures required fields are set, and checks that internal Markdown links are not broken.

```bash
# Lint the bundle in the current directory
okf lint

# Lint a bundle at a specific directory
okf lint /path/to/my-bundle

# Enforce strict conformance (treat validation warnings as errors, ideal for CI/CD)
okf lint /path/to/my-bundle --strict
```

#### Flags

- `path`: Path to the knowledge bundle directory (default: `.` or current directory).
- `--strict`: Fails with exit code 1 if any warnings are detected (such as missing recommended fields or orphaned internal links).
- `--host`: Host URL for remote concept resolution via OKF Hub (default: `http://localhost:8080`).
- `--api-key`: API key for authenticated OKF Hub requests.

#### Conformance Rules

- **Hard Rules (Fails with Error):**
  - Syntactically valid YAML frontmatter and files within the 10 MB size limit (`ParseError == ""`).
  - Presence of the `type` field in frontmatter.
  - Required fields for Attested Computations (`runtime` on `type: Attested Computation`, or `attestation.query` and `attestation.executor` on legacy `type: Attestation`).
  - Remote `hub://<bundle>/<concept>` links pointing to a missing concept in a reachable Hub bundle.
- **Soft Rules (Emits Warning; Fails with `--strict`):**
  - Presence of recommended fields (`title`, `description`).
  - Resolves and verifies all relative Markdown links (e.g. `[Users](users.md)`), `# Citations` URIs, and local frontmatter resource paths (`resource`, `sources[i].resource`, `computation`, `executor.resource`, `attester.resource`) to guarantee the graph is contained within the bundle root and lacks broken links.
  - Validates lifecycle `status` (`stable` or `draft`; warns on `deprecated` or unrecognized values) and trust signals (`sources`, `generated`, `verified`, `stale_after`).

#### CI/CD Automations (GitHub Actions & GitLab CI)

Automate OKF bundle linting on every push or pull request using official marketplace actions:

##### 🐙 GitHub Actions

Add `.github/workflows/okf-lint.yml` to your repository:

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
          path: "."
```

##### 🦊 GitLab CI/CD Component

Include in your `.gitlab-ci.yml`:

```yaml
include:
  - component: $CI_SERVER_FQDN/abcubed3/okf/lint@v1.0.0
    inputs:
      stage: test
      path: "."
```

_For complete publishing guides, see [Marketplace Publishing Guide](./docs/MARKETPLACE_PUBLISHING.md)._

---

### 2. Schema Harvester (`harvest`)

Extracts metadata from databases, API specs, protobuf files, git repositories, and web pages, generating structured OKF concepts automatically.

#### A. Database Schema Harvesting (`harvest db`)

Supports **PostgreSQL**, **MySQL**, **Cloud Spanner**, and **BigQuery**. Connects to the database, queries the information schema, and generates tables, columns, constraints, and relationships as OKF concepts.

```bash
# PostgreSQL Example (writes concepts to ./my-bundle/tables/<table_name>.md)
okf harvest db \
  --driver postgres \
  --conn "postgresql://postgres:password@localhost:5432/my_db?sslmode=disable" \
  --schema public \
  --output ./my-bundle

# MySQL Example
okf harvest db \
  --driver mysql \
  --conn "user:password@tcp(localhost:3306)/my_db" \
  --output ./my-bundle

# Cloud Spanner Example
okf harvest db \
  --driver spanner \
  --conn "projects/my-project/instances/my-instance/databases/my-db" \
  --output ./my-bundle

# BigQuery Example
okf harvest db \
  --driver bigquery \
  --conn "projects/my-project" \
  --dataset "my_dataset" \
  --output ./my-bundle
```

#### B. OpenAPI Spec Harvesting (`harvest openapi`)

Parses OpenAPI spec documents (JSON/YAML) and generates OKF concepts representing API endpoints (`endpoints/<method>-<path>.md`).

```bash
# Harvest an OpenAPI spec and output to a bundle
okf harvest openapi \
  --spec ../openapi-sample.yaml \
  --output ./my-bundle
```

#### C. Protobuf Schema Harvesting (`harvest proto`)

Extracts messages, RPC services, and field definitions from `.proto` schemas (`protobuf/<fqn>.md`).

```bash
okf harvest proto \
  --path ./protos/user_service.proto \
  --output ./my-bundle
```

#### D. Git Repository Harvesting (`harvest git`)

Extracts metadata from a Git repository, generating a `git/repository` summary concept, up to 15 recent `git/commits/<hash>` concepts, and `git/contributors/<slug>` concepts.

```bash
okf harvest git \
  --repo https://github.com/abcubed3/okf.git \
  --output ./my-bundle
```

#### E. Web Harvesting (`harvest web`)

Crawls target URLs (`--depth`, default `1`) to scrape documentation and structure it as OKF concepts. Optionally powered by Gemini (`--ai-enrich` using `LLMClassifier`) to classify concept types and generate descriptions, while `db`, `openapi`, `proto`, and `git` harvesters support `--ai-enrich`, `--model` (default `gemini-2.5-flash`), and `--api-key` via `CurateConcepts`.

```bash
okf harvest web \
  --url https://example.com/docs \
  --depth 2 \
  --output ./my-bundle/web \
  --ai-enrich
```

---

### 3. Context Assembler (`assemble`)

Traverses the concept relationship graph starting from a target concept ID. Follows relative markdown links up to a configured traversal depth to compile a unified context document for LLMs.

```bash
okf assemble tables/orders \
  --bundle ./testdata/sample \
  --depth 2 \
  --direction bidirectional \
  --format xml \
  --max-chars 16000
```

#### CLI Flags

- `--bundle`: Path to the OKF bundle (default: `.` or current directory).
- `--depth`: Maximum depth of link traversal (default: `2`).
- `--direction`: Traverse `outbound`, `inbound`, or `bidirectional` links (default: `bidirectional`).
- `--format`: Output format, either `xml` or `markdown` (default: `xml`).
- `--max-chars`: Maximum character budget for the output. If exceeded, traversal stops to prevent context overflow (default: `16000`).
- `--monitor`: Render a token-budget utilization progress bar to `stderr` (default: `false`).
- `--remote`: Emit an anonymous `assemble_run` telemetry event (default: `false`).

---

### 4. Model Context Protocol Server (`mcp`)

Exposes the OKF graph as an MCP Server. This allows LLM clients (like Antigravity, Claude Desktop or Cursor) to dynamically discover, search, retrieve, and assemble context from the bundle.

#### Stdio Transport (Standard Input/Output)

Ideal for local IDE and Desktop applications:

```bash
okf mcp --bundle ./testdata/sample --transport stdio
```

#### HTTP Transport

Ideal for remote integrations or network-based MCP clients:

```bash
okf mcp --bundle ./testdata/sample --transport http --port 8080
```

#### Claude Desktop Configuration

To register the OKF-go MCP server with Claude Desktop, add the server to your `claude_desktop_config.json` (typically located at `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "okf-knowledge": {
      "command": "okf",
      "args": ["mcp", "--bundle", "/path/to/my-bundle"]
    }
  }
}
```

_Note: If `okf` is not in your system `PATH`, provide the absolute path to your `okf` executable._

#### Exposed MCP Capabilities (`okf mcp` / `okf server`)

- **Resources:**
  - `okf://index`: A text/markdown list of all concepts in the bundle.
  - `okf://concept/{+id}`: Resolves the complete document for a concept by its ID (including slash-delimited concept IDs).
- **Prompts:**
  - `okf_concept_context`: Automatically generates an analysis prompt for a concept along with its assembled subgraph.
- **Tools:**
  - `list_concepts`: Returns summaries of all concepts.
  - `search_concepts`: Search matching terms across concept IDs, titles, descriptions, tags, and bodies, with optional `type` and `tag` filters.
  - `get_concept`: Fetch raw contents of a concept.
  - `assemble_context`: Run depth-traversal context assembly dynamically from the LLM.
  - `export_agent_skill`: Compile the bundle (or filtered tags) into an Agent Skill (`SKILL.md`) dynamically.
  - `ask_knowledge_base`: Perform grounded Q&A over the knowledge graph with trust verification.

---

### 5. Interactive HTML Portal Compiler (`doc`)

Compiles an OKF knowledge bundle into a fully self-contained static HTML documentation portal featuring search, tagging, and link graphs.

```bash
# Compile bundle in current directory to docs/ folder
okf doc

# Compile specific bundle to custom directory
okf doc --bundle /path/to/my-bundle --output /var/www/okf-portal
```

- `--bundle`: Path to the OKF bundle (default: `.`).
- `--output`: Output path for the static website files (default: `docs`).

---

### 6. Language Server Protocol Daemon (`lsp`)

Integrates OKF validation directly into your IDE. The LSP daemon runs over standard input/output to report conformance linter diagnostics (missing types, broken relative links) as you type.

```bash
# Start Language Server over Stdio
okf lsp
```

To use with Cursor or VS Code, configure your LSP client to invoke `okf lsp` for `markdown` files.

---

### 7. Knowledge Sync Daemon (`sync`)

Keeps your OKF knowledge bundle (`--bundle`, default `.`) in sync with configured remote connectors (`git` bidirectional sync, `google_drive` push + prototype pull, and stub `notion`, `confluence`, and `jira` connectors).

```bash
# Perform a single sync cycle
okf sync --bundle . --config okf.yaml

# Run continuously as a daemon
okf sync --bundle . --config okf.yaml --daemon --interval 300
```

#### Configuration (`okf.yaml`)

Create an `okf.yaml` file in the bundle directory to define connector settings:

```yaml
connectors:
  git:
    repo: "https://github.com/org/knowledge-bundle.git"
    branch: "main"
    token: "ghp_your_token_here"
  google_drive:
    folder_id: "1234567890abcdef"
    service_account: "/path/to/service-account-key.json" # or credentials_file / token_file
  notion:
    token: "secret_notion_api_token"
    parent_id: "notion_page_parent_uuid"
  confluence:
    domain: "your-company.atlassian.net"
    email: "user@company.com"
    token: "jira_api_token_here"
    space_key: "SPACEKEY"
  jira:
    domain: "your-company.atlassian.net"
    email: "user@company.com"
    token: "jira_api_token_here"
```

---

### 8. AI Agent Skills Engine (`skill`)

Bridge organizational knowledge directly into autonomous coding agents (Google Antigravity, Claude Code, Cursor, Windsurf, GitHub Copilot).

#### A. Exporting a Bundle to an Agent Skill (`okf skill export`)

Compiles an OKF bundle or filtered tag subset into an authoritative `SKILL.md` file containing verified schema dictionaries, business metric rules, and sanctioned computation queries:

```bash
# Compile entire bundle into an Agent Skill
okf skill export --bundle ./knowledge --output .agents/skills/data-warehouse/SKILL.md --name "data-warehouse" --desc "Warehouse domain skill"

# Filter by domain tags and types (or omit attestations with --no-attestations)
okf skill export --bundle ./knowledge --output .agents/skills/finance/SKILL.md --name "finance" --tags "finance,revenue" --types "Metric,Attested Computation"

# Using unified export command (alias: agent-skill)
okf export skill --bundle ./knowledge --output ./SKILL.md
```

#### B. Installing the Official OKF Skill (`okf skill install` / `okf skill init`)

Installs the official, self-contained OKF Agent Skill into your project or IDE configuration:

```bash
# Install into default workspace location (.agents/skills/okf, --platform agents)
okf skill install

# Target specific agent platforms
okf skill install --platform antigravity # (or agy) Installs to .agents/skills/okf
okf skill install --platform claude      # (or claude-code) Installs to .claude/skills/okf
okf skill install --platform cursor      # Installs to .cursor/skills/okf
okf skill install --dir /path/to/dir     # Installs to an explicit directory
```

The installed skill is packaged directly into the `okf` binary via Go `embed` and deploys a complete agent knowledge system:

- **`SKILL.md`**: Opinionated guidelines for autonomous agents on how to ground claims in evidence, respect trust verification tiers, honor sanctioned computations, and validate edits with `okf lint`.
- **`references/spec-v02.md`**: The complete OKF Specification v0.2 reference.
- **`references/create-bundle.md`**: Step-by-step guide for creating, structuring, and indexing new knowledge bundles.
- **`references/cli-cheatsheet.md`**: Cheatsheet for all `okf` CLI commands and options.
- **`examples/`**: Working sample concepts for metrics, database schemas, and attested computations.

---

### 9. Grounded AI Q&A (`ask`)

Query verified organizational knowledge from the command line using deterministic graph BFS context assembly combined with Gemini generative AI:

```bash
# Ask a question across the knowledge bundle
okf ask "How is monthly recurring revenue calculated and which table stores subscriptions?" --bundle ./knowledge

# Direct to a specific seed concept with deeper graph traversal
okf ask "What are the downstream dependencies of users?" --seed tables/users --depth 3 --bundle ./knowledge

# Output raw answer text with custom model and context budget
okf ask "What is the sanctioned query for MRR?" --bundle ./knowledge --max-chars 16000 --model gemini-3.8-flash --raw
```

---

### 10. Bundle Diff, Merge, JSON-LD Export, & OKF Hub (`diff`, `merge`, `export`, `auth`, `publish`, `push`, `pull`, `search`)

```bash
# Compare two bundles for structural and metadata drift
okf diff ./bundle-v1 ./bundle-v2

# Merge two bundles using union, ours, or theirs strategy
okf merge ./bundle-source ./bundle-target --strategy union --output ./bundle-merged

# Export a bundle to a Schema.org JSON-LD graph
okf export jsonld --bundle ./knowledge --output schema.jsonld

# Authenticate, publish, push, search, and pull bundles with OKF Hub
okf auth login --token "$OKF_HUB_API_KEY"
okf publish ./knowledge --host http://localhost:8080
okf push ./knowledge --host http://localhost:8080
okf search "revenue" --tag finance --host http://localhost:8080
okf pull hub://finance-bundle --host http://localhost:8080
```

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](./CONTRIBUTING.md) to learn how to set up your development environment, run linters, execute unit tests, and submit pull requests.

---

## Telemetry

When `--remote` is enabled on `okf mcp` (`mcp_start_remote`) or `okf assemble` (`assemble_run`), or when `install.sh` completes installation (`cli_installed`), an anonymous telemetry event is sent to `https://okfgo.dev/api/telemetry`. To use the local MCP server or context assembler without sending `--remote` events, leave `--remote=false` (the default):

```bash
OTEL_SDK_DISABLED=true okf mcp --remote=false
```

`--remote` defaults to `false`, but setting it explicitly makes the local-only behavior clear. `OTEL_SDK_DISABLED=true` disables OpenTelemetry SDK instrumentation; it does not disable direct HTTP telemetry events when `--remote` is set.

---

## 👥 Authors

- **abcubed3** — _Creator & Maintainer_ — [GitHub Profile](https://github.com/abcubed3)

---

## 📄 License

This project is licensed under the Apache License 2.0. See the [LICENSE](./LICENSE) file for the full text.
