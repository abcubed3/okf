---
type: BigQuery Table
title: Users Table
description: Store customer demographics and login details.
resource: https://console.cloud.google.com/bigquery?p=bigquery-public-data&d=ga4_obfuscated_sample_ecommerce&t=users
tags: [ecommerce, users]
timestamp: 2026-07-14T20:00:00Z
---

# Users Table

Demographic information and tracking IDs for store customers.

## Key Relationships
*   Referenced by [Orders Table](orders.md) via `user_id`.
See also remote link: [Stripe Charge](hub://stripe/api/charge) and [Fake](hub://stripe/api/fake)
