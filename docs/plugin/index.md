---
icon: lucide/puzzle
---

# Plugin System

Vesvai's plugin system allows you to extend its functionality by writing standalone
Go binaries. Plugins run in separate processes and communicate with Vesvai via RPC,
ensuring isolation and safety.

## Features

- **Process isolation** — Plugins run in their own processes. A crashing plugin
  won't bring down Vesvai.
- **Simple interface** — Implement a single `Boot` method to receive application
  configuration.
- **Auto-discovery** — Place executables in `~/.vesvai/plugins/` and they're loaded
  automatically.
- **Hot-pluggable** — Add or remove plugins without modifying Vesvai's source code.

## Architecture

```
┌─────────────────────────────────┐         ┌─────────────────────────┐
│         Vesvai Host             │         │      Plugin Process     │
│                                 │   RPC   │                         │
│  plugin.Module()                │◄───────►│  shared.Plugin.Boot()   │
│    ├─ Scan ~/.vesvai/plugins/   │         │    └─ deps.Config       │
│    ├─ Launch plugin subprocess  │         │                         │
│    └─ Call Boot(deps)           │         │  Your plugin logic      │
└─────────────────────────────────┘         └─────────────────────────┘
```

## Quick start

See the [Quickstart Guide](quickstart.md) for a step-by-step tutorial.

## Documentation

| Page | Description |
|---|---|
| [Quickstart](quickstart.md) | Build your first plugin in 5 minutes |
| [Interface](interface.md) | Plugin interface, types, and method signatures |
| [Development Guide](development.md) | Step-by-step plugin development |

## Configuration

See [Plugin Configuration](../vesvai/configurations/plugins.md) for configuration
options.
