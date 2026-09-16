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
	main := agent.New("explorer",
		agent.WithSystemPromptFn(func(providerID, modelID string) string {
			sys, err := generateExplorerPrompt(providerID, modelID)
			if err != nil {
				return ""
			}
			return sys
		}),
		agent.WithToolNames("glob", "grep", "list", "read", "bash", "webfetch", "websearch"),
		agent.WithMiddlewareNames("loop-detector", "redaction", "retry", "permission"),
	)

	return main, nil
}
