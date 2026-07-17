---
type: BigQuery Table
title: Orders Table
description: Contains one record per customer transaction.
resource: https://console.cloud.google.com/bigquery?p=bigquery-public-data&d=ga4_obfuscated_sample_ecommerce&t=orders
tags: [ecommerce, transactions]
timestamp: 2026-07-14T20:00:00Z
---

# Orders Table

The orders table stores customer transaction details including items purchased, totals, and transaction IDs.

## Schema
*   `transaction_id` (STRING): Unique identifier.
*   `user_id` (STRING): References [Users Table](users.md).
*   `value` (FLOAT64): Total value of order.
