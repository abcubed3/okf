# OKF CLI Command Cheatsheet

| Command | Purpose | Common Options |
| :--- | :--- | :--- |
| `okf lint [path]` | Validate YAML frontmatter, trust signals, and link integrity | `--strict` |
| `okf assemble <id>` | BFS context assembly starting from concept ID | `--bundle`, `--depth 2`, `--max-chars 16000`, `--format xml\|markdown`, `--monitor` |
| `okf harvest <source>` | Harvest schemas from databases, APIs, code, or web | `--out`, `--ai-enrich` |
| `okf mcp` | Run Model Context Protocol server | `--bundle`, `--transport stdio\|http`, `--port 8080` |
| `okf skill export` | Export OKF bundle as an Agent Skill (`SKILL.md`) | `--bundle`, `--output`, `--name`, `--tags`, `--types` |
| `okf skill install` | Install official OKF Agent Skill into project | `--dir`, `--platform antigravity\|claude\|cursor` |
| `okf ask "<query>"` | Grounded Q&A against knowledge graph using Gemini | `--bundle`, `--depth`, `--model` |
| `okf doc` | Compile bundle into interactive Cytoscape web docs | `--out`, `--title` |
| `okf export jsonld` | Export OKF bundle to Schema.org JSON-LD | `--bundle`, `--output` |
| `okf diff <a/> <b/>` | Compare two OKF bundles for drift | |
| `okf merge <a/> <b/>` | Merge two OKF bundles into one | `--out` |
