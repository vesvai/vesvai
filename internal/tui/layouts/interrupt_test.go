package layouts

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestCanInterruptWithMenuAndModal(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("click me")
	renderFrame(t, l, s)

	if !l.CanInterrupt() {
		t.Fatal("interrupt should be allowed on a clean screen")
	}

	l.HandleEvent(key(tcell.KeyCtrlP))
	if l.CanInterrupt() {
		t.Fatal("interrupt must not be allowed while a modal is open")
	}
	l.HandleEvent(key(tcell.KeyEsc))

	click(l, 6, 0)
	if l.CanInterrupt() {
		t.Fatal("interrupt must not be allowed while the menu is open")
	}
	l.HandleEvent(key(tcell.KeyEsc))
	if !l.CanInterrupt() {
		t.Fatal("interrupt should be allowed again after closing the menu")
	}
}
