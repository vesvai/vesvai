package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/llm"
)

type mockProvider struct {
	mu       sync.Mutex
	responses []*llm.Response
	streamChunks []llm.StreamChunk
	callCount int
	lastRequest *llm.Request
	_chatErr  error
	_chatStreamErr error
	_chatHandler func(req *llm.Request) *llm.Response
	_chatStreamHandler func(req *llm.Request, handler llm.StreamHandler) error
}

func newMockProvider() *mockProvider {
	return &mockProvider{}
}

func (p *mockProvider) Name() string { return "mock" }

func (p *mockProvider) Chat(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.callCount++
	p.lastRequest = req

	if p._chatErr != nil {
		return nil, p._chatErr
	}

	if p._chatHandler != nil {
		return p._chatHandler(req), nil
	}

	if len(p.responses) > 0 {
		resp := p.responses[0]
		p.responses = p.responses[1:]
		return resp, nil
	}

	return &llm.Response{
		Choices: []llm.Choice{
			{
				Message: &llm.Message{
					Role:    llm.RoleAssistant,
					Content: "default response",
				},
			},
		},
	}, nil
}

func (p *mockProvider) ChatStream(ctx context.Context, req *llm.Request, handler llm.StreamHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.callCount++
	p.lastRequest = req

	if p._chatStreamErr != nil {
		return p._chatStreamErr
	}

	if p._chatStreamHandler != nil {
		return p._chatStreamHandler(req, handler)
	}

	for _, chunk := range p.streamChunks {
		if err := handler(chunk); err != nil {
			return err
		}
	}

	return nil
}

func (p *mockProvider) setResponses(responses ...*llm.Response) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = responses
}

func (p *mockProvider) setStreamChunks(chunks ...llm.StreamChunk) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streamChunks = chunks
}

type mockAgent struct {
	name         string
	instructions string
	tools        []Tool
	config       AgentConfig
}

func (a *mockAgent) Instructions() string  { return a.instructions }
func (a *mockAgent) Tools() []Tool         { return a.tools }
func (a *mockAgent) Config() AgentConfig   { return a.config }

func newMockAgent(name string) *mockAgent {
	return &mockAgent{
		name:         name,
		instructions: "You are a test agent.",
		config: AgentConfig{
			Model:        "test-model",
			MaxSteps:     5,
			Temperature:  0.5,
			MaxTokens:    1024,
			SystemPrompt: "System prompt",
		},
	}
}

func TestWithAgentContext(t *testing.T) {
	ctx := WithAgentContext(context.Background(), "agent-1", "session-1")

	if got := AgentIDFromContext(ctx); got != "agent-1" {
		t.Errorf("AgentIDFromContext() = %q, want %q", got, "agent-1")
	}
	if got := SessionIDFromContext(ctx); got != "session-1" {
		t.Errorf("SessionIDFromContext() = %q, want %q", got, "session-1")
	}
}

func TestAgentIDFromContext_Empty(t *testing.T) {
	if got := AgentIDFromContext(context.Background()); got != "" {
		t.Errorf("AgentIDFromContext() = %q, want empty", got)
	}
}

func TestSessionIDFromContext_Empty(t *testing.T) {
	if got := SessionIDFromContext(context.Background()); got != "" {
		t.Errorf("SessionIDFromContext() = %q, want empty", got)
	}
}

func TestStepFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), stepKey, 7)
	if got := StepFromContext(ctx); got != 7 {
		t.Errorf("StepFromContext() = %d, want 7", got)
	}
}

func TestStepFromContext_Empty(t *testing.T) {
	if got := StepFromContext(context.Background()); got != 0 {
		t.Errorf("StepFromContext() = %d, want 0", got)
	}
}

func TestRunner_NewRunner(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus)

	if runner.provider != provider {
		t.Error("provider not set")
	}
	if runner.middlewares == nil {
		t.Error("middlewares not initialized")
	}
	if runner.eventBus != bus {
		t.Error("eventBus not set")
	}
}

func TestRunner_Run_SimpleResponse(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{
				Message: &llm.Message{
					Role:    llm.RoleAssistant,
					Content: "Hello!",
				},
			},
		},
		Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	resp, err := runner.Run(context.Background(), agent, "Hi")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello!")
	}
	if resp.Steps != 1 {
		t.Errorf("Steps = %d, want 1", resp.Steps)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
}

func TestRunner_Run_WithToolCalls(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	tool := NewFuncTool("search", "Search", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}, func(ctx context.Context, params map[string]any) (string, error) {
		return "search result for " + params["query"].(string), nil
	})

	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						Role:    llm.RoleAssistant,
						Content: "",
						ToolCalls: []llm.ToolCall{
							{
								ID:   "call_1",
								Type: "function",
								Function: llm.Function{
									Name:      "search",
									Arguments: `{"query":"test"}`,
								},
							},
						},
					},
				},
			},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						Role:    llm.RoleAssistant,
						Content: "Found: search result for test",
					},
				},
			},
			Usage: llm.Usage{TotalTokens: 50},
		},
	)

	agent := newMockAgent("tool-agent")
	agent.tools = []Tool{tool}

	runner := NewRunner(provider, bus)
	resp, err := runner.Run(context.Background(), agent, "search for test")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if resp.Content != "Found: search result for test" {
		t.Errorf("Content = %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ToolName != "search" {
		t.Errorf("ToolCalls[0].ToolName = %q, want search", resp.ToolCalls[0].ToolName)
	}
	if resp.ToolCalls[0].Result != "search result for test" {
		t.Errorf("ToolCalls[0].Result = %q", resp.ToolCalls[0].Result)
	}
	if resp.Steps != 2 {
		t.Errorf("Steps = %d, want 2", resp.Steps)
	}
}

func TestRunner_Run_ToolCallError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	tool := NewFuncTool("fail", "fail", map[string]any{
		"type": "object",
	}, func(ctx context.Context, params map[string]any) (string, error) {
		return "", errors.New("tool failed")
	})

	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{
								ID:       "call_1",
								Type:     "function",
								Function: llm.Function{Name: "fail", Arguments: "{}"},
							},
						},
					},
				},
			},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{Content: "handled error"},
				},
			},
		},
	)

	agent := newMockAgent("err-agent")
	agent.tools = []Tool{tool}

	runner := NewRunner(provider, bus)
	resp, err := runner.Run(context.Background(), agent, "do something")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Error == nil {
		t.Error("ToolCalls[0].Error should not be nil")
	}
}

func TestRunner_Run_MaxSteps(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	stepCount := int32(0)

	tool := NewFuncTool("loop", "loop", map[string]any{
		"type": "object",
	}, func(ctx context.Context, params map[string]any) (string, error) {
		return "ok", nil
	})

	provider._chatHandler = func(req *llm.Request) *llm.Response {
		n := atomic.AddInt32(&stepCount, 1)
		return &llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{
								ID:       fmt.Sprintf("call_%d", n),
								Type:     "function",
								Function: llm.Function{Name: "loop", Arguments: "{}"},
							},
						},
					},
				},
			},
		}
	}

	agent := newMockAgent("loop-agent")
	agent.config.MaxSteps = 3
	agent.tools = []Tool{tool}

	runner := NewRunner(provider, bus)
	_, err := runner.Run(context.Background(), agent, "loop forever")
	if err == nil {
		t.Fatal("Run() should error on max steps")
	}
}

func TestRunner_Run_ProviderError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider._chatErr = errors.New("provider down")

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	_, err := runner.Run(context.Background(), agent, "hello")
	if err == nil {
		t.Fatal("Run() should error on provider failure")
	}
}

func TestRunner_Run_NilEventBus(t *testing.T) {
	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	runner := NewRunner(provider, nil)
	agent := newMockAgent("test")

	resp, err := runner.Run(context.Background(), agent, "hi")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want ok", resp.Content)
	}
}

func TestRunner_Run_WithExistingSessionID(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	ctx := WithAgentContext(context.Background(), "custom-agent", "custom-session")
	resp, err := runner.Run(ctx, agent, "hi")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want ok", resp.Content)
	}
}

func TestRunner_Run_WithToolChoice(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")
	agent.config.ToolChoice = "auto"

	runner.Run(context.Background(), agent, "hi")

	provider.mu.Lock()
	defer provider.mu.Unlock()
	if provider.lastRequest == nil {
		t.Fatal("lastRequest is nil")
	}
	if provider.lastRequest.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %v, want auto", provider.lastRequest.ToolChoice)
	}
}

func TestRunner_RunStream_Simple(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Content: "Hello", IsDone: false},
		llm.StreamChunk{Content: " World", IsDone: false},
		llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	var chunks []string
	resp, err := runner.RunStream(context.Background(), agent, "hi", func(chunk StreamChunk) error {
		if chunk.Content != "" {
			chunks = append(chunks, chunk.Content)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if resp.Content != "Hello World" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello World")
	}
	if len(chunks) != 2 {
		t.Errorf("chunks len = %d, want 2", len(chunks))
	}
}

func TestRunner_RunStream_WithToolCalls(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	tool := NewFuncTool("calc", "calc", map[string]any{
		"type": "object",
	}, func(ctx context.Context, params map[string]any) (string, error) {
		return "42", nil
	})

	step := int32(0)
	provider._chatStreamHandler = func(req *llm.Request, handler llm.StreamHandler) error {
		n := atomic.AddInt32(&step, 1)
		if n == 1 {
			handler(llm.StreamChunk{
				ToolCalls: []llm.ToolCall{
					{Index: 0, ID: "call_1", Type: "function", Function: llm.Function{Name: "calc", Arguments: ""}},
				},
				IsDone: false,
			})
			handler(llm.StreamChunk{
				ToolCalls: []llm.ToolCall{
					{Index: 0, Function: llm.Function{Arguments: `{"x":1}`}},
				},
				IsDone: false,
			})
			handler(llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonToolCalls})
		} else {
			handler(llm.StreamChunk{Content: "result: 42", IsDone: false})
			handler(llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop})
		}
		return nil
	}

	agent := newMockAgent("stream-tool")
	agent.config.MaxSteps = 10
	agent.tools = []Tool{tool}

	runner := NewRunner(provider, bus)

	var toolCallInfo *ToolCallInfo
	var toolResultInfo *ToolResultInfo

	resp, err := runner.RunStream(context.Background(), agent, "calculate", func(chunk StreamChunk) error {
		if chunk.ToolCall != nil {
			toolCallInfo = chunk.ToolCall
		}
		if chunk.ToolResult != nil {
			toolResultInfo = chunk.ToolResult
		}
		return nil
	})

	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ToolName != "calc" {
		t.Errorf("ToolCalls[0].ToolName = %q, want calc", resp.ToolCalls[0].ToolName)
	}
	if toolCallInfo == nil {
		t.Error("ToolCall info not received in stream")
	}
	if toolResultInfo == nil {
		t.Error("ToolResult info not received in stream")
	}
}

func TestRunner_RunStream_CallbackError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Content: "hello", IsDone: false},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	callbackErr := errors.New("callback error")
	_, err := runner.RunStream(context.Background(), agent, "hi", func(chunk StreamChunk) error {
		return callbackErr
	})

	if err == nil {
		t.Fatal("RunStream() should error on callback error")
	}
}

func TestRunner_RunStream_ProviderError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider._chatStreamErr = errors.New("stream failed")

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	_, err := runner.RunStream(context.Background(), agent, "hi", func(chunk StreamChunk) error {
		return nil
	})

	if err == nil {
		t.Fatal("RunStream() should error on provider failure")
	}
}

func TestRunner_RunStream_MaxSteps(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	for i := 0; i < 20; i++ {
		provider.setStreamChunks(
			llm.StreamChunk{
				ToolCalls: []llm.ToolCall{
					{Index: 0, ID: fmt.Sprintf("c%d", i), Type: "function", Function: llm.Function{Name: "loop", Arguments: "{}"}},
				},
				IsDone: true,
				FinishReason: llm.FinishReasonToolCalls,
			},
		)
	}

	tool := NewFuncTool("loop", "loop", map[string]any{"type": "object"}, func(ctx context.Context, params map[string]any) (string, error) {
		return "ok", nil
	})

	agent := newMockAgent("loop")
	agent.config.MaxSteps = 2
	agent.tools = []Tool{tool}

	runner := NewRunner(provider, bus)
	_, err := runner.RunStream(context.Background(), agent, "loop", func(chunk StreamChunk) error {
		return nil
	})

	if err == nil {
		t.Fatal("RunStream() should error on max steps")
	}
}

func TestRunner_Run_WithMiddleware(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	var middlewareCalled int32
	mw := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		atomic.AddInt32(&middlewareCalled, 1)
		return next(ctx)
	}

	runner := NewRunner(provider, bus, mw)
	agent := newMockAgent("test")

	_, err := runner.Run(context.Background(), agent, "hi")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunner_Run_MiddlewareBlocksExecution(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	mw := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		return errors.New("blocked by middleware")
	}

	runner := NewRunner(provider, bus, mw)
	agent := newMockAgent("test")
	agent.tools = []Tool{
		NewFuncTool("test", "test", map[string]any{"type": "object"}, func(ctx context.Context, params map[string]any) (string, error) {
			return "result", nil
		}),
	}

	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{ID: "c1", Type: "function", Function: llm.Function{Name: "test", Arguments: "{}"}},
						},
					},
				},
			},
		},
	)

	resp, err := runner.Run(context.Background(), agent, "call tool")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Error == nil {
		t.Error("ToolCalls[0].Error should not be nil when middleware blocks")
	}
}

func TestRunner_Run_EventPublishing(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	var events []event.EventType
	var mu sync.Mutex

	bus.Subscribe(event.EventType(EventAgentStart), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		mu.Lock()
		events = append(events, e.Type())
		mu.Unlock()
		return nil
	}))
	bus.Subscribe(event.EventType(EventAgentComplete), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		mu.Lock()
		events = append(events, e.Type())
		mu.Unlock()
		return nil
	}))
	bus.Subscribe(event.EventType(EventAgentMessageReceived), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		mu.Lock()
		events = append(events, e.Type())
		mu.Unlock()
		return nil
	}))
	bus.Subscribe(event.EventType(EventAgentMessageSent), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		mu.Lock()
		events = append(events, e.Type())
		mu.Unlock()
		return nil
	}))

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	runner.Run(context.Background(), agent, "hi")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(events) < 2 {
		t.Errorf("not enough events published: %d", len(events))
	}

	hasStart := false
	hasComplete := false
	for _, et := range events {
		if et == event.EventType(EventAgentStart) {
			hasStart = true
		}
		if et == event.EventType(EventAgentComplete) {
			hasComplete = true
		}
	}
	if !hasStart {
		t.Error("EventAgentStart not published")
	}
	if !hasComplete {
		t.Error("EventAgentComplete not published")
	}
}

func TestRunner_Run_NoSystemPrompt(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")
	agent.config.SystemPrompt = ""
	agent.instructions = ""

	runner.Run(context.Background(), agent, "hi")

	provider.mu.Lock()
	defer provider.mu.Unlock()

	if provider.lastRequest == nil {
		t.Fatal("lastRequest is nil")
	}
	if len(provider.lastRequest.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1 (no system prompt)", len(provider.lastRequest.Messages))
	}
}

func TestRunner_Run_UseInstructionsAsFallback(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")
	agent.config.SystemPrompt = ""

	runner.Run(context.Background(), agent, "hi")

	provider.mu.Lock()
	defer provider.mu.Unlock()

	if provider.lastRequest == nil {
		t.Fatal("lastRequest is nil")
	}
	if len(provider.lastRequest.Messages) != 2 {
		t.Fatalf("Messages len = %d, want 2 (instructions + user)", len(provider.lastRequest.Messages))
	}
	if provider.lastRequest.Messages[0].Role != llm.RoleSystem {
		t.Errorf("Messages[0].Role = %q, want %q", provider.lastRequest.Messages[0].Role, llm.RoleSystem)
	}
}

func TestGetUsageOrDefault_Nil(t *testing.T) {
	u := getUsageOrDefault(nil)
	if u.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", u.TotalTokens)
	}
}

func TestGetUsageOrDefault_NotNil(t *testing.T) {
	usage := &llm.Usage{TotalTokens: 100}
	u := getUsageOrDefault(usage)
	if u.TotalTokens != 100 {
		t.Errorf("TotalTokens = %d, want 100", u.TotalTokens)
	}
}

func TestFindTool(t *testing.T) {
	tools := []Tool{
		NewFuncTool("a", "desc", nil, nil),
		NewFuncTool("b", "desc", nil, nil),
	}

	got, ok := findTool(tools, "b")
	if !ok {
		t.Fatal("findTool() ok = false")
	}
	if got.Name() != "b" {
		t.Errorf("Name() = %q, want b", got.Name())
	}

	_, ok = findTool(tools, "c")
	if ok {
		t.Error("findTool() ok = true for nonexistent")
	}
}

func TestParseToolArgsSafe(t *testing.T) {
	params := parseToolArgsSafe(`{"key":"val"}`)
	if params["key"] != "val" {
		t.Errorf("key = %v, want val", params["key"])
	}

	params = parseToolArgsSafe("invalid")
	if params != nil {
		t.Errorf("invalid JSON returned %v, want nil", params)
	}
}

func TestStreamChunk_Fields(t *testing.T) {
	chunk := StreamChunk{
		Content:      "text",
		Reasoning:    "reason",
		FinishReason: "stop",
		IsDone:       true,
	}

	if chunk.Content != "text" {
		t.Errorf("Content = %q", chunk.Content)
	}
	if chunk.Reasoning != "reason" {
		t.Errorf("Reasoning = %q", chunk.Reasoning)
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("FinishReason = %q", chunk.FinishReason)
	}
	if !chunk.IsDone {
		t.Error("IsDone = false")
	}
}

func TestToolCallInfo_Fields(t *testing.T) {
	info := &ToolCallInfo{
		ToolName: "search",
		Args:     map[string]any{"q": "test"},
	}
	if info.ToolName != "search" {
		t.Errorf("ToolName = %q", info.ToolName)
	}
	if info.Args["q"] != "test" {
		t.Errorf("Args[q] = %v", info.Args["q"])
	}
}

func TestToolResultInfo_Fields(t *testing.T) {
	info := &ToolResultInfo{
		ToolName: "search",
		Result:   "found",
		Duration: 100,
	}
	if info.ToolName != "search" {
		t.Errorf("ToolName = %q", info.ToolName)
	}
	if info.Result != "found" {
		t.Errorf("Result = %q", info.Result)
	}
	if info.Duration != 100 {
		t.Errorf("Duration = %d", info.Duration)
	}
}

func TestResponse_Fields(t *testing.T) {
	resp := &Response{
		Content: "hello",
		ToolCalls: []ToolCallResult{
			{ToolName: "search", Result: "found"},
		},
		Usage: llm.Usage{TotalTokens: 50},
		Steps: 3,
	}

	if resp.Content != "hello" {
		t.Errorf("Content = %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("ToolCalls len = %d", len(resp.ToolCalls))
	}
	if resp.Usage.TotalTokens != 50 {
		t.Errorf("TotalTokens = %d", resp.Usage.TotalTokens)
	}
	if resp.Steps != 3 {
		t.Errorf("Steps = %d", resp.Steps)
	}
}

func TestRunner_PublishEvent_NoMetadata(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	var received event.Event
	bus.Subscribe(event.EventType(EventAgentStart), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		received = e
		return nil
	}))

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "ok"}},
		},
	})

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	runner.Run(context.Background(), agent, "hi")

	time.Sleep(50 * time.Millisecond)

	if received == nil {
		t.Error("event not received")
	}
}

func TestRunner_PublishEvent_WithMetadata(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	var receivedEvent event.Event
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(event.EventType(EventAgentToolCall), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		receivedEvent = e
		wg.Done()
		return nil
	}))

	provider := newMockProvider()

	tool := NewFuncTool("test", "test", map[string]any{"type": "object"}, func(ctx context.Context, params map[string]any) (string, error) {
		return "result", nil
	})

	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{ID: "c1", Type: "function", Function: llm.Function{Name: "test", Arguments: "{}"}},
						},
					},
				},
			},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "done"}},
			},
		},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test-agent")
	agent.config.ID = "test-agent"
	agent.tools = []Tool{tool}

	runner.Run(context.Background(), agent, "do it")

	wg.Wait()

	if receivedEvent == nil {
		t.Fatal("no event received")
	}

	ae, ok := receivedEvent.(*AgentEvent)
	if !ok {
		t.Fatalf("event is not *AgentEvent, got %T", receivedEvent)
	}

	td, ok := ae.Data.(*ToolEventData)
	if !ok {
		t.Fatalf("event data is not *ToolEventData, got %T", ae.Data)
	}

	if td.AgentID != "test-agent" {
		t.Errorf("AgentID = %q, want test-agent", td.AgentID)
	}
}

func TestRunner_Run_InvalidToolArguments(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	tool := NewFuncTool("test", "test", map[string]any{"type": "object"}, func(ctx context.Context, params map[string]any) (string, error) {
		return "result", nil
	})

	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{ID: "c1", Type: "function", Function: llm.Function{Name: "test", Arguments: "invalid json"}},
						},
					},
				},
			},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "done"}},
			},
		},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")
	agent.tools = []Tool{tool}

	resp, err := runner.Run(context.Background(), agent, "call tool")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Error == nil {
		t.Error("ToolCalls[0].Error should not be nil for invalid arguments")
	}
}

func TestRunner_RunStream_ToolChoice(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Content: "response", IsDone: false},
		llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")
	agent.config.ToolChoice = "auto"

	_, err := runner.RunStream(context.Background(), agent, "hi", func(chunk StreamChunk) error {
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	if provider.lastRequest == nil {
		t.Fatal("lastRequest is nil")
	}
	if provider.lastRequest.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %v, want auto", provider.lastRequest.ToolChoice)
	}
}

func TestRunner_RunStream_WithReasoning(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Reasoning: "thinking...", IsDone: false},
		llm.StreamChunk{Content: "answer", IsDone: false},
		llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{TotalTokens: 25}},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	var reasoningChunks []string
	var contentChunks []string

	resp, err := runner.RunStream(context.Background(), agent, "think", func(chunk StreamChunk) error {
		if chunk.Reasoning != "" {
			reasoningChunks = append(reasoningChunks, chunk.Reasoning)
		}
		if chunk.Content != "" {
			contentChunks = append(contentChunks, chunk.Content)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if resp.Content != "answer" {
		t.Errorf("Content = %q, want answer", resp.Content)
	}
	if len(reasoningChunks) != 1 || reasoningChunks[0] != "thinking..." {
		t.Errorf("reasoningChunks = %v", reasoningChunks)
	}
	if len(contentChunks) != 1 || contentChunks[0] != "answer" {
		t.Errorf("contentChunks = %v", contentChunks)
	}
	if resp.Usage.TotalTokens != 25 {
		t.Errorf("TotalTokens = %d, want 25", resp.Usage.TotalTokens)
	}
}

func TestRunner_Run_StreamToolCallError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	tool := NewFuncTool("fail", "fail", map[string]any{
		"type": "object",
	}, func(ctx context.Context, params map[string]any) (string, error) {
		return "", errors.New("tool error")
	})

	step := int32(0)
	provider._chatStreamHandler = func(req *llm.Request, handler llm.StreamHandler) error {
		n := atomic.AddInt32(&step, 1)
		if n == 1 {
			handler(llm.StreamChunk{
				ToolCalls: []llm.ToolCall{
					{Index: 0, ID: "call_1", Type: "function", Function: llm.Function{Name: "fail", Arguments: "{}"}},
				},
				IsDone: true,
				FinishReason: llm.FinishReasonToolCalls,
			})
		} else {
			handler(llm.StreamChunk{Content: "handled", IsDone: false})
			handler(llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop})
		}
		return nil
	}

	agent := newMockAgent("stream-fail")
	agent.config.MaxSteps = 10
	agent.tools = []Tool{tool}

	runner := NewRunner(provider, bus)
	resp, err := runner.RunStream(context.Background(), agent, "fail", func(chunk StreamChunk) error {
		return nil
	})

	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Error == nil {
		t.Error("ToolCalls[0].Error should not be nil")
	}
}

func TestRunner_RunStream_DoneCallbackError(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Content: "hello", IsDone: false},
		llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	callCount := 0
	_, err := runner.RunStream(context.Background(), agent, "hi", func(chunk StreamChunk) error {
		callCount++
		if chunk.IsDone {
			return errors.New("done callback error")
		}
		return nil
	})

	if err == nil {
		t.Fatal("RunStream() should error on done callback error")
	}
}

func TestRunner_Run_ToolNotFound(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()

	provider.setResponses(
		&llm.Response{
			Choices: []llm.Choice{
				{
					Message: &llm.Message{
						ToolCalls: []llm.ToolCall{
							{ID: "c1", Type: "function", Function: llm.Function{Name: "nonexistent", Arguments: "{}"}},
						},
					},
				},
			},
		},
		&llm.Response{
			Choices: []llm.Choice{
				{Message: &llm.Message{Content: "done"}},
			},
		},
	)

	runner := NewRunner(provider, bus)
	agent := newMockAgent("test")

	resp, err := runner.Run(context.Background(), agent, "call nonexistent")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Error == nil {
		t.Error("ToolCalls[0].Error should not be nil for missing tool")
	}
}
