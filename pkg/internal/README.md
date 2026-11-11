# internal

## Overview
Groups private helper packages shared internally within `okf/pkg` and withheld from external module consumers via Go's `internal` visibility boundary.

## Subdirectories & Boundaries
- `links`: [`./links/README.md`](./links/README.md) - Stateless URL scheme prefix classifier (`[links/links.go:IsExternal]`) identifying `http://`, `https://`, `mailto:`, `ftp://`, and `file://` targets as external links.
