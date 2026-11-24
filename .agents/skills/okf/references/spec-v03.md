# Open Knowledge Format (OKF) — Version 0.3 Reference

**Version 0.3** (Canonical repository: https://github.com/open-knowledge-format/open-knowledge-format | Documentation: https://okf.md)

OKF is an open, vendor-neutral format for representing structured knowledge as UTF-8 Markdown files (CommonMark + GFM tables + footnotes) with YAML 1.2 frontmatter and RFC 3339 timestamps.

---

## 1. Bundle Structure, Coexistence & Reserved Files (§3, §8, §9)

```text
bundle/
  index.md              # Optional. Progressive disclosure index; root index.md MAY have frontmatter.
  log.md                # Optional. Reverse-chronological update history (no frontmatter).
  README.md             # Ignored by OKF consumers (no 'type' frontmatter).
  AGENTS.md             # Ignored by OKF consumers (no 'type' frontmatter).
  SKILL.md              # Ignored by OKF consumers (no 'type' frontmatter).
  metrics/
    index.md            # Optional nested directory index (MUST NOT have frontmatter).
    revenue.md          # Concept document (YAML frontmatter with non-empty 'type').
```

### Coexistence with Native Repository Files (§3, §11)
- Any non-reserved `.md` file lacking YAML frontmatter OR lacking a non-empty `type` field is **not an OKF concept** and **MUST** be ignored by consumers and validators.
- Malformed YAML frontmatter fences (`---`) remain validation errors.

### Reserved Filenames (§3.1, §8, §9)
- **`index.md`**: Reserved at every directory level; MUST NOT be used for concept documents (`MUST NOT` declare `type`).
  - **Bundle-root `index.md`** MAY carry YAML frontmatter declaring bundle-level metadata:
    ```yaml
    ---
    okf_version: "0.3"
    title: "Acme Analytics Knowledge Base"
    description: "Curated metrics, schemas, and sanctioned computations."
    license: "Apache-2.0"
    ---
    ```
    Producers regenerating root `index.md` **MUST preserve** its existing frontmatter.
  - **Nested `<subdir>/index.md`** files **MUST NOT** carry YAML frontmatter.
- **`log.md`**: Reserved at every directory level; **MUST NOT** carry YAML frontmatter. Date section headings **MUST** use ISO 8601 `## YYYY-MM-DD` form in reverse chronological order.

---

## 2. Concept Documents & Extension Model (§4)

```yaml
---
type: Metric                     # REQUIRED (non-empty string; open vocabulary)
title: Net Revenue               # Recommended
description: Recognized net revenue across active subscriptions. # Recommended
resource: analytics.net_revenue  # Optional canonical URI or asset identifier
tags: [finance, kpi]             # Optional list of categorization tags
x-acme-owner: team:finance       # Optional namespaced extension key
---
```

### `x-` Extension Model (§4.1)
- Future core OKF versions guarantee they will **never** define keys starting with `x-` (conventionally `x-<producer>-<name>`).
- Extension keys are permitted at **every level** (top-level frontmatter, root `index.md` frontmatter, `sources[]`, `usage_window`, `generated`, `verified[]`, `parameters[]`, `executor`, `attester`).
- Producers and consumers that edit or rewrite OKF files **MUST preserve** unrecognized and `x-*` keys.

---

## 3. Provenance, Current-Only Trust Tiers & Lifecycle (§5, §7)

All timestamp-valued fields (`generated.at`, `verified[].at`, `stale_after`, `sources[].last_modified`, `usage_window.from`, `usage_window.to`) **MUST** use ISO 8601 / RFC 3339 datetimes with an explicit UTC offset (e.g., `2026-11-18T14:00:00Z`).

### Provenance: `sources` & `usage_window` (§5.1)
```yaml
sources:
  - id: rev-policy
    resource: https://wiki.acme/finance/revenue-recognition # REQUIRED per entry
    title: Revenue recognition policy
    author: team:finance
    usage_count: 1200
    last_modified: 2026-11-08T00:00:00Z
usage_window:
  from: 2026-10-01T00:00:00Z
  to: 2026-11-01T00:00:00Z
```
Per-claim attribution in the Markdown body uses footnotes keyed to `sources[].id` (e.g., `[^rev-policy]`).

### Actor Convention (§7) & Current-Only Trust Tiers (§5.2, §5.3)
- **Actors**: `human:<id>` (person), `team:<id>` (group), `process:<id>` (automated job), `<producer>/<version>` (agent/tool).
- **`generated`**: `{ by: <actor>, at: <RFC3339> }` — marks the last meaningful content change.
- **`verified`**: List of `{ by: <actor>, at: <RFC3339> }` (a bare single mapping MUST be coerced to a 1-element list by consumers).
- **Current-Only Trust Tier Rule (v0.3 §5.3)**:
  - A verification event is **current** iff `generated` (or `generated.at`) is absent, OR `verified[].at >= generated.at`.
  - Older verifications remain in `verified` as audit history, but do **not** count toward the active trust tier once `generated.at` advances past them.
  - Derived tiers from current verifications:
    1. **`human-reviewed`**: At least one current `human:<id>` verification.
    2. **`machine-confirmed`**: Current non-`human:` verifications only (`team:`, `process:`, `<agent>/<model>`).
    3. **`unverified`**: No current verifications.

### Lifecycle: `status`, `superseded_by`, and `stale_after` (§5.4, §5.5)
- `status`: `draft` | `stable` (default when omitted) | `deprecated`.
- `superseded_by` (v0.3): Optional path or URL on `status: deprecated` concepts pointing to the active replacement concept (e.g., `superseded_by: /metrics/net-revenue.md`). Doubles as a safe forwarding stub when renaming or moving concepts.
- `stale_after`: RFC 3339 timestamp; content is stale when `now >= stale_after`.

---

## 4. Cross-Linking & Path Resolution (§6)

Applies to Markdown links and path-bearing frontmatter fields (`resource`, `sources[].resource`, `superseded_by`, `computation`, `executor.resource`, `attester.resource`):
1. **Explicit URI scheme** (`https://`, `postgresql://`, `hub://`): External / absolute URI.
2. **Leading slash (`/`)**: Resolves from the **bundle root** (recommended for move-resilience).
3. **Relative path**: Resolves from the **directory of the file containing it** (matching standard Markdown/GitHub behavior). For legacy `okf_version: "0.1"` / `"0.2"` bundles, consumers MAY fall back to bundle-root resolution for bare paths.

---

## 5. Attested Computations (§10)

```markdown
---
type: Attested Computation
title: Revenue for fiscal year
description: Recognized revenue for a fiscal year, per Finance's definition.
status: stable
runtime: bigquery
parameters:
  - name: year
    type: integer
    required: true
executor:
  resource: /references/skills/run-on-bq.md
  receipt: [job_id, executed_sql, result]
attester:
  resource: /references/attesters/revenue.py
generated: { by: reference_agent/gemini-2.5-pro, at: 2026-11-18T10:00:00Z }
verified: { by: human:ahormati, at: 2026-11-18T12:00:00Z }
---

# Computation

```sql
SELECT SUM(amount) AS revenue
FROM finance.recognized_revenue
WHERE fiscal_year = @year
```
```

- Discovered by frontmatter scan (`type: Attested Computation`).
- Supply computation either **inline** (single fenced code block under `# Computation`) OR via `computation: <path>`.
- Agents MAY supply values only for declared `parameters` (`name` and `type` required per entry) and MUST NOT alter the computation.
- Deterministic `attester` inspects the execution receipt and returns a verdict containing at least `ok` (`true`/`false`).

---

## 6. Conformance & Versioning (§11, §12)

- **Producers** MUST write parseable YAML 1.2 frontmatter with non-empty `type` on concepts, MUST NOT use `index.md` / `log.md` for concepts, and MUST preserve unknown/`x-*` keys.
- **Consumers** MUST ignore untyped `.md` files, MUST coerce bare `verified` mappings to 1-element lists, MUST tolerate `null` / missing optional fields / unknown `type`s / broken links / unknown `okf_version` values (best-effort consumption).
- **Validators / Linters** separate blocking `MUST` conformance errors from advisory `SHOULD` / trust / freshness warnings.
