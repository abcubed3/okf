# `cli` Component Package

## Overview

Package `cli` (`github.com/abcubed3/okf/internal/cli`) implements the command-line interface routing, flag parsing, and subcommand execution handlers for the `okf` binary. It translates CLI arguments and flags into calls across internal domain packages (`pkg/parser`, `pkg/validator`, `pkg/harvester`, `pkg/assembly`, `pkg/ai`, `pkg/bundle`, `pkg/generator`, `pkg/export`, `pkg/publish`, `pkg/pull`, `pkg/search`, `pkg/server/mcp`, `pkg/server/lsp`, `pkg/skill`, `pkg/sync`, `pkg/config`, and `pkg/telemetry`).

## Interface

- `[root.go:Version]`, `[root.go:Commit]`, `[root.go:Date]` — Exported build metadata string variables (defaults: `"dev"`, `"none"`, `"unknown"`).
- `[root.go:Execute]` — `func Execute(args []string, version, commit, date string) error`: Entry point dispatching top-level subcommands `lint`, `harvest`, `assemble`, `server` / `mcp`, `lsp`, `doc`, `sync`, `export`, `diff`, `merge`, `publish`, `push`, `pull`, `search`, `skill`, `ask`, `auth`, `version` / `-v` / `--version` / `-version`, and `help` / `-h` / `--help`.
- `[ask.go:RunAsk]` — `func RunAsk(args []string) error`: Executes `okf ask "<question>" [flags]` with flags `[ask.go:-bundle]` (string, default `"."`), `[ask.go:-seed]` (string, default `""`), `[ask.go:-depth]` (int, default `2`), `[ask.go:-max-chars]` (int, default `16000`), `[ask.go:-model]` (string, default `""`), `[ask.go:-api-key]` (string, default `""`), and `[ask.go:-raw]` (bool, default `false`).
- `[assemble.go:RunAssemble]` — `func RunAssemble(args []string) error`: Executes `okf assemble <concept-id> [flags]` with flags `[assemble.go:-bundle]` (string, default `"."`), `[assemble.go:-depth]` (int, default `2`), `[assemble.go:-max-chars]` (int, default `16000`), `[assemble.go:-direction]` (string, default `"bidirectional"`), `[assemble.go:-format]` (string, default `"xml"`), `[assemble.go:-monitor]` (bool, default `false`), and `[assemble.go:-remote]` (bool, default `false`).
- `[auth.go:RunAuth]` — `func RunAuth(args []string) error`: Dispatches `okf auth` subcommands (`login`).
- `[auth.go:RunAuthLogin]` — `func RunAuthLogin(args []string) error`: Authenticates CLI with OKF Hub using `[auth.go:-token]` (string, default `""`) or local HTTP callback `GET /callback?token=<token>` on `:8989`.
- `[diff_merge.go:RunDiff]` — `func RunDiff(args []string) error`: Compares two knowledge bundles `okf diff <bundle-path-a> <bundle-path-b>`.
- `[diff_merge.go:RunMerge]` — `func RunMerge(args []string) error`: Merges two knowledge bundles `okf merge <bundle-path-source> <bundle-path-target> [flags]` with flags `[diff_merge.go:-strategy]` (string, default `"union"`) and `[diff_merge.go:-output]` (string, default `""`).
- `[doc.go:RunDoc]` — `func RunDoc(args []string) error`: Generates static HTML documentation with flags `[doc.go:-bundle]` (string, default `"."`) and `[doc.go:-output]` (string, default `"docs"`).
- `[export.go:RunExport]` — `func RunExport(args []string) error`: Routes `okf export <type> [flags]` for `jsonld` (`[export.go:runExportJSONLD]`: `[export.go:-bundle]` default `"."`, `[export.go:-output]` default `"schema.jsonld"`) and `skill` / `agent-skill` (delegated to `[skill.go:runSkillExport]`).
- `[harvest.go:RunHarvest]` — `func RunHarvest(args []string) error`: Dispatches `okf harvest <type> [flags]` across subcommands:
  - `db` (`[harvest.go:runHarvestDB]`): `[harvest.go:-driver]` (string, default `""`), `[harvest.go:-conn]` (string, default `""`), `[harvest.go:-dataset]` (string, default `""`), `[harvest.go:-schema]` (string, default `""`), `[harvest.go:-output]` (string, default `"concepts"`), `[harvest.go:-ai-enrich]` (bool, default `false`), `[harvest.go:-model]` (string, default `"gemini-2.5-flash"`), `[harvest.go:-api-key]` (string, default `""`).
  - `openapi` (`[harvest.go:runHarvestOpenAPI]`): `[harvest.go:-spec]` (string, default `""`), `[harvest.go:-output]` (string, default `"concepts"`), `[harvest.go:-ai-enrich]` (bool, default `false`), `[harvest.go:-model]` (string, default `"gemini-2.5-flash"`), `[harvest.go:-api-key]` (string, default `""`).
  - `proto` (`[harvest.go:runHarvestProto]`): `[harvest.go:-path]` (string, default `""`), `[harvest.go:-output]` (string, default `"concepts"`), `[harvest.go:-ai-enrich]` (bool, default `false`), `[harvest.go:-model]` (string, default `"gemini-2.5-flash"`), `[harvest.go:-api-key]` (string, default `""`).
  - `git` (`[harvest.go:runHarvestGit]`): `[harvest.go:-repo]` (string, default `"."`), `[harvest.go:-output]` (string, default `"concepts"`), `[harvest.go:-ai-enrich]` (bool, default `false`), `[harvest.go:-model]` (string, default `"gemini-2.5-flash"`), `[harvest.go:-api-key]` (string, default `""`).
  - `web` (`[harvest.go:runHarvestWeb]`): `[harvest.go:-url]` (string, default `""`), `[harvest.go:-depth]` (int, default `1`), `[harvest.go:-output]` (string, default `"concepts"`), `[harvest.go:-ai-enrich]` (bool, default `false`), `[harvest.go:-api-key]` (string, default `""`).
- `[lint.go:RunLint]` — `func RunLint(args []string) error`: Validates an OKF bundle (optional positional `bundlePath`, default `"."`) with flags `[lint.go:-host]` (string, default `"http://localhost:8080"`), `[lint.go:-api-key]` (string, default `""`), and `[lint.go:-strict]` (bool, default `false`).
- `[publish.go:RunPublish]` — `func RunPublish(args []string) error`: Publishes bundle at optional positional `bundlePath` (default `"."`) to OKF Hub with flags `[publish.go:-host]` (string, default `"http://localhost:8080"`) and `[publish.go:-api-key]` (string, default `""`).
- `[publish.go:RunPull]` — `func RunPull(args []string) error`: Downloads a bundle specified by required positional `hubURI` from OKF Hub with flags `[publish.go:-host]` (string, default `"http://localhost:8080"`) and `[publish.go:-api-key]` (string, default `""`).
- `[push.go:RunPush]` — `func RunPush(args []string) error`: Pushes incremental concept updates for optional positional `bundlePath` (default `"."`) to OKF Hub with flags `[push.go:-host]` (string, default `"http://localhost:8080"`) and `[push.go:-api-key]` (string, default `""`).
- `[search.go:RunSearch]` — `func RunSearch(args []string) error`: Searches OKF Hub bundles using optional positional `query` (default `""`) with flags `[search.go:-tag]` (string, default `""`), `[search.go:-host]` (string, default `"http://localhost:8080"`), and `[search.go:-api-key]` (string, default `""`).
- `[server.go:RunServer]` — `func RunServer(args []string) error`: Starts the OKF MCP server (or remote proxy) with flags `[server.go:-bundle]` (string, default `"."`), `[server.go:-transport]` (string, default `"stdio"`), `[server.go:-port]` (int, default `8080`), and `[server.go:-remote]` (bool, default `false`).
- `[server.go:RunLsp]` — `func RunLsp(args []string) error`: Starts the Language Server Protocol (LSP) server over stdio.
- `[skill.go:RunSkill]` — `func RunSkill(args []string) error`: Dispatches `okf skill` subcommands:
  - `export` (`[skill.go:runSkillExport]`): `[skill.go:-bundle]` (string, default `"."`), `[skill.go:-output]` (string, default `"SKILL.md"`), `[skill.go:-name]` (string, default `""`), `[skill.go:-desc]` (string, default `""`), `[skill.go:-tags]` (string, default `""`), `[skill.go:-types]` (string, default `""`), `[skill.go:-no-attestations]` (bool, default `false`).
  - `install` / `init` (`[skill.go:runSkillInstall]`): `[skill.go:-dir]` (string, default `""`), `[skill.go:-platform]` (string, default `"agents"`; supports `"agents"`, `"antigravity"` / `"agy"`, `"claude"` / `"claude-code"`, `"cursor"`, falling back to `.agents/skills/okf` for unrecognized values).
- `[sync.go:RunSync]` — `func RunSync(args []string) error`: Runs the bi-directional knowledge sync engine with flags `[sync.go:-bundle]` (string, default `"."`), `[sync.go:-config]` (string, default `"okf.yaml"`), `[sync.go:-daemon]` (bool, default `false`), and `[sync.go:-interval]` (int, default `300`).

## Invariants

- `[root.go:Execute]` — Empty `args` prints usage text and returns `nil`; unrecognized subcommands print usage and return `unknown command: <subcommand>`.
- `[ask.go:RunAsk]`, `[ask.go:partitionFlagsAndPositional]` — Prints usage and returns `nil` when `args` is `"help"`, `"-h"`, `"--help"`, or triggers `flag.ErrHelp`; otherwise requires a non-empty trimmed question string, with recognized value flags allowed either before or after positional question words.
- `[assemble.go:RunAssemble]` — Requires positional `startID`; `[assemble.go:-direction]` must be `"outbound"`, `"inbound"`, or `"bidirectional"`; `MaxTokens` is computed as `*maxChars / 4`.
- `[auth.go:RunAuth]`, `[auth.go:RunAuthLogin]` — Requires subcommand `login`; when `[auth.go:-token]` is omitted, `GET /callback` requires a non-empty `token` query parameter (returning HTTP `400` and an error otherwise).
- `[diff_merge.go:RunDiff]` — Requires at least two positional bundle path arguments and returns an error (`"differences found"`) whenever `len(d.Changes) > 0`.
- `[diff_merge.go:RunMerge]` — Requires at least two positional bundle path arguments (`sourcePath`, `targetPath`) and `[diff_merge.go:-strategy]` in `{"union", "ours", "theirs"}`; defaults output directory to `sourcePath` when `[diff_merge.go:-output]` is empty, and when the output path is a Git URL, warns and writes merged concepts into a temporary clone directory that is deleted on return via `defer cleanupOut()`.
- `[export.go:RunExport]` — Requires positional `exportType` matching `"jsonld"`, `"skill"`, or `"agent-skill"`.
- `[harvest.go:RunHarvest]` — Requires positional `harvestType` matching `"db"`, `"openapi"`, `"proto"`, `"git"`, or `"web"`:
  - `[harvest.go:runHarvestDB]` requires non-empty `[harvest.go:-driver]` (`"bigquery"`, `"postgres"`, `"spanner"`, or `"mysql"`) and `[harvest.go:-conn]`; BigQuery requires `[harvest.go:-dataset]` (accepting `<PROJECT>` or `projects/<PROJECT>` in `-conn`) or a 4-segment `projects/<PROJECT>/datasets/<DATASET>` connection string.
  - `[harvest.go:runHarvestOpenAPI]` requires `[harvest.go:-spec]`; `[harvest.go:runHarvestProto]` requires `[harvest.go:-path]`; `[harvest.go:runHarvestWeb]` requires `[harvest.go:-url]` and (when `[harvest.go:-ai-enrich]` is set) a non-empty `--api-key` or `GEMINI_API_KEY` environment variable.
  - In `db`, `openapi`, `proto`, and `git` harvesters, `pkg/ai.CurateConcepts` failures under `[harvest.go:-ai-enrich]` emit a warning rather than aborting execution.
- `[lint.go:RunLint]` — Fails when validation reports `>0` errors, or when `[lint.go:-strict]` is `true` and `>0` warnings exist; defaults displayed `OKFVersion` to `"0.2 (default)"` when unset.
- `[publish.go:RunPublish]`, `[push.go:RunPush]` — Require a non-empty API key resolved via `pkg/config.GetAPIKey`; `[push.go:RunPush]` enforces a 30-second HTTP timeout and HTTP `2xx` status on `POST {host}/api/bundles/push`.
- `[publish.go:RunPull]` — Requires positional argument `hubURI`.
- `[server.go:RunServer]`, `[server.go:proxyRemoteMCP]` — Uses HTTP transport only when `[server.go:-transport]` equals `"http"` (falling back to stdio otherwise unless `[server.go:-remote]` is `true`); `[server.go:proxyRemoteMCP]` is an unimplemented prototype stub that blocks indefinitely on `<-context.Background().Done()`.
- `[skill.go:RunSkill]`, `[skill.go:runSkillExport]` — Requires a valid subcommand (`export`, `install`, `init`, `help`, `-h`, `--help`); `export` defaults skill name to `filepath.Base(absBundlePath)` (or `"okf-knowledge"` for `.` / `/`) when `[skill.go:-name]` is empty, and sets `IncludeAttestations` to `!*noAttestations`.

## Side Effects

- `[root.go:Execute]` — Mutates package-level `[root.go:Version]`, `[root.go:Commit]`, and `[root.go:Date]` variables and writes usage/version text to `os.Stdout`; `[assemble.go:renderTokenMonitor]` writes token-budget progress bars to `os.Stderr` when `-monitor` is `true`.
- `[harvest.go]` — Blank imports register `mysql` (`github.com/go-sql-driver/mysql`), `spanner` (`github.com/googleapis/go-sql-spanner`), and `postgres` (`github.com/lib/pq`) drivers globally in `database/sql` at package initialization.
- **Filesystem Reads, Temp Clones, & Writes**:
  - Bundle resolution (`pkg/parser.ResolvePath` and `pkg/parser.ParseBundle`) across `ask`, `assemble`, `diff`, `merge`, `doc`, `export`, `lint`, `publish`, `push`, `server`, `skill export`, and `sync` reads local directories or clones remote bundles into temporary directories cleaned up on return.
  - Reads `OKF_HUB_API_KEY` from process environment (`os.Getenv`) and `~/.okf/config.json` via `pkg/config.GetAPIKey` in `lint`, `publish`, `pull`, `push`, and `search`.
  - `[harvest.go:runHarvestGit]` creates temp directory `okf-harvest-git-*` (`os.MkdirTemp`) and clones remote Git URLs via `github.com/go-git/go-git/v5.PlainClone`.
  - Writes output artifacts to disk in `[auth.go:saveToken]` (user config via `pkg/config.SaveConfig`), `[diff_merge.go:RunMerge]` and `[harvest.go:RunHarvest]` (`pkg/harvester.WriteConcepts`), `[doc.go:RunDoc]` (`pkg/generator.Generate`), `[export.go:runExportJSONLD]` (`os.WriteFile`), `[publish.go:RunPull]` (`pkg/pull.PullBundle`), `[skill.go:runSkillExport]` (`os.MkdirAll`, `os.WriteFile`), and `[skill.go:runSkillInstall]` (`pkg/skill.InstallOfficialSkill`).
- **Network, External Services, & Blocking Servers**:
  - Makes Gemini LLM API calls in `[ask.go:RunAsk]` (`pkg/ai.AskBundle`), `[harvest.go:RunHarvest]` (`pkg/ai.CurateConcepts`), and `[harvest.go:runHarvestWeb]` (`pkg/harvester.NewLLMClassifier`, reading `GEMINI_API_KEY` via `os.Getenv`).
  - Connects to BigQuery (`cloud.google.com/go/bigquery.NewClient`), SQL databases (`database/sql.Open`), and target web pages (`pkg/harvester.NewWebHarvester`) in `[harvest.go:RunHarvest]`.
  - Communicates over HTTP with OKF Hub in `[lint.go:RunLint]`, `[publish.go:RunPublish]`, `[publish.go:RunPull]`, `[push.go:RunPush]` (`POST {host}/api/bundles/push`), and `[search.go:RunSearch]`.
  - Emits telemetry events (`pkg/telemetry.SendEvent`) when `--remote` is set in `[assemble.go:RunAssemble]` (`"assemble_run"`) and `[server.go:RunServer]` (`"mcp_start_remote"`).
  - Binds local network listeners or blocking stdio/daemon loops in `[auth.go:RunAuthLogin]` (HTTP server on `:8989`), `[server.go:RunServer]` (MCP HTTP server on `:<port>` or stdio), `[server.go:RunLsp]` (LSP over `os.Stdin`/`os.Stdout`), and `[sync.go:RunSync]` (`pkg/sync.Run`).

## Verification

- Run: `go test ./...`
