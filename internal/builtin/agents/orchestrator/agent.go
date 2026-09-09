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
	sys, err := generateOrchestratorPrompt()
	if err != nil {
		return nil, err
	}

	main := agent.New("orchestrator",
		agent.WithSystemPrompt(sys),
		agent.WithTools(file.Tools(fs)...),
		agent.WithToolNames("ask", "bash", "subagent", "wait-for-subagents", "subagents-status", "subagent-message", "list-todo", "update-todo"),
		agent.WithMiddlewareNames("loop-detector", "redaction", "retry"),
	)

	return main, nil
}
