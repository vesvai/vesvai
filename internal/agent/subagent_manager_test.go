package agent

import (
	"context"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/llm"
)

func TestSubagentManager_StartAndGet(t *testing.T) {
	m := NewSubagentManager()
	run := m.Start("run-1", "explorer", "find files", "orchestrator")
	if run.Name != "explorer" || run.Status != SubagentStatusRunning {
		t.Errorf("run = %+v", run)
	}

	got, ok := m.Get("run-1")
	if !ok || got != run {
		t.Error("Get should return the same run")
	}
}

func TestSubagentManager_SetResult(t *testing.T) {
	m := NewSubagentManager()
	run := m.Start("run-1", "planner", "plan", "orch")

	run.SetResult("done", nil)
	if run.Status != SubagentStatusCompleted {
		t.Errorf("status = %s, want completed", run.Status)
	}
	if run.Result != "done" {
		t.Errorf("result = %q", run.Result)
	}
}

func TestSubagentManager_SetResultWithError(t *testing.T) {
	m := NewSubagentManager()
	run := m.Start("run-1", "planner", "plan", "orch")

	err := context.DeadlineExceeded
	run.SetResult("", err)
	if run.Status != SubagentStatusFailed {
		t.Errorf("status = %s, want failed", run.Status)
	}
}

func TestSubagentManager_Active(t *testing.T) {
	m := NewSubagentManager()
	m.Start("r1", "a", "p", "parent")
	m.Start("r2", "b", "p", "parent")
	run2, _ := m.Get("r2")
	run2.SetResult("done", nil)

	active := m.Active()
	if len(active) != 1 {
		t.Fatalf("active = %d, want 1", len(active))
	}
	if active[0].Name != "a" {
		t.Errorf("active[0].Name = %q, want a", active[0].Name)
	}
}

func TestSubagentManager_All(t *testing.T) {
	m := NewSubagentManager()
	m.Start("r1", "a", "p", "parent")
	m.Start("r2", "b", "p", "parent")
	all := m.All()
	if len(all) != 2 {
		t.Fatalf("all = %d, want 2", len(all))
	}
}

func TestSubagentManager_Cancel(t *testing.T) {
	m := NewSubagentManager()
	m.Start("r1", "a", "p", "parent")
	if !m.Cancel("r1") {
		t.Error("Cancel should succeed for running task")
	}
	run, _ := m.Get("r1")
	if run.Status != SubagentStatusCancelled {
		t.Errorf("status = %s, want cancelled", run.Status)
	}
	if m.Cancel("r1") {
		t.Error("Cancel should fail for already-cancelled task")
	}
}

func TestSubagentManager_Snapshot(t *testing.T) {
	m := NewSubagentManager()
	run := m.Start("r1", "explorer", "find", "orch")
	run.AppendContent("found something")
	run.AppendReasoning("thinking...")
	run.AppendToolCall(SubagentToolEvent{ToolName: "glob", Result: "5 files"})

	snap := run.Snapshot()
	if snap.Content != "found something" || snap.Reasoning != "thinking..." {
		t.Errorf("snapshot content/reasoning mismatch")
	}
	if len(snap.ToolCalls) != 1 || snap.ToolCalls[0].ToolName != "glob" {
		t.Errorf("snapshot tool calls mismatch")
	}
}

func TestMailbox_PostAndCollect(t *testing.T) {
	mb := NewMailbox()
	mb.Post("orchestrator", "explorer", "find src files")
	mb.Post("orchestrator", "planner", "design API")

	if !mb.HasMessages("explorer") {
		t.Error("explorer should have messages")
	}

	explorerMsgs := mb.Collect("explorer")
	if len(explorerMsgs) != 1 || explorerMsgs[0].Content != "find src files" {
		t.Errorf("explorer msgs = %+v", explorerMsgs)
	}

	if mb.HasMessages("explorer") {
		t.Error("explorer mailbox should be empty after collect")
	}
}

func TestSubagentTool_BackgroundDispatch(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setStreamChunks(
		llm.StreamChunk{Content: "bg result", IsDone: false},
		llm.StreamChunk{IsDone: true, FinishReason: llm.FinishReasonStop},
	)

	runner := NewRunner(provider, bus, NewToolRegistry())
	agent := newMockAgent("inner-bg")

	tool := NewSubagentTool(runner, agent, "bgworker")
	result, err := tool.Handle(context.Background(), map[string]any{
		"prompt":     "do background work",
		"background": true,
	})
	if err != nil {
		t.Fatalf("Handle background error: %v", err)
	}
	if result == "" {
		t.Error("background dispatch should return a run ID message")
	}

	time.Sleep(100 * time.Millisecond)

	active := runner.Subagents().Active()
	if len(active) != 0 {
		t.Errorf("after sleep, active = %d, want 0 (bg should finish)", len(active))
	}

	all := runner.Subagents().All()
	if len(all) != 1 {
		t.Fatalf("All() = %d, want 1", len(all))
	}
	if all[0].Content != "bg result" {
		t.Errorf("bg run content = %q, want 'bg result'", all[0].Content)
	}
}

func TestSubagentTool_SetRunner(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	agent := newMockAgent("inner")

	tool := NewSubagentTool(nil, agent, "test")
	if tool.getRunner() != nil {
		t.Error("runner should be nil initially")
	}

	runner := NewRunner(provider, bus, NewToolRegistry())
	tool.SetRunner(runner)
	if tool.getRunner() != runner {
		t.Error("runner should be bound after SetRunner")
	}
}

func TestMessageTool_PostAndCollect(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	runner := NewRunner(provider, bus, NewToolRegistry())

	msgTool := NewMessageTool(runner)
	collectTool := NewCollectMessagesTool(runner)

	ctx := WithAgentContext(context.Background(), "orchestrator", "sess-1")
	_, err := msgTool.Handle(ctx, map[string]any{
		"to":      "explorer",
		"content": "find all Go files",
	})
	if err != nil {
		t.Fatalf("message: %v", err)
	}

	ctx2 := WithAgentContext(context.Background(), "explorer", "sess-1")
	result, err := collectTool.Handle(ctx2, nil)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if result == "" || result == "no messages" {
		t.Errorf("collect result = %q", result)
	}
}
func TestSubagentTool_InheritsParentModel(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	provider := newMockProvider()
	provider.setResponses(&llm.Response{
		Choices: []llm.Choice{
			{Message: &llm.Message{Content: "child result"}},
		},
	})

	runner := NewRunner(provider, bus, NewToolRegistry())
	child := newMockAgent("child")
	child.config.Model = ""
	tool := NewSubagentTool(runner, child, "explorer")

	ctx := WithAgentContext(context.Background(), "orchestrator", "sess-1")
	ctx = WithModelContext(ctx, "gpt-4o")

	_, err := tool.Handle(ctx, map[string]any{"prompt": "explore", "background": false})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if provider.lastRequest == nil || provider.lastRequest.Model != "gpt-4o" {
		t.Errorf("child request model = %q, want gpt-4o (inherited from parent)", provider.lastRequest.Model)
	}
}

func TestResolveModel_PrefersAgentModel(t *testing.T) {
	cfg := AgentConfig{Model: "agent-model"}
	ctx := WithModelContext(context.Background(), "parent-model")
	if got := resolveModel(ctx, cfg); got != "agent-model" {
		t.Errorf("resolveModel = %q, want agent-model", got)
	}
}

func TestResolveModel_InheritsParent(t *testing.T) {
	ctx := WithModelContext(context.Background(), "parent-model")
	if got := resolveModel(ctx, AgentConfig{}); got != "parent-model" {
		t.Errorf("resolveModel = %q, want parent-model", got)
	}
}
