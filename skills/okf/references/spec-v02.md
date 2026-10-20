# OKF Specification v0.2 Quick Reference

## Core Frontmatter Schema
```yaml
---
# Mandatory
type: string

# Recommended Metadata
title: string
description: string
resource: string
tags: list[string]
timestamp: RFC3339 string

# Trust Layer (§5)
status: "stable" | "draft" | "deprecated"
stale_after: RFC3339 string

sources:
  - resource: string
    title: string
    author: string
    usage_count: integer
    last_modified: RFC3339 string

generated:
  by: string
  at: RFC3339 string

verified:
  - by: string
    at: RFC3339 string

# Attestation Computation (§6)
attestation:
  query: string
  executor: string
  attester: string
---
```

## Actor Convention (§7)
- `human:<identifier>` indicates human review. Maps to Trust Tier: `human-reviewed`.
- Any other string (e.g., `agent:curator-v1`, `process:linter`) maps to Trust Tier: `machine-confirmed`.
- Empty or omitted `by` field maps to `unverified`.
