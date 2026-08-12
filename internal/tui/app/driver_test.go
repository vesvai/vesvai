package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/tui"
)

type scriptedProvider struct {
	mu      sync.Mutex
	steps   [][]llm.StreamChunk
	calls   int
	reqs    []*llm.Request
	streams bool
}

func newScriptedProvider(steps ...[]llm.StreamChunk) *scriptedProvider {
	return &scriptedProvider{steps: steps, streams: true}
}

func (p *scriptedProvider) Name() string { return "scripted" }
func (p *scriptedProvider) ListModels(context.Context) ([]llm.Model, error) {
	return []llm.Model{{ID: "mock-1"}}, nil
}

func (p *scriptedProvider) Chat(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return nil, errors.New("unexpected non-stream call")
}

func (p *scriptedProvider) ChatStream(ctx context.Context, req *llm.Request, handler llm.StreamHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reqs = append(p.reqs, req)
	var chunks []llm.StreamChunk
	if p.calls < len(p.steps) {
		chunks = p.steps[p.calls]
		p.calls++
	} else {
		chunks = []llm.StreamChunk{{Content: "…", IsDone: true, FinishReason: llm.FinishReasonStop}}
	}
	for _, c := range chunks {
		if err := handler(c); err != nil {
			return err
		}
	}
	return nil
}

type streamTool struct{}

func (t *streamTool) Name() string        { return "streamer" }
func (t *streamTool) Description() string { return "streams work" }
func (t *streamTool) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (t *streamTool) Handle(ctx context.Context, params map[string]any) (string, error) {
	return "done", nil
}
func (t *streamTool) HandleStream(ctx context.Context, params map[string]any, callback agent.StreamCallback) (string, error) {
	callback(agent.StreamChunk{SubagentStart: &agent.SubagentStartInfo{Name: "streamer", Prompt: "work"}})
	callback(agent.StreamChunk{SubagentID: "sub-run-1", Content: "streaming…"})
	callback(agent.StreamChunk{SubagentDone: &agent.SubagentDoneInfo{Name: "streamer", Result: "finished"}})
	return "finished", nil
}

func newTestDriver(t *testing.T, p llm.Provider) (*AgentDriver, *agent.Runner) {
	t.Helper()
	registry := agent.NewToolRegistry()
	registry.Register(&streamTool{})
	runner := agent.NewRunner(p, nil, registry)
	return &AgentDriver{runner: runner}, runner
}

func collectEvents(t *testing.T, d *AgentDriver, req RunRequest) []tui.StreamEvent {
	t.Helper()
	var evs []tui.StreamEvent
	d.Run(context.Background(), req, func(ev tui.StreamEvent) {
		evs = append(evs, ev)
	})
	return evs
}

func kindOf(evs []tui.StreamEvent) []tui.StreamEventKind {
	out := make([]tui.StreamEventKind, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.Kind)
	}
	return out
}

func TestAgentDriverMapsFullRun(t *testing.T) {
	p := newScriptedProvider(
		[]llm.StreamChunk{
			{Reasoning: "think…"},
			{Content: "hello"},
			{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{PromptTokens: 10, CompletionTokens: 5}},
			{ToolCalls: []llm.ToolCall{{Index: 0, ID: "c1", Type: "function", Function: llm.Function{Name: "streamer", Arguments: `{}`}}}},
			{IsDone: true, FinishReason: llm.FinishReasonToolCalls},
		},
		[]llm.StreamChunk{
			{Content: "final answer"},
			{IsDone: true, FinishReason: llm.FinishReasonStop, Usage: &llm.Usage{PromptTokens: 20, CompletionTokens: 30}},
		},
	)
	d, _ := newTestDriver(t, p)

	evs := collectEvents(t, d, RunRequest{Text: "do the thing"})

	wantKinds := []tui.StreamEventKind{
		tui.EventStart,
		tui.EventReasoning,
		tui.EventContent,
		tui.EventToolCall,
		tui.EventSubagentStart,
		tui.EventSubagentChunk,
		tui.EventSubagentDone,
		tui.EventToolResult,
		tui.EventContent,
		tui.EventDone,
	}
	got := kindOf(evs)
	if len(got) != len(wantKinds) {
		t.Fatalf("event kinds = %v, want %v", got, wantKinds)
	}
	for i, w := range wantKinds {
		if got[i] != w {
			t.Fatalf("event[%d] = %v, want %v (all: %v)", i, got[i], w, got)
		}
	}

	if evs[len(evs)-1].Usage.TotalTokens != 65 {
		t.Fatalf("final usage = %+v, want total 65", evs[len(evs)-1].Usage)
	}
	if evs[3].ToolCall.ToolName != "streamer" {
		t.Fatalf("tool call name = %q, want streamer", evs[3].ToolCall.ToolName)
	}
	if evs[5].SubagentID != "sub-run-1" {
		t.Fatalf("subagent chunk id = %q, want sub-run-1", evs[5].SubagentID)
	}
}

func TestAgentDriverMentionsRouteToSubagent(t *testing.T) {
	p := newScriptedProvider(
		[]llm.StreamChunk{{Content: "ok"}, {IsDone: true, FinishReason: llm.FinishReasonStop}},
	)
	d, _ := newTestDriver(t, p)

	collectEvents(t, d, RunRequest{Text: "@explorer look around"})

	if p.calls == 0 {
		t.Fatal("provider never called")
	}
	req := p.reqs[0]
	found := false
	for _, m := range req.Messages {
		if m.Role == llm.RoleUser {
			if s, ok := m.Content.(string); ok && s == "look around" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("user message not stripped to 'look around': %+v", req.Messages)
	}
}

func TestAgentDriverMentionBoundary(t *testing.T) {
	name, rest := stripLeadingMention("@plannerize the plan")
	if name != "orchestrator" || rest != "@plannerize the plan" {
		t.Fatalf("plannerize: name=%q rest=%q", name, rest)
	}
	name, rest = stripLeadingMention("@orchestrator go do it")
	if name != "orchestrator" || rest != "go do it" {
		t.Fatalf("orchestrator: name=%q rest=%q", name, rest)
	}
	name, rest = stripLeadingMention("fix the bug")
	if name != "orchestrator" || rest != "fix the bug" {
		t.Fatalf("plain: name=%q rest=%q", name, rest)
	}
}

func TestAgentDriverSeedsHistory(t *testing.T) {
	p := newScriptedProvider(
		[]llm.StreamChunk{{Content: "ok"}, {IsDone: true, FinishReason: llm.FinishReasonStop}},
	)
	d, _ := newTestDriver(t, p)

	history := []llm.Message{
		llm.UserMessage("earlier question"),
		llm.AssistantMessage("earlier answer"),
	}
	collectEvents(t, d, RunRequest{Text: "follow up", History: history})

	req := p.reqs[0]
	userCount := 0
	for _, m := range req.Messages {
		if m.Role == llm.RoleUser {
			userCount++
		}
	}
	if userCount != 2 {
		t.Fatalf("user messages in request = %d, want 2 (history + prompt)", userCount)
	}
}

func TestAgentDriverErrorEmitsEventError(t *testing.T) {
	failing := &failProvider{err: errors.New("boom")}
	d, _ := newTestDriver(t, failing)

	evs := collectEvents(t, d, RunRequest{Text: "hi"})

	if len(evs) < 2 {
		t.Fatalf("events = %v, want start + error", kindOf(evs))
	}
	last := evs[len(evs)-1]
	if last.Kind != tui.EventError || last.Error == nil || !strings.Contains(last.Error.Error(), "boom") {
		t.Fatalf("last event = %+v, want EventError with 'boom'", last)
	}
}

type failProvider struct {
	err error
}

func (f *failProvider) Name() string { return "fail" }
func (f *failProvider) ListModels(context.Context) ([]llm.Model, error) {
	return nil, f.err
}
func (f *failProvider) Chat(context.Context, *llm.Request) (*llm.Response, error) {
	return nil, f.err
}
func (f *failProvider) ChatStream(context.Context, *llm.Request, llm.StreamHandler) error {
	return f.err
}

func TestTUIApproverChannel(t *testing.T) {
	a := NewTUIApprover()
	ctx := context.Background()
	done := make(chan permission.Decision, 1)
	go func() {
		d, err := a.RequestApproval(ctx, "bash", map[string]any{"command": "ls"}, "needs approval")
		if err != nil {
			done <- permission.DecisionDeny
			return
		}
		done <- d
	}()

	select {
	case req := <-a.reqs:
		if req.toolName != "bash" {
			t.Fatalf("toolName = %q, want bash", req.toolName)
		}
		req.resp <- permission.DecisionAllowAlways
	case <-done:
		t.Fatal("approver returned before the UI answered")
	}

	if d := <-done; d != permission.DecisionAllowAlways {
		t.Fatalf("decision = %v, want AllowAlways", d)
	}
}
