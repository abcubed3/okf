---
type: "BigQuery Table"
title: "Subscriptions Table"
description: "Core subscription contracts and billing status events."
resource: "my-project.analytics.subscriptions"
tags:
  - "billing"
  - "schema"
timestamp: "2026-08-01T12:00:00Z"

status: "stable"
stale_after: "2027-01-01T00:00:00Z"

generated:
  by: "okf-harvester/v0.2"
  at: "2026-08-01T12:00:00Z"

verified:
  - by: "human:lead-dba@company.com"
    at: "2026-08-01T12:15:00Z"
---

# Subscriptions Table

`my-project.analytics.subscriptions` stores customer billing lifecycle records.

## Schema Definition

| Column | Type | Nullable | Description |
| :--- | :--- | :--- | :--- |
| `subscription_id` | STRING | No | Unique subscription identifier. |
| `customer_id` | STRING | No | Foreign key linking to customer account. |
| `status` | STRING | No | Current lifecycle status: `active`, `canceled`, `past_due`, `trial`. |
| `amount` | NUMERIC | No | Normalized monthly contract value in USD. |
| `created_at` | TIMESTAMP | No | Record creation timestamp. |
| `updated_at` | TIMESTAMP | Yes | Last status update timestamp. |

## Related Concepts
- Metric: [Monthly Recurring Revenue (MRR)](../metrics/mrr.md)
- Attested Computation: [Sanctioned MRR Calculation Query](../attestations/mrr_query.md)
