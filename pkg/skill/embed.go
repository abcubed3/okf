package skill

// OfficialSkillMarkdown contains the official OKF Agent Skill content.
const OfficialSkillMarkdown = `---
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
1. **Author or Edit Concepts:** Create or update OKF concept documents ('Metric', 'Table', 'Playbook', 'Attestation', etc.).
2. **Validate Bundles:** Check bundle integrity, YAML schemas, internal link graph, and trust signals using 'okf lint'.
3. **Harvest Metadata:** Extract schemas from PostgreSQL, MySQL, BigQuery, Spanner, OpenAPI, Protobuf, Git, or Web pages using 'okf harvest'.
4. **Assemble LLM Context:** Prune and retrieve subgraphs using deterministic Breadth-First Search (BFS) via 'okf assemble'.
5. **Serve Context over MCP:** Start or interact with the OKF Model Context Protocol server via 'okf mcp'.
6. **Generate Agent Skills:** Compile OKF knowledge bundles into domain-specific 'SKILL.md' files using 'okf skill export'.
7. **Ask Grounded Questions:** Query verified organizational knowledge using 'okf ask'.

---

## 📐 OKF v0.2 Specification Standards

### 1. Bundle Directory Structure
An OKF bundle is a directory containing UTF-8 Markdown files:
` + "```text" + `
okf-bundle/
├── index.md                 # Root progressive disclosure index (optional, recommended)
├── log.md                   # Chronological audit log of updates (optional)
├── metrics/                 # Grouped concept documents
│   └── mrr.md
├── tables/
│   └── subscriptions.md
└── attestations/
    └── mrr_query.md
` + "```" + `

### 2. Concept Document Format
Every concept document is a Markdown file with a YAML frontmatter block enclosed in '---' lines.

#### Frontmatter Fields
- 'type' (string, required): e.g., 'Metric', 'PostgreSQL Table', 'Attestation', 'Playbook'.
- 'title' (string, recommended): Human-readable title.
- 'description' (string, recommended): Short summary (1-2 sentences).
- 'resource' (string, recommended): Physical resource identifier.
- 'tags' (list of string, recommended): Categorization tags.
- 'timestamp' (string, RFC3339): Creation or modification time.

#### OKF v0.2 Trust Signals (§5)
` + "```yaml" + `
status: "stable" # "stable" | "draft" | "deprecated"
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
  - by: "human:lead-architect@company.com"
    at: "2026-08-01T12:15:00Z"
` + "```" + `

### 3. Attested Computation Concepts (§6)
An 'Attestation' represents an audited, sanctioned computation logic (SQL query or formula) approved by domain experts:

` + "```yaml" + `
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
    FROM ` + "`my-project.analytics.subscriptions`" + `
    WHERE status = 'active'
    GROUP BY 1
  executor: "BigQuery / SQL-2024"
  attester: "head-of-finance@company.com"

verified:
  - by: "human:compliance-officer@company.com"
    at: "2026-08-01T12:20:00Z"
---
` + "```" + `

---

## 🛠️ OKF CLI Workflows

- Validate bundle: 'okf lint .'
- Assemble BFS context: 'okf assemble <concept-id> --bundle . --depth 2 --max-chars 16000'
- Harvest schemas: 'okf harvest <source> [flags]'
- Start MCP server: 'okf mcp --bundle . --transport stdio'
- Export Agent Skill: 'okf skill export --bundle . --output .agents/skills/data/SKILL.md'
- Ask Grounded Question: 'okf ask "<query>" --bundle .'
`

// OfficialSpecV02Markdown contains the quick reference for OKF v0.2.
const OfficialSpecV02Markdown = `# OKF Specification v0.2 Quick Reference

## Trust Signals (§5)
- Lifecycle status: 'status: stable | draft | deprecated'
- Freshness: 'stale_after: RFC3339'
- Provenance: 'sources: [{resource, title, author, usage_count, last_modified}]'
- Audit: 'generated: {by, at}'
- Verification: 'verified: [{by, at}]' where 'human:<id>' yields 'human-reviewed'.
`

// OfficialCLICheatsheetMarkdown contains the CLI cheatsheet.
const OfficialCLICheatsheetMarkdown = `# OKF CLI Cheatsheet

- 'okf lint [path]': Validate bundle conformance and link integrity
- 'okf assemble <id>': Deterministic BFS context assembly
- 'okf harvest <source>': Ingest databases, APIs, code, or web
- 'okf mcp': Run Model Context Protocol server
- 'okf skill export': Compile OKF bundle into an Agent Skill (SKILL.md)
- 'okf skill install': Install official OKF Agent Skill into project
- 'okf ask "<query>"': Grounded Q&A against knowledge graph
`
