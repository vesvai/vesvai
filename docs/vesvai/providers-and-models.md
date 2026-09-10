---
icon: lucide/bot-message-square
---

# Providers & Models

Vesvai talks directly to LLM APIs using your own credentials. A **provider** is a
named, preconfigured endpoint; a **driver** is the wire protocol it speaks.

## Providers

32 providers are registered out of the box. Every provider exposes a model list that
Vesvai fetches at startup and caches locally.

| Provider | Driver | Default base URL |
|---|---|---|
| `ai21` | openai | `https://api.ai21.com/studio/v1` |
| `anthropic` | claude | `https://api.anthropic.com` |
| `anyscale` | openai | `https://api.endpoints.anyscale.com/v1` |
| `baichuan` | openai | `https://api.baichuan-ai.com/v1` |
| `cerebras` | openai | `https://api.cerebras.ai/v1` |
| `cohere` | openai | `https://api.cohere.com/compatibility/v1` |
| `deepinfra` | openai | `https://api.deepinfra.com/v1/openai` |
| `deepseek` | openai | `https://api.deepseek.com/v1` |
| `fireworks` | openai | `https://api.fireworks.ai/inference/v1` |
| `google` | gemini | `https://generativelanguage.googleapis.com/v1beta` |
| `groq` | openai | `https://api.groq.com/openai/v1` |
| `lepton` | openai | `https://api.lepton.ai/v1` |
| `minimax` | openai | `https://api.minimax.chat/v1` |
| `mistral` | openai | `https://api.mistral.ai/v1` |
| `moonshot` | openai | `https://api.moonshot.cn/v1` |
| `novita` | openai | `https://api.novita.ai/v3/openai` |
| `nvidia` | openai | `https://integrate.api.nvidia.com/v1` |
| `openai` | openai | `https://api.openai.com/v1` |
| `opencode-go` | openai | `https://opencode.ai/zen/go/v1` |
| `opencode-zen` | openai | `https://opencode.ai/zen/v1` |
| `openrouter` | openai | `https://openrouter.ai/api/v1` |
| `perplexity` | openai | `https://api.perplexity.ai` |
| `sambanova` | openai | `https://api.sambanova.ai/v1` |
| `stepfun` | openai | `https://api.stepfun.com/v1` |
| `tencent` | openai | `https://hunyuan.cloud.tencent.com/v1` |
| `together` | openai | `https://api.together.xyz/v1` |
| `upstage` | openai | `https://api.upstage.ai/v1/solar` |
| `voyage` | openai | `https://api.voyageai.com/v1` |
| `writer` | openai | `https://api.writer.com/v1` |
| `xai` | openai | `https://api.x.ai/v1` |
| `yi` | openai | `https://api.lingyiwanwu.com/v1` |
| `zhipu` | openai | `https://open.bigmodel.cn/api/paas/v4` |

### Authentication

| Driver | Auth mechanism |
|---|---|
| openai | `Authorization: Bearer <api_key>` |
| claude | `x-api-key: <api_key>` plus `anthropic-version: 2023-06-01` |
| gemini | `?key=<api_key>` query parameter |

Requests also send `HTTP-Referer: https://github.com/vesvai/vesvai` and
`X-Title: vesvai` unless you override the headers in config.

## Adding a provider

### Interactive

```bash
vesvai login
```

Pick a provider from the list and paste the API key (input is masked). The key may be
left empty for endpoints that do not need one.

### Non-interactive

```bash
vesvai login --provider groq --api-key gsk_...
```

The provider is synced first — Vesvai fetches its model list and only saves the
provider if that succeeds (35 second timeout). This guarantees a freshly added
provider has usable models immediately.

### TUI

Open Settings with `Ctrl+P` → **General** → **Provider**. Existing providers show
*Use existing configuration* / *Configure again*; new ones prompt for an API key.

### Config file

Keys are stored in plaintext in `~/.vesvai/vesvai.json`:

```json
{
  "providers": [
    { "provider": "openai", "api_key": "sk-..." },
    { "provider": "groq", "api_key": "gsk_...", "timeout": 60 }
  ]
}
```

Keys are masked in `vesvai providers list` and `vesvai config show`.

## Custom and OpenAI-compatible endpoints

To use a gateway, proxy, or local runtime, configure a raw **driver** instead of a
named provider. `base_url` is required for drivers:

```json
{
  "providers": [
    {
      "driver": "openai",
      "base_url": "http://localhost:11434/v1",
      "api_key": "ollama",
      "timeout": 120,
      "headers": { "X-Custom": "value" }
    }
  ]
}
```

The same works for `"driver": "claude"` and `"driver": "gemini"` endpoints that
implement those wire protocols.

## Managing providers

```bash
vesvai providers list                    # name, masked key, cached model count
vesvai providers refresh                 # re-fetch models for every provider
vesvai providers refresh --provider xai  # re-fetch one provider
vesvai providers remove groq             # remove provider and its cached models
vesvai models                            # list cached models
vesvai models --provider deepseek        # filter by provider
```

Model lists are cached in `~/.vesvai/cache.db` (or `cache.json` with the JSON cache
driver). `vesvai cache clear` empties the cache; `vesvai providers refresh` bypasses
it and fetches from the network.

## Model selection

When a run starts, Vesvai resolves a provider/model in this order:

1. **Explicit** — `--provider` and `--model` must match a known model (by ID or
   display name), otherwise the run fails.
2. **Model only** — the model is searched across all providers. A single match is
   used automatically; multiple matches open a picker.
3. **Provider only** — a picker lists that provider's models.
4. **Preferred** — the provider/model of the most recent non-subagent session in the
   current project directory, falling back to the first model of the first provider.
   Resolution has a 30 second timeout.

```bash
vesvai run --provider openai --model gpt-4o "Explain this error"
vesvai run --model deepseek-chat "Refactor this function"
vesvai run --select-model "Do something"
```

The same resolution logic backs the HTTP `/api/run` endpoint and the ACP server.

## Model metadata

Model lists are enriched with metadata fetched from
`https://models.opencode.ai/api.json` and cached under the `models_cache` key. For
each model this includes:

| Field | Used for |
|---|---|
| `max_input_tokens`, `max_output_tokens`, `max_tokens` | Context-window usage indicator and request limits |
| `input_cost_per_token`, `output_cost_per_token`, `cache_read_cost_per_token` | Cost display in the TUI status bar and session usage |
| `supports_reasoning`, `reasoning_options` | The TUI **Reasoning** setting (`default`, `low`, `medium`, `high`) |
| `modalities.input` | Whether image/audio attachments are allowed |
| `tool_call`, `structured_output`, `temperature` | Capability checks |

Lookup tries `provider/model` first, then the bare model ID, then a substring
fallback. Unknown models still work; they simply have no metadata.

## Reasoning effort

Models that support reasoning expose effort levels. Set it per run in the TUI
(Settings → General → Reasoning) or via the SDK's `ReasoningEffort` field. The value
is persisted with the session and shown in the status bar.

## Attachments

Image and audio attachments are validated against the model's input modalities before
sending. If the model does not support them, Vesvai refuses the attachment with an
error instead of failing at the API. See
[Adding Context](features/adding-context.md).

## Failures and retries

The built-in `retry` middleware wraps every LLM call in all agents:

- Temporary provider errors (HTTP `429` and `5xx`) are retried with a backoff of
  **1s, 3s, 10s, 30s, 1m, 5m**, then every 5 minutes.
- Other errors are retried too, unless the run was cancelled.
- Streaming calls are only retried before any content, reasoning, or tool call has
  been emitted.
- While waiting, the CLI and TUI show a transient
  `Request failed (attempt N): ... retrying in Xs` message.

There is currently no automatic failover to a secondary provider — configure retries
and switch models manually if a provider is down.

## Troubleshooting

```bash
vesvai doctor                            # connectivity + model count per provider
vesvai logs --level ERROR                # recent errors
vesvai providers refresh --provider X    # force a fresh model fetch
```

See [Troubleshooting](troubleshooting.md) for details.
