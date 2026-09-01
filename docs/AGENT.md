# Vesvai System Architecture & Specification Document

## Core Architecture
*   **Event-Driven Design:** The entire system must operate on an event-driven architecture utilizing a high-performance event bus.
*   **Hook System (Filters & Actions):** The system must feature a WordPress-style hook system. All providers, drivers, agents, and tools must be registrable via hooks. Hooks must be placed throughout the entire codebase, making the system 100% extendable via plugins.
*   **Lightweight & High Performance:** The system must run with minimum memory footprint and maximum performance, utilizing buffers where necessary. Background processes (like loading skills or initializing LSPs) must not block the user. Users sending messages should only wait for pending background tasks (LSP/MCP) to finish if required.
*   **Headless Daemon & Webhook Gateway:** The core system can run as a background daemon (`vesvai server`) exposing a REST and WebSocket API. This strictly enforces the UI/Core separation, allowing CLI clients, IDE extensions, or future Web interfaces to connect to the same core engine simultaneously.
*   **Secret Redaction Middleware:** A global middleware layer on the event bus and LLM request pipeline that automatically detects and redacts sensitive patterns (API keys, JWT tokens, private keys) before prompts are transmitted to third-party LLM providers.
*   **Plugin System:** Users can install plugins under the `~/.vesvai/plugins` directory. Thanks to the hook system, plugins can modify or extend the core system safely.
*   **Global & Project Configurations:** The system must read both global (`~/.vesvai/`) and project-level configs (e.g., `.claude`, `.agents`). Adding new configurations should be modular and easy.
*   **UI/Core Separation:** The presentation layer (CLI, TUI, future Web/Mobile) must strictly consume the core system's APIs, interfaces, hooks, and events. It must contain zero core logic to ensure future cross-platform compatibility.

## LLM & Provider Management
*   **Multi-Provider Support:** Support for multiple LLM providers. For example, an OpenAI driver can power providers like `opencode`, `groq`, `openai`, etc. Dedicated drivers and providers for Claude, Gemini, etc., must also exist.
*   **Circuit Breakers & Rate Limit Fallbacks:** The provider pipeline includes robust circuit breakers. If a provider hits rate limits (HTTP 429) or experiences outages (HTTP 5xx), the circuit breaker trips and automatically falls back to a user-configured secondary provider or model without disrupting the Orchestrator's workflow.
*   **Multimodal Capabilities:** LLMs must support file and audio inputs, and if supported by the model, generate multimodal outputs.
*   **Standardized LLM Structs:** `internal/llm/` must contain standardized core types (`Response`, `Request`, `Message`, `Tool`, `Attachment`, etc.) shared across all providers.
*   **Drivers & Providers Architecture:** Drivers should be located under `internal/llm/drivers/` and providers under `internal/llm/providers/`.
*   **Advanced Parameter Handling:** Complete control over reasoning, tool calls, `temperature`, `n` values, etc. No missing configuration options.
*   **Streaming:** All LLMs must fully support response streaming.
*   **Provider Registration & Health Checks:** Providers are registered globally under `~/.vesvai/`. When a new provider is added, the system must ping its `/models` endpoint. If it fails, return an error. If successful, cache the provider's model list under `~/.vesvai/` for user selection.
*   **Model Duplication Awareness:** The system must gracefully handle the same model existing across multiple different providers.
*   **Model Capabilities API:** A dedicated GET endpoint must provide system model metadata (ID, context window size, image in/out support, audio in/out support, etc.).
*   **Utils:** Shared utilities like HTTP clients must be located under `internal/utils/`.

## Agentic System
*   **Custom Agent Framework:** A highly advanced, custom agent system located under `internal/agent/`. Agents must be fully configurable (model, system prompt, temperature, etc.) and support middleware.
*   **System Agents:** Default agents are defined under `internal/agents/`. These can be used as standalone agents or as sub-agents for others.
*   **Dynamic System Prompts:** Agents must support dynamic, regex-based system prompts tied to models (e.g., specific prompts for `gemini-*`, `gpt-*`, and fallbacks). Prompts can be Markdown or XML formatted and contain dynamic variables. A robust Prompt Builder must be included.
*   **Smart Router (Dynamic LLM Selection):** While users can manually select models, the system supports dynamic routing. A dedicated routing LLM (configured in `~/.vesvai/`) evaluates the API model capabilities, benchmarks (e.g., SWE-bench), and task difficulty to optimize performance and token cost. It dynamically routes orchestrators to high-context models, planners to reasoning models, minor tasks to cheaper models, and image inputs to vision-capable models.
*   **Task Focus:** Every agent must focus entirely on executing its assigned task in the best possible way.
*   **Scalability:** The architecture must seamlessly handle everything from minor quick tasks to massive, multi-hour projects.

### Agent Roles & Orchestration
*   **Main Orchestrator Agent:** The primary controller. It delegates tasks to sub-agents and avoids doing direct work, though it can take initiative for very minor, obvious changes. Its core goal is successful task completion via delegation. 
    *   **Autonomous Skill Creation:** If the Orchestrator researches and learns a new concept (e.g., an undocumented API structure or a novel project setup), it can autonomously create and save a new skill for future sessions to reuse.
*   **Explorer Sub-agent:** Used to understand pre-existing projects. The Orchestrator deploys it to analyze the entire system architecture and return a comprehensive summary.
*   **Planner Sub-agent:** Creates highly detailed execution plans. Returns data to the Orchestrator in a table format including: `id`, `title`, `description`, `depends_on`, and `parallel_with`.
*   **Developer Sub-agent:** The primary coding agent. Tasked by the Orchestrator to write and implement code.
*   **Testing & QA Sub-agent:** An automated regression loop. The Orchestrator deploys this agent to run test suites (pytest, jest, go test) after Developer edits. It parses `stderr`/`stdout` test failures and feeds them directly back to the Developer sub-agent in a self-correcting loop before marking tasks complete.
*   **Dynamic Skill & Role Assignment:** The Orchestrator can dynamically assign skills to sub-agents upon creation. It assigns Task IDs and **must** give sub-agents logical, descriptive names based on their task for user traceability.
*   **Sub-agent Messaging:** A detailed messaging protocol. Sub-agents can communicate with each other or the Orchestrator, but only when necessary (e.g., to ask questions or report blockers).
*   **Sub-agent Persistence (Wake-up):** Sub-agents are not destroyed upon task completion. They remain dormant and can be "woken up" by the Orchestrator via new messages if issues arise later, saving initialization costs and preserving context.
*   **Parallel Background Execution:** The Orchestrator can run sub-agents in parallel in the background or wait for them. A `wait_for` tool (accepting agent names) allows the Orchestrator to sleep until specific sub-agents finish, waking automatically upon completion to prevent infinite loops.

## Tools, Skills & Filesystem
*   **Tool Registration:** System tools reside under `internal/tools/`, are registered to the hook system, and can be assigned to agents using their string names.
*   **Core Tools:**
    *   **Bash Tool:** Highly secure execution.
    *   **File Tools:** `read`, `edit`, `write`.
    *   **List Tools:** `glob`, `grep`.
    *   **Todo Tools:** `list-todo`, `update-todo` (add, delete, etc.).
    *   **Web Tools:** `web-search` (using DuckDuckGo HTML with custom parser) and `web-fetch` (formats HTML to Markdown via `html-to-markdown/v2`).
    *   **Ask-User Tool:** Allows agents to dynamically query the user (supports images, videos, multiple choice, or batched questions).
*   **Dynamic Skill Engine (`internal/skill/`):** The system features a dedicated module for managing reusable knowledge and workflows.
    *   **Registration & Listing:** The `internal/skill` module is responsible for loading, parsing frontmatter metadata, registering, and listing all available skills (both global and project-specific).
    *   **Agent-Driven Creation:** Agents (primarily the Orchestrator) possess the ability to dynamically generate and persist new skills. If an agent learns a complex workflow, API, or project pattern, it can call a specific function/tool to format and save this knowledge as a new skill under the project's or global `.vesvai/skills/` directory for immediate future use.
*   **Custom Filesystem & Permissions:** The system uses a custom virtual filesystem for security.
    *   Respects `.gitignore` and `.vesvaignore` to hide files from the agent.
    *   Detects and blocks agents trying to escape the project directory.
    *   Project permissions are stored in `.vesvai/`. Instead of auto-denying blocked actions, it prompts the user for permission. Denials are recorded to auto-deny future identical requests.
    *   Global permissions stored dynamically in `~/.vesvai/` (e.g., setting the Bash tool to `AllowAll` via middleware). Tool default is `AllowAll`.
*   **Judge LLM (Pre-execution Validation):** Sensitive tools like `bash` have a Judge system. Before execution, the tool prompt and partial chat history are sent to a Judge LLM (configured in `~/.vesvai/`) for automatic approval, bypassing manual user prompts.
*   **Smart Edit Validation:** If the `edit` tool is called on a file the LLM hasn't read, it throws an error forcing a read. File hashes are stored after reading; if the file is modified externally before an edit, the system throws an error forcing the LLM to re-read it.
*   **LSP Support:** Language Server Protocol support. Defined globally in `~/.vesvai/` or per-project in `vesvai.json`.
    *   After `read`, `edit`, or `write` tool executions, LSP diagnostics are returned to the agent. Errors automatically trigger a feedback loop to the agent.
*   **MCP Support:** Model Context Protocol support. Remote or local MCPs can be defined in `vesvai.json` or globally in `~/.vesvai/`. MCP tools are injected into the agent.

## Context, State & Memory Management
*   **Session Database:** SQLite-based session management. All history is persisted.
*   **Main vs. Sub-Sessions:** The user's conversation with the Orchestrator is the Main Session. Conversations with/between sub-agents are saved as sub-sessions linked via `parent_id`.
*   **Deterministic Replay / Forking:** Because all interactions are saved to the database, users can "fork" a session from a specific past state (message/tool execution) to explore alternative agent trajectories or debug failures.
*   **Session Restoration:** Users can load past sessions and even send manual messages directly to sub-agents within restored sessions.
*   **Auto-Titling:** A background LLM (configured in `~/.vesvai/`) automatically generates a 3-5 word title based on the first session message.
*   **Persistent Todos:** Tasks generated by the Planner are saved as Todos in the project's `.vesvai/` directory for cross-session persistence.
*   **Context Compacter System:** When an agent's context window fills:
    *   The Compacter Agent summarizes recent messages to free up space.
    *   If it fills again, it performs a deep compaction, merging the entire history into a single, highly dense summary message. The agent resumes using only this compacted message, discarding raw history.
*   **AST-Aware Context Pruning:** To maximize context efficiency, the system utilizes Abstract Syntax Tree (AST) parsing for supported languages. Instead of dumping entire 2,000-line files into the prompt, the agent can request a lightweight symbol graph (function signatures, struct definitions) and pull only the specific code blocks it needs.
*   **Message Type Segregation:** Inter-agent messages must be stored as distinct types in the database to allow clear visual separation in future Web/TUI interfaces, even if they are ultimately sent as prompts.

## Error Handling & System Resilience
*   **Domain-Driven Error Standardization:** The system enforces strict, typed errors (e.g., `ErrProviderRateLimit`, `ErrContextExceeded`, `ErrLSPTimeout`, `ErrToolExecutionFailed`). This ensures the event bus and UI can react deterministically to specific failures.
*   **Agentic Self-Correction (Error Feedback Loop):** System errors (like a failing bash command, malformed JSON, or a rejected Git push) do not crash the session. Instead, they are caught, wrapped in a system message, and fed back to the active agent. The agent is prompted to read the error log (e.g., `stderr`) and attempt a fix on its own.
*   **Checkpointing & Snapshotting Engine:** For massive, multi-hour projects, the system takes periodic SQLite snapshots of the state graph (active sessions, memory, open files, pending todos). If the host machine loses power or panics, the system resumes seamlessly from the last checkpoint without wasting LLM tokens or losing work.
*   **Graceful Degradation:** If non-critical background services fail (e.g., an LSP crashes or the notification daemon hangs), the system logs the error to the console/TUI but allows the LLM Orchestrator to continue working uninterrupted.
*   **Panic Recovery (Global Catch):** Critical subsystems and goroutines/threads utilize global panic recovery wrappers to prevent application crashes. Fatal errors safely flush buffers and commit final session states to SQLite before gracefully shutting down the daemon.

## Interactions & Notifications
*   **Notification System:** Hook-driven dynamic notifications (e.g., when the `ask-user` tool is triggered, or the main LLM finishes). Initial implementation uses `github.com/gen2brain/beeep` for desktop notifications, designed to be extendable to Web/Mobile push via plugins.
*   **Prompt Injectors (Shortcuts):**
    *   **Skills (`/`):** Typing `/skill-name` replaces the text with the actual skill prompt content before sending.
    *   **Mentions (`@`):** Used to tag attachments (e.g., `@uploaded-image.png`), force route to specific agents (e.g., `@explorer`), or force the Orchestrator to prioritize a specific file (e.g., `@src/index.tsx`).

## Command Line Interface (CLI)
*   **Standardized CLI Commands:** A highly capable CLI using standard flags and commands (`vesvai [command] [flags]`). The CLI interacts directly with the core engine.
*   **Config & File Locations:** 
    *   `~/.vesvai/config.json` -> Global configurations, default models, API keys.
    *   `~/.vesvai/providers/` -> Cached provider and model data.
    *   `~/.vesvai/plugins/` -> Global system plugins.
    *   `./.vesvai/` (Project Dir) -> Project-specific session databases, permissions, ignores, skills, and cross-session todos.
*   **Provider Management via CLI:** Commands to manage LLM providers quickly without modifying JSON files directly.
    *   `vesvai provider add <name> --api-key <key> --driver <driver>`
    *   `vesvai provider list`
    *   `vesvai provider check <name>` (Pings the `/models` endpoint to verify health).
*   **Server Command:** `vesvai server --port <port>` starts the core event-driven backend in daemon mode, allowing the TUI or API clients to connect.
*   **Direct Run Command (`vesvai run`):** Allows users to pass a prompt directly into the terminal without opening the TUI. The output (agent thoughts, tool executions, final answer) streams natively to `stdout`/`stderr`.
    *   Example: `vesvai run "Create a react auth context" --model deepseek-v4 --temp 0.2 --agent orchestrator`
*   **Deep Debug Mode:** Accessed via the `--debug` flag or `VESVAI_DEBUG=1` environment variable.
    *   Hooks directly into the central Event Bus.
    *   Streams verbose logs of *every* system event to the terminal: exact JSON HTTP requests to providers, raw tool execution outputs, AST extraction logs, sub-agent spawning events, context compaction triggers, and hook fires. 
    *   Crucial for plugin developers and system debugging.

## Terminal User Interface (TUI)
*   **Framework:** Built using `tcell`.
*   **Textarea Input:**
    *   Starts centered. Supports `Ctrl`/`Shift` keybinds.
    *   Auto-expands from 3 up to 6 rows.
    *   Typing `/` or `@` (as new words) opens a dropdown list navigable with arrow keys and Enter. Live filtering as the user types.
    *   Pasted files/images automatically appear as attachment pills above the textarea.
*   **Chat Viewport:** Upon sending a message, the textarea animates down to the bottom, revealing the chat viewport.
*   **Message & Tool Rendering:**
    *   LLM text, Tool usage, and "Thinking..." states must be visible.
    *   "Thinking..." should blink/animate while active, becoming solid when finished.
    *   **Custom Tool UIs:** E.g., `Read` just shows the filename. `Write` shows a truncated, highlighted snippet of the code. `Edit` shows a highlighted diff. Full syntax highlighting support required.
*   **Command Palette (Ctrl + P):** Opens a modal menu for: New Session, Load Session, Connect Provider, Change Model, etc.
    *   *Load Session:* Opens a searchable, paginated list of main sessions. Switching sessions pauses the current one (with an optional confirmation modal).
*   **High-Performance Lazy Loading:** When loading a session, only recent messages/tools are fetched. Scrolling up triggers paginated lazy loading of the history.
*   **Emergency Stop:** Pressing `ESC` twice during generation instantly halts the Orchestrator and all running sub-agents.
*   **Inline Sub-agent Rendering:** Sub-agents appear inline in the chat similar to tools.
    *   Displays high-level status (current tool, thinking, answering) rather than raw logs.
    *   Clicking the sub-agent swaps the viewport to that sub-agent's isolated chat history. A "Back" button returns to the Main Agent. Users can interact directly in the sub-agent's chat.
*   **Mouse Support:** The entire TUI must support mouse clicks and scrolling.
*   **Agent View Mode (Node Graph):**
    *   A shortcut toggles from Chat View to Agent View.
    *   Displays a visual node graph (circles) of the active agent architecture.
    *   Sub-agents are connected to their parents.
    *   Active/working agents have animated borders. Idle agents are dimmed.
    *   Inter-agent messaging is visualized on the graph.
    *   Clicking a node reveals details: Name, Current Action, Context Window %, Task ID, and a button to jump directly into its chat.