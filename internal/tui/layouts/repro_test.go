package layouts

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestPaletteStaysOpenWhileStreaming(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	l.Layout(image.Rect(0, 0, 80, 25))
	l.NotifyModelChange()
	l.Render(s, tui.DefaultDark())
	s.Show()

	l.HandleEvent(key(tcell.KeyCtrlP))
	l.Render(s, tui.DefaultDark())
	s.Show()

	for i := 0; i < 10; i++ {
		model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "step "})
		model.Apply(tui.StreamEvent{Kind: tui.EventContent, Content: "tick "})
		l.NotifyModelChange()
		l.Render(s, tui.DefaultDark())
		l.Tick(time.Duration(i) * 50 * time.Millisecond)
		l.Render(s, tui.DefaultDark())
		s.Show()
	}

	rows := renderRaw(s, l)
	joined := frameText(rows)
	if !l.hasModal() {
		t.Fatalf("modal stack cleared during streaming:\n%s", joined)
	}
	if !strings.Contains(joined, "Actions") {
		t.Fatalf("palette vanished during streaming:\n%s", joined)
	}
}
