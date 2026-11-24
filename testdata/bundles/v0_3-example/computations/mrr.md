---
type: "Attested Computation"
title: "Sanctioned MRR Computation"
description: "Audited BigQuery SQL computation for active subscription MRR."
status: "stable"
runtime: "bigquery"
parameters:
  - name: "as_of_date"
    type: "date"
    required: true
executor:
  resource: "/references/run-bq.md"
  receipt: ["job_id", "executed_sql", "result"]
attester:
  resource: "/references/check-sql.py"
generated:
  by: "okf-ai-curator/v0.3"
  at: "2026-11-18T12:00:00Z"
verified:
  by: "human:head-of-finance@company.com"
  at: "2026-11-18T12:20:00Z"
---

# Computation

```sql
SELECT
  DATE_TRUNC(created_at, MONTH) AS revenue_month,
  SUM(amount) AS total_mrr
FROM `my-project.analytics.subscriptions`
WHERE status = 'active' AND DATE(created_at) <= @as_of_date
GROUP BY 1
```
