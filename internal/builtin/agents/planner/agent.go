package planner

import (
	"fmt"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	_ "github.com/vesvai/vesvai/internal/builtin/middlewares"
	"github.com/vesvai/vesvai/internal/builtin/tools/file"
	"github.com/vesvai/vesvai/internal/builtin/tools/plan"
	"github.com/vesvai/vesvai/internal/vfs"
)

const plansScope = vfs.PlansDir

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

	main := agent.New("planner",
		agent.WithSystemPromptFn(func(providerID, modelID string) string {
			sys, err := generatePlannerPrompt(providerID, modelID)
			if err != nil {
				return ""
			}
			return sys
		}),
		agent.WithTools(file.Tools(plans)...),
		agent.WithToolNames("bash", "webfetch", "websearch", "todoread", "todowrite"),
		agent.WithMiddlewareNames("loop-detector", "redaction", "retry", "permission"),
	)
	main.AttachReminder(plan.PlanModeReminder())

	return main, nil
}
