---
name: okf
description: >-
  Create, edit, validate, harvest, assemble, serve, export, and query Open
  Knowledge Format (OKF) bundles and concepts. Use when a task involves OKF
  documents, the okf CLI, provenance and trust signals, or attested
  computations; not for generic Markdown, YAML, database-schema, or MCP tasks
  unrelated to OKF.
metadata:
  author: "abcubed3.wagnerjt"
  version: 1.0.0
  date: 2026-10-22
  repository: https://github.com/abcubed3/okf/skills
  upstream: https://github.com/GoogleCloudPlatform/open-knowledge-format
---

# Open Knowledge Format (OKF)

The **Open Knowledge Format (OKF)** is a vendor-neutral, lightweight specification developed by Google Cloud for representing structured organizational metadata (runbooks, metrics, database schemas, API definitions, and sanctioned computations) as plain Markdown files with YAML frontmatter.

OKF bridges the gap between structured metadata repositories and unstructured text documentation, serving as the foundational substrate for **AI agents**, **Retrieval-Augmented Generation (RAG)**, and the **Model Context Protocol (MCP)**.

---

## When to Activate This Skill

Use this skill when a task involves creating, editing, validating, harvesting,
assembling, serving, exporting, or querying an OKF knowledge bundle or its
concepts.

Typical requests include:

- Creating or enriching OKF concepts or bundles.
- Running or integrating `okf` commands such as `lint`, `harvest`, `assemble`,
  `mcp`, `skill`, or `ask`.
- Understanding OKF frontmatter, provenance, trust signals, or attested computations.

Do not use this skill for generic Markdown, YAML, database-schema, or MCP tasks
that do not involve OKF.

---

## 📐 OKF v0.2 Specification Standards

### Key Terminology

- **Knowledge Bundle** (or **bundle**): A self-contained, hierarchical collection of knowledge documents in a directory tree of `.md` files. The unit of distribution (git repo, tarball, or subdirectory).
- **Concept**: A single unit of knowledge within a bundle, represented as one markdown document (table, metric, playbook, API, etc.)
- **Concept ID**: The path of the concept's file within the bundle, with the `.md` suffix removed. Example: `platforms/github.md` -> ID `platforms/github`
- **Frontmatter**: YAML block delimited by `---` at the top of a markdown file.
- **Body**: Everything after the frontmatter.
- **Link**: Standard markdown link from one concept to another.
- **Source**: A material a concept derives from that is recorded in the `sources` frontmatter field.
- **Provenance**: The set of sources a concept derives from.
- **Actor**: Identifying who or what performed an action, using the convention `<producer>/<version>` for agents, `human:<id>` for people, and `process:<id>` for automation.
- **Trust tier**: Derived from a concept's `verified`: unverified, machine-confirmed, or human-reviewed.
- **Attested Computation**: A concept (`type: Attested Computation`) carrying a sanctioned way to compute a value, so a consumer can confirm the value was produced by running it.

### 1. Bundle Directory Structure

An OKF bundle is a directory containing UTF-8 Markdown files:

```text
okf-bundle/
├── index.md                 # Root index; recommended by this skill
├── log.md                   # Change history; recommended by this skill
├── metrics/                 # Grouped concept documents
│   └── mrr.md
├── tables/
│   └── subscriptions.md
└── attestations/
    └── mrr_query.md
```

### 2. Concept Document Format

#### Required & Recommended Frontmatter Fields

| Field         | Type             | Required?   | Description                                                                   |
| :------------ | :--------------- | :---------- | :---------------------------------------------------------------------------- |
| `type`        | string           | **Yes**     | Concept type (e.g., `Metric`, `PostgreSQL Table`, `Attestation`, `Playbook`). |
| `title`       | string           | Recommended | Human-readable title.                                                         |
| `description` | string           | Recommended | Short summary (1-2 sentences) of the concept.                                 |
| `resource`    | string           | Recommended | Physical resource identifier (e.g., `db.schema.table`, `/v1/users`).          |
| `tags`        | list of string   | Recommended | Categorization tags (e.g., `finance`, `analytics`, `core`).                   |
| `timestamp`   | string (RFC3339) | Recommended | Generation or last update timestamp.                                          |

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
>
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

## Create a Bundle

When the user wants to create an OKF bundle from scratch:

### Workflow

1. Inspect the workspace and clarify the bundle's scope and audience.
2. Choose a simple directory structure and create one concept document per knowledge unit.
3. Follow this skill's default convention: create a bundle-root `index.md` with `okf_version: "0.2"` frontmatter and a bundle-root `log.md`. The format spec permits omitting either.
4. Ground claims in available sources and add useful cross-links. Never invent facts; claim human verification only when it occurred.
5. Run `okf lint` and report the bundle structure, assumptions, and unresolved gaps.

### 1. Determine scope and structure

Ask: What knowledge are we capturing? (tables, metrics, APIs, playbooks, etc.)
Organize into a directory tree that makes sense for the domain.

### 2. Create concept documents

Each concept = one `.md` file. Minimal conformant example:

```markdown
---
type: Metric
---

# Monthly Recurring Revenue (MRR)

Sum of all active subscriptions normalized to a monthly amount.
```

Full v0.2 example with provenance and trust:

```markdown
---
type: Metric
title: Monthly Recurring Revenue
description: Sum of all active subscription revenue normalized to monthly.
tags: [revenue, saas, kpi]
status: stable
generated: { by: human:ftelles, at: 2026-08-25T10:00:00Z }
verified: { by: human:finance-lead, at: 2026-08-25T14:00:00Z }
stale_after: 2026-12-31T00:00:00Z
sources:
  - id: stripe-docs
    resource: https://stripe.com/docs/billing/subscriptions
    title: Stripe Subscription Billing
    author: team:stripe-docs
    last_modified: 2026-06-01T00:00:00Z
---

# Monthly Recurring Revenue (MRR)

## Definition

Sum of all active subscriptions normalized to a monthly amount.[^stripe-docs]
Excludes one-time fees and overages.

## Formula

`MRR = Σ(active_subscription_monthly_value)`

## Related

- [Churn Rate](./churn.md) uses MRR as denominator
- [ARR](./arr.md) = MRR × 12

[^stripe-docs]: Stripe Subscription Billing
```

For more examples across domains, see [references/examples.md](references/examples.md).

### 3. Cross-link concepts

Use standard markdown links. Two forms:

- **Absolute** (bundle-relative, starts with `/`): `[customers](/tables/customers.md)` — **preferred** (stable when files move)
- **Relative**: `[churn](./churn.md)`

Links assert relationships. The kind of relationship is conveyed by surrounding prose, not by the link syntax. Broken links are explicitly permitted — they represent knowledge not yet written.

### 4. Add provenance with footnotes (v0.2)

When claims reference external sources, use `sources` in frontmatter and footnotes in body:

```yaml
sources:
  - id: ga4-schema
    resource: https://developers.google.com/analytics/bigquery/export-schema
    title: GA4 BigQuery Export schema
```

```markdown
The `events_` table is sharded daily as `events_YYYYMMDD`.[^ga4-schema]

[^ga4-schema]: GA4 BigQuery Export schema
```

### 5. Generate the bundle index

Always create a bundle-root `index.md` for progressive disclosure. Declare the
target spec version in its frontmatter. Nested `index.md` files may be added
where useful, but have no frontmatter.

```markdown
---
okf_version: "0.2"
---

# My Knowledge Bundle

- [Metrics](./metrics/index.md) - Business metrics
- [Tables](./tables/index.md) - Data schemas
```

Nested index example:

```markdown
# Metrics

- [MRR](./mrr.md) - Monthly recurring revenue
- [Churn](./churn.md) - Monthly churn rate
- [NPS](./nps.md) - Net Promoter Score
```

Entries should include the description from the linked concept's frontmatter.

### 6. Generate the bundle log

Always create a bundle-root `log.md` with a chronological change history,
newest first, using ISO 8601 date headings:

```markdown
# Update Log

## 2026-08-25

- **Creation**: Added MRR, Churn, and NPS metrics.
- **Creation**: Established directory structure.

## 2026-08-20

- **Initialization**: Bundle created.
```

### 7. Verify conformance

Three rules — all must pass:

1. Every non-reserved `.md` file has parseable YAML frontmatter
2. Every frontmatter has a non-empty `type` field
3. The root `index.md` and `log.md` follow their defined structures

### Output

After bundle creation, briefly report:

1. The bundle structure.
2. The `okf lint` result, if run.
3. Assumptions and unresolved gaps.

Include full file contents only when requested or useful. Example tree:

```text
heavy-knowledge/
├── index.md
├── log.md
├── offerings/
│   ├── my-offering.md
├── platforms/
│   ├── github.md
│   ├── google.md
│   └── aws.md
└── playbooks/
    └── support-playbook.md
```

---

## Enrich Concepts

When the user wants to improve an existing bundle:

1. Inspect the bundle and available evidence; preserve existing content and conventions.
2. Add relevant details, such as schemas or examples, and support sourced claims with provenance. Weave useful relationships into prose as cross-links.
3. Record the actual generator. Add `verified` only after human review; do not infer missing facts.
4. Run `okf lint` and report changed concepts and unresolved gaps.

---

## Create an Attested Computation (v0.2)

Attested Computations are concepts that carry not just what a value _means_ but a sanctioned way to _compute_ it. Use them when you need verifiable, reproducible calculations.

### When to use

- Financial metrics where compliance requires audit trails
- KPIs that must be computed consistently across reports
- Any calculation where "did the sanctioned thing run" matters

### Structure

```markdown
---
type: Attested Computation
title: Revenue for fiscal year
description: Recognized revenue for a fiscal year, per Finance's definition.
status: stable
runtime: bigquery
parameters:
  - { name: year, type: integer, required: true }
executor:
  resource: references/skills/run-on-bq.md
  receipt: [job_id, executed_sql, result]
attester:
  resource: references/attesters/revenue.py
generated: { by: reference_agent/gemini-2.5-pro, at: 2026-06-20T22:53:05Z }
verified: { by: human:ahormati, at: 2026-06-25T09:00:00Z }
stale_after: 2026-09-23T00:00:00Z
sources:
  - id: rev-policy
    resource: https://wiki.acme/finance/revenue-recognition
    title: Revenue recognition policy
---

# Computation

    SELECT SUM(amount) AS revenue
    FROM finance.recognized_revenue
    WHERE fiscal_year = @year

The computation binds only the declared `parameters`, per the recognition
policy.[^rev-policy]

[^rev-policy]: Revenue recognition policy
```

### Key rules

1. **Agent fills parameters only** — The agent supplies _values_ for declared `parameters`, never edits the computation itself
2. **Computation can be inline or external** — Use `# Computation` heading for inline, or `computation:` field for external file
3. **Executor produces receipt** — Evidence the attester inspects
4. **Attester is deterministic** — No LLM, just code that verifies the receipt

### Linking to computations

Other concepts link to Attested Computations:

```markdown
---
type: Metric
title: Revenue
---

# Definition

Recognized revenue for a fiscal year, computed by
[the revenue computation](../computations/revenue.md).
```

---

---

## 🛠️ CLI Quick Reference

Run commands from the bundle root or pass `--bundle <path>` where supported.
See the [CLI reference](./references/cli-cheatsheet.md) for installation and
all commands and flags.

### Validate

```bash
okf lint ./bundle
okf lint ./bundle --strict
```

### Assemble context or ask a question

```bash
okf assemble metrics/mrr --bundle ./bundle --depth 2 --max-chars 16000
okf ask "How is active subscription MRR calculated?" --bundle ./bundle
```

### Harvest metadata

```bash
# Harvest from PostgreSQL / MySQL / Spanner
okf harvest postgres --dsn "postgres://user:pass@localhost:5432/db" --out ./concepts

# Harvest OpenAPI / Swagger specification
okf harvest openapi --spec ./openapi.yaml --out ./concepts/api

# Harvest Git codebase architecture
okf harvest git --repo . --out ./concepts/architecture

# AI-enrich during harvest (Gemini business descriptions)
okf harvest postgres --dsn "$DB_URL" --out ./concepts --ai-enrich
```

### Serve over MCP

```bash
# Use stdio for MCP clients
okf mcp --bundle ./bundle --transport stdio

# Use HTTP only when network access is configured appropriately
okf mcp --bundle ./bundle --transport http --port 8080
```

### Export an Agent Skill

```bash
okf skill export --bundle ./bundle --output .agents/skills/data/SKILL.md --name data
```

---

## 🤖 Agent Rules

### Ground claims in evidence

- Never invent facts, schemas, URLs, or metric definitions. Omit unknowns or ask when missing information blocks the work.
- Preserve unknown frontmatter fields and existing bundle conventions. Treat concept types as free-form.

### Represent trust honestly

- Record provenance only for sources actually used, and identify the actual generator.
- Add `verified` only after human review. Check `stale_after` and flag stale concepts.
- When a relevant attested computation exists, use its sanctioned logic; don't substitute an ad hoc calculation.

### Make focused, consistent edits

- Add only useful metadata, sections, and links. Follow the bundle's existing link conventions; don't force reciprocal links.
- When editing an existing bundle, update its `log.md` chronologically if present; create one for a new bundle.

### Validate and report

- Run `okf lint` after changes. Report its actual result and any unresolved warnings or gaps; don't claim validation you didn't run.
- Use the [OKF v0.2 specification](./references/spec-v02.md) and [CLI reference](./references/cli-cheatsheet.md) when needed.
