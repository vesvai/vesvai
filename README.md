<div align="center">

```
__        __                   _
\ \      / /__  _____   ____ _(_)
 \ \    / / _ \/ __\ \ / / _` | |
  \ \  / /  __/\__ \\ V / (_| | |
   \_\/_/ \___||___/ \_/ \__,_|_|
```

# Vesvai

**An AI coding agent that speaks your language, runs your way, and stays out of your way.**

[Docs](https://docs.vesv.ai) · [GitHub](https://github.com/vesvai/vesvai) · [Discussions](https://github.com/vesvai/vesvai/discussions)

</div>

---

Vesvai is not another chat wrapper. It's a full agent engine written in Go — provider-agnostic, sandboxed, and designed to work the way you already work. No hosted service, no vendor lock-in, no magic that breaks when you look away.

Run it in your terminal. Run it headless in CI. Embed it in your own tool. Point it at any LLM provider you want. The agent reads your code, edits files, runs commands, searches the web, and delegates to specialist subagents — all while you stay in control.

---

## Pick Your Interface

<table>
<tr>
<td width="50%" valign="top">

### `vesvai` (TUI)

The full experience. A terminal UI built on tcell with streaming markdown, syntax-highlighted diffs, live context/cost tracking, and 26 themes.

```bash
curl -fsSL https://vesv.ai/install | bash
vesvai
```

</td>
<td width="50%" valign="top">

### `vesvai run` (Headless)

Pipe it. Script it. Drop it in a GitHub Action. Same engine, zero interaction.

```bash
vesvai run "Review this PR for bugs"
git diff main | vesvai "Summarize changes"
```

</td>
</tr>
<tr>
<td width="50%" valign="top">

### HTTP API

REST endpoint with SSE streaming. Build dashboards, Slack bots, or your own frontend.

```bash
vesvai serve --port 8080
curl -N http://localhost:8080/v1/chat -d '{"message":"hello"}'
```

</td>
<td width="50%" valign="top">

### Go SDK

Embed the engine directly. No subprocess, no network hop. Your program, your agent.

```go
eng, _ := sdk.Open(ctx, sdk.Options{Workspace: "."})
resp, _ := eng.Chat(ctx, sdk.ChatRequest{Input: "Fix the leak"})
```

</td>
</tr>
</table>

---

## What It Actually Does

**Reads your project.** Vesvai builds a mental model of your codebase — file structure, dependencies, conventions. It references code with `file:line` precision.

**Edits with diffs.** Every change shows as a reviewable diff. Revert what you don't like. Sessions track everything so you can rewind.

**Runs your commands.** Install deps, run tests, start servers. Long-running processes stay in the background. Errors get caught and fixed in real time.

**Plans before it acts.** Toggle Plan mode to let the agent explore, ask questions, and strategize. Then switch to Act mode and watch it execute — with approval gates on every change.

**Delegates to specialists.** The orchestrator spawns subagents (explorer, planner, developer) that run concurrently with their own tool access and context. They persist across turns.

**Stays sandboxed.** A virtual filesystem respects `.gitignore` and `.vesvaignore`. The agent can't escape your project. Five permission modes, an LLM judge for risky ops, and double-`Esc` emergency stop.

---

## 32 Providers. Your Keys. No Proxy.

Vesvai talks directly to LLM APIs using your credentials. Nothing goes through a hosted service.

```
anthropic   openai        google       deepseek     groq
openrouter  mistral       xai          cerebras     ollama
cohere      fireworks     together     sambanova    nvidia
```

Plus AI21, Anyscale, Baichuan, DeepInfra, Lepton, MiniMax, Moonshot, Novita, Perplexity, StepFun, Tencent, Upstage, Voyage, Writer, Yi, Zhipu, and any OpenAI-compatible endpoint.

Point it at a local model. Point it at a corporate gateway. Point it at whatever you want.

---

## Extend It

<div align="center">

| Skill | MCP Server | Plugin |
|-------|------------|--------|
| Markdown rules the agent loads on demand | Connect to databases, APIs, cloud services | Go hooks for logging, auditing, custom tools |
| `.vesvai/skills/` or project-level | Community servers or ask Vesvai to build one | WordPress-style |

</div>

**Skills** are markdown files with frontmatter. Drop them in `.vesvai/skills/` and the agent picks them up:

```markdown
---
description: How we write tests in this project
---
Always use table-driven tests. Never use external test fixtures.
```

**MCP servers** give the agent superpowers it doesn't have natively. Query your database. Manage your K8s cluster. Send a Slack message. Use [community servers](https://github.com/modelcontextprotocol/servers) or build your own.

**Plugins** tap into the event bus. Middleware for secret redaction, permission enforcement, or custom logging — all hookable without forking.

---

## Quick Start

```bash
# Install
curl -fsSL https://vesv.ai/install | bash

# Add your first provider
vesvai login

# Run it
vesvai
```

Or with npm:

```bash
npm i -g vesvai
```

Or build from source:

```bash
git clone https://github.com/vesvai/vesvai
cd vesvai
make build
./bin/vesvai
```

---

## Build

```bash
make build   # Compile binary
make test    # Run tests
make lint    # Run golangci-lint
make fmt     # Format with gofmt
```

Requires Go 1.26.5+. No runtime dependencies — the binary is self-contained.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Join [Discord](https://discord.gg/vesvai) → `#contributors`.

---

<div align="center">

**[Apache 2.0](./LICENSE) © 2026 Vesvai**

</div>
