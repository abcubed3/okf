# OKF CLI Command Cheatsheet

## Installation

On macOS or Linux, use the official installer:

```bash
curl -sSfL https://okfgo.dev/install.sh | sh
```

Alternatively, download a platform-specific binary from the
[GitHub Releases](https://github.com/abcubed3/okf/releases) page, or install
with Go:

```bash
go install github.com/abcubed3/okf/cmd/okf@latest
```

Ensure the installed binary is available on `PATH`. See the
[README installation guide](https://github.com/abcubed3/okf#installation-and-distribution)
for platform details.

## Commands

| Command                | Purpose                                                      | Common Options                                                                      |
| :--------------------- | :----------------------------------------------------------- | :---------------------------------------------------------------------------------- |
| `okf lint [path]`      | Validate YAML frontmatter, trust signals, and link integrity | `--strict`, `--host`, `--api-key`                                                   |
| `okf assemble <id>`    | BFS context assembly starting from concept ID                | `--bundle`, `--depth 2`, `--max-chars 16000`, `--format xml\|markdown`, `--monitor` |
| `okf harvest <source>` | Harvest schemas from `db`, `openapi`, `proto`, `git`, or `web` | `--output`, `--ai-enrich`, `--model`, `--api-key`                                   |
| `okf mcp`              | Run Model Context Protocol server                            | `--bundle`, `--transport stdio\|http`, `--port 8080`, `--remote`                    |
| `okf skill export`     | Export OKF bundle as an Agent Skill (`SKILL.md`)             | `--bundle`, `--output`, `--name`, `--desc`, `--tags`, `--types`, `--no-attestations` |
| `okf skill install`    | Install official OKF Agent Skill into project                | `--dir`, `--platform agents\|antigravity\|claude\|cursor`                           |
| `okf ask "<query>"`    | Grounded Q&A against knowledge graph using Gemini            | `--bundle`, `--seed`, `--depth`, `--max-chars`, `--model`, `--api-key`, `--raw`     |
| `okf doc`              | Compile bundle into interactive Cytoscape web docs           | `--bundle`, `--output`                                                              |
| `okf export jsonld`    | Export OKF bundle to Schema.org JSON-LD                      | `--bundle`, `--output`                                                              |
| `okf diff <a/> <b/>`   | Compare two OKF bundles for drift                            |                                                                                     |
| `okf merge <a/> <b/>`  | Merge two OKF bundles into one                               | `--strategy union\|ours\|theirs`, `--output`                                        |
