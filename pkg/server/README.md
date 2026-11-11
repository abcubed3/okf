# `pkg/server`

## Overview

`pkg/server` groups protocol server transport adapters that expose Open Knowledge Format (OKF) bundles, validation diagnostics, graph context assembly, skill compilation, and grounded Q&A to external editors, IDEs, and AI agent harnesses. It contains no direct Go source files of its own and delegates protocol handling to its `lsp` and `mcp` child packages.

## Subdirectories & Boundaries

- `lsp`: [`./lsp/README.md`](./lsp/README.md) - Language Server Protocol (LSP 3.16) stdio server (`okf-lsp`) built on `github.com/tliron/glsp` that maintains an in-memory OKF bundle (`NewServer`, `Run`), publishes real-time `textDocument/publishDiagnostics` validation notifications on document open, full-document change, and save events (reading from disk on save when `params.Text == nil`), and clears diagnostics on document close.
- `mcp`: [`./mcp/README.md`](./mcp/README.md) - Model Context Protocol (MCP) server (`okf-server`) over stdio (`Server.StartStdio`) and streamable HTTP (`Server.StartHTTP`) with `fsnotify` live bundle hot-reloading (`Server.StartWatcher`), exposing `okf://index`, `okf://concept/{+id}`, prompt `okf_concept_context`, and tools (`list_concepts`, `search_concepts`, `get_concept`, `assemble_context`, `export_agent_skill`, `ask_knowledge_base`).
