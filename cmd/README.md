# cmd

## Overview
Command-line binary entrypoints for the Open Knowledge Format (`okf`) module. This directory contains no direct source files and groups executable `main` packages built from this repository.

## Subdirectories & Boundaries
- `okf`: [./okf/README.md](./okf/README.md) - Binary entrypoint package (`package main`) that defines build-time version metadata (`Version`, `Commit`, `Date`), delegates argument execution to `github.com/abcubed3/okf/internal/cli`, and handles top-level error reporting and process exit codes.
