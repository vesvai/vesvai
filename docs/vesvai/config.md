---
icon: lucide/cog
---

# Config

Vesvai is configured with a single JSON file per machine plus a few project-local
files. There is no YAML or TOML variant.

## File locations

| Scope | Path | Purpose |
|---|---|---|
| Global | `~/.vesvai/vesvai.json` | Providers, server, permission, theme, storage drivers, MCP and LSP servers |
| Project | `<project>/.vesvai/` | Project data: `todos/`, `subagents/`, `plans/`, `rules/` |
| Project | `<project>/.mcp.json` | Project MCP servers |
| Project | `<project>/.lsp.json` | Project language servers |
| Global | `~/.vesvai/permissions.json` | Remembered tool approvals and rejections |

The global file is created with defaults on first run. A missing file is not an
error — Vesvai falls back to built-in defaults. An unparsable file is an error.

`vesvai files` prints every resolved path and whether it exists.

## Viewing the config

```bash
vesvai config show
```

Prints the resolved configuration as pretty-printed JSON with every API key masked.
There are no `config get` / `config set` subcommands — edit the file directly.

## Reference

### `providers[]`

Configured LLM providers. See [Providers & Models](providers-and-models.md).

| Key | Type | Default | Description |
|---|---|---|---|
| `provider` | string | — | Name of a registered provider (for example `"openai"`). Wins over `driver` |
| `driver` | string | — | Raw wire protocol: `"openai"`, `"claude"`, or `"gemini"`. Requires `base_url` |
| `api_key` | string | — | API key, stored in plaintext |
| `base_url` | string | — | Override the endpoint. Required for raw drivers, ignored by named providers |
| `timeout` | int | `120` | HTTP timeout in seconds |
| `max_retries` | int | — | Reserved; currently not read by the runtime |
| `headers` | object | — | Extra HTTP headers merged onto the driver defaults |

Either `provider` or `driver` must be set.

### `logger`

| Key | Type | Default | Description |
|---|---|---|---|
| `driver` | string | `"sqlite"` | `"sqlite"` writes `~/.vesvai/logs.db`; `"console"` keeps an in-memory buffer printed to stdout |
| `max_log_count` | int | `1000` | Maximum retained log records |

### `cache`

| Key | Type | Default | Description |
|---|---|---|---|
| `driver` | string | `"sqlite"` | `"sqlite"` uses `~/.vesvai/cache.db`; `"json"` uses `~/.vesvai/cache.json` |

### `session`

| Key | Type | Default | Description |
|---|---|---|---|
| `driver` | string | `"sqlite"` | `"sqlite"` uses `~/.vesvai/sessions.db`; `"json"` stores one file per session under `~/.vesvai/sessions/` |

### `server`

Used by `vesvai serve`. See [HTTP](usage/http.md) and [ACP](usage/acp.md).

| Key | Type | Default | Description |
|---|---|---|---|
| `host` | string | `"127.0.0.1"` | Bind address |
| `port` | int | `8080` | Listen port |
| `required_headers` | object | — | Required request headers for auth. A non-empty value must match (case-insensitively); an empty value only checks presence |
| `headers` | object | — | Extra headers added to every HTTP API response |

### `theme`

| Key | Type | Default | Description |
|---|---|---|---|
| `theme` | string | `"dark"` | TUI theme name. Changed at runtime with `Ctrl+T` and saved back to this file |

See [TUI themes](usage/tui.md#themes) for the list.

### `permission`

Controls when tools may run. See [Permissions](features/permissions.md).

| Key | Type | Default | Description |
|---|---|---|---|
| `default` | string | `"semi-ask"` | Mode when no rule matches: `allow`, `semi-ask`, `ask`, `semi-judge`, `judge` |
| `rules` | object | — | Per-tool mode overrides, for example `{"bash": "ask", "write": "allow"}` |
| `judge_provider` | string | — | Provider used by the judge LLM. Falls back to the preferred model |
| `judge_model` | string | — | Model used by the judge LLM |

### `mcp_servers`

Map of server name to [MCP server config](configurations/mcp.md).

| Key | Type | Description |
|---|---|---|
| `command` | string | Executable to spawn (stdio transport) |
| `args` | string[] | Command arguments |
| `env` | object | Extra environment variables (merged over the inherited environment) |
| `url` | string | SSE endpoint. When set, the server is treated as a remote SSE server |
| `headers` | object | HTTP headers for SSE requests |

### `language_servers`

Map of server name to [language server config](configurations/lsp.md).

| Key | Type | Description |
|---|---|---|
| `command` | string | Language server executable |
| `args` | string[] | Command arguments |
| `env` | object | Extra environment variables |
| `filetypes` | string[] | File extensions this server handles |
| `rootMarkers` | string[] | Files used to detect the project root |
| `download` | string | Static binary URL to download when the command is not found |
| `install` | string | Shell command to install the server when the command is not found |
| `version` | string | Informational version |
| `enabled` | bool | `false` removes a server inherited from a lower-priority layer |

## Complete example

```json
{
  "providers": [
    { "provider": "openai", "api_key": "sk-..." },
    {
      "provider": "groq",
      "api_key": "gsk_...",
      "timeout": 60,
      "headers": { "X-Custom": "value" }
    },
    {
      "driver": "openai",
      "base_url": "http://localhost:11434/v1",
      "api_key": "ollama"
    }
  ],
  "logger": { "driver": "sqlite", "max_log_count": 1000 },
  "cache": { "driver": "sqlite" },
  "session": { "driver": "sqlite" },
  "server": {
    "host": "127.0.0.1",
    "port": 8080,
    "required_headers": { "Authorization": "Bearer secret" },
    "headers": { "X-Served-By": "vesvai" }
  },
  "theme": "dark",
  "permission": {
    "default": "semi-ask",
    "judge_provider": "openai",
    "judge_model": "gpt-4o-mini",
    "rules": { "bash": "ask", "write": "allow" }
  },
  "mcp_servers": {
    "db": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres"],
      "env": { "DEBUG": "true" }
    },
    "remote": {
      "url": "https://example.com/sse",
      "headers": { "Authorization": "Bearer token" }
    }
  },
  "language_servers": {
    "gopls": {
      "command": "gopls",
      "filetypes": ["go"],
      "rootMarkers": ["go.mod"]
    }
  }
}
```

## Project-local configuration

Only MCP and LSP servers support project-level files. Other settings are global.

```json title=".mcp.json"
{
  "mcpServers": {
    "db": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-postgres"] }
  }
}
```

```json title=".lsp.json"
{
  "languageServers": {
    "gopls": { "command": "gopls", "filetypes": ["go"], "rootMarkers": ["go.mod"] }
  }
}
```

Project entries override global entries with the same name. Setting
`"enabled": false` on a project LSP entry removes an inherited server.

## Environment variables

Vesvai does not read API keys or config paths from environment variables. The only
variable that affects behavior is `HOME` (via the OS home directory), which determines
where `~/.vesvai/` lives. Spawned MCP and LSP subprocesses inherit the parent process
environment plus any `env` entries you configure.
