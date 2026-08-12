package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func key(k tcell.Key) *tcell.EventKey {
	return tcell.NewEventKey(k, 0, tcell.ModNone)
}

func runeKey(r rune) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
}

func TestPaletteOpensWithCtrlP(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	for _, want := range []string{"Actions", "New session", "Switch session", "Share session", "Connect provider", "Change model"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("palette missing %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(joined, "Session") || !strings.Contains(joined, "Provider") {
		t.Fatalf("category headers missing:\n%s", joined)
	}
}

func TestPaletteSearchFiltersActions(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "model" {
		l.HandleEvent(runeKey(r))
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	if !strings.Contains(joined, "Change model") {
		t.Fatalf("filtered action missing:\n%s", joined)
	}
	for _, gone := range []string{"New session", "Share session", "Connect provider"} {
		if strings.Contains(joined, gone) {
			t.Fatalf("unfiltered action %q still visible:\n%s", gone, joined)
		}
	}
}

func TestPaletteRunsSelectedActionAndCloses(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	got := ""
	l.OnAction = func(id string) { got = id; l.CloseAllModals() }
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "share" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))

	if got != "share-session" {
		t.Fatalf("OnAction = %q, want share-session", got)
	}
	rows := renderFrame(t, l, s)
	if strings.Contains(frameText(rows), "Actions") {
		t.Fatalf("palette still open after running an action:\n%s", frameText(rows))
	}
}

func TestPaletteEscapeCloses(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	if !l.hasModal() {
		t.Fatal("palette should be open")
	}
	l.HandleEvent(key(tcell.KeyEsc))
	if l.hasModal() {
		t.Fatal("palette should close on Escape")
	}
}

func TestPaletteCtrlPTogglesClosed(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	l.HandleEvent(key(tcell.KeyCtrlP))
	if l.hasModal() {
		t.Fatal("second Ctrl+P must close the palette")
	}
}

func TestSwitchSessionOpensPickerAndSelects(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	got := ""
	l.OnSessionSelect = func(id string) { got = id }
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "switch" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))

	if !l.hasModal() {
		t.Fatal("session picker should be open")
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "Switch session") || !strings.Contains(joined, "Refactor the TUI layout") {
		t.Fatalf("session picker content missing:\n%s", joined)
	}

	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyEnter))
	if got != "s3" {
		t.Fatalf("OnSessionSelect = %q, want s3", got)
	}
	if l.hasModal() {
		t.Fatal("picker should close after selecting")
	}
}

func TestPaletteDownArrowNavigates(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyEnter))

	if !l.hasModal() {
		t.Fatal("switch-session should open the picker")
	}
	rows := renderFrame(t, l, s)
	if !strings.Contains(frameText(rows), "Switch session") {
		t.Fatalf("session picker not shown:\n%s", frameText(rows))
	}
}

func TestStatusbarShowsTransientActionMessage(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	model.SetStatusMsg("provider connected")
	l.NotifyModelChange()
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "provider connected") {
		t.Fatalf("transient message missing:\n%s", joined)
	}
}

func TestNewSessionActionResetsConversation(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	l.OnAction = func(id string) {
		if id == "new-session" || id == "clear-conversation" {
			model.Conv.Reset()
			model.SessionName = ""
			model.SetStatusMsg("new session")
		}
	}
	model.Conv.AddUser("old content")
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	l.HandleEvent(key(tcell.KeyEnter))

	if len(model.Conv.Messages) != 0 {
		t.Fatalf("conversation not reset, %d messages remain", len(model.Conv.Messages))
	}
	if model.SessionName != "" {
		t.Fatalf("session name = %q, want empty", model.SessionName)
	}
}

func TestPaletteHintInheritsSelectionBackground(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()
	s.SetSize(80, 25)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	l.Render(s, tui.DefaultDark())
	s.Show()

	cx, bg, row, ok := findRuneInBox(s, 5, 18, 'C')
	if !ok {
		t.Fatalf("hint 'Ctrl+N' not found in palette box")
	}
	sel := tui.DefaultDark().Selection
	if bg != sel {
		t.Fatalf("hint bg = %v, want selection %v", bg, sel)
	}
	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	_, bgLeft, _ := cells[row*w+cx-1].Style.Decompose()
	if bgLeft != sel {
		t.Fatalf("cell left of hint bg = %v, want selection %v", bgLeft, sel)
	}

	l.HandleEvent(key(tcell.KeyDown))
	l.Render(s, tui.DefaultDark())
	s.Show()
	_, bg, _, ok = findRuneInBox(s, 5, 18, 'C')
	if !ok {
		t.Fatalf("hint 'Ctrl+N' not found after moving cursor")
	}
	if bg == sel {
		t.Fatal("unselected hint must not use the selection background")
	}
}

func findRuneInBox(s tcell.SimulationScreen, y0, y1 int, r rune) (int, tcell.Color, int, bool) {
	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	for y := y0; y <= y1; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if c.Runes != nil && len(c.Runes) > 0 && c.Runes[0] == r {
				_, bg, _ := c.Style.Decompose()
				return x, bg, y, true
			}
		}
	}
	return -1, 0, 0, false
}

func TestSessionPickerOmitsModelName(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "switch" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	if !strings.Contains(joined, "Refactor the TUI layout") || !strings.Contains(joined, "2m ago") {
		t.Fatalf("session entries missing:\n%s", joined)
	}
	for _, gone := range []string{"deepseek-v4", "claude-3.5-haiku", "gemini-2.5-flash"} {
		if strings.Contains(joined, gone) {
			t.Fatalf("model name %q still shown in picker:\n%s", gone, joined)
		}
	}
}

func TestSessionMetaInheritsSelectionBackground(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "switch" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))
	l.Render(s, tui.DefaultDark())
	s.Show()

	cx, bg, _, ok := findRuneInBox(s, 4, 22, '2')
	if !ok {
		t.Fatalf("meta '2m ago' not found in picker")
	}
	sel := tui.DefaultDark().Selection
	if bg != sel {
		t.Fatalf("meta bg = %v, want selection %v", bg, sel)
	}
	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	_, bgAfter, _ := cells[8*w+cx+1].Style.Decompose()
	if bgAfter != sel {
		t.Fatalf("cell after meta bg = %v, want selection %v", bgAfter, sel)
	}

	l.HandleEvent(key(tcell.KeyDown))
	l.Render(s, tui.DefaultDark())
	s.Show()
	_, bg, _, ok = findRuneInBox(s, 4, 22, '2')
	if !ok {
		t.Fatalf("meta '2m ago' not found after moving cursor")
	}
	if bg == sel {
		t.Fatal("unselected meta must not use the selection background")
	}
}
