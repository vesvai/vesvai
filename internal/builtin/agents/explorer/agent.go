package explorer

import (
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	_ "github.com/vesvai/vesvai/internal/builtin/middlewares"
)

func Register() {
	agents.Register(newExplorerAgent)
}

func newExplorerAgent() (*agent.Agent, error) {
	sys, err := generateExplorerPrompt()
	if err != nil {
		return nil, err
	}

	main := agent.New("explorer",
		agent.WithSystemPrompt(sys),
		agent.WithToolNames("glob", "grep", "list", "read", "bash", "web-fetch", "web-search"),
		agent.WithMiddlewareNames("loop-detector", "redaction"),
	)

	return main, nil
}
