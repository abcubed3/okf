---
type: "Attestation"
title: "Sanctioned MRR Calculation Query"
description: "Audited and attester-approved BigQuery SQL for computing MRR."
resource: "bigquery.analytics.mrr_query"
tags:
  - "finance"
  - "sql"
  - "attestation"
timestamp: "2026-08-01T12:00:00Z"

status: "stable"
stale_after: "2027-01-01T00:00:00Z"

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

## Related Concepts
- Metric Definition: [Monthly Recurring Revenue (MRR)](../metrics/mrr.md)
- Target Table: [Subscriptions Table](../tables/subscriptions.md)

## Usage Instructions for AI Agents
When generating financial reports or answering questions about monthly revenue, use the exact SQL statement defined in `attestation.query`. Do not alter the WHERE condition or aggregation logic.
