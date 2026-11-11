# Package `assembly` (`pkg/assembly`)

## Overview

Package `assembly` (`github.com/abcubed3/okf/pkg/assembly`) constructs an in-memory directed relationship graph over an Open Knowledge Format `bundle.Bundle` (`github.com/abcubed3/okf/pkg/bundle`) and assembles budget-constrained context payloads for LLMs and AI agents. `BuildGraph` parses Markdown concept bodies into Goldmark ASTs to resolve internal concept-to-concept links into outbound and inbound adjacency lists (`ConceptGraph`), while `AssembleContext` performs breadth-first traversal from a starting concept within configurable link-depth, character, and heuristic token budgets, serializing visited concepts as XML or Markdown (`AssemblyResult`).

## Interface

- `[assembler.go:Direction]` — String enum type specifying graph traversal direction.
- `[assembler.go:DirectionOutbound]` — Constant (`"outbound"`) following outgoing links from the start concept.
- `[assembler.go:DirectionInbound]` — Constant (`"inbound"`) following incoming links referencing the start concept.
- `[assembler.go:DirectionBidirectional]` — Constant (`"bidirectional"`) following both outbound and inbound links.
- `[assembler.go:AssemblyOptions]` — Configuration struct controlling traversal and serialization (`MaxDepth`, `MaxTokens`, `MaxCharacters`, `Direction`, `Format`).
- `[assembler.go:DefaultOptions]` — `func DefaultOptions() AssemblyOptions` returning defaults (`MaxDepth: 2`, `MaxTokens: 4000`, `MaxCharacters: 16000`, `Direction: DirectionBidirectional`, `Format: "xml"`).
- `[assembler.go:NodeStat]` — JSON-serializable struct recording `ID`, `Tokens`, and `Characters` for each included concept node.
- `[assembler.go:AssemblyResult]` — JSON-serializable struct containing `Context`, `TotalTokens`, `TotalCharacters`, and `Nodes` (`[]NodeStat`).
- `[assembler.go:AssembleContext]` — `func AssembleContext(g *ConceptGraph, startID string, opts AssemblyOptions) (*AssemblyResult, error)` executing bounded BFS context assembly from `startID`.
- `[graph.go:ConceptNode]` — Graph vertex holding `*bundle.Concept`, `OutLinks []string`, and `InLinks []string`.
- `[graph.go:ConceptGraph]` — Directed concept graph mapping concept IDs to `*ConceptNode` (`Nodes map[string]*ConceptNode`).
- `[graph.go:BuildGraph]` — `func BuildGraph(b *bundle.Bundle) *ConceptGraph` extracting internal Markdown links across a bundle into a `ConceptGraph`.

## Invariants

- `[assembler.go:AssembleContext]` — Returns an error if `startID` does not exist in `g.Nodes`.
- `[assembler.go:AssembleContext]` — Traverses `g` in BFS order using a `visited` set so each concept appears at most once, enqueuing neighbors only while `item.depth < opts.MaxDepth`.
- `[assembler.go:AssembleContext]` — Estimates tokens heuristically as `len(conceptContent) / 4` and terminates BFS immediately (breaking out of traversal rather than skipping to smaller nodes) when `opts.MaxCharacters > 0 && charCount+conceptLen > opts.MaxCharacters` or `opts.MaxTokens > 0 && tokenCount+conceptTokens > opts.MaxTokens`.
- `[assembler.go:AssembleContext]` — Defaults unrecognized `opts.Direction` values to `DirectionOutbound`; any `opts.Format` other than case-insensitive `"markdown"` renders XML wrapped in `<context>\n...\n</context>\n` (outer wrapper tags are excluded from `TotalCharacters` and `TotalTokens`).
- `[assembler.go:formatSingleConcept]` — XML-escapes `Type`, `Title`, `Desc`, and `Body` via `xml.EscapeText` (quoting `id` with `%q`), whereas YAML-marshaled `Frontmatter` inside `<metadata>` is unescaped and inline-formats any YAML marshal error instead of failing.
- `[graph.go:BuildGraph]` — Allocates a `ConceptNode` for every entry in `b.Concepts`, skipping AST link extraction for concepts with non-empty `c.ParseError`.
- `[graph.go:BuildGraph]` — Walks Goldmark AST `*ast.Link` nodes (ignoring links inside code blocks), skips external URLs (`links.IsExternal`), pure `#` fragments, paths escaping bundle root (`..`), and targets absent from `b.GetConcept` after stripping `#fragment` and `.md`; deduplicates `OutLinks` per source node and populates `InLinks` as the exact inverse of `OutLinks`.

## Side Effects

- None

## Verification

- Run: `go test ./...`
