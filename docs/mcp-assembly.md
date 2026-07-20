# Context Assembly & MCP

Once you have harvested metadata into an OKF bundle, you need a way to serve it to Large Language Models (LLMs) or AI Agents. `okf` provides two powerful mechanisms for this: static assembly and dynamic Model Context Protocol (MCP) serving.

## Graph-based Breadth-First Search (BFS) Assembly

OKF concepts are connected via markdown links (e.g., a Database Table links to its Database Schema, which links to its Project). When an LLM asks a question about a specific concept, it often needs the surrounding context.

The `okf assemble` command performs a deterministic Breadth-First Search (BFS) starting from a target concept, recursively resolving linked dependencies up to a maximum depth or character budget. It outputs the result in an XML structure optimized for LLM ingestion.

```bash
# Assemble context for a specific table up to 10,000 characters
okf assemble \
  --bundle ./okf-bundle \
  --concept tables/users \
  --max-chars 10000 \
  --format xml
```

## Model Context Protocol (MCP)

For interactive agentic workflows, `okf` includes a built-in MCP server. MCP allows AI agents (like Claude Desktop, Google Vertex Agents, or any custom MCP client) to dynamically query the OKF graph at runtime.

The server exposes an `assemble_context` tool that agents can call to fetch structured metadata exactly when they need it to write a query, understand a schema, or troubleshoot an architecture.

### Running the MCP Server

```bash
# Start the MCP server using Standard I/O transport
okf mcp --transport stdio --bundle ./okf-bundle
```

### Claude Desktop Integration

To expose your OKF bundle to Claude Desktop, add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "okf": {
      "command": "/usr/local/bin/okf",
      "args": ["mcp", "--transport", "stdio", "--bundle", "/absolute/path/to/okf-bundle"]
    }
  }
}
```

Once configured, you can simply ask Claude: *"Write a SQL query to find all active users based on our schema."* Claude will autonomously use the OKF MCP tool to look up the exact table schemas before writing the query.
