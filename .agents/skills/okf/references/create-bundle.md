# Creating an OKF Bundle

Use this reference when creating a bundle from scratch. It supplements the
workflow in `../SKILL.md`; the normative format details are in
[spec-v02.md](./spec-v02.md).

## Structure

Organize concepts in directories that fit the domain. Use one Markdown file per
concept; group related concepts where useful.

```text
my-bundle/
├── index.md
├── log.md
├── metrics/
│   └── mrr.md
├── tables/
│   └── subscriptions.md
└── playbooks/
    └── billing.md
```

This skill recommends always creating a root `index.md` and `log.md`, although
OKF v0.2 makes both optional.

## Concepts

Every concept document needs YAML frontmatter with a non-empty `type`. Add
recommended metadata when useful; don't pad concepts with empty fields or
guess information that isn't supported.

```markdown
---
type: Metric
title: Monthly Recurring Revenue
description: Active subscription revenue normalized to a monthly amount.
tags:
  - revenue
  - saas
---

# Monthly Recurring Revenue

## Definition

Revenue from active subscriptions, normalized to a monthly amount.
```

For claims based on external sources, record only sources actually consulted in
`sources` and cite them in the body where helpful:

```yaml
sources:
  - resource: https://example.com/billing-docs
    title: Billing documentation
```

```markdown
The billing period is monthly.[^billing-docs]

[^billing-docs]: Billing documentation
```

Use `generated` to identify the actual producer. Add `verified` only after
human review; never imply a human verified content that they have not reviewed.

## Links and indexes

Use Markdown links to express relationships in context. Follow the bundle's
existing link style; do not force reciprocal links. An index entry should link
to a concept or subdirectory and, where available, use the concept's
frontmatter description.

The root `index.md` may declare the targeted format version in frontmatter.
This skill's default is:

```markdown
---
okf_version: "0.2"
---

# My Knowledge Bundle

- [Monthly Recurring Revenue](./metrics/mrr.md) - Active subscription revenue normalized monthly.
- [Subscriptions](./tables/subscriptions.md) - Subscription data schema.
```

Nested `index.md` files have no frontmatter:

```markdown
# Metrics

- [Monthly Recurring Revenue](./mrr.md) - Active subscription revenue normalized monthly.
```

## Change log

Create a root `log.md` with date-grouped entries in reverse chronological order.
Use the actual current date and record meaningful initial creation changes:

```markdown
# Update Log

## YYYY-MM-DD

- **Creation**: Added initial concepts and bundle structure.
```

When changing an existing bundle, add a concise entry to its log if it has one.

## Validate and report

Run `okf lint <bundle-path>` after creating the bundle. Confirm that concept
documents have parseable frontmatter and a non-empty `type`, and that the root
index and log follow the guidance above. Report the bundle structure, the actual
lint result, assumptions, and unresolved gaps. Include full file contents only
when requested or useful.
