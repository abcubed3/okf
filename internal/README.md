# `internal`

## Overview

`internal` groups private Go packages scoped exclusively to the `github.com/abcubed3/okf` module that are not exported for external library consumption. Currently, it houses the command-line interface dispatcher and subcommand handlers under `./cli`.

## Subdirectories & Boundaries

- `cli`: [`./cli/README.md`](./cli/README.md) - Implements top-level `okf` CLI subcommand routing (`Execute`), flag parsing, and command execution handlers (`RunLint`, `RunHarvest`, `RunAssemble`, `RunServer`, `RunLsp`, `RunDoc`, `RunSync`, `RunExport`, `RunDiff`, `RunMerge`, `RunPublish`, `RunPull`, `RunPush`, `RunSearch`, `RunSkill`, `RunAsk`, `RunAuth`, `RunAuthLogin`) delegating to `pkg/*` domain services.

## Verification

- Run: `go test ./...`
