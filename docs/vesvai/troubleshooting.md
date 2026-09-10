---
icon: lucide/bug
---

# Troubleshooting

## First checks

```bash
vesvai version      # confirm what you are running
vesvai files        # where config, logs, cache, and sessions live
vesvai doctor       # test every configured provider and check for updates
vesvai config show  # print the resolved config (API keys masked)
```

## Logs

Vesvai logs to SQLite by default. Every command, request, tool call, and error is
recorded in `~/.vesvai/logs.db`.

```bash
vesvai logs                          # most recent 50 entries
vesvai logs --level ERROR            # only errors
vesvai logs --search "provider"      # substring search in message and level
vesvai logs --level DEBUG --size 200 --page 2
```

If `logger.driver` is `console`, `vesvai logs` prints only the current process's
in-memory buffer instead. Switch back to `"sqlite"` in
[Config](config.md#logger) to persist logs.

To increase verbosity, set the logger driver to `sqlite` (the default already records
debug-level events) and query with `--level DEBUG`.

## Cache

Model lists and model metadata are cached locally. Stale cache is the usual cause of
"model not found" after a provider adds new models.

```bash
vesvai cache clear                        # clear everything
vesvai cache clear --provider openai      # clear one provider
vesvai providers refresh                  # re-fetch models from the network
vesvai providers refresh --provider openai
```

Cache files: `~/.vesvai/cache.db` (SQLite driver) or `~/.vesvai/cache.json` (JSON
driver).

## Common problems

??? question "`cli: no providers configured` or an empty provider list"

    No provider has been added yet. Run `vesvai login` or add an entry to the
    `providers` array in `~/.vesvai/vesvai.json`.

??? question "`cli: model \"x\" not found`"

    The model is not in the local cache. Refresh and try again:

    ```bash
    vesvai providers refresh
    vesvai models
    ```

    Model matching is case-sensitive and checks both the model ID and its display
    name.

??? question "`cli: timed out selecting model`"

    Model resolution waits 30 seconds for the LLM manager to become ready. This
    happens when providers are still syncing, no provider is reachable, or the cache
    is empty. Run `vesvai doctor` to see which providers fail, then retry. Passing
    `--provider` and `--model` explicitly skips the selection step.

??? question "HTTP 401 / 403 from a provider"

    The API key is missing, expired, or lacks access to the model. Re-run
    `vesvai login --provider <name>`, or check the key in `vesvai config show`.

??? question "HTTP 429 or 5xx responses"

    Temporary errors are retried automatically with a 1s, 3s, 10s, 30s, 1m, 5m
    backoff. If the run keeps failing, wait for the rate limit to reset, switch to
    another provider, or lower concurrency.

??? question "`Model does not support image/audio attachments`"

    The selected model's metadata does not list the attachment modality as an input.
    Choose a multimodal model, or remove the attachment.

??? question "A tool is blocked by a permission prompt or denied"

    Vesvai asks before running gated tools (or consults the judge LLM, depending on
    the mode). Denials are remembered in `~/.vesvai/permissions.json`, keyed by the
    tool call arguments. See [Permissions](features/permissions.md).

    To reset every remembered decision:

    ```bash
    rm ~/.vesvai/permissions.json
    ```

??? question "An MCP server does not connect"

    Check the server definition and run Vesvai with the `sqlite` logger, then inspect
    `vesvai logs --search mcp`. Common causes:

    - The `command` is not on `PATH` (stdio transport).
    - The `url` is not reachable or does not emit an `endpoint` event (SSE transport).
    - The server fails its `initialize` handshake or lists no tools.

    Connection failures are logged as warnings and the server is skipped; the rest of
    Vesvai keeps working.

    ```bash
    vesvai mcp list
    vesvai mcp tools --server db
    ```

??? question "No LSP diagnostics appear in file reads"

    Diagnostics require a language server whose `filetypes` match the file. Servers
    start lazily on first access and are stopped after 10 minutes idle. If the binary
    is missing, Vesvai tries the configured `download` URL or `install` command.

    ```bash
    vesvai lsp list
    vesvai lsp diag --file src/main.go
    ```

    See [LSP](configurations/lsp.md).

??? question "Config file fails to parse"

    `~/.vesvai/vesvai.json` must be valid JSON. Validate it with your editor or
    `jq . ~/.vesvai/vesvai.json`. A missing file is fine; a malformed one aborts
    startup.

??? question "The TUI looks wrong or the terminal is unsupported"

    Try a different theme with `Ctrl+T` (the choice is saved). The TUI enables mouse
    and bracketed paste support when the terminal provides it; if the terminal
    misbehaves, resize the window or switch to the CLI with `vesvai run`.

??? question "The agent keeps running and I want to stop it"

    Press ++esc++ twice within two seconds. The first press shows
    *Press Esc to interrupt*; the second cancels the run and all subagents.
    ++ctrl+c++ quits the whole application.

??? question "Sessions disappeared or moved"

    Sessions are filtered to the current project directory by default. List them all
    with:

    ```bash
    vesvai sessions list --all
    ```

    Session storage lives in `~/.vesvai/sessions.db` (SQLite driver) or
    `~/.vesvai/sessions/` (JSON driver). See [Sessions](features/sessions.md).

## Reset to defaults

Remove pieces selectively, or everything:

```bash
rm ~/.vesvai/permissions.json    # forget tool approvals
vesvai cache clear               # rebuild model cache
rm ~/.vesvai/vesvai.json         # regenerate default config (loses providers)
rm -rf ~/.vesvai                 # full reset: config, sessions, logs, cache
```

## Reporting issues

If the problem persists, open an issue at
[github.com/vesvai/vesvai/issues](https://github.com/vesvai/vesvai/issues) with:

1. The output of `vesvai version` and your OS/architecture.
2. Relevant log lines (`vesvai logs --level ERROR`).
3. A redacted copy of `vesvai config show` (keys are masked already).
4. Steps to reproduce, including the exact command or prompt.
