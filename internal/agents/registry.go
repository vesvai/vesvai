package agents

import (
	"context"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents/explorer"
	"github.com/vesvai/vesvai/internal/agents/orchestrator"
	"github.com/vesvai/vesvai/internal/agents/planner"
	"github.com/vesvai/vesvai/internal/hook"
)

type AgentEntry struct {
	Name  string
	Agent agent.Agent
}

type Registry struct {
	Runner       *agent.Runner
	Planner      *planner.Agent
	Explorer     *explorer.Agent
	Orchestrator *orchestrator.Agent
	agents       map[string]agent.Agent

	subagentTools []agent.Tool
}

type Config struct {
	Runner       *agent.Runner
	ExtraTools   []agent.Tool
	PlannerOpts  []agent.AgentOption
	ExplorerOpts []agent.AgentOption
}

func RegisterCoreAgents(hooks *hook.Hooks) {
	hooks.AddFilter(hook.HookAgentsCollect, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		existing, _ := value.([]AgentEntry)
		return append(existing,
			AgentEntry{Name: "planner", Agent: planner.New()},
			AgentEntry{Name: "explorer", Agent: explorer.New()},
			AgentEntry{Name: "orchestrator", Agent: orchestrator.New()},
		)
	}, 50)
}

func CollectAgentsFromHooks(hooks *hook.Hooks, ctx context.Context) []AgentEntry {
	result := hooks.ApplyFilter(ctx, hook.HookAgentsCollect, []AgentEntry{})
	entries, _ := result.([]AgentEntry)
	return entries
}

func NewRegistry(cfg Config) *Registry {
	p := planner.New(cfg.PlannerOpts...)
	e := explorer.New(cfg.ExplorerOpts...)
	o := orchestrator.New()

	agentMap := make(map[string]agent.Agent)
	agentMap["planner"] = p
	agentMap["explorer"] = e
	agentMap["orchestrator"] = o

	subagentTools := agent.BuildSubagentTools(cfg.Runner, []agent.SubagentConfig{
		{Name: "planner", Agent: p},
		{Name: "explorer", Agent: e},
		{Name: "orchestrator", Agent: o},
	})

	if len(cfg.ExtraTools) > 0 {
		allTools := make([]agent.Tool, 0, len(subagentTools)+len(cfg.ExtraTools))
		allTools = append(allTools, subagentTools...)
		allTools = append(allTools, cfg.ExtraTools...)
		subagentTools = allTools
	}

	return &Registry{
		Runner:        cfg.Runner,
		Planner:       p,
		Explorer:      e,
		Orchestrator:  o,
		agents:        agentMap,
		subagentTools: subagentTools,
	}
}

func NewRegistryFromHooks(runner *agent.Runner, hooks *hook.Hooks, ctx context.Context) *Registry {
	entries := CollectAgentsFromHooks(hooks, ctx)

	agentMap := make(map[string]agent.Agent)
	var p *planner.Agent
	var e *explorer.Agent
	var o *orchestrator.Agent

	var subagentConfigs []agent.SubagentConfig
	for _, entry := range entries {
		agentMap[entry.Name] = entry.Agent
		subagentConfigs = append(subagentConfigs, agent.SubagentConfig{
			Name:  entry.Name,
			Agent: entry.Agent,
		})

		switch entry.Name {
		case "planner":
			if a, ok := entry.Agent.(*planner.Agent); ok {
				p = a
			}
		case "explorer":
			if a, ok := entry.Agent.(*explorer.Agent); ok {
				e = a
			}
		case "orchestrator":
			if a, ok := entry.Agent.(*orchestrator.Agent); ok {
				o = a
			}
		}
	}

	subagentTools := agent.BuildSubagentTools(runner, subagentConfigs)

	return &Registry{
		Runner:        runner,
		Planner:       p,
		Explorer:      e,
		Orchestrator:  o,
		agents:        agentMap,
		subagentTools: subagentTools,
	}
}

func (r *Registry) SubagentTools() []agent.Tool {
	return r.subagentTools
}

func (r *Registry) SubagentConfigs() []agent.SubagentConfig {
	var configs []agent.SubagentConfig
	for name, a := range r.agents {
		configs = append(configs, agent.SubagentConfig{Name: name, Agent: a})
	}
	return configs
}

func (r *Registry) Get(name string) (agent.Agent, bool) {
	a, ok := r.agents[name]
	return a, ok
}

func (r *Registry) All() map[string]agent.Agent {
	result := make(map[string]agent.Agent, len(r.agents))
	for k, v := range r.agents {
		result[k] = v
	}
	return result
}
