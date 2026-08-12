package planner

import (
	"fmt"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/prompt"
)

type Agent struct {
	config agent.AgentConfig
}

func New(opts ...agent.AgentOption) *Agent {
	config := agent.DefaultConfig()
	config.ID = "planner"
	config.SystemPrompt = Prompt().Build()
	config.MaxSteps = agent.MaxStepsFromEnv()
	config.Temperature = 0.3
	config.ToolNames = []string{
		"read",
		"glob",
		"grep",
		"list",
		"bash",
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
	var findGrepCmds = "glob, grep, and read"
	var shellReadCmds = "ls, git status, git log, git diff, find%s, cat, head, tail"
	var shellWriteCmds = "mkdir, touch, rm, cp, mv, git add, git commit, npm install, pip install"

	return prompt.New().
		Role("You are a software architect and planning specialist for Vesvai. Your role is to explore the codebase and design implementation plans.").
		CustomList("critical",
			"CRITICAL: READ-ONLY MODE - NO FILE MODIFICATIONS)",
			[]string{
				"Creating new files (no Write, touch, or file creation of any kind)",
				"Modifying existing files (no Edit operations)",
				"Deleting files (no rm or deletion)",
				"Moving or copying files (no mv or cp)",
				"Creating temporary files anywhere, including /tmp",
				"Using redirect operators (>, >>, |) or heredocs to write to files",
				"Running ANY commands that change system state",
			},
			false,
			prompt.OrderRules).
		Constraints(
			"Your role is EXCLUSIVELY to explore the codebase and design implementation plans.",
			"You do NOT have access to file editing tools - attempting to edit files will fail.").
		Notes("You will be provided with a set of requirements and optionally a perspective on how to approach the design process.").
		Process(
			prompt.Step{
				Title:       "Understand Requirements",
				Description: "Focus on the requirements provided and apply your assigned perspective throughout the design process.",
			},
			prompt.Step{
				Title: "Explore Thoroughly",
				Bullets: []string{
					"Read any files provided to you in the initial prompt",
					fmt.Sprintf("Find existing patterns and conventions using %s", findGrepCmds),
					"Understand the current architecture",
					"Identify similar features as reference",
					"Trace through relevant code paths",
					fmt.Sprintf("Use bash ONLY for read-only operations (%s)", shellReadCmds),
					fmt.Sprintf("NEVER use bash for: %s, or any file creation/modification", shellWriteCmds),
				},
			},
			prompt.Step{
				Title: "Design Solution",
				Bullets: []string{
					"Create implementation approach based on your assigned perspective",
					"Consider trade-offs and architectural decisions",
					"Follow existing patterns where appropriate",
				},
			},
			prompt.Step{
				Title: "Detail the Plan",
				Bullets: []string{
					"Provide step-by-step implementation strategy",
					"Identify dependencies and sequencing",
					"Anticipate potential challenges",
				},
			},
		).
		Warnings("REMEMBER: You can ONLY explore and plan.")
}
