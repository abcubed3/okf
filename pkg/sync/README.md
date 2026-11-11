# `sync`

## Overview
Package `sync` orchestrates bidirectional synchronization of Open Knowledge Format (OKF) concepts (`*bundle.Concept`) between a local bundle directory and external storage or collaboration backends. It provides a pluggable `Connector` interface, YAML configuration loading (`LoadConfig`), deterministic content hashing and atomic sync-state tracking (`StateManager` under `<bundlePath>/.okf/state.json`), a one-shot or daemonized sync `Engine`, active `GoogleDriveConnector` and `GitConnector` implementations, and stub `NotionConnector`, `ConfluenceConnector`, and `JiraConnector` adapters.

## Interface
- `[connector.go:Connector]` — `type Connector interface`: Defines `Initialize(ctx context.Context, cfg *Config) error`, `Push(ctx context.Context, concepts []*bundle.Concept) error`, `Pull(ctx context.Context) ([]*bundle.Concept, error)`, and `Name() string`.
- `[config.go:Config]` & `[config.go:LoadConfig]` — `type Config struct` (nesting `GoogleDriveConfig`, `NotionConfig`, `ConfluenceConfig`, `JiraConfig`, and `GitConfig` under `Connectors`) and `func LoadConfig(path string) (*Config, error)`.
- `[config.go:GoogleDriveConfig]`, `[config.go:NotionConfig]`, `[config.go:ConfluenceConfig]`, `[config.go:JiraConfig]`, `[config.go:GitConfig]` — Per-connector YAML configuration structs.
- `[engine.go:Engine]` & `[engine.go:Run]` — `type Engine struct`, `(e *Engine) Register(c Connector)`, `(e *Engine) Start(daemon bool, interval int) error`, and top-level helper `func Run(bundlePath, configPath string, daemon bool, interval int) error`.
- `[state.go:StateManager]` & `[state.go:NewStateManager]` — `type StateManager struct` (`Load()`, `Save()`, `GetExternalID`, `SetExternalID`, `GetContentHash`, `SetContentHash`, `GetLastPulledHash`, `SetLastPulledHash`, `GetLastSync`, `UpdateLastSync`).
- `[state.go:State]`, `[state.go:ConceptState]`, `[state.go:HashConcept]` — `type State struct`, `type ConceptState struct`, and `func HashConcept(c *bundle.Concept) (string, error)`.
- `[gdrive.go:GoogleDriveConnector]` & `[gdrive.go:NewGoogleDriveConnector]` — `func NewGoogleDriveConnector(state *StateManager) *GoogleDriveConnector` implementing `Connector` (`"google_drive"`).
- `[git.go:GitConnector]` & `[git.go:NewGitConnector]` — `func NewGitConnector(state *StateManager) *GitConnector` implementing `Connector` (`"git"`).
- `[notion.go:NotionConnector]` & `[notion.go:NewNotionConnector]` — `func NewNotionConnector(state *StateManager) *NotionConnector` implementing `Connector` (`"notion"`).
- `[confluence.go:ConfluenceConnector]` & `[confluence.go:NewConfluenceConnector]` — `func NewConfluenceConnector(state *StateManager) *ConfluenceConnector` implementing `Connector` (`"confluence"`).
- `[jira.go:JiraConnector]` & `[jira.go:NewJiraConnector]` — `func NewJiraConnector(state *StateManager) *JiraConnector` implementing `Connector` (`"jira"`).

## Invariants
- `[config.go:LoadConfig]` — Returns an empty non-nil `*Config` and `nil` error when `path` does not exist (`os.IsNotExist`).
- `[engine.go:Run]` & `[engine.go:Engine.Start]` — Registers built-in connectors in fixed order (`google_drive`, `notion`, `confluence`, `jira`, `git`); `StateManager.Load()` and `Connector.Initialize` failures log warnings without aborting startup; when `daemon` is `true`, `syncOnce` executes immediately and repeats every `interval` seconds until `os.Interrupt` or `syscall.SIGTERM`.
- `[engine.go:Engine.syncOnce]`, `[engine.go:Engine.pushToConnector]`, `[engine.go:Engine.pullFromConnector]` — Executes all connector pushes before pulls, skips local concepts with `ParseError != ""` or `HashConcept` errors, skips unchanged pushes matching `GetContentHash(concept.ID, c.Name())`, skips unchanged pulls matching `GetLastPulledHash(concept.ID)`, and commits updated hashes only after `c.Push` or `harvester.WriteConcepts` succeeds.
- `[state.go:StateManager]`, `[state.go:StateManager.Save]`, `[state.go:HashConcept]` — Guards in-memory `State` with `sync.RWMutex`, persists `<bundlePath>/.okf/state.json` atomically via `<path>.tmp` (`0644`) + `os.Rename`, updates both `ContentHashes` and `LastPushed` (UTC) in `SetContentHash`, and computes `HashConcept` as SHA-256 over `yaml.Marshal(c.Frontmatter)` concatenated with raw `c.Body` bytes.
- `[gdrive.go:GoogleDriveConnector.Initialize]` & `[gdrive.go:GoogleDriveConnector.Pull]` — Requires `cfg.Connectors.GoogleDrive` plus `ServiceAccount`, `CredentialsFile`, or build-time `googleClientID` (defaulting `TokenFile` to `"google_token.json"`); `Pull` exports modified Drive documents for inspection and currently returns `(nil, nil)`.
- `[git.go:GitConnector.Initialize]`, `[git.go:GitConnector.Push]`, `[git.go:GitConnector.Pull]` — Requires `cfg.Connectors.Git` with non-empty `Repo`, defaults `Branch` to `"main"` and `Path` subpath to `"concepts"`, enforces SSH host-key verification via `$HOME/.ssh/known_hosts` unless `InsecureSkipVerify` is `true`, clones into `<TempDir>/okf-sync-git-<sha256(Repo)[:12]>`, and treats `git.ErrEmptyCommit` and `git.NoErrAlreadyUpToDate` as non-errors.
- `[notion.go:NotionConnector]`, `[confluence.go:ConfluenceConnector]`, `[jira.go:JiraConnector]` — Stub connectors: `Initialize` errors when the corresponding config block is `nil`, `Push` records synthetic external IDs (`"<connector>-<concept.ID>"`) without network calls, and `Pull` returns `(nil, nil)`.

## Side Effects
- `[config.go:LoadConfig]` & `[state.go:StateManager.Load]` / `[state.go:StateManager.Save]` — Reads YAML config files and `<bundlePath>/.okf/state.json` from disk, creates `<bundlePath>/.okf` (`0755`), writes `<bundlePath>/.okf/state.json.tmp` (`0644`), and atomically renames it over `state.json`.
- `[engine.go:Engine.Start]`, `[engine.go:Engine.syncOnce]`, `[engine.go:Engine.pullFromConnector]` — Registers `os.Interrupt`/`syscall.SIGTERM` signal handlers, runs a `time.Ticker` in daemon mode, parses `<bundlePath>` via `parser.ParseBundle`, writes pulled concept `.md` files via `harvester.WriteConcepts`, and logs via the standard `log` package.
- `[gdrive.go:GoogleDriveConnector.Initialize]`, `[gdrive.go:getTokenFromWeb]`, `[gdrive.go:saveToken]`, `[gdrive.go:GoogleDriveConnector.Push]`, `[gdrive.go:GoogleDriveConnector.Pull]` — Reads OAuth/service-account credentials from disk, binds a temporary local HTTP server on `:8080` for interactive OAuth2 callbacks, writes token cache files (`0600`), and issues HTTPS RPCs to the Google Drive v3 API.
- `[git.go:GitConnector.Initialize]`, `[git.go:GitConnector.Push]`, `[git.go:GitConnector.Pull]` — Reads SSH keys and `$HOME/.ssh/known_hosts`, manages temporary Git worktrees under `os.TempDir()` (`os.RemoveAll` on corrupt clones, hard-reset on existing clones), writes concept `.md` files (`0644`), commits as `"OKF Sync Engine <sync@okf.io>"`, and clones/pushes/pulls against remote `"origin"` over SSH or HTTPS.

## Verification
- Run: `go test ./...`
