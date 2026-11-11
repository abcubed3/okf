# Package `mcp` (`pkg/server/mcp`)

## Overview

Package `mcp` (`github.com/abcubed3/okf/pkg/server/mcp`) implements the Model Context Protocol (MCP) server adapter (`okf-server` v1.0.0) for Open Knowledge Format bundles using `github.com/modelcontextprotocol/go-sdk/mcp`. It loads an OKF bundle from disk (`pkg/parser`), builds an in-memory relationship graph (`pkg/assembly`) and lowercase search index, watches non-hidden bundle directories via `fsnotify` for live hot-reloads on `.md` file changes, and serves MCP resources, prompts, and tools over either standard I/O (`StartStdio`) or streamable HTTP (`StartHTTP`).

## Interface

- `[mcp.go:Server]` — MCP server struct wrapping `*sdk.Server`, `bundlePath`, optional HTTP bearer/query `Token string`, and `sync.RWMutex`-guarded `*bundle.Bundle`, `*assembly.ConceptGraph`, and `searchTexts` index.
- `[mcp.go:NewMCPServer]` — `func NewMCPServer(bundlePath string) (*Server, error)` resolving and parsing `bundlePath`, building the concept graph and search index, and registering MCP capabilities:
  - Resource `okf://index` and resource template `okf://concept/{+id}`
  - Prompt `okf_concept_context` (arguments: `id`, optional `depth`)
  - Tools: `list_concepts`, `search_concepts`, `get_concept`, `assemble_context`, `export_agent_skill`, and `ask_knowledge_base`
- `[mcp.go:Server.StartStdio]` — `func (s *Server) StartStdio(ctx context.Context) error` starting the `fsnotify` watcher and serving MCP over `sdk.StdioTransport`.
- `[mcp.go:Server.StartHTTP]` — `func (s *Server) StartHTTP(addr string) error` starting the `fsnotify` watcher and serving streamable HTTP (`sdk.NewStreamableHTTPHandler`) with localhost `Origin` validation and optional `Token` authentication.
- `[mcp.go:Server.StartWatcher]` — `func (s *Server) StartWatcher(ctx context.Context) error` watching non-hidden subdirectories under `bundlePath` and hot-reloading bundle state when `.md` files are created, modified, or removed.
- `[mcp.go:ListConceptsArgs]` / `[mcp.go:ListConceptsResult]` — Input/output structs for the `list_concepts` MCP tool.
- `[mcp.go:ConceptSummary]` — Summary DTO (`ID`, `Type`, `Title`, `Description`, `Tags`) returned by `list_concepts` and `search_concepts`.
- `[mcp.go:SearchConceptsArgs]` / `[mcp.go:SearchConceptsResult]` — Input (`Query`, `Type`, `Tag`) and output structs for the `search_concepts` MCP tool.
- `[mcp.go:GetConceptArgs]` / `[mcp.go:GetConceptResult]` — Input (`ID`) and output (`ID`, `Path`, `Frontmatter`, `Body`) structs for the `get_concept` MCP tool.
- `[mcp.go:AssembleContextArgs]` / `[mcp.go:AssembleContextResult]` — Input (`ID`, `Depth`, `Direction`, `MaxChars`, `Format`) and output (`Context`, `TotalTokens`, `TotalCharacters`, `Nodes`) structs for the `assemble_context` MCP tool.
- `[mcp.go:ExportAgentSkillArgs]` / `[mcp.go:ExportAgentSkillResult]` — Input (`Name`, `Description`, `Tag`, `Type`, `IncludeAttestations`) and output (`Name`, `SkillMarkdown`) structs for the `export_agent_skill` MCP tool.
- `[mcp.go:AskKnowledgeBaseArgs]` / `[mcp.go:AskKnowledgeBaseResult]` — Input (`Query`, `SeedConcept`, `Depth`) and output (`Query`, `Answer`, `SeedConcept`, `ReferencedConcepts`, `Attestations`) structs for the `ask_knowledge_base` MCP tool.

## Invariants

- `[mcp.go:Server]` — Guards concurrent reads of `s.bundle`, `s.graph`, and `s.searchTexts` across all resource, prompt, and tool handlers with `s.mu.RLock()`, and exclusive hot-reload swaps in `reloadBundle` with `s.mu.Lock()`.
- `[mcp.go:Server.reloadBundle]` — Preserves the last valid in-memory `s.bundle`, `s.graph`, and `s.searchTexts` unchanged if `parser.ParseBundle` returns an error during hot-reload.
- `[mcp.go:Server.StartHTTP]` — Rejects HTTP requests carrying a non-empty `Origin` header with `403 Forbidden` unless `Origin` begins with `http://localhost`, `http://127.0.0.1`, `https://localhost`, or `https://127.0.0.1`.
- `[mcp.go:Server.StartHTTP]` — When `s.Token` is non-empty, requires non-`OPTIONS` requests to supply a matching `?token=` query parameter or `Authorization: Bearer <token>` header, returning `401 Unauthorized` otherwise.
- `[mcp.go:Server.StartWatcher]` — Skips hidden directories (prefixed with `.` other than `.`) during `filepath.Walk` registration and triggers `reloadBundle` exclusively on `Write`, `Create`, or `Remove` events for `.md` files.
- `[mcp.go:NewMCPServer]` — Prompt `okf_concept_context` requires a non-empty `"id"` argument (defaulting `depth` to `2`); resource `okf://concept/{+id}` and `get_concept` return an error if the requested concept ID is absent from `s.bundle`.
- `[mcp.go:Server.handleSearchConcepts]` — Applies case-insensitive exact matching on non-empty `Type` and `Tag` filters and requires all lowercase alphanumeric tokens from `Query` to match as substrings of `s.searchTexts[id]`.
- `[mcp.go:Server.handleAssembleContext]` — Derives `opts.MaxTokens = *args.MaxChars / 4` whenever `MaxChars` is provided.
- `[mcp.go:Server.handleExportAgentSkill]` — Defaults skill `Name` to `"okf-domain-knowledge"` when nil/empty and `IncludeAttestations` to `true` when nil.
- `[mcp.go:Server.handleAskKnowledgeBase]` — Defaults traversal `Depth` to `2` when `args.Depth` is nil.

## Side Effects

- `[mcp.go:NewMCPServer]` — Resolves `bundlePath` via `filepath.Abs` and reads/parses bundle files from disk via `parser.ParseBundle`.
- `[mcp.go:Server.StartWatcher]` — Walks `bundlePath` on disk, registers OS `fsnotify` descriptors, spawns a background watcher goroutine that mutates in-memory server state via `reloadBundle`, and writes status/error logs to `os.Stderr`.
- `[mcp.go:Server.reloadBundle]` — Re-reads and parses the OKF bundle directory from disk and logs reload outcomes to `os.Stderr`.
- `[mcp.go:Server.StartStdio]` — Starts the filesystem watcher, logs warnings to `os.Stderr`, and performs blocking JSON-RPC I/O over standard I/O (`sdk.StdioTransport`).
- `[mcp.go:Server.StartHTTP]` — Starts the filesystem watcher, logs warnings to `os.Stderr`, and binds a TCP listener at `addr` via `http.Server.ListenAndServe`.
- `[mcp.go:Server.handleAskKnowledgeBase]` — Invokes `ai.AskBundle`, issuing external Google Gemini API network requests when an API key is configured.

## Verification

- Run: `go test ./...`
