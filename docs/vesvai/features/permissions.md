---
icon: lucide/shield-check
---

# Permissions

The permission middleware decides which [tool](../configurations/tools.md) calls may
run. It combines a per-tool mode, a filesystem sandbox, and — depending on the mode
— an interactive prompt or a judge LLM.

## Modes

| Mode | Behavior |
|---|---|
| `allow` | Always runs. The filesystem sandbox is lifted for the call |
| `semi-ask` | Runs first; only asks the user if the sandbox denies it (or the tool is gated by policy) |
| `ask` | Always asks the user before running |
| `semi-judge` | Runs first; only sends to the judge LLM if the sandbox denies it |
| `judge` | Always sends to the judge LLM before running |

The effective mode for a tool is `permission.rules[<tool>]`, falling back to
`permission.default`, falling back to `semi-ask`. An invalid configured mode parses
to `ask`.

## Built-in defaults

| Tool | Default mode |
|---|---|
| `read`, `write`, `edit`, `delete`, `list`, `glob`, `grep` | `semi-ask` |
| `bash` | `semi-judge` |
| `todoread`, `todowrite`, `subagent`, `wait-for-subagents`, `subagents-status`, `subagent-message` | `allow` |
| `ask` | `allow` (never gated) |
| `websearch`, `webfetch` | `semi-ask` |

MCP tools default to `permission.default`. Override any tool with the
`permission.rules` map:

```json
{
  "permission": {
    "default": "semi-ask",
    "rules": { "bash": "ask", "write": "allow" },
    "judge_provider": "openai",
    "judge_model": "gpt-4o-mini"
  }
}
```

## The filesystem sandbox

File tools operate inside a virtual filesystem rooted at the workspace. Escaping the
root — absolute paths outside the workspace, `..` traversal, symlink escapes — fails
with an out-of-bounds error. `.gitignore` and `.vesvaignore` rules hide ignored
files.

For `semi-ask` / `semi-judge`, the tool runs first: if it succeeds, no prompt or
judge is involved. Only an out-of-bounds result triggers the gate. After the user or
judge approves a path, the call is re-run with that specific path permitted. An
`allow`-mode call runs with the sandbox fully lifted.

The planner agent is write-scoped to `.vesvai/plans/`; the explorer is read-only.

## Bash policy

`bash` uses `semi-judge`, but simple, safe commands never reach the judge:

- The command starts with a whitelisted binary — `go`, `npm`, `ls`, `pwd`, `node`,
  or `git` — **and**
- Contains no shell metacharacters (`;`, `|`, `&`, `>`, `<`, `$`, whitespace
  control chars, `&&`, `||`).

Everything else is gated. Interactive shells or destructive commands therefore
trigger a judge or prompt, while `git status` and `npm run build` run directly.

## The prompt flow

When a call needs approval, the user is asked (in the TUI and CLI):

```
Allow the "bash" tool call?
  Allow / Allow All / Reject
```

| Choice | Effect |
|---|---|
| **Allow** | Runs this call once |
| **Allow All** | Runs this call and remembers it; same arguments never prompt again |
| **Reject** | Denies the call; an optional reason is collected and remembered |

Dismissing the prompt denies the call. The `ask` tool is exempt from all gating.

## The judge flow

In `judge` / `semi-judge` mode a dedicated **judge agent** reviews the tool call
against the last few conversation messages and returns a structured
`{allow, reason}` verdict. Configure which model judges with
`judge_provider` / `judge_model`; by default the preferred model is used.

## Remembered decisions

Allow and reject decisions are stored in `~/.vesvai/permissions.json`, keyed by a
hash of the canonicalized tool arguments.

- A previously allowed call runs immediately with the sandbox permitted.
- A previously rejected call fails immediately with
  `permission denied for tool "<name>": <reason>` (marked *previously rejected*).

Reset all remembered decisions:

```bash
rm ~/.vesvai/permissions.json
```

## Denied calls

A denied call fails the tool execution with `permission denied for tool
"<name>": <reason>`. The agent sees the denial as a tool error and should adjust
its approach.

## Interrupting a run

Pressing ++esc++ twice within two seconds in the TUI cancels the running agent and
all subagents — including any in-flight permission prompt. See
[TUI](../usage/tui.md#interrupting-the-agent).