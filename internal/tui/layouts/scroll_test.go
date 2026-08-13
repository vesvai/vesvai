package layouts

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func buildScrolledConversation(t *testing.T) (*MainLayout, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	t.Cleanup(s.Fini)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	for i := 0; i < 40; i++ {
		model.Conv.AddUser(fmt.Sprintf("message number %d", i))
	}
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentStart, Subagent: &tui.SubagentInfo{Name: "explorer", Prompt: "inspect"}})
	var long strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&long, "transcript line %d\n\n", i)
	}
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "explorer", Content: long.String()})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentDone, SubagentID: "explorer", SubagentResult: &tui.SubagentResult{Name: "explorer"}})
	return l, s
}

func TestFirstScrollAfterSubagentView(t *testing.T) {
	l, s := buildScrolledConversation(t)
	renderFrame(t, l, s)

	rows := renderFrame(t, l, s)
	subRow := -1
	for i, r := range rows {
		if strings.Contains(r, "explorer") && strings.Contains(r, "✔") {
			subRow = i
			break
		}
	}
	if subRow < 0 {
		t.Fatalf("subagent line not found:\n%s", frameText(rows))
	}
	click(l, 10, subRow)
	if !l.Viewport().InSubagentView() {
		t.Fatal("subagent view did not open")
	}
	renderFrame(t, l, s)
	l.HandleEvent(key(tcell.KeyUp))
	l.HandleEvent(key(tcell.KeyUp))

	l.HandleEvent(key(tcell.KeyEsc))
	if l.Viewport().InSubagentView() {
		t.Fatal("Esc should return to the main chat")
	}

	l.HandleEvent(key(tcell.KeyUp))
	vp := l.Viewport()
	if vp.Following() {
		t.Fatal("first scroll should leave follow mode")
	}
	want := vp.MaxOffset() - 1
	if got := vp.Offset(); got != want {
		t.Fatalf("first scroll offset = %d, want %d (maxOffset %d) — top-jump regression", got, want, vp.MaxOffset())
	}

	l.HandleEvent(key(tcell.KeyUp))
	if got := vp.Offset(); got != want-1 {
		t.Fatalf("second scroll offset = %d, want %d", got, want-1)
	}
}

func TestFirstScrollInSubagentView(t *testing.T) {
	l, s := buildScrolledConversation(t)
	renderFrame(t, l, s)

	rows := renderFrame(t, l, s)
	subRow := -1
	for i, r := range rows {
		if strings.Contains(r, "explorer") && strings.Contains(r, "✔") {
			subRow = i
			break
		}
	}
	click(l, 10, subRow)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyUp))
	vp := l.Viewport()
	if vp.Following() {
		t.Fatal("first scroll in subagent view should leave follow mode")
	}
	want := vp.MaxOffset() - 1
	if got := vp.Offset(); got != want {
		t.Fatalf("subagent first scroll offset = %d, want %d", got, want)
	}
}

func TestFollowScrollStillWorks(t *testing.T) {
	l, s := buildScrolledConversation(t)
	renderFrame(t, l, s)

	vp := l.Viewport()
	if !vp.Following() {
		t.Fatal("viewport should start following")
	}
	l.HandleEvent(key(tcell.KeyTab))
	l.HandleEvent(key(tcell.KeyUp))
	l.HandleEvent(key(tcell.KeyUp))
	if vp.Following() {
		t.Fatal("should have left follow mode after scrolling up")
	}
	for i := 0; i < 5; i++ {
		l.HandleEvent(key(tcell.KeyDown))
	}
	if !vp.Following() {
		t.Fatal("scrolling past the bottom should re-enter follow mode")
	}
}
