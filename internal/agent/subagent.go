package agent

import (
	"context"
	"fmt"
)

type SubagentTool struct {
	agent  Agent
	runner *Runner
	name   string
}

func NewSubagentTool(runner *Runner, agent Agent, name string) *SubagentTool {
	return &SubagentTool{
		agent:  agent,
		runner: runner,
		name:   name,
	}
}

func (t *SubagentTool) Name() string {
	return t.name
}

func (t *SubagentTool) Description() string {
	return "Delegate a task to a sub-agent. The sub-agent will execute the prompt and return its result."
}

func (t *SubagentTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task or question to give to the sub-agent",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *SubagentTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("prompt is required and must be a string")
	}

	resp, err := t.runner.Run(ctx, t.agent, prompt)
	if err != nil {
		return "", fmt.Errorf("sub-agent %q failed: %w", t.name, err)
	}

	return resp.Content, nil
}

func (t *SubagentTool) HandleStream(ctx context.Context, params map[string]any, callback StreamCallback) (string, error) {
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("prompt is required and must be a string")
	}

	resp, err := t.runner.RunStream(ctx, t.agent, prompt, callback)
	if err != nil {
		return "", fmt.Errorf("sub-agent %q failed: %w", t.name, err)
	}

	return resp.Content, nil
}

type SubagentConfig struct {
	Name string

	Agent Agent
}

func BuildSubagentTools(runner *Runner, configs []SubagentConfig) []Tool {
	tools := make([]Tool, len(configs))
	for i, cfg := range configs {
		tools[i] = NewSubagentTool(runner, cfg.Agent, cfg.Name)
	}
	return tools
}
