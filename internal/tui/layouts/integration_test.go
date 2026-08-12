package layouts

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func renderFrame(t *testing.T, l *MainLayout, s tcell.SimulationScreen) []string {
	t.Helper()
	s.SetSize(80, 25)
	l.Layout(image.Rect(0, 0, 80, 25))
	l.NotifyModelChange()
	l.Render(s, tui.DefaultDark())
	s.Show()
	cells, _, _ := s.GetContents()
	rows := make([]string, 25)
	for y := 0; y < 25; y++ {
		var sb strings.Builder
		for x := 0; x < 80; x++ {
			c := cells[y*80+x]
			if c.Runes != nil {
				sb.WriteRune(c.Runes[0])
			} else {
				sb.WriteByte(' ')
			}
		}
		rows[y] = sb.String()
	}
	return rows
}

func frameText(rows []string) string { return strings.Join(rows, "\n") }

func toggleItem(model *tui.Model, id string) {
	model.Conv.TogglePartByID(id)
}

func TestLifecycleEndToEnd(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())

	model.Conv.AddUser("build a tool")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "Breaking this into steps."})
	rows := renderFrame(t, l, s)

	joined := frameText(rows)
	if !strings.Contains(joined, "Thinking") {
		t.Fatalf("thinking header missing:\n%s", joined)
	}

	model.Apply(tui.StreamEvent{
		Kind:     tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{ToolName: "bash", Args: map[string]any{"command": "ls"}},
	})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "bash") || !strings.Contains(joined, "running") {
		t.Fatalf("running tool card missing:\n%s", joined)
	}
	if !strings.Contains(joined, "⚙ bash") {
		t.Fatalf("tool line missing:\n%s", joined)
	}

	model.Apply(tui.StreamEvent{
		Kind: tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{
			ToolName: "bash",
			Result:   "file.txt\nother.txt",
			Duration: 900 * time.Millisecond,
		},
	})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "✔") || !strings.Contains(joined, "900ms") {
		t.Fatalf("success state missing:\n%s", joined)
	}
	if strings.Contains(joined, "file.txt") {
		t.Fatal("collapsed tool result should be hidden")
	}

	msg := model.Conv.Current()
	toggleItem(model, tui.ToolPartID(msg, 0))
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "file.txt") {
		t.Fatalf("expanded result missing:\n%s", joined)
	}

	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "## Plan\n```go\npackage main\n```\n"})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "┌─go") || !strings.Contains(joined, "package main") {
		t.Fatalf("code block missing:\n%s", joined)
	}
	if !strings.Contains(joined, "▍ Plan") {
		t.Fatalf("heading text missing:\n%s", joined)
	}

	model.Apply(tui.StreamEvent{Kind: tui.EventDone, Usage: &tui.Usage{PromptTokens: 10, CompletionTokens: 5}})
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "✔ idle") {
		t.Fatalf("statusbar idle missing:\n%s", joined)
	}
	if !strings.Contains(joined, "15 · step 1") {
		t.Fatalf("token usage missing:\n%s", joined)
	}
}

func TestViewportScrollingAndFollow(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())

	model.Conv.AddUser("hello")
	for i := 0; i < 10; i++ {
		model.Apply(tui.StreamEvent{Kind: tui.EventStart})
		model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "message block " + strings.Repeat("x", 30) + "\n"})
		model.Apply(tui.StreamEvent{Kind: tui.EventDone})
		model.Conv.AddUser("turn " + string(rune('a'+i)))
	}

	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "turn ") {
		t.Fatalf("latest content not visible while following:\n%s", joined)
	}
	if strings.Contains(joined, "hello") {
		t.Fatalf("old content still visible after scroll\n%s", joined)
	}
}

func TestScrollUpIndicatorClickResumesFollow(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())

	for i := 0; i < 12; i++ {
		model.Conv.AddUser("user line " + string(rune('a'+i)))
		model.Apply(tui.StreamEvent{Kind: tui.EventStart})
		model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "assistant reply " + strings.Repeat("y", 40) + "\n"})
		model.Apply(tui.StreamEvent{Kind: tui.EventDone})
	}
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyTab))
	l.HandleEvent(key(tcell.KeyPgUp))
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "↓") {
		t.Fatalf("scroll indicator missing:\n%s", joined)
	}

	v := l.Viewport()
	if !v.IndicatorVisible() {
		t.Fatal("indicator should be visible after scrolling up")
	}
	x0, _, y := v.IndicatorHitbox()
	l.HandleEvent(tcell.NewEventMouse(x0+1, y, tcell.Button1, tcell.ModNone))
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "assistant reply ") {
		t.Fatalf("bottom not visible after clicking indicator:\n%s", joined)
	}
	if strings.Contains(joined, "↓") {
		t.Fatalf("indicator should disappear once following:\n%s", joined)
	}
}

func renderRaw(s tcell.SimulationScreen, l *MainLayout) []string {
	s.Show()
	cells, _, _ := s.GetContents()
	w, h := s.Size()
	rows := make([]string, h)
	for y := 0; y < h; y++ {
		var sb strings.Builder
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if c.Runes != nil {
				sb.WriteRune(c.Runes[0])
			} else {
				sb.WriteByte(' ')
			}
		}
		rows[y] = sb.String()
	}
	return rows
}

func TestFocusCycleDoesNotWipeScreen(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("persistent message")
	renderFrame(t, l, s)

	for i := 0; i < 3; i++ {
		l.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
		l.Render(s, tui.DefaultDark())
	}
	rows := renderRaw(s, l)
	joined := frameText(rows)
	if !strings.Contains(joined, "persistent message") {
		t.Fatalf("content wiped by focus cycle:\n%s", joined)
	}
	if !strings.Contains(joined, "✔ idle") || !strings.Contains(joined, "⇥ focus") {
		t.Fatalf("statusbar wiped by focus cycle:\n%s", joined)
	}
}

func TestTypingRendersWithoutFocusChange(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	for _, r := range "hello" {
		l.HandleEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
		l.Render(s, tui.DefaultDark())
	}
	rows := renderRaw(s, l)
	joined := frameText(rows)
	if !strings.Contains(joined, "❯ hello") {
		t.Fatalf("keystrokes not rendered:\n%s", joined)
	}
}

func TestResizeKeepsEverythingVisible(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("resize survivor")
	renderFrame(t, l, s)

	for _, size := range [][2]int{{60, 20}, {100, 32}, {40, 12}} {
		s.SetSize(size[0], size[1])
		l.Layout(image.Rect(0, 0, size[0], size[1]))
		l.Render(s, tui.DefaultDark())
	}
	rows := renderRaw(s, l)
	joined := frameText(rows)
	if !strings.Contains(joined, "resize survivor") {
		t.Fatalf("content lost after resize:\n%s", joined)
	}
	if !strings.Contains(joined, "❯") || !strings.Contains(joined, "✔ idle") || !strings.Contains(joined, "⇥ focus") {
		t.Fatalf("chrome lost after resize:\n%s", joined)
	}
	for y := 10; y < 12; y++ {
		if strings.TrimSpace(rows[y]) == "" {
			t.Fatalf("row %d left blank after resize:\n%s", y, joined)
		}
	}
}
