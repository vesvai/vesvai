package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestViewportShowsLogoWhenEmpty(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	if !strings.Contains(joined, "██╗   ██╗") {
		t.Fatalf("logo missing:\n%s", joined)
	}
	if !strings.Contains(joined, "AI-powered development assistant") {
		t.Fatalf("tagline missing:\n%s", joined)
	}
	first := -1
	tagRow := -1
	for i, r := range rows {
		if strings.Contains(r, "██") && first < 0 {
			first = i
		}
		if strings.Contains(r, "AI-powered") {
			tagRow = i
		}
	}
	if first < 0 || tagRow < 0 {
		t.Fatal("logo rows not found")
	}
	above := first
	below := 17 - tagRow
	if diff := above - below; diff > 2 || diff < -2 {
		t.Fatalf("logo off-center: above=%d below=%d", above, below)
	}
}

func TestLogoDisappearsAfterFirstMessage(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	model.Conv.AddUser("hello agent")
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	if strings.Contains(joined, "██╗   ██╗") {
		t.Fatalf("logo still shown after first message:\n%s", joined)
	}
	if !strings.Contains(joined, "hello agent") {
		t.Fatalf("first message not shown:\n%s", joined)
	}
}

func TestLogoReappearsAfterNewSession(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("old")
	renderFrame(t, l, s)

	model.Conv.Reset()
	l.NotifyModelChange()
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "██╗   ██╗") {
		t.Fatalf("logo did not return after reset:\n%s", joined)
	}
}

func TestEmptyStateInputsDoNotPanic(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	click(l, 40, 8)
	click(l, 5, 16)
	l.HandleEvent(key(tcell.KeyTab))
	l.HandleEvent(key(tcell.KeyTab))
	l.HandleEvent(key(tcell.KeyTab))
	l.HandleEvent(runeKey(']'))
	l.HandleEvent(runeKey('['))
	l.HandleEvent(key(tcell.KeyEnter))
	l.HandleEvent(runeKey(' '))
	l.HandleEvent(tcell.NewEventMouse(40, 8, tcell.WheelUp, tcell.ModNone))

	rows := renderFrame(t, l, s)
	if !strings.Contains(frameText(rows), "██╗   ██╗") {
		t.Fatalf("logo lost:\n%s", frameText(rows))
	}
	if l.menu != nil {
		t.Fatal("empty-state click must not open a menu")
	}
}
