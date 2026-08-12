package layouts

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestSubagentStatusLineFlow(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())

	model.Conv.AddUser("check the layout")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})

	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner", Prompt: "plan the work"},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner", Content: "step one, step two…"})
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "planner") || !strings.Contains(joined, "Thinking") {
		t.Fatalf("thinking status line missing:\n%s", joined)
	}
	if strings.Contains(joined, "step one") {
		t.Fatalf("subagent transcript leaked into the chat:\n%s", joined)
	}

	model.Apply(tui.StreamEvent{
		Kind:     tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{ToolName: "read", SubagentID: "planner", Args: map[string]any{"path": "x"}},
	})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "read") {
		t.Fatalf("tool name missing on the status line:\n%s", joined)
	}
	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "While planner works, I continue.\n"})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "While planner works") {
		t.Fatalf("main content missing:\n%s", joined)
	}

	model.Apply(tui.StreamEvent{
		Kind:       tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{ToolName: "read", SubagentID: "planner", Result: "ok", Duration: 400 * time.Millisecond},
	})
	model.Apply(tui.StreamEvent{
		Kind:       tui.EventSubagentDone,
		SubagentID: "planner",
		SubagentResult: &tui.SubagentResult{
			Name: "planner", Result: "here is the long final answer", Duration: 2100 * time.Millisecond,
		},
	})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "✔ planner · 1 tools · 2.1s") {
		t.Fatalf("completion summary missing:\n%s", joined)
	}
	if strings.Contains(joined, "long final answer") {
		t.Fatalf("subagent result text leaked into the chat:\n%s", joined)
	}
}

func TestSubagentStatusLineUsesSubagentColor(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("go")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner", Prompt: "plan the work"},
	})
	renderFrame(t, l, s)

	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	pal := tui.DefaultDark()
	found := false
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 {
			fg, _, _ := c.Style.Decompose()
			if fg == pal.Subagent {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("no cell with the subagent color found")
	}
	_ = w
}

func TestSubagentLineVisibleWhileWorking(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("go")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "content before delegation\n"})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner", Prompt: "plan"},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "main agent continues here\n"})
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	subRow := -1
	beforeRow := -1
	afterRow := -1
	for i, r := range rows {
		if strings.Contains(r, "planner") && strings.Contains(r, "Thinking") {
			subRow = i
		}
		if strings.Contains(r, "content before delegation") {
			beforeRow = i
		}
		if strings.Contains(r, "main agent continues") {
			afterRow = i
		}
	}
	if subRow < 0 {
		t.Fatalf("subagent line not visible while working:\n%s", joined)
	}
	if afterRow < 0 {
		t.Fatalf("main content not visible:\n%s", joined)
	}
	if beforeRow < 0 || subRow <= beforeRow || subRow >= afterRow {
		t.Fatalf("subagent line must sit in place between content blocks (before=%d sub=%d after=%d):\n%s", beforeRow, subRow, afterRow, joined)
	}
}

func TestSubagentClickOpensChatView(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("main user message")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner", Prompt: "plan the work"},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner", Content: "secret transcript content\nwith a second line"})
	model.Apply(tui.StreamEvent{
		Kind:           tui.EventSubagentDone,
		SubagentID:     "planner",
		SubagentResult: &tui.SubagentResult{Name: "planner", Duration: 900 * time.Millisecond},
	})
	rows := renderFrame(t, l, s)

	if strings.Contains(frameText(rows), "secret transcript") {
		t.Fatalf("transcript leaked into main chat:\n%s", frameText(rows))
	}

	subRow := -1
	for i, r := range rows {
		if strings.Contains(r, "planner") && strings.Contains(r, "✔") {
			subRow = i
			break
		}
	}
	if subRow < 0 {
		t.Fatalf("subagent line not found:\n%s", frameText(rows))
	}
	click(l, 10, subRow)

	if !l.Viewport().InSubagentView() {
		t.Fatal("viewport should switch to the subagent chat view")
	}
	rows = renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "← back") {
		t.Fatalf("back button missing:\n%s", joined)
	}
	if !strings.Contains(joined, "secret transcript content") {
		t.Fatalf("transcript not loaded:\n%s", joined)
	}
	if strings.Contains(joined, "main user message") {
		t.Fatalf("main chat still visible in subagent view:\n%s", joined)
	}

	click(l, 74, 18)
	if l.Viewport().InSubagentView() {
		t.Fatal("back button should return to the main chat")
	}
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "main user message") {
		t.Fatalf("main chat not restored:\n%s", joined)
	}
	if strings.Contains(joined, "secret transcript") {
		t.Fatalf("transcript still visible after going back:\n%s", joined)
	}
}

func TestSubagentViewCounterAndBack(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("hi")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner"},
	})
	for i := 0; i < 30; i++ {
		model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner",
			Content: fmt.Sprintf("transcript line %02d\n\n", i)})
	}
	model.Apply(tui.StreamEvent{
		Kind:           tui.EventSubagentDone,
		SubagentID:     "planner",
		SubagentResult: &tui.SubagentResult{Name: "planner", Duration: 100 * time.Millisecond},
	})
	rows := renderFrame(t, l, s)

	subRow := -1
	for i, r := range rows {
		if strings.Contains(r, "planner") {
			subRow = i
			break
		}
	}
	click(l, 10, subRow)
	if !l.Viewport().InSubagentView() {
		t.Fatal("should be in subagent view")
	}
	rows = renderFrame(t, l, s)
	joined := frameText(rows)

	if !strings.Contains(joined, "← back") {
		t.Fatalf("back button missing while following:\n%s", joined)
	}
	if l.Viewport().IndicatorVisible() {
		t.Fatal("line counter must not show while following")
	}

	l.HandleEvent(tcell.NewEventMouse(10, 5, tcell.WheelUp, tcell.ModNone))
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !l.Viewport().IndicatorVisible() {
		t.Fatalf("line counter missing after scrolling up:\n%s", joined)
	}
	if !strings.Contains(joined, "← back") {
		t.Fatalf("back button must stay visible when scrolled:\n%s", joined)
	}
	x0, x1, y := l.Viewport().IndicatorHitbox()
	bx0, _, by := l.Viewport().BackHitbox()
	if y != by || x1+1 != bx0 {
		t.Fatalf("counter (%d..%d) and back (%d..) not adjacent at row %d", x0, x1, bx0, y)
	}

	click(l, x0+1, y)
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if l.Viewport().IndicatorVisible() {
		t.Fatalf("counter should disappear after resuming follow:\n%s", joined)
	}
	if !strings.Contains(joined, "← back") {
		t.Fatalf("back button missing after resuming follow:\n%s", joined)
	}
}

func TestSubagentEscReturnsToMain(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("hi")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner"},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner", Content: "notes"})
	model.Apply(tui.StreamEvent{
		Kind:           tui.EventSubagentDone,
		SubagentID:     "planner",
		SubagentResult: &tui.SubagentResult{Name: "planner", Duration: 100 * time.Millisecond},
	})
	rows := renderFrame(t, l, s)

	subRow := -1
	for i, r := range rows {
		if strings.Contains(r, "planner") {
			subRow = i
			break
		}
	}
	click(l, 10, subRow)
	if !l.Viewport().InSubagentView() {
		t.Fatal("should be in subagent view")
	}
	l.HandleEvent(key(tcell.KeyEsc))
	if l.Viewport().InSubagentView() {
		t.Fatal("Esc should return to the main chat")
	}
	rows = renderFrame(t, l, s)
	if !strings.Contains(frameText(rows), "hi") {
		t.Fatalf("main chat not restored:\n%s", frameText(rows))
	}
}
