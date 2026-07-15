---
type: Playbook
title: Database Cleanup Playbook
description: Standard operational procedures for cleaning stale user records.
tags: [ops, cleanup]
timestamp: 2026-07-14T20:15:00Z
---

# Database Cleanup Playbook

Follow these steps to safely remove users who have not made orders:
1. Run the stale user search query.
2. Cross-reference the [Orders Table](../tables/orders.md) to ensure no orders are active.
3. Archive stale users.
