---
icon: lucide/package-plus
---

# Plugins

Vesvai supports plugins that extend its functionality. Plugins are standalone binaries
that run in separate processes and communicate with Vesvai via RPC.

## Configuration

Add plugin configuration to `~/.vesvai/vesvai.json`:

```json
{
  "plugins": {
    "enabled": true,
    "exclude": ["broken-plugin"]
  }
}
```

### Options

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | boolean | `true` | Enable or disable the plugin system |
| `exclude` | string[] | `[]` | List of plugin names to skip during loading |

### Plugin directory

Plugins are loaded from `~/.vesvai/plugins/`. Vesvai scans this directory at startup
for executable files and attempts to load each one as a plugin.

```bash
mkdir -p ~/.vesvai/plugins
```

## Managing plugins

```bash
vesvai plugin list              # List installed plugins
vesvai plugin info <name>       # Show plugin details
```

## Lifecycle

1. At startup, after all core modules are initialized, Vesvai scans the plugin
   directory for executables.
2. Each executable is launched as a separate process via
   [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin).
3. The plugin's `Boot` method is called with a `Deps` struct containing the
   application configuration.
4. If a plugin fails to load, a warning is logged and Vesvai continues normally.
5. On shutdown, all plugin processes are terminated.

## Troubleshooting

```bash
vesvai plugin list               # Check if plugins are detected
vesvai logs --search plugin      # Check plugin-related logs
```

Common issues:

- **Plugin not appearing** — The executable must be in `~/.vesvai/plugins/` and have
  execute permission.
- **Plugin fails to load** — Check logs for the specific error. Common causes include
  missing dependencies or incompatible versions.
- **Plugin excluded** — Check the `exclude` list in your config.
