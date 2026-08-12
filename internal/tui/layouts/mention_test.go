package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestMentionEndToEnd(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	for _, r := range "@pl" {
		l.HandleEvent(runeKey(r))
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "planner") {
		t.Fatalf("mention picker not shown above the textarea:\n%s", joined)
	}
	taY := -1
	pickY := -1
	for i, r := range rows {
		if strings.Contains(r, "❯ @pl") {
			taY = i
		}
		if strings.Contains(r, "planner") && !strings.Contains(r, "❯") {
			pickY = i
		}
	}
	if taY < 0 || pickY < 0 || pickY >= taY {
		t.Fatalf("picker not above the textarea (picker=%d textarea=%d):\n%s", pickY, taY, joined)
	}

	l.HandleEvent(key(tcell.KeyEnter))
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if !strings.Contains(joined, "@planner") {
		t.Fatalf("mention block not inserted:\n%s", joined)
	}

	l.HandleEvent(key(tcell.KeyBackspace))
	rows = renderFrame(t, l, s)
	joined = frameText(rows)
	if strings.Contains(joined, "planner") {
		t.Fatalf("mention block not deleted as a whole:\n%s", joined)
	}
}

func TestMentionInFullLayoutRendersGold(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	for _, r := range "@pl" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))
	rows := renderFrame(t, l, s)

	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	pal := tui.DefaultDark()
	gold := false
	for i := 0; i < len(cells); i++ {
		c := cells[i]
		if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == '@' {
			fg, _, _ := c.Style.Decompose()
			if fg == pal.Mention {
				gold = true
				break
			}
		}
	}
	if !gold {
		t.Fatalf("mention not gold in the layout frame:\n%s", strings.Join(rows, "\n"))
	}
	_ = w
}
