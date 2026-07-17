# Performance & Benchmarks

This document captures the benchmark results for the OKF CLI's core packages, explains how to reproduce them, and provides interpretation guidance.

---

## 🏁 Running Benchmarks

Use the `task` runner:

```bash
task benchmark
```

This runs all `Benchmark*` functions across every package with memory allocation profiling enabled:

```
go test ./... -bench=. -benchmem -run=^$
```

The `-run=^$` flag skips unit tests so only benchmarks execute. Remove it to run both:

```bash
go test ./... -bench=. -benchmem
```

To target a single package or benchmark:

```bash
# Only the assembly package
go test ./pkg/assembly/... -bench=BenchmarkAssembleContext -benchmem

# Only the parser package
go test ./pkg/parser/... -bench=BenchmarkParseConceptReader -benchmem
```

---

## 📊 Benchmark Reference Results

Baseline measured on **Apple M2 Max** (`darwin/arm64`, 12 cores).

> [!NOTE]
> Go benchmark results vary by host CPU and load. Always compare against a consistent baseline on the same machine. The numbers below serve as a reference regression floor.

### `pkg/parser` — Concept Document Parser

| Benchmark | Iterations | Time/op | Memory/op | Allocs/op |
|---|---|---|---|---|
| `BenchmarkParseConceptReader` | 116,179 | **10.3 µs** | 20.9 KB | 165 |

**Scenario**: Parses a single Markdown concept document with YAML frontmatter (type, title, description, tags) and a body. Tests the full `ParseConceptReader` call path.

**Interpretation**: The parser comfortably handles ~97,000 document parses per second. The 165 allocations are dominated by YAML unmarshalling and string splitting of the frontmatter block. This is acceptable for file-system-bound I/O workloads.

---

### `pkg/assembly` — Graph-Based Context Assembler

| Benchmark | Iterations | Time/op | Memory/op | Allocs/op |
|---|---|---|---|---|
| `BenchmarkAssembleContext` | 33,189 | **38.3 µs** | 80.6 KB | 427 |

**Scenario**: BFS traversal over a 200-node graph (each concept linking to two neighbours), with `MaxDepth=3`, `MaxCharacters=16000`, bidirectional traversal, XML output format.

**Interpretation**: The assembler completes a full BFS traversal and XML serialization of a complex graph in ~38.3 µs per call. At this rate, the assembler can serve ~26,000 context requests per second for a 200-concept bundle, well within latency bounds for interactive LLM tooling.

---

### `pkg/validator` — Bundle Conformance Validator

| Benchmark | Iterations | Time/op | Memory/op | Allocs/op |
|---|---|---|---|---|
| `BenchmarkValidateBundle` | 9,643 | **125 µs** | 199.2 KB | 2,400 |

**Scenario**: Full validation pass over a 100-concept bundle, where each concept includes all frontmatter fields, a body, and an outgoing markdown link forming a circular chain. Tests YAML verification, required-field checks, recommended-field warnings, and broken-link detection.

**Interpretation**: Full validation of a 100-concept bundle completes in ~125 µs, giving ~8,000 full-bundle validations per second. The allocation count (~2,400) is driven by the link-resolution pass which performs string operations per concept per link. This is optimally fast for use as a pre-commit hook or CI check.

---

### `pkg/bundle` — Core Bundle Queries

| Benchmark | Iterations | Time/op | Memory/op | Allocs/op |
|---|---|---|---|---|
| `BenchmarkBundleListTypes` | 638,608 | **1.8 µs** | 32 B | 1 |
| `BenchmarkBundleListTags` | 238,450 | **5.1 µs** | 1.6 KB | 6 |
| `BenchmarkBundleConceptsByType` | 424,624 | **2.9 µs** | 1.07 KB | 9 |

**Scenario**: Queries metadata across a 100-concept bundle with varied tags and types. Lists unique sorted types, unique sorted tags, and queries/sorts concepts matching a specific type.

**Interpretation**: Performing in-memory lookups, deduplication, and sorting of bundle attributes takes only a few microseconds, making dynamic metadata lookups (e.g., during validation or LSP completions) extremely fast and lightweight.

---

### `pkg/export` — JSON-LD Serializer

| Benchmark | Iterations | Time/op | Memory/op | Allocs/op |
|---|---|---|---|---|
| `BenchmarkExportBundleToJSONLD` | 11,061 | **106 µs** | 66.2 KB | 356 |

**Scenario**: Serializes a 50-concept bundle containing a mix of Datasets (including parsing markdown schema tables), WebAPIs, and TechArticles into schema.org JSON-LD graph representation.

**Interpretation**: Serialization and schema extraction of a medium-sized bundle completes in just ~106 µs. This makes publishing structured metadata on-the-fly extremely fast.

---

### `pkg/generator` — Documentation Portal Compiler

| Benchmark | Iterations | Time/op | Memory/op | Allocs/op |
|---|---|---|---|---|
| `BenchmarkGenerate` | 772 | **1.6 ms** | 483 KB | 2,202 |

**Scenario**: Compiles a static HTML documentation portal with full-text search index, metadata graphs, and templates from a multi-concept knowledge bundle directory on disk to a temporary output folder.

**Interpretation**: Compiling and writing a documentation site with full assets from files on disk completes in under ~1.6 ms. This provides instantaneous rebuilds during interactive editing or build-step tasks.

---

## 🧪 Benchmark Descriptions

| Benchmark | File | What It Measures |
|---|---|---|
| `BenchmarkParseConceptReader` | [`pkg/parser/parser_test.go`](pkg/parser/parser_test.go) | YAML frontmatter parsing + Markdown body splitting |
| `BenchmarkAssembleContext` | [`pkg/assembly/assembler_test.go`](pkg/assembly/assembler_test.go) | BFS graph traversal, character-budget pruning, XML serialization |
| `BenchmarkValidateBundle` | [`pkg/validator/validator_test.go`](pkg/validator/validator_test.go) | Full conformance pass: type checks, field warnings, broken link detection |
| `BenchmarkBundleListTypes` | [`pkg/bundle/bundle_test.go`](pkg/bundle/bundle_test.go) | Resolving and sorting unique concept types across the bundle |
| `BenchmarkBundleListTags` | [`pkg/bundle/bundle_test.go`](pkg/bundle/bundle_test.go) | Resolving, deduplicating, and sorting unique tags across the bundle |
| `BenchmarkBundleConceptsByType` | [`pkg/bundle/bundle_test.go`](pkg/bundle/bundle_test.go) | Querying and sorting concepts matching a specific type |
| `BenchmarkExportBundleToJSONLD` | [`pkg/export/jsonld_test.go`](pkg/export/jsonld_test.go) | Schema extraction from markdown bodies and JSON-LD serialization |
| `BenchmarkGenerate` | [`pkg/generator/generator_test.go`](pkg/generator/generator_test.go) | Full static HTML documentation portal generation and writing to disk |

---

## 📈 Tracking Regressions

Use the [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) tool to compare two benchmark runs statistically.

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Capture a baseline
go test ./... -bench=. -benchmem -run=^$ -count=10 | tee baseline.txt

# Make your changes, then capture a new run
go test ./... -bench=. -benchmem -run=^$ -count=10 | tee after.txt

# Compare with statistical significance
benchstat baseline.txt after.txt
```

A `+` indicates a regression; a `-` indicates an improvement. Changes smaller than the noise threshold (~2–5%) are expected to show `(p=N/A)`.

> [!TIP]
> Run with `-count=10` or more iterations to get statistically meaningful results via `benchstat`.

---

## 🔍 Profiling

To generate a CPU or memory profile for deep performance analysis:

```bash
# CPU profile for assembly package
go test ./pkg/assembly/... -bench=BenchmarkAssembleContext -benchmem -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof

# Memory profile for validator
go test ./pkg/validator/... -bench=BenchmarkValidateBundle -benchmem -memprofile=mem.prof
go tool pprof -http=:8080 mem.prof
```
