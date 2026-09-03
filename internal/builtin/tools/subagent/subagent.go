package subagent

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"
	"strings"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agent/agents"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/llm"
)

func agentOutput(res *agent.RunResult, runErr error) string {
	if res != nil && strings.TrimSpace(res.Output) != "" {
		return res.Output
	}
	if res != nil {
		var b strings.Builder
		for _, m := range res.History {
			if m.Role != llm.RoleAssistant {
				continue
			}
			if c, ok := m.Content.(string); ok && strings.TrimSpace(c) != "" {
				b.WriteString(c)
				b.WriteString("\n")
			}
		}
		if s := strings.TrimSpace(b.String()); s != "" {
			return s
		}
	}
	if runErr != nil {
		return runErr.Error()
	}
	return ""
}

func SubAgentTools(reader HistoryReader) {
	tools.Register(subAgentTool())
	tools.Register(waitForSubAgentsTool())
	tools.Register(subAgentsStatusTool())
	tools.Register(subAgentMessageTool(reader))
}

type subagentSpec struct {
	Name    string   `json:"name"`
	Agent   string   `json:"agent"`
	Task    string   `json:"task"`
	TaskIDs []string `json:"task_id"`
}

func subAgentTool() tool.Tool {
	available := agents.List()
	return tool.NewSpec(
		"subagent",
		"Delegate tasks to one or more subagents in a single call. Each entry spawns a registered agent type with its own task and optional todo linkage. Choose unique, role-specific names that describe the task (e.g. 'refactor-auth-middleware'). Use 'background' to spawn all of them in the background and return immediately (collect results later with wait-for-subagents), or foreground (default) to run all of them concurrently and wait for every one to finish, returning all results. Available agents: "+strings.Join(available, ", ")+".",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subagents": map[string]any{
					"type":        "array",
					"description": "One or more subagents to spawn. All names must be unique.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{
								"type":        "string",
								"description": "Unique name for this subagent run. Must describe the task and be distinct from any other subagent name.",
							},
							"agent": map[string]any{
								"type":        "string",
								"enum":        available,
								"description": "Registered agent type to run. Choose the agent best suited for the task.",
							},
							"task": map[string]any{
								"type":        "string",
								"description": "Instructions for the subagent: what to do, constraints, and what to report back.",
							},
							"task_id": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "string"},
								"description": "Optional todo ID(s) this subagent is working on (e.g. ['todo-1', 'todo-3']).",
							},
						},
						"required": []string{"name", "agent", "task"},
					},
				},
				"background": map[string]any{
					"type":        "boolean",
					"description": "If true, run all subagents in the background and return immediately. If false (default), run all subagents concurrently and wait for every one to finish.",
				},
			},
			"required": []string{"subagents"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				SubAgents  []subagentSpec `json:"subagents"`
				Background bool           `json:"background"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("subagent: invalid arguments: %w", err)
			}
			if len(params.SubAgents) == 0 {
				return "", fmt.Errorf("subagent: subagents array must not be empty")
			}

			parent := agent.FromContext(ctx)
			if parent == nil {
				return "", fmt.Errorf("subagent: no parent agent in context")
			}
			if parent.Provider == nil {
				return "", fmt.Errorf("subagent: parent agent has no provider configured")
			}

			seen := make(map[string]bool, len(params.SubAgents))
			for i := range params.SubAgents {
				s := &params.SubAgents[i]
				if strings.TrimSpace(s.Name) == "" {
					return "", fmt.Errorf("subagent: name is required for every entry")
				}
				if strings.TrimSpace(s.Task) == "" {
					return "", fmt.Errorf("subagent: task is required for every entry")
				}
				if !agents.Has(s.Agent) {
					return "", fmt.Errorf("subagent: unknown agent %q (available: %s)", s.Agent, strings.Join(agents.List(), ", "))
				}
				if seen[s.Name] {
					return "", fmt.Errorf("subagent: duplicate name %q within batch (names must be unique)", s.Name)
				}
				seen[s.Name] = true
				if _, ok := store.get(s.Name); ok {
					return "", fmt.Errorf("subagent: duplicate name %q (names must be unique)", s.Name)
				}
			}

			if parent.Bus != nil {
				store.subscribe(parent.Bus)
			}

			spawned := make([]*SubAgent, 0, len(params.SubAgents))
			for i := range params.SubAgents {
				s := &params.SubAgents[i]
				sa, err := store.spawn(s.Name, s.Agent, s.TaskIDs, params.Background)
				if err != nil {
					return "", err
				}
				spawned = append(spawned, sa)
			}

			runOne := func(s *subagentSpec, sa *SubAgent, runCtx context.Context) {
				sub, err := agents.New(s.Agent)
				if err != nil {
					store.finish(sa, "", err)
					return
				}
				sub.Provider = parent.Provider
				sub.Model = parent.Model
				if parent.Bus != nil {
					sub.Bus = parent.Bus
					store.setAgentID(sa.Name, sub.ID)
					res, runErr := sub.Run(runCtx, s.Task)
					store.finish(sa, agentOutput(res, runErr), runErr)
					return
				}
				store.start(sa)
				res, runErr := sub.Run(runCtx, s.Task)
				store.finish(sa, agentOutput(res, runErr), runErr)
			}

			if params.Background {
				names := make([]string, 0, len(params.SubAgents))
				for i := range params.SubAgents {
					go runOne(&params.SubAgents[i], spawned[i], context.Background())
					names = append(names, params.SubAgents[i].Name)
				}
				return fmt.Sprintf("Started %d subagents in background: %s\n", len(names), strings.Join(names, ", ")), nil
			}

			for i := range params.SubAgents {
				go runOne(&params.SubAgents[i], spawned[i], ctx)
			}
			for _, sa := range spawned {
				select {
				case <-sa.done:
				case <-ctx.Done():
					return "", fmt.Errorf("subagent: %w", ctx.Err())
				}
			}

			var b strings.Builder
			for _, sa := range spawned {
				done, _ := store.get(sa.Name)
				if done.Status == StatusFailed {
					fmt.Fprintf(&b, "Subagent %q failed: %s\n", done.Name, done.Err)
					continue
				}
				fmt.Fprintf(&b, "Subagent %q finished.\nResponse:\n%s\n", done.Name, done.Output)
			}
			return b.String(), nil
		},
	)
}
