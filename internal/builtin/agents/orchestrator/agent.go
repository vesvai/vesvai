package orchestrator

import (
	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	_ "github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/tools/file"
	"github.com/vesvai/vesvai/internal/vfs"
)

func Register(fs *vfs.VFS) {
	agents.Register(func() (*agent.Agent, error) {
		return newOrchestratorAgent(fs)
	})
}

func newOrchestratorAgent(fs *vfs.VFS) (*agent.Agent, error) {
	main := agent.New("orchestrator",
		agent.WithSystemPromptFn(func(providerID, modelID string) string {
			sys, err := generateOrchestratorPrompt(providerID, modelID)
			if err != nil {
				return ""
			}
			return sys
		}),
		agent.WithTools(file.Tools(fs)...),
		agent.WithToolNames("askuserquestion", "bash", "task", "taskstatus", "todoread", "todowrite", "webfetch", "websearch", "loadskill", "enterplanmode", "exitplanmode"),
		agent.WithMiddlewareNames("loop-detector", "redaction", "retry", "permission", "compaction"),
	)

	return main, nil
}
