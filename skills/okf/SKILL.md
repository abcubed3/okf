---
name: okf
description: >-
  Author, validate, assemble, query, and maintain Open Knowledge Format (OKF v0.2)
  knowledge bundles, concepts, metrics, database schemas, trust signals, and
  attested computations using the okf CLI, MCP server, and agent skills engine.
  Use when working with OKF files (*.md with YAML frontmatter), creating or editing
  metrics, playbooks, tables, or attestations, running 'okf lint', 'okf harvest',
  'okf assemble', 'okf mcp', 'okf skill', or 'okf ask', or integrating organizational
  knowledge into AI agent context.
---

# Open Knowledge Format (OKF) Agent Skill

The **Open Knowledge Format (OKF)** is a vendor-neutral, lightweight specification developed by Google Cloud for representing structured organizational metadata (runbooks, metrics, database schemas, API definitions, and sanctioned computations) as plain Markdown files with YAML frontmatter.

OKF bridges the gap between structured metadata repositories and unstructured text documentation, serving as the foundational substrate for **AI agents**, **Retrieval-Augmented Generation (RAG)**, and the **Model Context Protocol (MCP)**.

---

## 🎯 When to Activate This Skill

Activate this skill whenever you need to:
1. **Author or Edit Concepts:** Create or update OKF concept documents (`Metric`, `Table`, `Playbook`, `Attestation`, etc.).
2. **Validate Bundles:** Check bundle integrity, YAML schemas, internal link graph, and trust signals using `okf lint`.
3. **Harvest Metadata:** Extract schemas from PostgreSQL, MySQL, BigQuery, Spanner, OpenAPI, Protobuf, Git, or Web pages using `okf harvest`.
4. **Assemble LLM Context:** Prune and retrieve subgraphs using deterministic Breadth-First Search (BFS) via `okf assemble`.
5. **Serve Context over MCP:** Start or interact with the OKF Model Context Protocol server via `okf mcp`.
6. **Generate Agent Skills:** Compile OKF knowledge bundles into domain-specific `SKILL.md` files using `okf skill export`.
7. **Ask Grounded Questions:** Query verified organizational knowledge using `okf ask`.

---

## 📐 OKF v0.2 Specification Standards

### 1. Bundle Directory Structure
An OKF bundle is a directory containing UTF-8 Markdown files:
```text
okf-bundle/
├── index.md                 # Root progressive disclosure index (optional, recommended)
├── log.md                   # Chronological audit log of updates (optional)
├── metrics/                 # Grouped concept documents
│   └── mrr.md
├── tables/
│   └── subscriptions.md
└── attestations/
    └── mrr_query.md
```

### 2. Concept Document Format
Every concept document is a Markdown file with a YAML frontmatter block enclosed in `---` lines.

#### Required & Recommended Frontmatter Fields
| Field | Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `type` | string | **Yes** | Concept type (e.g., `Metric`, `PostgreSQL Table`, `Attestation`, `Playbook`). |
| `title` | string | Recommended | Human-readable title. |
| `description`| string | Recommended | Short summary (1-2 sentences) of the concept. |
| `resource` | string | Recommended | Physical resource identifier (e.g., `db.schema.table`, `/v1/users`). |
| `tags` | list of string | Recommended | Categorization tags (e.g., `finance`, `analytics`, `core`). |
| `timestamp`| string (RFC3339) | Recommended | Generation or last update timestamp. |

#### OKF v0.2 Trust Signals (§5)
OKF v0.2 introduces a cryptographic/auditable trust layer to prevent AI agents from using stale or unverified data:

```yaml
# 1. Lifecycle status
status: "stable" # "stable" | "draft" | "deprecated"

# 2. Freshness expiration
stale_after: "2027-01-01T00:00:00Z"

# 3. Provenance sources
sources:
  - resource: "https://stripe.com/docs/billing/subscriptions"
    title: "Stripe Subscription Lifecycle Specs"
    author: "Billing Architecture Team"
    usage_count: 142
    last_modified: "2026-06-15T10:00:00Z"

# 4. Generator audit
generated:
  by: "okf-harvester/v0.2"
  at: "2026-08-01T12:00:00Z"

# 5. Verification audit & Trust Tiers
verified:
  - by: "human:lead-architect@company.com"
    at: "2026-08-01T12:15:00Z"
```

> [!IMPORTANT]
> **Trust Tier Derivation (§5.3 & §7):**
> - If `by` begins with `human:` (e.g. `human:alice`), trust tier is **`human-reviewed`**.
> - If `by` is a machine/process (e.g. `agent:curator-v1`, `process:ci-check`), trust tier is **`machine-confirmed`**.
> - If missing or empty, trust tier is **`unverified`**.

### 3. Attested Computation Concepts (§6)
An `Attestation` represents an audited, sanctioned computation logic (SQL query, calculation script, or deterministic formula) approved by domain experts:

```yaml
---
type: "Attestation"
title: "Sanctioned MRR Calculation Query"
description: "Audited and attester-approved BigQuery SQL for computing MRR."
resource: "bigquery.analytics.mrr_query"
status: "stable"

attestation:
  query: |
    SELECT
      DATE_TRUNC(created_at, MONTH) AS revenue_month,
      SUM(amount) AS total_mrr
    FROM `my-project.analytics.subscriptions`
    WHERE status = 'active'
    GROUP BY 1
  executor: "BigQuery / SQL-2024"
  attester: "head-of-finance@company.com"

generated:
  by: "okf-ai-curator"
  at: "2026-08-01T12:00:00Z"

verified:
  - by: "human:compliance-officer@company.com"
    at: "2026-08-01T12:20:00Z"
---

# Sanctioned MRR Calculation Query

This concept defines the sanctioned computation logic for financial reporting.
Always reference this query rather than writing ad-hoc calculations for MRR.
```

---

## 🛠️ OKF CLI Workflows

The `okf` CLI provides a complete suite of commands. Always run commands from the workspace root or pass `--bundle <path>`.

### 1. Conformance Validation (`okf lint`)
Run the linter after creating or modifying concepts to ensure zero errors and valid link topology:
```bash
# Validate local bundle
okf lint .

# Enforce strict conformance (fail on warnings such as missing recommended fields)
okf lint ./bundle --strict
```

### 2. Context Assembly (`okf assemble`)
Deterministic Breadth-First Search (BFS) graph traversal to gather context for LLM prompts:
```bash
# Assemble context starting from a concept ID with token budget
okf assemble metrics/mrr --bundle . --depth 2 --max-chars 16000 --format xml

# Monitor token window consumption
okf assemble metrics/mrr --bundle . --monitor
```

### 3. Metadata Harvesting (`okf harvest`)
Automatically ingest schemas into OKF concepts:
```bash
# Harvest from PostgreSQL / MySQL / Spanner
okf harvest postgres --dsn "postgres://user:pass@localhost:5432/db" --out ./concepts

# Harvest OpenAPI / Swagger specification
okf harvest openapi --spec ./openapi.yaml --out ./concepts/api

# Harvest Protobuf definitions
okf harvest protobuf --proto-path ./proto --out ./concepts/proto

# Harvest Git codebase architecture
okf harvest git --repo . --out ./concepts/architecture

# AI-enrich during harvest (Gemini business descriptions)
okf harvest postgres --dsn "$DB_URL" --out ./concepts --ai-enrich
```

### 4. Model Context Protocol (`okf mcp`)
Run the MCP server to give AI agents runtime access to the knowledge base:
```bash
# Start MCP server over standard I/O (recommended for agent tools)
okf mcp --bundle ./bundle --transport stdio

# Start MCP server over HTTP
okf mcp --bundle ./bundle --transport http --port 8080
```

### 5. Compiling Agent Skills (`okf skill export`)
Compile an OKF bundle or filtered tag into an active Agent Skill (`SKILL.md`) for any repository:
```bash
# Export entire bundle as an Agent Skill
okf skill export --bundle ./bundle --output .agents/skills/data-warehouse/SKILL.md --name "data-warehouse"

# Export filtered by tags (e.g. finance)
okf skill export --bundle ./bundle --output .agents/skills/finance/SKILL.md --name "finance" --tags "finance,revenue"

# Export via unified export command
okf export skill --bundle ./bundle --output ./SKILL.md
```

### 6. Grounded AI Q&A (`okf ask`)
Ask questions directly against your OKF bundle, backed by BFS context assembly and source verification:
```bash
# Query the knowledge graph
okf ask "How is active subscription MRR calculated?" --bundle ./bundle
```

---

## 🤖 Agent Rules & Best Practices

1. **Never Hallucinate Schema or Metric Names:** Always verify table and column names by reading the concept files or calling `okf assemble` / `get_concept` before writing SQL.
2. **Honor Attested Queries:** If an `Attestation` concept exists for a calculation (e.g. MRR, churn, daily active users), **use the sanctioned query**. Do NOT invent custom calculation logic.
3. **Verify Trust Signals:**
   - Check if `stale_after` has passed. If stale, flag it to the user.
   - Prefer concepts with `verified` entries containing `human:` actors over unverified concepts.
4. **Maintain Bidirectional Link Graph:** When creating a new concept that references another concept, use relative markdown links:
   `[Users Table](../tables/users.md)`.
5. **Run `okf lint` on Every Modification:** Never conclude a task with broken YAML frontmatter or dead markdown links.
