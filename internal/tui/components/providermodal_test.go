package components

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func keyEvent(k tcell.Key, r rune) *tcell.EventKey {
	return tcell.NewEventKey(k, r, tcell.ModNone)
}

func TestProviderModalListStep(t *testing.T) {
	m := NewProviderModal([]string{"openrouter", "openai", "anthropic"})
	submitted := false
	m.OnSubmit = func(name, key string) { submitted = true }

	for _, r := range "open" {
		m.HandleEvent(keyEvent(tcell.KeyRune, r))
	}
	got := m.filtered()
	if len(got) != 2 || got[0] != "openrouter" || got[1] != "openai" {
		t.Fatalf("filtered = %v, want [openrouter openai]", got)
	}

	m.HandleEvent(keyEvent(tcell.KeyEnter, 0))
	if m.step != 1 {
		t.Fatalf("step = %d, want 1", m.step)
	}
	if m.selected != "openrouter" {
		t.Fatalf("selected = %q, want openrouter", m.selected)
	}

	for _, r := range "sk-abc" {
		m.HandleEvent(keyEvent(tcell.KeyRune, r))
	}
	m.HandleEvent(keyEvent(tcell.KeyEnter, 0))
	if !submitted {
		t.Fatal("OnSubmit not fired")
	}
}

func TestProviderModalKeyStepValidation(t *testing.T) {
	m := NewProviderModal([]string{"openrouter"})
	m.HandleEvent(keyEvent(tcell.KeyEnter, 0))

	m.HandleEvent(keyEvent(tcell.KeyEnter, 0))
	if m.errMsg == "" {
		t.Fatal("empty key should show an error")
	}

	m.HandleEvent(keyEvent(tcell.KeyEsc, 0))
	if m.step != 0 {
		t.Fatalf("step = %d, want 0 after esc", m.step)
	}
}

func TestProviderModalRendersListThenKey(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	m := NewProviderModal([]string{"openrouter", "openai"})
	m.Layout(CenteredRect(80, 25, 56, m.DesiredHeight()))
	m.MarkDirty()
	m.Render(s, tui.DefaultDark())
	s.Show()

	text := screenText(s)
	if !strings.Contains(text, "Connect provider") {
		t.Fatalf("title missing:\n%s", text)
	}
	if !strings.Contains(text, "openrouter") || !strings.Contains(text, "openai") {
		t.Fatalf("provider list missing:\n%s", text)
	}
	if strings.Contains(text, "api key") {
		t.Fatalf("key form shown before selection:\n%s", text)
	}

	m.HandleEvent(keyEvent(tcell.KeyEnter, 0))
	m.MarkDirty()
	m.Render(s, tui.DefaultDark())
	s.Show()
	text = screenText(s)
	if !strings.Contains(text, "provider: openrouter") || !strings.Contains(text, "api key") {
		t.Fatalf("key step missing:\n%s", text)
	}
}
