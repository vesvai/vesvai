package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/llm"
)

func TestSubagentTool_Name(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	if tool.Name() != "research" {
		t.Errorf("Name() = %q, want research", tool.Name())
	}
}

func TestSubagentTool_Description(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestSubagentTool_Schema(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	schema := tool.Schema()

	if schema["type"] != "object" {
		t.Errorf("Schema().type = %v", schema["type"])
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema().properties is not map")
	}
	if _, exists := props["prompt"]; !exists {
		t.Error("Schema().properties missing prompt")
	}
}

func TestSubagentTool_Handle_Success(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "sub-agent result"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	result, err := tool.Handle(context.Background(), map[string]any{
		"prompt": "do research",
	})

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result != "sub-agent result" {
		t.Errorf("Handle() = %q, want sub-agent result", result)
	}
}

func TestSubagentTool_Handle_MissingPrompt(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	_, err := tool.Handle(context.Background(), nil)
	if err == nil {
		t.Error("Handle() should error with nil params")
	}
}

func TestSubagentTool_Handle_EmptyPrompt(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	_, err := tool.Handle(context.Background(), map[string]any{"prompt": ""})
	if err == nil {
		t.Error("Handle() should error with empty prompt")
	}
}

func TestSubagentTool_Handle_NonStringPrompt(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	_, err := tool.Handle(context.Background(), map[string]any{"prompt": 123})
	if err == nil {
		t.Error("Handle() should error with non-string prompt")
	}
}

func TestSubagentTool_Handle_AgentError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider._chatErr = errors.New("agent failed")

	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "research")
	_, err := tool.Handle(context.Background(), map[string]any{"prompt": "do something"})
	if err == nil {
		t.Error("Handle() should error on agent failure")
	}
}

func TestSubagentTool_HandleStream_Success(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Content: "streaming", IsDone: false},
		llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "streamer")
	var chunks []string
	result, err := tool.HandleStream(context.Background(), map[string]any{
		"prompt": "stream this",
	}, func(chunk StreamChunk) error {
		if chunk.Content != "" {
			chunks = append(chunks, chunk.Content)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("HandleStream() error = %v", err)
	}
	if result != "streaming" {
		t.Errorf("HandleStream() = %q, want streaming", result)
	}
	if len(chunks) != 1 {
		t.Errorf("chunks len = %d, want 1", len(chunks))
	}
}

func TestSubagentTool_HandleStream_MissingPrompt(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "streamer")
	_, err := tool.HandleStream(context.Background(), nil, func(chunk StreamChunk) error {
		return nil
	})
	if err == nil {
		t.Error("HandleStream() should error with nil params")
	}
}

func TestSubagentTool_HandleStream_EmptyPrompt(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "streamer")
	_, err := tool.HandleStream(context.Background(), map[string]any{"prompt": ""}, func(chunk StreamChunk) error {
		return nil
	})
	if err == nil {
		t.Error("HandleStream() should error with empty prompt")
	}
}

func TestSubagentTool_HandleStream_AgentError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider._chatStreamErr = errors.New("stream failed")

	runner := NewRunner(provider, bus)
	agent := newMockAgent("inner")

	tool := NewSubagentTool(runner, agent, "streamer")
	_, err := tool.HandleStream(context.Background(), map[string]any{
		"prompt": "stream",
	}, func(chunk StreamChunk) error {
		return nil
	})
	if err == nil {
		t.Error("HandleStream() should error on agent failure")
	}
}

func TestBuildSubagentTools(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)

	configs := []SubagentConfig{
		{Name: "research", Agent: newMockAgent("r")},
		{Name: "code", Agent: newMockAgent("c")},
	}

	tools := BuildSubagentTools(runner, configs)
	if len(tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(tools))
	}
	if tools[0].Name() != "research" {
		t.Errorf("tools[0].Name() = %q, want research", tools[0].Name())
	}
	if tools[1].Name() != "code" {
		t.Errorf("tools[1].Name() = %q, want code", tools[1].Name())
	}
}

func TestBuildSubagentTools_Empty(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)

	tools := BuildSubagentTools(runner, nil)
	if len(tools) != 0 {
		t.Errorf("tools len = %d, want 0", len(tools))
	}
}

func TestSubagentTool_Integration_WithRunner(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	innerProvider := newMockProvider()
	innerProvider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "inner result"}},
		},
	})

	innerRunner := NewRunner(innerProvider, bus)
	innerAgent := newMockAgent("inner")

	subtool := NewSubagentTool(innerRunner, innerAgent, "delegate")

	outerProvider := newMockProvider()
	outerProvider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{
								ID:       "call_1",
								Type:     "function",
								Function: llm.Function{Name: "delegate", Arguments: `{"prompt":"analyze this"}`},
							},
						},
					},
				},
			},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "final: inner result"}},
			},
		},
	)

	outerRunner := NewRunner(outerProvider, bus)
	outerAgent := newMockAgent("outer")
	outerAgent.tools = []Tool{subtool}

	resp, err := outerRunner.Run(context.Background(), outerAgent, "delegate analysis")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.Content != "final: inner result" {
		t.Errorf("Content = %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Result != "inner result" {
		t.Errorf("ToolCalls[0].Result = %q", resp.ToolCalls[0].Result)
	}
}
