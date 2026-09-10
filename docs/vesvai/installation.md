---
icon: lucide/download
---

# Vesvai Installation

Vesvai ships as a single static binary with no runtime dependencies. Release builds
are produced for Linux, macOS, and Windows on both `amd64` and `arm64`.

## Requirements

| | |
|---|---|
| **Runtime** | None — the binary is self-contained (`CGO_ENABLED=0`) |
| **Build from source** | Go 1.26.5 or newer |
| **Terminal** | Any modern terminal. The TUI uses 24-bit color and mouse/paste support when available |

## Install

=== "Install script"

    ```bash
    curl -fsSL https://vesv.ai/install | bash
    ```

    Downloads the latest release binary for your platform and installs it to a
    directory on your `PATH`.

=== "npm"

    ```bash
    npm i -g vesvai
    ```

    Installs the `vesvai` binary globally via npm.

=== "Prebuilt binary"

    Download the archive for your platform from the
    [GitHub releases page](https://github.com/vesvai/vesvai/releases):

    | Platform | Archive |
    |---|---|
    | Linux | `vesvai_linux_amd64.tar.gz`, `vesvai_linux_arm64.tar.gz` |
    | macOS | `vesvai_darwin_amd64.tar.gz`, `vesvai_darwin_arm64.tar.gz` |
    | Windows | `vesvai_windows_amd64.zip`, `vesvai_windows_arm64.zip` |

    ```bash
    # Example: Linux amd64
    tar -xzf vesvai_linux_amd64.tar.gz
    sudo mv vesvai /usr/local/bin/
    ```

    Each release also publishes `checksums.txt` so you can verify your download:

    ```bash
    sha256sum -c checksums.txt --ignore-missing
    ```

=== "go install"

    ```bash
    go install github.com/vesvai/vesvai/cmd/vesvai@latest
    ```

    The binary is placed in `$(go env GOPATH)/bin`. Make sure that directory is on
    your `PATH`.

=== "Build from source"

    ```bash
    git clone https://github.com/vesvai/vesvai.git
    cd vesvai
    make build        # produces ./bin/vesvai
    make run          # or run it immediately
    ```

    Other useful targets: `make test`, `make lint`, `make fmt`, `make vet`,
    `make clean`.

## Verify

```bash
vesvai version
# vesvai version 0.1.0
```

## First run

The first launch creates the global config directory and a default config file, and
creates a project-local `.vesvai/` directory in the current working directory.

```bash
vesvai login
```

The login command opens an interactive provider picker, then asks for an API key
(masked). Keys are stored in `~/.vesvai/vesvai.json`. You can skip the interactive
prompt:

```bash
vesvai login --provider openai --api-key sk-...
```

Once at least one provider is configured, start the TUI:

```bash
vesvai
```

Or run a one-shot headless prompt:

```bash
vesvai run "Summarize this repository"
echo "Fix the failing test" | vesvai
```

See [CLI usage](usage/cli.md) for every command.

## Update

Vesvai can update itself in place from GitHub releases:

```bash
vesvai update
```

`vesvai doctor` also checks for updates after testing provider connectivity and
offers to install a newer version when one is available.

## File locations

Run `vesvai files` to print the resolved paths and whether each exists.

| Purpose | Path |
|---|---|
| Global config | `~/.vesvai/vesvai.json` |
| Logs | `~/.vesvai/logs.db` (SQLite driver) |
| Model cache | `~/.vesvai/cache.db` (SQLite driver) |
| Sessions | `~/.vesvai/sessions.db` (SQLite driver) |
| Permission approvals | `~/.vesvai/permissions.json` |
| Global skills | `~/.vesvai/skills/` |
| Global rules | `~/.vesvai/rules/` |
| Downloaded LSP binaries | `~/.vesvai/lsps/` |
| Project data | `<project>/.vesvai/` |
| Project MCP config | `<project>/.mcp.json` |
| Project LSP config | `<project>/.lsp.json` |

## Uninstall

```bash
rm "$(command -v vesvai)"    # remove the binary
rm -rf ~/.vesvai             # remove config, sessions, logs, cache, skills, rules
```

Project-local data lives in each project's `.vesvai/` directory and can be removed
individually.
