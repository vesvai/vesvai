package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

func TestChatAppendAndSelect(t *testing.T) {
	c := NewChat()
	c.AppendItem(&ChatItem{Kind: ItemUser, Text: "hello"})
	c.AppendItem(&ChatItem{Kind: ItemAssistant, Text: "hi"})
	if !c.HasItems() {
		t.Fatal("expected items")
	}
	if c.sel != 1 {
		t.Errorf("selection = %d, want last item", c.sel)
	}
	c.HandleKey(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if c.sel != 0 {
		t.Errorf("after up selection = %d, want 0", c.sel)
	}
}

func TestChatActivateThinking(t *testing.T) {
	c := NewChat()
	it := &ChatItem{Kind: ItemThinking, Reasoning: "secret reasoning"}
	c.AppendItem(it)
	c.SetOnActivate(func(item *ChatItem) { item.Expanded = !item.Expanded })
	c.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if !it.Expanded {
		t.Error("Enter should expand the thinking item")
	}
}

func TestChatPrepend(t *testing.T) {
	c := NewChat()
	c.AppendItem(&ChatItem{Kind: ItemUser, Text: "b"})
	old := len(c.items)
	c.PrependItems([]*ChatItem{{Kind: ItemUser, Text: "a"}})
	if len(c.items) != old+1 {
		t.Errorf("items = %d, want %d", len(c.items), old+1)
	}
	if c.items[0].Text != "a" {
		t.Errorf("first item = %q, want a", c.items[0].Text)
	}
}

func TestChatClickSelects(t *testing.T) {
	c := NewChat()
	var activated *ChatItem
	c.AppendItem(&ChatItem{Kind: ItemUser, Text: "first"})
	thinking := &ChatItem{Kind: ItemThinking}
	c.AppendItem(thinking)
	c.SetOnActivate(func(item *ChatItem) { activated = item })
	c.HandleClick(5, 7, 0)
	if activated != thinking {
		t.Error("click on row 1 should activate the thinking item")
	}
}

func TestChatSetItemsResets(t *testing.T) {
	c := NewChat()
	c.AppendItem(&ChatItem{Kind: ItemUser, Text: "x"})
	c.SetItems([]*ChatItem{{Kind: ItemAssistant, Text: "y"}})
	if len(c.items) != 1 || c.items[0].Text != "y" {
		t.Errorf("items after SetItems = %+v", c.items)
	}
}

func TestChatDrawSmoke(t *testing.T) {
	styles.RegisterDefaults()
	styles.Set("dark")
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	defer s.Fini()
	s.SetSize(100, 30)

	c := NewChat()
	c.AppendItem(&ChatItem{Kind: ItemUser, Text: "Please fix the bug in #main"})
	c.AppendItem(&ChatItem{Kind: ItemThinking, Reasoning: "Let me think about this\nstep by step."})
	c.AppendItem(&ChatItem{Kind: ItemAssistant, Text: "I'll look at the code first.\n\n```go\npackage main\n```"})
	c.AppendItem(&ChatItem{Kind: ItemTool, ToolName: "edit", ToolArgs: `{"filePath":"a.go","oldString":"x","newString":"y"}`, Diff: ComputeDiff("x", "y"), ToolOutput: "edited"})
	c.AppendItem(&ChatItem{Kind: ItemSubagent, SubagentName: "developer", SubagentStatus: "running"})
	c.AppendItem(&ChatItem{Kind: ItemFinished})

	c.Draw(s, layout.Region{Left: 0, Top: 0, Width: 100, Height: 28}, true)
	s.Show()
}

func TestChatDrawExpandedSmoke(t *testing.T) {
	styles.RegisterDefaults()
	styles.Set("dark")
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	defer s.Fini()
	s.SetSize(100, 30)

	c := NewChat()
	tool := &ChatItem{Kind: ItemTool, ToolName: "edit", ToolArgs: `{"oldString":"a","newString":"b"}`, Diff: ComputeDiff("a\nb", "a\nc"), ToolOutput: "done"}
	tool.Expanded = true
	c.AppendItem(tool)
	think := &ChatItem{Kind: ItemThinking, Reasoning: "thinking text\nmore"}
	think.Expanded = true
	c.AppendItem(think)
	c.Draw(s, layout.Region{Left: 0, Top: 0, Width: 100, Height: 28}, true)
	s.Show()
}
