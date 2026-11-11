# skills

## Overview
Package `skills` embeds the Open Knowledge Format (`okf`) agent skill assets (`SKILL.md`, reference guides, and examples) into the Go binary via `embed.FS` and exposes helpers to read a skill's `SKILL.md` content or materialize an entire embedded skill directory onto the local filesystem.

## Interface
- `[embed.go:Skill]`: `type Skill string` — typed identifier for an embedded skill.
- `[embed.go:OKF]`: `const OKF Skill = "okf"` — identifier for the embedded Open Knowledge Format skill.
- `[embed.go:ReadSkill]`: `func ReadSkill(name Skill) ([]byte, error)` — validates `name` and returns the raw bytes of `<name>/SKILL.md` from the embedded filesystem.
- `[embed.go:CopySkill]`: `func CopySkill(name Skill, destination string) error` — validates `name` and recursively copies the embedded `<name>` directory tree (including `references` and `examples`) into `destination`.
- Embedded skill assets (`//go:embed all:okf`):
  - [okf/SKILL.md](./okf/SKILL.md) — Open Knowledge Format (OKF v0.2) agent skill definition and workflow instructions.
  - [okf/references/spec-v02.md](./okf/references/spec-v02.md), [okf/references/create-bundle.md](./okf/references/create-bundle.md), [okf/references/cli-cheatsheet.md](./okf/references/cli-cheatsheet.md) — Reference specification, bundle creation templates, and CLI cheatsheet.
  - [okf/examples/index.md](./okf/examples/index.md), [okf/examples/metrics/mrr.md](./okf/examples/metrics/mrr.md), [okf/examples/attestations/mrr_query.md](./okf/examples/attestations/mrr_query.md), [okf/examples/tables/subscriptions.md](./okf/examples/tables/subscriptions.md) — Sample OKF bundle documents.

## Invariants
- `[embed.go:validateName]`: `name` must satisfy `fs.ValidPath(name)` and must not contain `"/"` (`!fs.ValidPath(name) || strings.Contains(name, "/")`), restricting skill names to a single valid path segment before any `embed.FS` lookup in `ReadSkill` or `CopySkill`.
- `[embed.go:skillFS]`: `//go:embed all:okf` embeds the entire `okf` directory tree (including hidden or underscore-prefixed files) at compile time.
- `[embed.go:ReadSkill]`: Expects `<name>/SKILL.md` (`path.Join(string(name), "SKILL.md")`) to exist within `skillFS`.
- `[embed.go:CopySkill]`: Preserves the relative directory structure of `fs.Sub(skillFS, string(name))` under `destination` via `filepath.FromSlash`, creating directories with mode `0755` and writing files with mode `0644`.

## Side Effects
- `[embed.go:CopySkill]`: Mutates the local filesystem under `destination` by creating directories with `os.MkdirAll(..., 0755)` and creating or truncating files with `os.WriteFile(..., 0644)`.
- `[embed.go:ReadSkill]`: None (reads purely from the in-memory `embed.FS`).

## Verification
- Run: `go test ./...`
