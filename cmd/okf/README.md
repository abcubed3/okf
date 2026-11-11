# okf (`cmd/okf`)

## Overview
Binary entrypoint package (`package main`) for the Open Knowledge Format (`okf`) command-line interface. It defines build-time metadata variables (`Version`, `Commit`, `Date`) and delegates CLI argument execution to `github.com/abcubed3/okf/internal/cli`.

## Interface
- `[main.go:Version]` — Package-level string variable (`default: "dev"`) holding the CLI version injected at build time.
- `[main.go:Commit]` — Package-level string variable (`default: "none"`) holding the Git commit hash injected at build time.
- `[main.go:Date]` — Package-level string variable (`default: "unknown"`) holding the build timestamp injected at build time.
- `[main.go:main]` — Binary entrypoint `func main()` that passes `os.Args[1:]`, `Version`, `Commit`, and `Date` to `cli.Execute`.

## Invariants
- None

## Side Effects
- `[main.go:main]` — Reads process arguments from `os.Args[1:]` and invokes `cli.Execute(os.Args[1:], Version, Commit, Date)`.
- `[main.go:main]` — Writes formatted error output (`"Error: %v\n"`) to `os.Stderr` via `fmt.Fprintf` when `cli.Execute` returns an error.
- `[main.go:main]` — Terminates the OS process with exit status `1` via `os.Exit(1)` when `cli.Execute` returns an error.

## Verification
- Run: `go test ./...`
