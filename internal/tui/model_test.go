package tui

import (
	"errors"
	"testing"
	"time"
)

func TestModelApplyFullLifecycle(t *testing.T) {
	m := NewModel("demo")

	m.Apply(StreamEvent{Kind: EventStart})
	if !m.Busy {
		t.Fatal("expected busy after start")
	}
	if m.Conv.Current() == nil || m.Conv.Current().Role != RoleAssistant {
		t.Fatal("expected an assistant message")
	}

	m.Apply(StreamEvent{Kind: EventReasoning, Reasoning: "thinking…"})
	m.Apply(StreamEvent{Kind: EventContent, Content: "hello "})
	m.Apply(StreamEvent{Kind: EventContent, Content: "world"})

	m.Apply(StreamEvent{
		Kind:     EventToolCall,
		ToolCall: &ToolCallInfo{ToolName: "bash", Args: map[string]any{"command": "ls"}},
	})
	cur := m.Conv.Current()
	if len(cur.Tools) != 1 || cur.Tools[0].State != ToolRunning {
		t.Fatal("expected one running tool")
	}

	m.Apply(StreamEvent{
		Kind: EventToolResult,
		ToolResult: &ToolResultInfo{
			ToolName: "bash",
			Result:   "file.txt",
			Duration: 500 * time.Millisecond,
		},
	})
	if cur.Tools[0].State != ToolSuccess {
		t.Fatal("expected success state")
	}

	m.Apply(StreamEvent{Kind: EventDone, Usage: &Usage{PromptTokens: 1, CompletionTokens: 2}})
	if m.Busy {
		t.Fatal("expected idle after done")
	}
	if m.Conv.Current().Status != StatusDone {
		t.Fatal("expected done status")
	}
	if m.Conv.Current().Content != "hello world" {
		t.Fatalf("content = %q", m.Conv.Current().Content)
	}
	if m.Usage.TotalTokens != 3 || m.Usage.PromptTokens != 1 {
		t.Fatalf("usage = %+v", m.Usage)
	}
}

func TestModelApplyError(t *testing.T) {
	m := NewModel("demo")
	m.Apply(StreamEvent{Kind: EventStart})
	m.Apply(StreamEvent{Kind: EventError, Error: errors.New("boom")})
	if m.Busy {
		t.Fatal("expected idle after error")
	}
	if m.Conv.Current().Status != StatusError {
		t.Fatal("expected error status")
	}
	if m.Err == nil || m.Err.Error() != "boom" {
		t.Fatalf("err = %v", m.Err)
	}
}

func TestConversationRevisionBumpsOnMutation(t *testing.T) {
	c := &Conversation{}
	r0 := c.Revision()
	c.AddUser("hi")
	if c.Revision() == r0 {
		t.Fatal("revision must bump on AddUser")
	}
	r1 := c.Revision()
	c.AppendContent("x")
	if c.Revision() == r1 {
		t.Fatal("revision must bump on AppendContent")
	}
}

func TestToolBlockIDsAreStable(t *testing.T) {
	c := &Conversation{}
	m := c.StartAssistant()
	c.AppendThinking("hmm")
	tc := c.AddToolCall("bash", nil)
	if got := ThinkingPartID(m, 0); got != "m1:think0" {
		t.Fatalf("thinking id = %q", got)
	}
	if got := ToolPartID(m, 0); got != "m1:t0" {
		t.Fatalf("tool id = %q", got)
	}
	_ = tc
}

func TestPartsFollowArrivalOrder(t *testing.T) {
	m := NewModel("demo")
	m.Apply(StreamEvent{Kind: EventStart})
	m.Apply(StreamEvent{Kind: EventReasoning, Reasoning: "first thoughts "})
	m.Apply(StreamEvent{Kind: EventReasoning, Reasoning: "continue"})
	m.Apply(StreamEvent{Kind: EventToolCall, ToolCall: &ToolCallInfo{ToolName: "read"}})
	m.Apply(StreamEvent{Kind: EventReasoning, Reasoning: "more thinking"})
	m.Apply(StreamEvent{Kind: EventContent, Content: "the answer"})

	cur := m.Conv.Current()
	var kinds []PartKind
	for _, p := range cur.Parts {
		kinds = append(kinds, p.Kind)
	}
	want := []PartKind{PartThinking, PartTool, PartThinking, PartContent}
	if len(kinds) != len(want) {
		t.Fatalf("part kinds = %v, want %v", kinds, want)
	}
	for i, k := range want {
		if kinds[i] != k {
			t.Fatalf("part[%d] = %v, want %v (all: %v)", i, kinds[i], k, kinds)
		}
	}
	if cur.Parts[0].Thinking != "first thoughts continue" {
		t.Fatalf("merged thinking = %q", cur.Parts[0].Thinking)
	}
	if cur.Parts[2].Thinking != "more thinking" {
		t.Fatalf("second thinking = %q", cur.Parts[2].Thinking)
	}
	if !m.Conv.TogglePartByID(ThinkingPartID(cur, 1)) {
		t.Fatal("toggle did not match thinking part 1")
	}
	if !cur.Parts[2].ThinkingExpanded {
		t.Fatal("second thinking part not expanded")
	}
	if cur.Parts[0].ThinkingExpanded {
		t.Fatal("first thinking part must stay collapsed")
	}
	if !m.Conv.TogglePartByID(ToolPartID(cur, 0)) {
		t.Fatal("toggle did not match tool part 0")
	}
	if !cur.Parts[1].Tool.Expanded {
		t.Fatal("tool part not expanded")
	}
	if m.Conv.TogglePartByID("nope") {
		t.Fatal("bogus id must not match")
	}
}

func TestSubagentLifecycle(t *testing.T) {
	m := NewModel("demo")

	m.Apply(StreamEvent{Kind: EventStart})
	m.Apply(StreamEvent{Kind: EventSubagentStart, Subagent: &SubagentInfo{Name: "explorer", Prompt: "inspect"}})
	m.Apply(StreamEvent{Kind: EventSubagentChunk, SubagentID: "explorer", Content: "found "})
	m.Apply(StreamEvent{Kind: EventSubagentChunk, SubagentID: "explorer", Content: "the bug"})

	m.Apply(StreamEvent{
		Kind:     EventToolCall,
		ToolCall: &ToolCallInfo{ToolName: "read", SubagentID: "explorer", Args: map[string]any{"path": "x"}},
	})
	m.Apply(StreamEvent{
		Kind:       EventToolResult,
		ToolResult: &ToolResultInfo{ToolName: "read", SubagentID: "explorer", Result: "ok", Duration: 500 * time.Millisecond},
	})

	m.Apply(StreamEvent{
		Kind:     EventToolCall,
		ToolCall: &ToolCallInfo{ToolName: "bash", Args: map[string]any{"command": "ls"}},
	})

	cur := m.Conv.Current()
	if len(cur.Subagents) != 1 {
		t.Fatalf("subagents = %d, want 1", len(cur.Subagents))
	}
	sa := cur.Subagents[0]
	if sa.Status != StatusRunning {
		t.Fatalf("subagent status = %v, want running", sa.Status)
	}
	if sa.Content != "found the bug" {
		t.Fatalf("subagent content = %q", sa.Content)
	}
	if len(sa.Tools) != 1 || sa.Tools[0].Name != "read" || sa.Tools[0].State != ToolSuccess {
		t.Fatalf("subagent tools = %+v", sa.Tools)
	}
	if len(cur.Tools) != 1 || cur.Tools[0].Name != "bash" {
		t.Fatalf("main tools = %+v", cur.Tools)
	}

	m.Apply(StreamEvent{
		Kind:       EventSubagentDone,
		SubagentID: "explorer",
		SubagentResult: &SubagentResult{
			Name: "explorer", Result: "all good", Duration: 2 * time.Second,
		},
	})
	if sa.Status != StatusDone {
		t.Fatalf("subagent status = %v, want done", sa.Status)
	}
	if sa.Duration != 2*time.Second || sa.Result != "all good" {
		t.Fatalf("subagent result = %+v", sa)
	}
}

func TestSubagentError(t *testing.T) {
	m := NewModel("demo")
	m.Apply(StreamEvent{Kind: EventStart})
	m.Apply(StreamEvent{Kind: EventSubagentStart, Subagent: &SubagentInfo{Name: "planner"}})
	m.Apply(StreamEvent{
		Kind:           EventSubagentDone,
		SubagentID:     "planner",
		SubagentResult: &SubagentResult{Name: "planner", Error: errors.New("boom"), Duration: 100 * time.Millisecond},
	})
	sa := m.Conv.Current().Subagents[0]
	if sa.Status != StatusError || sa.Error == nil || sa.Error.Error() != "boom" {
		t.Fatalf("subagent error state = %+v", sa)
	}
}
