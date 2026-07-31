package agent

import (
	"context"
	"fmt"
)

func NewMessageTool(runner *Runner) Tool {
	return NewFuncTool(
		"message",
		"Send a message to another agent by name. The message is stored in that agent's mailbox and can be collected when the agent next runs. Use this to communicate with background sub-agents or pass information between agents.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"to": map[string]any{
					"type":        "string",
					"description": "The name of the agent to send the message to (e.g. 'planner', 'explorer')",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The message content to send",
				},
			},
			"required": []string{"to", "content"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			to, ok := params["to"].(string)
			if !ok || to == "" {
				return "", fmt.Errorf("to is required and must be a string")
			}
			content, ok := params["content"].(string)
			if !ok || content == "" {
				return "", fmt.Errorf("content is required and must be a string")
			}

			from := AgentIDFromContext(ctx)
			runner.Mailbox().Post(from, to, content)
			return fmt.Sprintf("message sent to %s", to), nil
		},
	)
}

func NewCollectMessagesTool(runner *Runner) Tool {
	return NewFuncTool(
		"collect-messages",
		"Retrieve and clear any messages waiting in your mailbox from other agents. Returns all pending messages as a formatted string.",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			name := AgentIDFromContext(ctx)
			msgs := runner.Mailbox().Collect(name)
			if len(msgs) == 0 {
				return "no messages", nil
			}
			var result string
			for i, m := range msgs {
				if i > 0 {
					result += "\n"
				}
				result += fmt.Sprintf("From %s: %s", m.From, m.Content)
			}
			return result, nil
		},
	)
}
