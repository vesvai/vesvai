---
name: init
description: Instructs the /init command to analyze a codebase and create or improve a AGENTS.md file
---

Analyze this codebase and create a AGENTS.md file, which will be given to future instances of Vesvai to operate in this repository.

About AGENTS.md:
Modern AI coding tools converge on a simple idea: give the agent a single, well-structured Markdown file that explains how your repo “works,” and prepend that file to every LLM call so the agent never has to guess about architecture, commands, or conventions. Community gists, RFCs, and vendor playbooks all recommend the same core sections—overview, project map, build/test scripts, code style, security, and other relevant information. This file is intended to be that single source of truth for Vesvai.

What to add:
1. Commands that will be commonly used, such as how to build, lint, and run tests. Include the necessary commands to develop in this codebase, such as how to run a single test.
2. High-level code architecture and structure so that future instances can be productive more quickly. Focus on the "big picture" architecture that requires reading multiple files to understand.

Usage Notes:
- If there's already a AGENTS.md, suggest improvements to it.
- When you make the initial AGENTS.md, do not repeat yourself and do not include obvious instructions like "Provide helpful error messages to users", "Write unit tests for all new utilities", "Never include sensitive information (API keys, tokens) in code or commits".
- Avoid listing every component or file structure that can be easily discovered.
- Don't include generic development practices.
- If there are Cursor rules (in .cursor/rules/ or .cursorrules) or Copilot rules (in .github/copilot-instructions.md), make sure to include the important parts.
- If there is a README.md, make sure to include the important parts.
- Do not make up information such as "Common Development Tasks", "Tips for Development", "Support and Documentation" unless this is expressly included in other files that you read.
- Be sure to prefix the file with the following text:

```
# AGENTS.md

This file provides guidance to Vesvai when working with code in this repository.
```

Required Sections:
1. Project Overview – one-paragraph description and elevator pitch.
2. Repository Structure – bullet list mirroring the directory tree; explain each top-level folder in ≤ 1 sentence.
3. Build & Development Commands – shell-ready commands for install, test, lint, type-check, run, debug, deploy; use fenced code blocks.
4. Code Style & Conventions – formatting rules, naming patterns, lint config, commit-message template.
5. Architecture Notes – high-level diagram in Mermaid **or** ASCII plus a prose explanation of major components and data flow.
6. Testing Strategy – unit, integration, e2e tools and how to run them locally + in CI.
7. Security & Compliance – secrets handling, dependency-scanning, guardrails, license notes.
8. Agent Guardrails – boundaries for automated agents (files never touched, required reviews, rate limits).
9. Extensibility Hooks – plugin points, env vars, feature flags.
10. Further Reading – relative links to deeper docs (docs/ARCH.md, ADRs, etc.).

Style & Rules
* Write in concise, direct English; max line length ≈ 100 chars.
* Use Markdown only—no HTML.
* Prefer ordered lists for sequences, tables only where tabular data adds clarity.
* Do NOT invent details; if information is missing, insert a `> TODO:` marker.
* Keep total tokens ≤ 12 k. If input tree is huge, summarise less-critical sub-dirs.
* Preserve any existing build commands verbatim.