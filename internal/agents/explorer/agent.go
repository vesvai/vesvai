package explorer

import (
	"fmt"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/prompt"
)

type Agent struct {
	config agent.AgentConfig
}

var explorerTools = []string{
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

func New(opts ...agent.AgentOption) *Agent {
	config := agent.DefaultConfig()
	config.ID = "explorer"
	config.SystemPrompt = Prompt().Build()
	config.MaxSteps = agent.MaxStepsFromEnv()
	config.Temperature = 0.2
	config.ToolNames = explorerTools
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
	var shellReadCmds = "ls, git status, git log, git diff, find%s, cat, head, tail"
	var shellWriteCmds = "mkdir, touch, rm, cp, mv, git add, git commit, npm install, pip install"

	return prompt.New().
		Role("You are a file search specialist for Vesvai. You excel at thoroughly navigating and exploring codebases.").
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
			"Your role is EXCLUSIVELY to search and analyze existing code.",
			"You do NOT have access to file editing tools - attempting to edit files will fail.").
		Strengths(
			"Rapidly finding files using glob patterns",
			"Searching code and text with powerful regex patterns",
			"Reading and analyzing file contents",
		).
		Guidelines(
			"glob",
			"grep",
			"Use read when you know the specific file path you need to read",
			fmt.Sprintf("Use bash ONLY for read-only operations (%s)", shellReadCmds),
			fmt.Sprintf("NEVER use bash for: %s, or any file creation/modification", shellWriteCmds),
			"Adapt your search approach based on the thoroughness level specified by the caller",
			"Communicate your final report directly as a regular message - do NOT attempt to create files",
		).
		Notes(strings.Join([]string{
			"You are meant to be a fast agent that returns output as quickly as possible. In order to achieve this you must:",
			"- Make efficient use of the tools that you have at your disposal: be smart about how you search for files and implementations",
			"- Wherever possible you should try to spawn multiple parallel tool calls for grepping and reading files",
		}, "\n")).
		Goal("Complete the user's search request efficiently and report your findings clearly.").
		Tools(explorerTools...).
		Skills()
}
