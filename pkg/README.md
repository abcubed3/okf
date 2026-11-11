# pkg

## Overview
Go library packages (`github.com/abcubed3/okf/pkg/...`) implementing the Open Knowledge Format (OKF v0.1/v0.2) domain model, Markdown/YAML bundle parser, conformance and trust-signal validator, metadata harvesters, bounded BFS context assembler, static documentation portal generator, JSON-LD and Agent Skill exporters, Gemini-backed Q&A and curation, OKF Hub client operations, knowledge sync engine, telemetry, and LSP/MCP protocol servers. This directory contains no direct Go source files and groups 15 public library packages, 1 module-internal helper directory (`internal`), and 1 protocol server grouping directory (`server`).

## Subdirectories & Boundaries
- `ai`: [./ai/README.md](./ai/README.md) - Gemini-backed concurrent schema curation (`CurateConcepts`) and graph-grounded bundle Q&A (`AskBundle`, `FindSeedConcepts`) with offline fallback.
- `assembly`: [./assembly/README.md](./assembly/README.md) - In-memory bidirectional concept link graph builder (`BuildGraph`) and token/character-bounded BFS context assembler (`AssembleContext`).
- `bundle`: [./bundle/README.md](./bundle/README.md) - Core OKF `Bundle` and `Concept` domain models, OKF v0.2 trust-tier and staleness evaluation, structural diffing (`Diff`), and strategy-based bundle merging (`Merge`).
- `config`: [./config/README.md](./config/README.md) - Local CLI credential storage (`~/.okf/config.json`) and OKF Hub API key precedence resolution (`GetAPIKey`, `LoadConfig`, `SaveConfig`).
- `export`: [./export/README.md](./export/README.md) - Schema.org JSON-LD graph serializer (`ExportBundleToJSONLD`, `GetJSONLDString`) mapping OKF concepts into `Dataset`, `WebAPI`, and `TechArticle` entities.
- `generator`: [./generator/README.md](./generator/README.md) - Static interactive HTML catalog compiler (`Generate`, `HTMLTemplate`) producing `viz.html`, JSON-LD metadata, and per-concept `concepts/<id>.json` payloads.
- `harvester`: [./harvester/README.md](./harvester/README.md) - Metadata harvesters (`Harvester`, `WriteConcepts`) for BigQuery, MySQL, PostgreSQL, Cloud Spanner, Git repositories, OpenAPI specs, Protobuf schemas, and web pages with optional Gemini classification.
- `internal`: [./internal/README.md](./internal/README.md) - Private helper boundary grouping internal utilities (`./internal/links/README.md`) withheld from external module consumers.
- `parser`: [./parser/README.md](./parser/README.md) - Concurrent OKF Markdown/YAML bundle parser (`ParseBundle`, `ParseBundleWithOptions`, `ParseConceptReader`, `ParseCitationsFromMarkdown`) and shallow Git URL resolver (`ResolvePath`).
- `publish`: [./publish/README.md](./publish/README.md) - OKF Hub bundle uploader (`PublishBundle`) posting parsed concepts and link edges to `POST <host>/api/bundles`.
- `pull`: [./pull/README.md](./pull/README.md) - OKF Hub bundle downloader (`PullBundle`) fetching `hub://` concepts from `GET <host>/api/concepts` and writing local Markdown concept files.
- `search`: [./search/README.md](./search/README.md) - OKF Hub bundle discovery client (`SearchBundles`, `SearchBundlesWithClient`) querying `GET <host>/api/bundles` by text and tag filters.
- `server`: [./server/README.md](./server/README.md) - Protocol server adapters grouping `./server/lsp/README.md` (LSP 3.16 stdio diagnostics server) and `./server/mcp/README.md` (MCP stdio and streamable HTTP server with `fsnotify` live reload).
- `skill`: [./skill/README.md](./skill/README.md) - Agent Skill compiler (`ExportBundleToSkill`) and embedded official OKF skill workspace installer (`InstallOfficialSkill`, `ResolveSkillInstallPath`).
- `sync`: [./sync/README.md](./sync/README.md) - One-shot and daemonized knowledge sync engine (`Engine`, `Run`, `StateManager`) with SHA-256 change tracking across active bidirectional `GitConnector`, Google Drive push + prototype pull (`GoogleDriveConnector`), and stub Notion/Confluence/Jira connectors.
- `telemetry`: [./telemetry/README.md](./telemetry/README.md) - Asynchronous fire-and-forget anonymous CLI event dispatcher (`SendEvent`) posting to `https://okfgo.dev/api/telemetry`.
- `validator`: [./validator/README.md](./validator/README.md) - Conformance and OKF v0.2 trust-signal linter (`ValidateBundle`, `ValidateConcept`) verifying required fields, timestamps, attestations, symlink-safe bundle path containment, and local/`hub://` links.
