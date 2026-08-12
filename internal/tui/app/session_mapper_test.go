package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestConvToMessagesRoundTrip(t *testing.T) {
	img := filepath.Join(t.TempDir(), "x.png")
	if err := os.WriteFile(img, []byte("fake-png"), 0644); err != nil {
		t.Fatal(err)
	}

	conv := &tui.Conversation{}

	u := conv.AddUser("look at this")
	u.Attachments = []*tui.Attachment{
		{Path: img, Name: "x.png", Kind: "image", Size: 10},
	}
	a := conv.StartAssistant()
	a.Thinking = "hmm…"
	a.Content = "here is the plan"
	tc := conv.AddToolCall("read", map[string]any{"path": "main.go"})
	conv.FinishToolCall(tc, "package main", nil, 0)
	conv.MarkDone()

	msgs := ConvToMessages(conv)

	if len(msgs) != 3 {
		t.Fatalf("messages = %d, want 3", len(msgs))
	}
	if msgs[0].Role != llm.RoleUser {
		t.Fatalf("msg[0] role = %v, want user", msgs[0].Role)
	}
	content, ok := msgs[0].Content.(llm.Content)
	if !ok || content.Text != "look at this" || len(content.Attachments) != 1 {
		t.Fatalf("user content = %#v, want text + 1 attachment", msgs[0].Content)
	}
	if msgs[1].Role != llm.RoleAssistant || msgs[1].Content != "here is the plan" {
		t.Fatalf("assistant msg = %#v", msgs[1])
	}
	if msgs[1].Reasoning != "hmm…" {
		t.Fatalf("reasoning = %v, want 'hmm…'", msgs[1].Reasoning)
	}
	if len(msgs[1].ToolCalls) != 1 || msgs[1].ToolCalls[0].Function.Name != "read" {
		t.Fatalf("tool calls = %#v", msgs[1].ToolCalls)
	}
	if msgs[2].Role != llm.RoleTool || msgs[2].Content != "package main" {
		t.Fatalf("tool msg = %#v", msgs[2])
	}
	if msgs[2].ToolCallID != msgs[1].ToolCalls[0].ID {
		t.Fatalf("tool result id %q != call id %q", msgs[2].ToolCallID, msgs[1].ToolCalls[0].ID)
	}

	out := &tui.Conversation{}
	MessagesToConv(out, msgs)

	if len(out.Messages) != 2 {
		t.Fatalf("reloaded messages = %d, want 2", len(out.Messages))
	}
	if out.Messages[0].Role != tui.RoleUser || out.Messages[0].Content != "look at this" {
		t.Fatalf("reloaded user = %#v", out.Messages[0])
	}
	if len(out.Messages[0].Attachments) != 1 {
		t.Fatalf("reloaded attachments = %d, want 1", len(out.Messages[0].Attachments))
	}
	if out.Messages[1].Thinking != "hmm…" {
		t.Fatalf("reloaded thinking = %q", out.Messages[1].Thinking)
	}
	if len(out.Messages[1].Tools) != 1 {
		t.Fatalf("reloaded tools = %d, want 1", len(out.Messages[1].Tools))
	}
	rt := out.Messages[1].Tools[0]
	if rt.Name != "read" || rt.Result != "package main" || rt.State != tui.ToolSuccess {
		t.Fatalf("reloaded tool = %#v", rt)
	}
}

func TestCurrentHistoryExcludesNewest(t *testing.T) {
	conv := &tui.Conversation{}
	conv.AddUser("first")
	conv.StartAssistant()
	conv.Current().Content = "reply"
	conv.MarkDone()
	conv.AddUser("second")

	hist := (&AgentDriver{}).CurrentHistory(conv)
	if len(hist) != 2 {
		t.Fatalf("history = %d, want 2 (excludes 'second')", len(hist))
	}
	if hist[0].Role != llm.RoleUser || hist[0].Content != "first" {
		t.Fatalf("history[0] = %#v", hist[0])
	}

	empty := &tui.Conversation{}
	empty.AddUser("only")
	if h := (&AgentDriver{}).CurrentHistory(empty); h != nil {
		t.Fatalf("single message history = %#v, want nil", h)
	}
}

func TestPartsRoundTripWithSubagent(t *testing.T) {
	conv := &tui.Conversation{}
	conv.AddUser("do the thing")

	m := tui.NewModel("demo")
	m.Conv = conv
	m.Apply(tui.StreamEvent{Kind: tui.EventStart})
	m.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "first thoughts "})
	m.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "more"})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "bash", Args: map[string]any{"command": "ls"}}})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "bash", Result: "files", Duration: 300 * time.Millisecond}})
	m.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "second thoughts"})
	m.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "delegating now"})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentStart, Subagent: &tui.SubagentInfo{Name: "explorer", Prompt: "inspect"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "explorer", Content: "thinking step", SubagentReasoning: true})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "read", SubagentID: "explorer", Args: map[string]any{"path": "x"}}})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "read", SubagentID: "explorer", Result: "code", Duration: 100 * time.Millisecond}})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "explorer", Content: "found the bug"})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentDone, SubagentID: "explorer", SubagentResult: &tui.SubagentResult{Name: "explorer", Result: "bug at line 3", Duration: 2 * time.Second}})
	m.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "summarizing"})
	m.Apply(tui.StreamEvent{Kind: tui.EventDone})

	msgs := ConvToMessages(conv)
	parts := ConvToSessionParts(conv)

	out := &tui.Conversation{}
	MessagesToConv(out, msgs)
	ApplySessionParts(out, parts)

	if len(out.Messages) != 2 {
		t.Fatalf("reloaded messages = %d, want 2", len(out.Messages))
	}
	am := out.Messages[1]

	var kinds []tui.PartKind
	for _, p := range am.Parts {
		kinds = append(kinds, p.Kind)
	}
	want := []tui.PartKind{
		tui.PartThinking, tui.PartTool, tui.PartThinking,
		tui.PartContent, tui.PartSubagent, tui.PartContent,
	}
	if len(kinds) != len(want) {
		t.Fatalf("part kinds = %v, want %v", kinds, want)
	}
	for i, k := range want {
		if kinds[i] != k {
			t.Fatalf("part[%d] = %v, want %v (all: %v)", i, kinds[i], k, kinds)
		}
	}

	if am.Parts[0].Thinking != "first thoughts more" {
		t.Fatalf("part[0].Thinking = %q", am.Parts[0].Thinking)
	}
	if am.Parts[2].Thinking != "second thoughts" {
		t.Fatalf("part[2].Thinking = %q", am.Parts[2].Thinking)
	}
	if am.Parts[1].Tool == nil || am.Parts[1].Tool.Name != "bash" || am.Parts[1].Tool.Result != "files" {
		t.Fatalf("tool part = %+v", am.Parts[1].Tool)
	}

	if len(am.Subagents) != 1 {
		t.Fatalf("subagents = %d, want 1", len(am.Subagents))
	}
	sa := am.Subagents[0]
	if sa.Name != "explorer" || sa.Prompt != "inspect" {
		t.Fatalf("subagent = %+v", sa)
	}
	if sa.Thinking != "thinking step" {
		t.Fatalf("subagent thinking = %q", sa.Thinking)
	}
	if sa.Content != "found the bug" {
		t.Fatalf("subagent content = %q", sa.Content)
	}
	if sa.Result != "bug at line 3" || sa.Duration != 2*time.Second {
		t.Fatalf("subagent result = %q dur=%v", sa.Result, sa.Duration)
	}
	if len(sa.Tools) != 1 || sa.Tools[0].Name != "read" || sa.Tools[0].Result != "code" || sa.Tools[0].State != tui.ToolSuccess {
		t.Fatalf("subagent tools = %+v", sa.Tools)
	}
	var saKinds []tui.PartKind
	for _, p := range sa.Parts {
		saKinds = append(saKinds, p.Kind)
	}
	wantSA := []tui.PartKind{tui.PartThinking, tui.PartTool, tui.PartContent}
	for i, k := range wantSA {
		if saKinds[i] != k {
			t.Fatalf("subagent part[%d] = %v, want %v", i, saKinds[i], k)
		}
	}
	if am.Parts[4].Subagent != sa {
		t.Fatal("subagent part must reference the restored subagent")
	}
}

func TestSessionStoreRoundTripParts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	conv := &tui.Conversation{}
	conv.AddUser("q")
	m := tui.NewModel("demo")
	m.Conv = conv
	m.Apply(tui.StreamEvent{Kind: tui.EventStart})
	m.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "think"})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "read"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "read", Result: "r"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentStart, Subagent: &tui.SubagentInfo{Name: "explorer", Prompt: "go"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "explorer", Content: "secret transcript"})
	m.Apply(tui.StreamEvent{Kind: tui.EventSubagentDone, SubagentID: "explorer", SubagentResult: &tui.SubagentResult{Name: "explorer", Result: "done"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "answer"})
	m.Apply(tui.StreamEvent{Kind: tui.EventDone})

	store, err := session.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess := &session.Session{
		ID:       "s1",
		Title:    "round trip",
		Messages: ConvToMessages(conv),
		Parts:    ConvToSessionParts(conv),
	}
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load("s1")
	if err != nil {
		t.Fatal(err)
	}

	out := &tui.Conversation{}
	MessagesToConv(out, loaded.Messages)
	ApplySessionParts(out, loaded.Parts)

	am := out.Messages[1]
	if len(am.Subagents) != 1 || am.Subagents[0].Content != "secret transcript" {
		t.Fatalf("subagent not restored from store: %+v", am.Subagents)
	}
	if len(am.Parts) != 4 {
		t.Fatalf("parts = %d, want 4 (thinking, tool, subagent, content)", len(am.Parts))
	}
	if am.Parts[3].Kind != tui.PartContent || am.Parts[3].Content != "answer" {
		t.Fatalf("last part = %+v", am.Parts[3])
	}
}

func TestToolResultsMatchByCallID(t *testing.T) {
	conv := &tui.Conversation{}
	conv.AddUser("q")
	m := tui.NewModel("demo")
	m.Conv = conv
	m.Apply(tui.StreamEvent{Kind: tui.EventStart})
	m.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "working"})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "read", Args: map[string]any{"path": "a"}}})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "read", Result: "AAA result"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "grep", Args: map[string]any{"pattern": "x"}}})
	m.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "grep", Result: "BBB result"}})
	m.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "done"})
	m.Apply(tui.StreamEvent{Kind: tui.EventDone})

	out := &tui.Conversation{}
	MessagesToConv(out, ConvToMessages(conv))
	ApplySessionParts(out, ConvToSessionParts(conv))

	tools := out.Messages[1].Tools
	if len(tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(tools))
	}
	if tools[0].Name != "read" || tools[0].Result != "AAA result" || tools[0].State != tui.ToolSuccess {
		t.Fatalf("tool[0] = %+v, want read/AAA result/success", tools[0])
	}
	if tools[1].Name != "grep" || tools[1].Result != "BBB result" || tools[1].State != tui.ToolSuccess {
		t.Fatalf("tool[1] = %+v, want grep/BBB result/success", tools[1])
	}

	if !out.TogglePartByID(tui.ToolPartID(out.Messages[1], 0)) {
		t.Fatal("toggle tool part 0 failed")
	}
	if !tools[0].Expanded {
		t.Fatal("tool[0] not expanded")
	}
}

func TestTitleFromText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"fix the bug", "fix the bug"},
		{"", "(untitled)"},
		{"  spaced  \n\n", "spaced"},
		{"a very long title that definitely exceeds the sixty character limit for session titles and keeps going", "a very long title that definitely exceeds the sixty chara..."},
	}
	for _, c := range cases {
		if got := titleFromText(c.in); got != c.want {
			t.Fatalf("titleFromText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
