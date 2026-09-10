---
icon: lucide/square-terminal
---

# Commands

Vesvai's slash commands are implemented as **skills**: type `/name` in the input
and the skill's instructions are injected into the conversation. Three commands are
built in — `/init`, `/batch`, and `/review` — and you can
[add your own](../configurations/skills.md#creating-your-own) with a `SKILL.md` file.

In the TUI, type `/` to open the picker and filter by name. The same token works in
the CLI (`vesvai run "/init"`) and inside subagent task messages.

## `/init`

Instructs Vesvai to analyze a codebase and create or improve an `AGENTS.md` file.
Use it on a repository that lacks agent guidance, or to refresh an outdated one.

The generated file follows a fixed structure:

- Project Overview
- Repository Structure
- Build & Development Commands
- Code Style & Conventions
- Architecture Notes
- Testing Strategy
- Security & Compliance
- Agent Guardrails
- Extensibility Hooks
- Further Reading

Rules the init command follows:

- The file must start with the `# AGENTS.md` heading.
- Content is limited to roughly 12k tokens; keep it dense and factual.
- Unresolved details are marked inline with `> TODO:` so you can fill them in.
- Information the codebase doesn't reveal is not invented — it is left as a
  placeholder or omitted.

The result is a file the agent framework reads on every run as
[Project Instructions](../configurations/rules.md#rules-vs-agentsmd).

## `/batch`

Instructs Vesvai to orchestrate a large, parallelizable change from exploration to
report, using subagents. This is the orchestrator's working contract for multi-file
work.

The workflow:

1. **Decompose** the task into independent chunks that can be parallelized.
2. **Explore** the codebase to map the affected areas.
3. **Plan** by writing a plan file under `.vesvai/plans/YYYY-MM-DD-<feature>.md`.
4. **Delegate** chunks to explorer, planner, and developer subagents via the
   `subagent` tool, running them concurrently.
5. **Verify** the results before accepting them.
6. **Report** the outcome, including what changed and what remains.

Supporting rules the batch command applies:

- Subagent names are unique and task-descriptive; results can be re-used from
  `.vesvai/subagents.json`.
- The plan file is the source of truth for what is being changed and why.
- Todo items use a hierarchy (`todo-1`, `todo-1.1`, ...), priorities, and
  `dependsOn` links so progress and dependencies stay visible.
- Every subagent either proves its work (tests, checks) or reports a failure.

The lifecycle tools — `subagent`, `wait-for-subagents`, `subagents-status`, and
`subagent-message` — back the whole flow. See
[Subagents](subagents.md).

## `/review`

Instructs Vesvai to review a pull request or changeset and produce structured
feedback. Use it when you want a second pass over a diff before merging.

The review pipeline:

1. **Assess size** — determine whether the review can be done in one pass or must be
   partitioned.
2. **Gather context** — an Explorer subagent collects relevant background from the
   codebase.
3. **Review hunks** — Developer subagents review the diff piece by piece, recording
   findings as:

   ```
   - **[File:Line]** [Severity] - [Issue] - [Fix]
   ```

4. **Verify findings** — a Verifier pass filters the list into `[Confirmed]`,
   `[Plausible]`, and `[Refuted]`, dropping noise and duplicates.
5. **Sweep for gaps** — look for issues the hunk-by-hunk review might have missed
   (security, tests, docs).
6. **Report** — produce inline review comments on the VCS, or a Markdown report.

## Other commands

These are the only literal slash commands in the chat loop:

| Input | Effect |
|---|---|
| `exit`, `quit`, `/exit`, `/quit`, `bye` | End the CLI `run --chat` loop |

Anything else starting with `/` that is not a registered skill is left in the
message text. There is no hidden command palette — Settings (++ctrl+p++ in the TUI)
and the `/` picker are the full command surface.