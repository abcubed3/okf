# `config`

## Overview
Package `config` manages local CLI user configuration and Hub API credential resolution for OKF. It persists user settings in `~/.okf/config.json` with restricted filesystem permissions and resolves API keys across CLI flags, environment variables, and the persisted configuration file.

## Interface
- `[config.go:Config]` — `type Config struct` (`APIKey string` serialized as `"api_key"`): JSON-serializable user configuration payload.
- `[config.go:GetConfigPath]` — `func GetConfigPath() (string, error)`: Resolves the absolute path to `~/.okf/config.json`.
- `[config.go:LoadConfig]` — `func LoadConfig() (*Config, error)`: Reads and unmarshals `~/.okf/config.json`.
- `[config.go:SaveConfig]` — `func SaveConfig(cfg *Config) error`: Marshals and writes `cfg` to `~/.okf/config.json`.
- `[config.go:GetAPIKey]` — `func GetAPIKey(providedKey string) string`: Resolves the active OKF Hub API key from explicit input, environment, or saved config.

## Invariants
- `[config.go:GetConfigPath]` — Always resolves the configuration path to `filepath.Join(os.UserHomeDir(), ".okf", "config.json")`.
- `[config.go:LoadConfig]` — Returns an empty `&Config{}` and `nil` error when `~/.okf/config.json` does not exist (`os.IsNotExist`).
- `[config.go:SaveConfig]` — Enforces mode `0700` on the `~/.okf` directory and mode `0600` on `config.json`.
- `[config.go:GetAPIKey]` — Evaluates credential sources in strict precedence order: non-empty `providedKey` parameter -> non-empty `OKF_HUB_API_KEY` environment variable -> non-empty `APIKey` from `LoadConfig()` (returning `""` if all sources are empty or loading fails).

## Side Effects
- `[config.go:GetConfigPath]` — Queries the operating system user home directory via `os.UserHomeDir()`.
- `[config.go:LoadConfig]` — Reads `~/.okf/config.json` from the local filesystem via `os.ReadFile`.
- `[config.go:SaveConfig]` — Creates `~/.okf` (`0700`) via `os.MkdirAll` and writes/truncates `~/.okf/config.json` (`0600`) via `os.WriteFile`.
- `[config.go:GetAPIKey]` — Reads process environment variable `OKF_HUB_API_KEY` via `os.Getenv` and conditionally reads `~/.okf/config.json` via `LoadConfig()`.

## Verification
- Run: `go test ./...`
