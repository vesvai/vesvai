package developer

import (
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	_ "github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/tools/file"
	"github.com/vesvai/vesvai/internal/vfs"
)

func Register(fs *vfs.VFS) {
	agents.Register(func() (*agent.Agent, error) {
		return newDeveloperAgent(fs)
	})
}

func newDeveloperAgent(fs *vfs.VFS) (*agent.Agent, error) {
	main := agent.New("developer",
		agent.WithSystemPromptFn(func(providerID, modelID string) string {
			sys, err := generateDeveloperPrompt(providerID, modelID)
			if err != nil {
				return ""
			}
			return sys
		}),
		agent.WithTools(file.Tools(fs)...),
		agent.WithToolNames("bash", "webfetch", "websearch", "todoread", "todowrite"),
		agent.WithMiddlewareNames("loop-detector", "redaction", "retry", "permission"),
	)

	return main, nil
}
