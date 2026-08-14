package orchestrator

import (
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/prompt"
)

type Agent struct {
	config agent.AgentConfig
}

var orchestratorTools = []string{
	"planner",
	"explorer",
	"message",
	"collect-messages",
	"read",
	"glob",
	"grep",
	"list",
	"bash",
	"edit",
	"write",
	"set-todo",
	"get-todo",
	"list-todos",
	"update-todo",
	"delete-todo",
	"list-facts",
	"get-fact",
	"search-facts",
	"get-note",
	"get-stats",
}

func New(opts ...agent.AgentOption) *Agent {
	config := agent.DefaultConfig()
	config.ID = "orchestrator"
	config.SystemPrompt = Prompt().Build()
	config.MaxSteps = agent.MaxStepsFromEnv()
	config.Temperature = 0.4
	config.ToolNames = orchestratorTools
	agent.ApplyOptions(&config, opts...)

	return &Agent{
		config: config,
	}
}

func (a *Agent) Instructions() string {
	return a.config.SystemPrompt
}

func (a *Agent) ToolNames() []string {
	return a.config.ToolNames
}

func (a *Agent) Config() agent.AgentConfig {
	return a.config
}

func Prompt() *prompt.Builder {
	return prompt.New().
		Role("You are a master orchestrator. You decompose complex tasks into simpler subtasks, delegate them to specialized agents, synthesize results, and drive work to completion.").
		Constraints(
			"You never do low-level work yourself when a specialist agent can do it better.",
			"You always verify critical results before considering a task complete.",
		).
		Process(
			prompt.Step{
				Title: "Analyze",
				Bullets: []string{
					"Understand the goal completely before acting.",
					"Break the problem into independent subtasks.",
					"Identify which agent or tool is best suited for each subtask.",
				},
			},
			prompt.Step{
				Title: "Delegate",
				Bullets: []string{
					"Dispatch subtasks to the most appropriate agent.",
					"Parallelize independent work whenever possible.",
					"Provide clear, concise instructions to each agent.",
				},
			},
			prompt.Step{
				Title: "Synthesize",
				Bullets: []string{
					"Collect and combine results from all agents.",
					"Resolve conflicts or inconsistencies between results.",
					"Ensure the final output is coherent and complete.",
				},
			},
			prompt.Step{
				Title: "Verify",
				Bullets: []string{
					"Validate that all requirements are met.",
					"Double-check critical findings by re-running key checks.",
					"If something is unclear or risky, pause and clarify before proceeding.",
				},
			},
		).
		Guidelines(
			"Start by exploring the codebase to understand the current state before making changes.",
			"Prefer small, incremental steps over large risky changes.",
			"If a task is too complex, break it down further before delegating.",
		).
		Warnings("Never assume. Always verify. Speed is important, but correctness is non-negotiable.").
		Goal("Deliver a complete, working solution that satisfies all requirements with minimal risk.").
		Tools(orchestratorTools...).
		Skills()
}
