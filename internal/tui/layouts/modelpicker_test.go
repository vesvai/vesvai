package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestChangeModelOpensPicker(t *testing.T) {
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
	l.HandleEvent(key(tcell.KeyEnter))

	if !l.hasModal() {
		t.Fatal("model picker should be open")
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	for _, want := range []string{"Switch model", "deepseek-v4", "DeepSeek", "Opus 4.5", "Anthropic", "gpt-4o-mini", "OpenAI"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("model picker missing %q:\n%s", want, joined)
		}
	}
}

func TestModelPickerSearchFilters(t *testing.T) {
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
	l.HandleEvent(key(tcell.KeyEnter))
	for _, r := range "anthropic" {
		l.HandleEvent(runeKey(r))
	}
	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	if !strings.Contains(joined, "Opus 4.5") || !strings.Contains(joined, "claude-3.5-haiku") {
		t.Fatalf("matching models missing:\n%s", joined)
	}
	for _, gone := range []string{"deepseek-v4", "gemini-2.5-flash", "gpt-4o-mini"} {
		if strings.Contains(joined, gone) {
			t.Fatalf("unfiltered model %q still visible:\n%s", gone, joined)
		}
	}
}

func TestModelPickerSelect(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	got := ""
	l.OnModelSelect = func(m tui.ModelInfo) { got = m.Name }
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "model" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))

	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyEnter))

	if got != "Opus 4.5" {
		t.Fatalf("OnModelSelect = %q, want Opus 4.5", got)
	}
	if l.hasModal() {
		t.Fatal("picker should close after selecting")
	}
}

func TestModelPickerEscapeCloses(t *testing.T) {
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
	l.HandleEvent(key(tcell.KeyEnter))
	if !l.hasModal() {
		t.Fatal("picker should be open")
	}
	l.HandleEvent(key(tcell.KeyEsc))
	if l.hasModal() {
		t.Fatal("picker should close on Escape")
	}
}

func TestModelPickerMetaInheritsSelectionBackground(t *testing.T) {
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
	l.HandleEvent(key(tcell.KeyEnter))
	l.Render(s, tui.DefaultDark())
	s.Show()

	_, bg, _, ok := findRuneInBox(s, 4, 22, 'D')
	if !ok {
		t.Fatalf("meta text not found in picker")
	}
	sel := tui.DefaultDark().Selection
	if bg != sel {
		t.Fatalf("meta bg = %v, want selection %v", bg, sel)
	}
	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	_, bgPad, _ := cells[11*w+60].Style.Decompose()
	if bgPad != sel {
		t.Fatalf("row padding bg = %v, want selection %v", bgPad, sel)
	}

	l.HandleEvent(key(tcell.KeyDown))
	l.Render(s, tui.DefaultDark())
	s.Show()
	_, bg, _, ok = findRuneInBox(s, 4, 22, 'D')
	if !ok {
		t.Fatalf("meta text not found after moving cursor")
	}
	if bg == sel {
		t.Fatal("unselected meta must not use the selection background")
	}
}

func TestModelSelectUpdatesStatusbar(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	wirePalette(l)
	l.OnModelSelect = func(m tui.ModelInfo) {
		model.Model = m.Name
		model.Provider = m.Provider
		model.Effort = m.Effort
		model.ContextWindow = m.ContextWindow
		model.SetStatusMsg("model: " + m.Name)
	}
	renderFrame(t, l, s)

	l.HandleEvent(key(tcell.KeyCtrlP))
	for _, r := range "model" {
		l.HandleEvent(runeKey(r))
	}
	l.HandleEvent(key(tcell.KeyEnter))
	l.HandleEvent(key(tcell.KeyDown))
	l.HandleEvent(key(tcell.KeyEnter))
	l.NotifyModelChange()
	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "Opus 4.5") || !strings.Contains(joined, "model: Opus 4.5") {
		t.Fatalf("statusbar not updated:\n%s", joined)
	}
}
