# parser

## Overview
Package `parser` (`github.com/abcubed3/okf/pkg/parser`) resolves local directory paths or shallow-clones remote Git repository URLs (`github.com/go-git/go-git/v5`) and parses Open Knowledge Format (OKF) Markdown documents—extracting `---`-fenced YAML frontmatter (`gopkg.in/yaml.v3`), Markdown bodies, and `# Citations` sections—into [`../bundle/README.md`](../bundle/README.md) domain models.

## Interface
- `[parser.go:DefaultMaxFileSize]` — `const DefaultMaxFileSize = 10 * 1024 * 1024`: Default 10 MB per-file size limit in bytes.
- `[parser.go:ParseOptions]` — `type ParseOptions struct { MaxFileSize int64 }`: Parser configuration options.
- `[parser.go:DefaultParseOptions]` — `func DefaultParseOptions() ParseOptions`: Returns `ParseOptions` populated with `DefaultMaxFileSize`.
- `[parser.go:ParseBundle]` — `func ParseBundle(ctx context.Context, rootPath string) (*bundle.Bundle, error)`: Traverses `rootPath` and parses an OKF bundle using `DefaultParseOptions()`.
- `[parser.go:ParseBundleWithOptions]` — `func ParseBundleWithOptions(ctx context.Context, rootPath string, opts ParseOptions) (*bundle.Bundle, error)`: Walks `rootPath` and parses candidate `.md` concept files concurrently.
- `[parser.go:ParseConceptFile]` — `func ParseConceptFile(fullPath, relPath, conceptID string) (*bundle.Concept, error)`: Opens `fullPath` and delegates to `ParseConceptReader`.
- `[parser.go:ParseConceptReader]` — `func ParseConceptReader(r io.Reader, relPath, conceptID string) (*bundle.Concept, error)`: Splits `---`-fenced YAML frontmatter from the Markdown body and extracts citations.
- `[parser.go:ParseCitationsFromMarkdown]` — `func ParseCitationsFromMarkdown(body string) []bundle.Citation`: Extracts numbered citation entries beneath `# Citations` headings.
- `[git_resolver.go:IsGitURL]` — `func IsGitURL(path string) bool`: Reports whether `path` matches a Git remote URL prefix or `.git` suffix.
- `[git_resolver.go:ResolvePath]` — `func ResolvePath(path string) (string, func(), error)`: Resolves a local path or shallow-clones a remote Git URL to a temporary directory, returning the local directory path and a cleanup closure.

## Invariants
- `[parser.go:ParseBundleWithOptions]` falls back to `DefaultMaxFileSize` (10 MB) when `opts.MaxFileSize <= 0`, and returns an error if `rootPath` cannot be resolved via `filepath.Abs` or is not an existing directory (`os.Stat` / `IsDir`).
- `[parser.go:ParseBundleWithOptions]` inspects only `.md` files, skips `log.md`, and excludes `index.md` files from `Bundle.Concepts`—parsing `index.md` files synchronously during `filepath.Walk` (without checking `opts.MaxFileSize` or `ctx.Done()`) and recording successfully parsed `index.md` files in `Bundle.IndexFiles` (with root `index.md` populating `Bundle.OKFVersion` and `index.md` open/parse errors silently ignored).
- `[parser.go:ParseBundleWithOptions]` derives each `Concept.ID` as the slash-normalized path relative to `absRoot` with `.md` stripped, caps worker goroutines at `min(runtime.NumCPU(), len(jobs))`, records per-file size violations (`> opts.MaxFileSize`) and `ParseConceptFile` errors on `Concept.ParseError` inside `Bundle.Concepts` rather than aborting bundle parsing, and stops workers early on `ctx.Done()` without returning `ctx.Err()` (yielding a partial `Bundle.Concepts` map).
- `[parser.go:ParseConceptReader]` parses YAML frontmatter via `gopkg.in/yaml.v3` only when trimmed line 1 equals `"---"` up to the next trimmed `"---"` line via `bufio.Scanner` (default 64 KB max token size), stripping leading blank lines immediately after the closing `"---"` fence (or unmarshaling all lines until EOF as frontmatter with an empty `Body` if no closing `"---"` line is encountered), and returning both a partial `*bundle.Concept` (`ID`, `Path`, `Body`, `Citations`) and a wrapped error when YAML unmarshaling fails.
- `[parser.go:ParseCitationsFromMarkdown]` extracts citations only under Markdown `#` headings (any level) whose trimmed title case-insensitively equals `"citations"` until the next `#` heading, matching `^\[(\d+)\]\s+(?:\[([^\]]+)\]\(([^)]+)\)|(\S+))` and defaulting `Title` to `URI` for bare URIs.
- `[git_resolver.go:IsGitURL]` returns `true` iff lowercased `path` starts with `"http://"`, `"https://"`, `"git@"`, `"ssh://"`, or `"git://"`, or ends with `".git"`.
- `[git_resolver.go:ResolvePath]` verifies non-Git paths via `os.Stat` (returning a no-op cleanup closure) and automatically removes its temporary directory if `git.PlainClone` fails.

## Side Effects
- `[parser.go:ParseBundleWithOptions]` walks and reads local filesystem directories and `.md` files (`filepath.Abs`, `os.Stat`, `filepath.Walk`, `os.Open`) and spawns up to `runtime.NumCPU()` worker goroutines plus a `sync.WaitGroup` closer goroutine.
- `[parser.go:ParseConceptFile]` opens and reads `fullPath` from the local filesystem via `os.Open`.
- `[git_resolver.go:ResolvePath]` stats local paths (`os.Stat`) and, for Git URLs, creates an `okf-git-*` directory in the OS temp directory (`os.MkdirTemp`), performs shallow remote network/Git clone I/O (`git.PlainClone` with `Depth: 1`), and deletes the temp directory via `os.RemoveAll` on clone error or when the caller invokes the returned cleanup function.

## Verification
- Run: `go test ./...`
