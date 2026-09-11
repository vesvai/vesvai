---
icon: lucide/heart-handshake
---

# Subagents

Vesvai is built around a main **orchestrator** agent that can delegate work to
specialized **subagents**. Each subagent runs its own loop with its own tools,
prompt, and middleware, and runs **concurrently** with its siblings.

## The agent roster

| Agent | Role | Write access | Tools |
|---|---|---|---|
| `orchestrator` | Top-level coordinator | Full workspace | All file tools, `askuserquestion`, `bash`, `subagent`, `wait-for-subagents`, `subagents-status`, `subagent-message`, `todoread`, `todowrite` |
| `planner` | Architecture and planning | `.vesvai/plans/` only | File tools write-scoped to plans, `bash`, `webfetch`, `websearch`, `todoread`, `todowrite` |
| `developer` | Implementation | Full workspace except `.vesvai/plans/` | All file tools, `bash`, `webfetch`, `websearch`, `todoread`, `todowrite` |
| `explorer` | Read-only research | None | `glob`, `grep`, `list`, `read`, `bash`, `webfetch`, `websearch` |

All agents share a common system prompt that establishes the interaction style
(tone, proactiveness, code conventions), injects the working environment, git
status, the [skills list](../configurations/skills.md), [AGENTS.md and rules](../configurations/rules.md), and
requires code references in `file:line` format. Every agent runs the same middleware
chain: loop detection, secret redaction, retry, and permissions.

By default an agent may run up to **150 iterations** per turn.

## Spawning subagents

The orchestrator uses the `subagent` tool to delegate. It accepts one or more
subagents and runs them all concurrently:

```json
{
  "subagents": [
    { "name": "explore-api", "agent": "explorer",
      "task": "Find all REST endpoints in internal/server and report file:line references." },
    { "name": "plan-refactor", "agent": "planner",
      "task": "Design a plan to split the server package. Write it to .vesvai/plans/." }
  ],
  "background": false
}
```

| Field | Description |
|---|---|
| `name` | Unique, descriptive name for this run |
| `agent` | One of `orchestrator`, `explorer`, `planner`, `developer` |
| `task` | Instructions: what to do, constraints, what to report back |
| `task_id` | Optional todo ids this subagent is working on |
| `background` | `false` (default) waits for all to finish; `true` returns immediately |

Foreground mode blocks until every subagent completes or fails. Background mode
returns a confirmation immediately and lets the orchestrator continue; results are
collected later.

## Delegation details

- Subagents inherit the parent's **provider**, **model**, and **event bus** — they
  stream into the same transcript.
- In the TUI, each subagent is a card with a live activity line, an output preview,
  usage, and a **History** view of its full transcript.
- The `task_id` field links a subagent to todo items, so progress is trackable via
  the todo tools.

## Working with background subagents

| Tool | Purpose |
|---|---|
| `subagents-status` | Check status: `pending`, `running`, `completed`, `failed`, `interrupted` |
| `wait-for-subagents` | Block until the named subagents finish and return their results |
| `subagent-message` | Send a follow-up message to a finished subagent, resuming it with its full history |

`subagent-message` re-instantiates the subagent (same agent type, same provider and
model), prepends its system prompt and stored conversation, and runs it again. This
is how you iterate on a completed investigation without losing context.

## Persistence and restart

Subagent state is persisted to `.vesvai/subagents.json` in the project directory.
If Vesvai restarts while subagents are running, those subagents are marked
`interrupted` (with reason "app restarted"). You can inspect them with
`subagents-status` and resume one with `subagent-message`.

Subagent conversations are also stored as sessions, so their history is durable and
replayable.

## `/batch` orchestration

For large, parallelizable changes, invoke the `/batch` skill. It turns the
orchestrator into a batch manager that explores, plans, delegates to subagents,
verifies results, and reports back — following the subagent lifecycle described
above. See [Commands](commands.md#batch).