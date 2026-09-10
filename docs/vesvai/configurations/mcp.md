---
icon: lucide/package-plus
---

# MCP

Vesvai is an MCP (Model Context Protocol) **client**. It connects to MCP servers at
startup, discovers their tools, and exposes them to agents alongside the built-in
tools.

## Configuration

=== "Global"

    Add servers to the `mcp_servers` map in `~/.vesvai/vesvai.json`:

    ```json
    {
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
      }
    }
    ```

=== "Project"

    Create `.mcp.json` in the project root:

    ```json
    {
      "mcpServers": {
        "db": {
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-postgres",
                   "postgresql://localhost:5432/db"]
        }
      }
    }
    ```

Project entries override global entries with the same name.

### Server options

| Key | Type | Description |
|---|---|---|
| `command` | string | Executable to spawn (stdio transport) |
| `args` | string[] | Command arguments |
| `env` | object | Extra environment variables, merged over the inherited environment |
| `url` | string | Remote endpoint (SSE transport). Takes precedence over `command` |
| `headers` | object | HTTP headers sent with SSE requests |

A server must define either `url` or `command`.

## Transports

### stdio

Vesvai spawns `command args...` as a child process:

- The child inherits Vesvai's full environment; `env` entries override individual
  variables.
- The child's stderr is forwarded to Vesvai's stderr.
- Messages are newline-delimited JSON (with optional `Content-Length` framing), max
  16 MiB per frame.
- The child is terminated with SIGTERM when Vesvai shuts down.

### SSE

Vesvai opens a `GET` request with `Accept: text/event-stream` and the configured
headers. The server must first emit an `endpoint` event telling Vesvai where to
POST messages. Subsequent requests include the `Mcp-Session-Id` header when the
server provides one. Responses arrive as `message` events.

## Lifecycle

1. At startup, one goroutine per configured server connects with a 15 second
   timeout: transport connect → `initialize` → `tools/list`.
2. Every discovered tool is registered in the global tool registry as
   `<server>__<tool>` (double underscore). Names are guaranteed not to collide
   between servers.
3. A server that fails to connect is logged as a warning and skipped; the rest of
   Vesvai continues normally.
4. On shutdown, every MCP tool is unregistered and every connection is closed.

The MCP protocol version used is `2024-11-05`.

## Managing servers

```bash
vesvai mcp list                       # all servers, global and project
vesvai mcp list --project             # project only
vesvai mcp tools                      # live tools from connected servers
vesvai mcp tools --server db          # filter by server
```

### Adding a server

Interactive:

```bash
vesvai mcp add
```

Fully specified:

```bash
# Local stdio server
vesvai mcp add --project --name db --transport local \
  --command npx --arg -y --arg @modelcontextprotocol/server-postgres \
  --env DEBUG=true

# Remote SSE server
vesvai mcp add --global --name remote --transport http \
  --url https://example.com/sse \
  --header Authorization="Bearer token"
```

| Flag | Description |
|---|---|
| `--global` / `--project` | Where to write the server |
| `--name` | Server name |
| `--transport` | `local` (stdio) or `http` (SSE) |
| `--command` | Executable for local servers |
| `--arg` | Argument (repeatable) |
| `--env` | `KEY=VALUE` environment variable (repeatable) |
| `--url` | SSE endpoint for http servers |
| `--header` | `KEY=VALUE` request header (repeatable) |

The interactive flow offers common header presets: `Authorization`, `X-API-Key`,
`Cookie`, and `User-Agent`.

### Removing a server

```bash
vesvai mcp remove db
```

If the server exists in both scopes, Vesvai asks which one to remove. If it exists
in only one, that scope is used automatically.

## Using MCP tools

MCP tools behave like built-in tools. The model sees them under their
`server__tool` names, and they are subject to the permission middleware's
`permission.default` mode unless overridden by a rule.

Tool results are returned as text; image content is summarized as `[image data]`.
A tool result flagged `isError` is surfaced as a tool error.

## TUI

Open Settings with ++ctrl+p++ and switch to the **MCP** tab. Each server shows its
tool count and connection state (`connected` / `not connected`). Selecting a server
opens a sub-list of its tools.

## Troubleshooting

```bash
vesvai logs --search mcp
vesvai mcp tools --server db
```

Common issues:

- **Command not found** — the executable must be on `PATH` for the Vesvai process.
- **SSE server never becomes ready** — the server must emit the `endpoint` event.
- **Tools missing** — the server connected but `tools/list` returned nothing, or the
  connection failed during `initialize`. Check the logs for a warning line.
