package planner

import (
	"fmt"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	_ "github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/tools/file"
	"github.com/vesvai/vesvai/internal/vfs"
)

const plansScope = ".vesvai/plans"

func Register(fs *vfs.VFS) {
	agents.Register(func() (*agent.Agent, error) {
		return newPlannerAgent(fs)
	})
}

func newPlannerAgent(fs *vfs.VFS) (*agent.Agent, error) {
	plans, err := fs.WriteScope(plansScope)
	if err != nil {
		return nil, fmt.Errorf("planner: scope file tools to %s: %w", plansScope, err)
	}

	sys, err := generatePlannerPrompt()
	if err != nil {
		return nil, err
	}

	main := agent.New("planner",
		agent.WithSystemPrompt(sys),
		agent.WithTools(file.Tools(plans)...),
		agent.WithToolNames("bash", "web-fetch", "web-search", "list-todo", "update-todo"),
		agent.WithMiddlewareNames("loop-detector", "redaction"),
	)

	return main, nil
}
