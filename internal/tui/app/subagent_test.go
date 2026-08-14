package app

import (
	"context"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents/planner"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/tui"
)

type readTool struct{}

func (t *readTool) Name() string        { return "read" }
func (t *readTool) Description() string { return "reads a file" }
func (t *readTool) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"path": map[string]any{"type": "string"},
	}}
}
func (t *readTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	return "package main", nil
}

func newSubagentTestDriver(t *testing.T, p llm.Provider) (*AgentDriver, *agent.Runner) {
	t.Helper()
	registry := agent.NewToolRegistry()
	registry.Register(&readTool{})
	runner := agent.NewRunner(p, nil, registry)

	sub := agent.NewSubagentTool(runner, planner.New(), "planner")
	sub.SetRunner(runner)
	registry.Register(sub)

	return &AgentDriver{runner: runner}, runner
}

func TestSubagentStreamingToTUI(t *testing.T) {
	p := newScriptedProvider(
		[]llm.StreamChunk{
			{Content: "delegating to planner…"},
			{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{PromptTokens: 5}},
			{ToolCalls: []llm.ToolCall{{Index: 0, ID: "p1", Type: "function", Function: llm.Function{Name: "planner", Arguments: `{"prompt":"plan the work"}`}}}},
			{IsDone: true, FinishReason: llm.FinishReasonToolCalls},
		},
		[]llm.StreamChunk{
			{Reasoning: "child thinking…"},
			{Content: "scanning the layout"},
			{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{PromptTokens: 3}},
			{ToolCalls: []llm.ToolCall{{Index: 0, ID: "c1", Type: "function", Function: llm.Function{Name: "read", Arguments: `{"path":"main.go"}`}}}},
			{IsDone: true, FinishReason: llm.FinishReasonToolCalls},
		},
		[]llm.StreamChunk{
			{Content: "the plan is ready"},
			{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{PromptTokens: 2, CompletionTokens: 9}},
		},
		[]llm.StreamChunk{
			{Content: "delegation complete"},
			{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{PromptTokens: 4, CompletionTokens: 7}},
		},
	)
	d, _ := newSubagentTestDriver(t, p)

	model := tui.NewModel("demo")
	var evs []tui.StreamEvent
	d.Run(context.Background(), RunRequest{Text: "do it"}, func(ev tui.StreamEvent) {
		evs = append(evs, ev)
		model.Apply(ev)
	})

	cur := model.Conv.Current()
	if cur == nil {
		t.Fatal("no current message")
	}
	if len(cur.Subagents) != 1 {
		t.Fatalf("subagent blocks = %d, want 1", len(cur.Subagents))
	}
	sa := cur.Subagents[0]
	if sa.ContentText() == "" || !strings.Contains(sa.ContentText(), "scanning the layout") {
		t.Fatalf("subagent content = %q, want child's streamed content", sa.ContentText())
	}
	if sa.ThinkingText() != "child thinking…" {
		t.Fatalf("subagent thinking = %q, want 'child thinking…'", sa.ThinkingText())
	}
	if len(sa.Tools) != 1 || sa.Tools[0].Name != "read" {
		t.Fatalf("subagent tools = %+v, want [read]", sa.Tools)
	}
	if sa.Result != "scanning the layout\n\nthe plan is ready" {
		t.Fatalf("subagent result = %q, want accumulated child output", sa.Result)
	}

	for _, ev := range evs {
		if ev.Kind == tui.EventToolResult && ev.ToolResult.ToolName == "planner" {
			if ev.ToolResult.Result != "scanning the layout\n\nthe plan is ready" {
				t.Fatalf("planner tool result = %q, want accumulated child output", ev.ToolResult.Result)
			}
		}
	}
}
