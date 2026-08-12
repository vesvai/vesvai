package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestInterleavedPartsRenderInOrder(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("do it")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "first thoughts"})
	model.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "bash", Args: map[string]any{"command": "ls"}}})
	model.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "bash", Result: "out"}})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "second thoughts"})
	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "final answer"})
	model.Apply(tui.StreamEvent{Kind: tui.EventDone})

	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	if strings.Count(joined, "Thinking") != 2 {
		t.Fatalf("expected 2 thinking blocks, got:\n%s", joined)
	}
	order := []string{"Thinking", "bash", "Thinking", "final answer"}
	pos := -1
	for _, want := range order {
		p := strings.Index(joined[pos+1:], want)
		if p < 0 {
			t.Fatalf("segment %q missing in order after %q:\n%s", want, order, joined)
		}
		pos += p + 1
	}
}

func TestInterleavedThinkingTogglesIndependently(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("do it")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "AAAA first"})
	model.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "bash"}})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "BBBB second"})
	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "answer"})
	model.Apply(tui.StreamEvent{Kind: tui.EventDone})

	cur := model.Conv.Current()

	toggleItem(model, tui.ThinkingPartID(cur, 0))
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "AAAA first") {
		t.Fatalf("first thinking body missing after expand:\n%s", joined)
	}
	if strings.Contains(joined, "BBBB second") {
		t.Fatalf("second thinking body leaked while collapsed:\n%s", joined)
	}

	toggleItem(model, tui.ThinkingPartID(cur, 0))
	toggleItem(model, tui.ThinkingPartID(cur, 1))
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if strings.Contains(joined, "AAAA first") {
		t.Fatalf("first thinking body still visible after collapse:\n%s", joined)
	}
	if !strings.Contains(joined, "BBBB second") {
		t.Fatalf("second thinking body missing after expand:\n%s", joined)
	}
}
