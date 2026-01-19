# ADK Setup Instructions

This project is configured to work with the Google Agent Development Kit (ADK). To enable your AI assistant to access the ADK documentation, you need to configure an MCP server.

## Prerequisites

Ensure you have `uv` installed. It is used to run the documentation server.
If you didn't have it, install it by running:
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

## IDE Configuration

You need to add the `adk-docs-mcp` server to your IDE's MCP configuration file (typically `mcp_config.json` or similar in your settings configuration directory).

Since a standard `mcp_config.json` was not found in `~/.config/`, please locate your IDE's MCP configuration file (often accessible via the "Manage MCP Servers" menu in your AI assistant panel) and add the following entry:

```json
{
  "mcpServers": {
    "adk-docs-mcp": {
      "command": "uvx",
      "args": [
        "--from",
        "mcpdoc",
        "mcpdoc",
        "--urls",
        "AgentDevelopmentKit:https://google.github.io/adk-docs/llms.txt",
        "--transport",
        "stdio"
      ]
    }
  }
}
```

After adding this configuration, restart your MCP servers or IDE if necessary.
