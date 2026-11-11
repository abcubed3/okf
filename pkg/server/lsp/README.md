# Package `lsp` (`pkg/server/lsp`)

## Overview

Package `lsp` (`github.com/abcubed3/okf/pkg/server/lsp`) implements the Language Server Protocol (LSP 3.16) stdio server (`okf-lsp`) for Open Knowledge Format workspaces, built on `github.com/tliron/glsp`. Upon workspace initialization, it parses the workspace OKF bundle via `github.com/abcubed3/okf/pkg/parser`, updates in-memory `bundle.Concept` entries as Markdown documents are opened, edited, or saved, validates modified concepts against the active bundle via `github.com/abcubed3/okf/pkg/validator`, and publishes real-time `textDocument/publishDiagnostics` notifications to connected editors.

## Interface

- `[server.go:Server]` — Struct holding per-server LSP state (`sync.RWMutex`, `workspaceRoot string`, and `activeBundle *bundle.Bundle`).
- `[server.go:NewServer]` — `func NewServer() *Server` allocating an empty LSP `Server` instance.
- `[server.go:Run]` — `func Run()` binding LSP 3.16 lifecycle and `textDocument` handlers (`initialize`, `initialized`, `shutdown`, `$/setTrace`, `didOpen`, `didChange`, `didSave`, `didClose`) and serving JSON-RPC over standard I/O via `glsp`.

## Invariants

- `[server.go:Server]` — Guards `workspaceRoot` and `activeBundle` map reads/writes with `Server.mu` (`sync.RWMutex`), holding `Server.mu` in `parseAndValidate` across `activeBundle.Concepts[conceptID]` assignment and `validator.ValidateConcept` execution before publishing diagnostics.
- `[server.go:initialize]` — Advertises `protocol.TextDocumentSyncKindFull` synchronization and server metadata `{Name: "okf-lsp", Version: "0.1.0"}`; prefers `params.RootURI` over `params.RootPath` when determining `workspaceRoot`.
- `[server.go:textDocumentDidChange]` — Requires full-document change events (`protocol.TextDocumentContentChangeEventWhole`); returns `nil` without re-validating if `ContentChanges` is empty or incremental.
- `[server.go:textDocumentDidSave]` — Validates `*params.Text` when provided or falls back to reading `uriToPath(params.TextDocument.URI)` from disk via `os.ReadFile` when `params.Text == nil`.
- `[server.go:parseAndValidate]` — No-ops when `workspaceRoot` is empty or `filepath.Rel` fails; computes `conceptID` by normalizing the workspace-relative path to forward slashes and stripping the `.md` suffix.
- `[server.go:parseAndValidate]` — Emits diagnostics with source `"okf"` anchored at line `0` (`Character: 0..100`), mapping `validator.SeverityWarning` to `DiagnosticSeverityWarning` and all other severities to `DiagnosticSeverityError`, always publishing a non-nil `[]protocol.Diagnostic` slice.

## Side Effects

- `[server.go:Run]` — Performs blocking JSON-RPC communication (with `debug=true` logging) over process `os.Stdin`/`os.Stdout` via `srv.RunStdio()`.
- `[server.go:initialized]` — Reads and parses workspace files from disk via `parser.ParseBundle(context.Background(), s.workspaceRoot)` to populate `s.activeBundle`.
- `[server.go:shutdown]` — Mutates global GLSP trace state via `protocol.SetTraceValue(protocol.TraceValueOff)`.
- `[server.go:setTrace]` — Mutates global GLSP trace state via `protocol.SetTraceValue(params.Value)`.
- `[server.go:parseAndValidate]` — Lazily initializes `s.activeBundle` if `nil`, mutates `s.activeBundle.Concepts[conceptID]` in memory, and sends `textDocument/publishDiagnostics` notifications over the LSP connection via `context.Notify`.
- `[server.go:textDocumentDidClose]` — Sends an empty `textDocument/publishDiagnostics` notification via `context.Notify` to clear diagnostics for closed documents.

## Verification

- Run: `go test ./...`
