---
icon: lucide/zap
---

# Quickstart

Build a working Vesvai plugin in under 5 minutes.

## Prerequisites

- Go 1.26 or later
- Vesvai installed and working

## Step 1: Create the project

```bash
mkdir my-vesvai-plugin
cd my-vesvai-plugin
go mod init my-vesvai-plugin
go get github.com/hashicorp/go-plugin@latest
go get github.com/vesvai/vesvai@latest
go get github.com/hashicorp/go-hclog@latest
```

## Step 2: Write the plugin

Create `main.go`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/hashicorp/go-hclog"
    "github.com/hashicorp/go-plugin"
    "github.com/vesvai/vesvai/internal/plugin/shared"
)

// MyPlugin implements the shared.Plugin interface
type MyPlugin struct{}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Description() string {
    return "My first Vesvai plugin"
}

func (p *MyPlugin) Boot(deps shared.Deps) error {
    fmt.Println("My plugin loaded!")
    fmt.Printf("Config has %d providers\n", len(deps.Config.Providers))
    return nil
}

func main() {
    logger := hclog.New(&hclog.LoggerOptions{
        Name:   "my-plugin",
        Output: os.Stderr,
        Level:  hclog.Info,
    })

    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: shared.Handshake,
        Plugins: map[string]plugin.Plugin{
            "vesvai": &shared.VesvaiPluginRPC{
                Impl: &MyPlugin{},
            },
        },
        Logger: logger,
    })
}
```

## Step 3: Build and install

```bash
# Build
go build -o my-plugin .

# Install
mkdir -p ~/.vesvai/plugins
cp my-plugin ~/.vesvai/plugins/
chmod +x ~/.vesvai/plugins/my-plugin
```

## Step 4: Verify

```bash
vesvai plugin list
```

Output:

```
NAME        VERSION  DESCRIPTION
----        -------  -----------
my-plugin   1.0.0    My first Vesvai plugin

Total: 1 plugins
```

## Step 5: Check details

```bash
vesvai plugin info my-plugin
```

Output:

```
Name:        my-plugin
Version:     1.0.0
Description: My first Vesvai plugin
Path:        /home/user/.vesvai/plugins/my-plugin
```

## Next steps

- Read the [Interface Reference](interface.md) for all available types and methods
- See the [Development Guide](development.md) for more advanced patterns
- Check the [Configuration](../vesvai/configurations/plugins.md) for options
