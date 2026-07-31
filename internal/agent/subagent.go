package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type SubagentTool struct {
	agent  Agent
	runner *Runner
	name   string
	mu     sync.RWMutex
}

func NewSubagentTool(runner *Runner, agent Agent, name string) *SubagentTool {
	return &SubagentTool{
		agent:  agent,
		runner: runner,
		name:   name,
	}
}

func (t *SubagentTool) SetRunner(r *Runner) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.runner = r
}

func (t *SubagentTool) getRunner() *Runner {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.runner
}

func (t *SubagentTool) Name() string {
	return t.name
}

func (t *SubagentTool) Description() string {
	return fmt.Sprintf(
		"Delegate a task to the %s sub-agent. The sub-agent will execute the prompt and return its result. "+
			"Set background=true to run concurrently (returns a run ID immediately). "+
			"Set wait_for to a list of sub-agent names to wait for their completion before this agent proceeds.",
		t.name,
	)
}

func (t *SubagentTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task or question to give to the sub-agent",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "If true, run the sub-agent in the background and return immediately with a run ID. Default false (blocking).",
			},
			"wait_for": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of sub-agent names to wait for before proceeding. Only meaningful with background=false.",
			},
		},
		"required": []string{"prompt"},
	}
}

type modelOverrideAgent struct {
	agent Agent
	model string
}

func (m *modelOverrideAgent) Instructions() string { return m.agent.Instructions() }
func (m *modelOverrideAgent) ToolNames() []string  { return m.agent.ToolNames() }

func (m *modelOverrideAgent) Config() AgentConfig {
	cfg := m.agent.Config()
	if m.model != "" {
		cfg.Model = m.model
	}
	return cfg
}

func (t *SubagentTool) childWithModel(ctx context.Context) Agent {
	model := ModelFromContext(ctx)
	if model == "" {
		return t.agent
	}
	return &modelOverrideAgent{agent: t.agent, model: model}
}

func (t *SubagentTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("prompt is required and must be a string")
	}

	background := false
	if b, ok := params["background"].(bool); ok {
		background = b
	}

	runner := t.getRunner()
	if runner == nil {
		return "", fmt.Errorf("sub-agent %q has no runner bound", t.name)
	}

	if !background {
		if waitList, ok := params["wait_for"].([]any); ok && len(waitList) > 0 {
			if err := t.waitForSubagents(ctx, runner, waitList); err != nil {
				return "", err
			}
		}
	}

	parentID := AgentIDFromContext(ctx)
	runID := generateRunID(t.name)
	run := runner.Subagents().Start(runID, t.name, prompt, parentID)
	child := t.childWithModel(ctx)

	if background {
		bgCtx := WithModelContext(context.Background(), ModelFromContext(ctx))
		bgCtx = WithAgentContext(bgCtx, AgentIDFromContext(ctx), SessionIDFromContext(ctx))
		go func() {
			_, err := t.runStreamWithTracking(bgCtx, runner, child, run, prompt, nil)
			_ = err
		}()
		return fmt.Sprintf("subagent %q started in background (run ID: %s)", t.name, runID), nil
	}

	result, err := t.runWithTracking(ctx, runner, child, run, prompt)
	if err != nil {
		return "", fmt.Errorf("sub-agent %q failed: %w", t.name, err)
	}
	return result, nil
}

func (t *SubagentTool) runWithTracking(ctx context.Context, runner *Runner, child Agent, run *SubagentRun, prompt string) (string, error) {
	resp, err := runner.Run(ctx, child, prompt)
	if err != nil {
		run.SetResult("", err)
		return "", err
	}

	if resp.Content != "" {
		run.AppendContent(resp.Content)
	}
	for _, tc := range resp.ToolCalls {
		run.AppendToolCall(SubagentToolEvent{
			ToolName: tc.ToolName,
			Args:     tc.Args,
			Result:   tc.Result,
			Error:    tc.Error,
		})
	}

	run.SetResult(resp.Content, nil)
	return resp.Content, nil
}

func (t *SubagentTool) waitForSubagents(ctx context.Context, runner *Runner, names []any) error {
	for _, n := range names {
		name, ok := n.(string)
		if !ok || name == "" {
			continue
		}
		if err := t.waitForName(ctx, runner, name); err != nil {
			return err
		}
	}
	return nil
}

func (t *SubagentTool) waitForName(ctx context.Context, runner *Runner, name string) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			active := runner.Subagents().Active()
			stillRunning := false
			for _, r := range active {
				if r.Name == name {
					stillRunning = true
					break
				}
			}
			if !stillRunning {
				return nil
			}
		}
	}
}

func (t *SubagentTool) HandleStream(ctx context.Context, params map[string]any, callback StreamCallback) (string, error) {
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("prompt is required and must be a string")
	}

	background := false
	if b, ok := params["background"].(bool); ok {
		background = b
	}

	runner := t.getRunner()
	if runner == nil {
		return "", fmt.Errorf("sub-agent %q has no runner bound", t.name)
	}

	parentID := AgentIDFromContext(ctx)
	runID := generateRunID(t.name)
	run := runner.Subagents().Start(runID, t.name, prompt, parentID)
	child := t.childWithModel(ctx)

	if background {
		bgCtx := WithModelContext(context.Background(), ModelFromContext(ctx))
		bgCtx = WithAgentContext(bgCtx, AgentIDFromContext(ctx), SessionIDFromContext(ctx))
		go func() {
			_, err := t.runStreamWithTracking(bgCtx, runner, child, run, prompt, nil)
			_ = err
		}()
		return fmt.Sprintf("subagent %q started in background (run ID: %s)", t.name, runID), nil
	}

	return t.runStreamWithTracking(ctx, runner, child, run, prompt, callback)
}

func (t *SubagentTool) runStreamWithTracking(ctx context.Context, runner *Runner, child Agent, run *SubagentRun, prompt string, callback StreamCallback) (string, error) {
	resp, err := runner.RunStream(ctx, child, prompt, func(chunk StreamChunk) error {
		if chunk.Content != "" {
			run.AppendContent(chunk.Content)
		}
		if chunk.Reasoning != "" {
			run.AppendReasoning(chunk.Reasoning)
		}
		if chunk.ToolResult != nil {
			run.AppendToolCall(SubagentToolEvent{
				ToolName: chunk.ToolResult.ToolName,
				Result:   chunk.ToolResult.Result,
				Error:    chunk.ToolResult.Error,
				Duration: chunk.ToolResult.Duration,
			})
		}
		if callback != nil {
			return callback(chunk)
		}
		return nil
	})

	if err != nil {
		run.SetResult("", err)
		return "", err
	}

	run.SetResult(resp.Content, nil)
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
