package subagent

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"

	"github.com/vesvai/vesvai/internal/agent/tool"
)

func subAgentsStatusTool() tool.Tool {
	return tool.NewSpec(
		"subagents-status",
		"Show the status of subagents. By default returns every subagent; pass agent_names to filter. Statuses: pending, running, completed, failed, interrupted (a subagent that was running when the app restarted).",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_names": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Optional names of subagents to show. Omit to show all.",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				AgentNames []string `json:"agent_names"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("subagents-status: invalid arguments: %w", err)
			}

			if len(params.AgentNames) == 0 {
				return "Subagents:\n" + formatStatuses(store.all()), nil
			}
			agents, err := store.filter(params.AgentNames)
			if err != nil {
				return "", fmt.Errorf("subagents-status: %w", err)
			}
			return "Subagents:\n" + formatStatuses(agents), nil
		},
	)
}
