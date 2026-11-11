# Package `skill` (`pkg/skill`)

## Overview

Package `skill` (`github.com/abcubed3/okf/pkg/skill`) compiles Open Knowledge Format (`github.com/abcubed3/okf/pkg/bundle`) knowledge bundles into AI Agent Skill documents (`SKILL.md`) and installs the embedded official OKF Agent Skill (`github.com/abcubed3/okf/skills`) into agent-specific workspace directories.

## Interface

- `[skill.go:ExportOptions]` — Configuration struct (`Name`, `Description`, `Tags`, `Types`, `IncludeAttestations`) controlling concept filtering and `SKILL.md` generation.
- `[skill.go:SkillFrontmatter]` — ``type SkillFrontmatter struct { Name string `yaml:"name"`; Description string `yaml:"description"` }`` serialized as the YAML frontmatter block of generated `SKILL.md` files.
- `[skill.go:ExportBundleToSkill]` — `func ExportBundleToSkill(b *bundle.Bundle, opts ExportOptions) (string, error)` filtering, deterministically sorting, and rendering a `*bundle.Bundle` into a complete `SKILL.md` string.
- `[skill.go:InstallOptions]` — Configuration struct (`TargetDir`, `Platform`) specifying destination directory or target agent harness for skill installation.
- `[skill.go:ResolveSkillInstallPath]` — `func ResolveSkillInstallPath(opts InstallOptions) string` mapping `InstallOptions` to the target workspace skill directory path.
- `[skill.go:InstallOfficialSkill]` — `func InstallOfficialSkill(opts InstallOptions) (string, error)` resolving the absolute destination path and extracting embedded `skills.OKF` assets onto disk.

## Invariants

- `[skill.go:ExportBundleToSkill]` — Rejects a `nil` `*bundle.Bundle` (`"bundle cannot be nil"`) and returns an error when zero concepts match the configured `Tags`/`Types` filters.
- `[skill.go:ExportBundleToSkill]` — Defaults blank `Name` to `"okf-domain-knowledge"`, normalizes `Name` via `strings.TrimSpace`, space-to-hyphen replacement, and `strings.ToLower`, and populates a default `Description` from the normalized name when empty.
- `[skill.go:filterConcepts]` — Evaluates `Tags` and `Types` case-insensitively using logical OR within `Tags`, logical OR within `Types`, and logical AND across non-empty `Tags` and `Types`.
- `[skill.go:ExportBundleToSkill]` — Sorts matched concept IDs and grouped concept type headers lexicographically; defaults empty `Frontmatter.Type` to `"General"` and empty `Frontmatter.Title` to `Concept.ID`.
- `[skill.go:ExportBundleToSkill]` — Classifies a concept as an attestation if `Frontmatter.Type` equals `"Attestation"` or `"Attested Computation"` (case-insensitive) or `Concept.SanctionedQuery()` is non-empty, rendering the audited SQL section (including OKF v0.2 `Runtime`, `Executor.Resource`, and `Attester.Resource` metadata) only when `opts.IncludeAttestations` is `true` and at least one attestation matched.
- `[skill.go:ResolveSkillInstallPath]` — Non-empty `TargetDir` takes precedence over `Platform`; otherwise lowercase trimmed `Platform` maps `"claude"`/`"claude-code"` to `.claude/skills/okf`, `"cursor"` to `.cursor/skills/okf`, and `"antigravity"`/`"agy"`/`"agents"`/`""` (or any unrecognized value) to `.agents/skills/okf`.

## Side Effects

- `[skill.go:ExportBundleToSkill]` — Reads the system clock via `time.Now().UTC()` to embed an `RFC3339` generation timestamp in the Markdown footer.
- `[skill.go:InstallOfficialSkill]` — Resolves relative paths against the process working directory via `filepath.Abs` and writes embedded `skills.OKF` files to the local filesystem via `skills.CopySkill`.

## Verification

- Run: `go test ./...`
