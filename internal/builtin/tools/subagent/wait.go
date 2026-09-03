package subagent

import (
	"context"
	"fmt"
	json "github.com/goccy/go-json"

	"github.com/vesvai/vesvai/internal/agent/tool"
)

func waitForSubAgentsTool() tool.Tool {
	return tool.NewSpec(
		"wait-for-subagents",
		"Wait for background subagents to finish and return their results. Blocks until every named subagent has completed or failed. Use this after starting subagents with the subagent tool in background mode.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_names": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Names of the subagents to wait for (as given when they were spawned).",
				},
			},
			"required": []string{"agent_names"},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				AgentNames []string `json:"agent_names"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("wait-for-subagents: invalid arguments: %w", err)
			}
			if len(params.AgentNames) == 0 {
				return "", fmt.Errorf("wait-for-subagents: agent_names is required")
			}

			agents, err := store.waitFor(ctx, params.AgentNames)
			if err != nil {
				return "", fmt.Errorf("wait-for-subagents: %w", err)
			}
			return "Subagents finished:\n" + formatStatuses(agents), nil
		},
	)
}
