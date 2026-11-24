---
type: "Attested Computation"
title: "Sanctioned MRR Calculation Query"
description: "Audited and attester-approved BigQuery SQL for computing MRR."
resource: "bigquery.analytics.mrr_query"
tags:
  - "finance"
  - "sql"
  - "attestation"
status: "stable"
runtime: "bigquery"
parameters:
  - name: "as_of_month"
    type: "date"
    required: false
stale_after: "2027-01-01T00:00:00Z"

sources:
  - id: "rev-policy"
    resource: "https://stripe.com/docs/billing/subscriptions"
    title: "Revenue Recognition & Subscription Policy"
    author: "team:finance-data"

generated:
  by: "okf-ai-curator"
  at: "2026-08-01T12:00:00Z"

verified:
  - by: "human:compliance-officer@company.com"
    at: "2026-08-01T12:20:00Z"
---

# Computation

    SELECT
      DATE_TRUNC(created_at, MONTH) AS revenue_month,
      SUM(amount) AS total_mrr
    FROM `my-project.analytics.subscriptions`
    WHERE status = 'active'
    GROUP BY 1

This concept defines the sanctioned computation logic for financial reporting.[^rev-policy]

## Related Concepts
- Metric Definition: [Monthly Recurring Revenue (MRR)](/metrics/mrr.md)
- Target Table: [Subscriptions Table](/tables/subscriptions.md)

## Usage Instructions for AI Agents
When generating financial reports or answering questions about monthly revenue, use the exact SQL statement under `# Computation`. Do not alter the `WHERE` condition or aggregation logic.

[^rev-policy]: Revenue Recognition & Subscription Policy
