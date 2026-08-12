package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func click(l *MainLayout, x, y int) {
	l.HandleEvent(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone))
}

func TestUserMessageClickOpensMenuAtPosition(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("click me")
	renderFrame(t, l, s)

	click(l, 6, 0)

	if l.menu == nil {
		t.Fatal("context menu did not open")
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	for _, want := range []string{"Fork", "Revert", "Copy"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("menu missing %q:\n%s", want, joined)
		}
	}
	b := l.menu.Bounds()
	if b.Min.X != 6 || b.Min.Y != 0 {
		t.Fatalf("menu positioned at %v, want (6,0)", b.Min)
	}
}

func TestMenuKeyboardSelectRunsAction(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("click me")
	got := ""
	l.OnUserAction = func(id string, _ *tui.Message) { got = id }
	renderFrame(t, l, s)

	click(l, 6, 0)
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyEnter))

	if got != "copy" {
		t.Fatalf("OnUserAction = %q, want copy", got)
	}
	if l.menu != nil {
		t.Fatal("menu should close after running an action")
	}
}

func TestMenuMouseSelectRunsAction(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("click me")
	got := ""
	l.OnUserAction = func(id string, _ *tui.Message) { got = id }
	renderFrame(t, l, s)

	click(l, 6, 0)
	click(l, 8, 3)

	if got != "copy" {
		t.Fatalf("OnUserAction = %q, want copy", got)
	}
}

func TestMenuEscapeCloses(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("click me")
	renderFrame(t, l, s)

	click(l, 6, 0)
	if l.menu == nil {
		t.Fatal("menu should be open")
	}
	l.HandleEvent(key(tcell.KeyEsc))
	if l.menu != nil {
		t.Fatal("menu should close on Escape")
	}
}

func TestMenuOutsideClickCloses(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("click me")
	renderFrame(t, l, s)

	click(l, 6, 0)
	click(l, 60, 20)
	if l.menu != nil {
		t.Fatal("menu should close on an outside click")
	}
}

func TestCopyActionFillsClipboard(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("precious content")
	l.OnUserAction = func(id string, m *tui.Message) {
		if id == "copy" {
			l.Textarea().SetClipboard(m.Content)
		}
	}
	renderFrame(t, l, s)

	click(l, 6, 0)
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyEnter))

	l.HandleEvent(tcell.NewEventKey(tcell.KeyInsert, 0, tcell.ModShift))
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "precious content") {
		t.Fatalf("clipboard content not pasted:\n%s", joined)
	}
}

func TestForkActionTruncatesConversation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	m1 := model.Conv.AddUser("first")
	model.Conv.AddUser("second")
	got := ""
	l.OnUserAction = func(id string, m *tui.Message) {
		if id == "fork" {
			model.Conv.TruncateAfter(m)
		}
		got = id
	}
	renderFrame(t, l, s)

	click(l, 6, 0)
	l.HandleEvent(key(tcell.KeyEnter))

	if got != "fork" {
		t.Fatalf("OnUserAction = %q, want fork", got)
	}
	if len(model.Conv.Messages) != 1 || model.Conv.Messages[0] != m1 {
		t.Fatalf("conversation not truncated: %d messages", len(model.Conv.Messages))
	}
	rows := renderFrame(t, l, s)
	if strings.Contains(frameText(rows), "second") {
		t.Fatalf("truncated message still visible:\n%s", frameText(rows))
	}
}

func TestAssistantClickDoesNotOpenMenu(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("hello")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "assistant reply text"})
	model.Apply(tui.StreamEvent{Kind: tui.EventDone})
	renderFrame(t, l, s)

	click(l, 10, 5)
	if l.menu != nil {
		t.Fatal("menu must not open on assistant content")
	}
}
