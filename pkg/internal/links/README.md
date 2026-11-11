# links

## Overview
Package `links` (`github.com/abcubed3/okf/pkg/internal/links`) provides internal URL scheme classification shared across `pkg` packages to distinguish external URIs from bundle-relative file links.

## Interface
- `[links.go:IsExternal]` — `func IsExternal(url string) bool`: Performs case-insensitive prefix matching (`strings.ToLower`) and returns `true` if `url` starts with `http://`, `https://`, `mailto:`, `ftp://`, or `file://`.

## Invariants
- None

## Side Effects
- None

## Verification
- Run: `go test ./...`
