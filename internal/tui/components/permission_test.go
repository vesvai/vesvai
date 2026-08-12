package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/permission"
	"github.com/vesvai/vesvai/internal/tui"
)

func TestPermissionModalKeys(t *testing.T) {
	cases := []struct {
		key  *tcell.EventKey
		want permission.Decision
	}{
		{key(tcell.KeyRune, 'a'), permission.DecisionAllow},
		{key(tcell.KeyRune, 'A'), permission.DecisionAllowAlways},
		{key(tcell.KeyRune, 'd'), permission.DecisionDeny},
		{key(tcell.KeyRune, 'y'), permission.DecisionAllow},
		{key(tcell.KeyRune, 'n'), permission.DecisionDeny},
		{key(tcell.KeyEsc, 0), permission.DecisionDeny},
	}
	for _, c := range cases {
		got := permission.DecisionDeny
		m := NewPermissionModal("bash", map[string]any{"command": "ls"}, "needs approval")
		m.OnDecision = func(d permission.Decision) { got = d }
		if !m.HandleEvent(c.key) {
			t.Fatalf("key %v not handled", c.key)
		}
		if got != c.want {
			t.Fatalf("key %v: decision = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestPermissionModalDraw(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	m := NewPermissionModal("bash", map[string]any{"command": "ls -la"}, "tool=bash action=prompt")
	m.Layout(CenteredRect(80, 25, 56, m.DesiredHeight()))
	m.MarkDirty()
	m.Render(s, tui.DefaultDark())
	s.Show()

	text := screenText(s)
	for _, want := range []string{"Permission required", "bash", "ls -la", "tool=bash action=prompt", "allow", "deny"} {
		if !containsText(text, want) {
			t.Fatalf("modal missing %q:\n%s", want, text)
		}
	}
}

func TestConfirmModalYesNo(t *testing.T) {
	confirm := 0
	cancel := 0
	m := NewConfirmModal("Delete session", "Delete 'x'?")
	m.OnConfirm = func() { confirm++ }
	m.OnCancel = func() { cancel++ }

	if !m.HandleEvent(key(tcell.KeyRune, 'y')) || confirm != 1 {
		t.Fatalf("y: confirm = %d, want 1", confirm)
	}
	if !m.HandleEvent(key(tcell.KeyRune, 'n')) || cancel != 1 {
		t.Fatalf("n: cancel = %d, want 1", cancel)
	}
	if !m.HandleEvent(key(tcell.KeyEsc, 0)) || cancel != 2 {
		t.Fatalf("esc: cancel = %d, want 2", cancel)
	}
}

func key(k tcell.Key, r rune) *tcell.EventKey {
	return tcell.NewEventKey(k, r, tcell.ModNone)
}

func screenText(s tcell.SimulationScreen) string {
	cells, w, _ := s.GetContents()
	out := make([]byte, 0, len(cells))
	for i, c := range cells {
		if len(c.Runes) > 0 {
			out = append(out, string(c.Runes[0])...)
		}
		if i%w == w-1 {
			out = append(out, '\n')
		}
	}
	return string(out)
}

func containsText(text, sub string) bool {
	return len(text) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(text); i++ {
			if text[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
