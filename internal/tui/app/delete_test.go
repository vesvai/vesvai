package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/layouts"
)

func newDeleteApp(t *testing.T) (*App, *stubBackend) {
	t.Helper()
	b := &stubBackend{}
	a := NewWithDriver(&stubDriver{stubBackend: b})
	a.model = tui.NewModel("demo")
	a.layout = layouts.NewMainLayout(a.model, tui.DefaultDark())
	a.backend = b
	a.sessionID = "sess-1"
	a.model.SessionName = "my session"
	a.model.Conv.AddUser("hello")
	return a, b
}

func keyRune(r rune) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
}

func TestDeleteCurrentSessionConfirm(t *testing.T) {
	a, b := newDeleteApp(t)

	a.handleAction("delete-session")

	top := a.layout.TopModal()
	if top == nil {
		t.Fatal("confirm modal should be open")
	}

	top.HandleEvent(keyRune('y'))

	if len(b.deleted) != 1 || b.deleted[0] != "sess-1" {
		t.Fatalf("deleted = %v, want [sess-1]", b.deleted)
	}
	if a.sessionID != "" {
		t.Fatalf("sessionID = %q, want empty after delete", a.sessionID)
	}
	if len(a.model.Conv.Messages) != 0 {
		t.Fatalf("conversation should be reset after delete, got %d messages", len(a.model.Conv.Messages))
	}
	if a.layout.TopModal() != nil {
		t.Fatal("modals should be closed after confirming")
	}
}

func TestDeleteCurrentSessionCancel(t *testing.T) {
	a, b := newDeleteApp(t)

	a.handleAction("delete-session")
	top := a.layout.TopModal()
	if top == nil {
		t.Fatal("confirm modal should be open")
	}
	top.HandleEvent(keyRune('n'))

	if len(b.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", b.deleted)
	}
	if a.sessionID != "sess-1" {
		t.Fatalf("sessionID = %q, want sess-1", a.sessionID)
	}
	if a.layout.TopModal() != nil {
		t.Fatal("confirm modal should close on cancel")
	}
}

func TestDeleteCurrentSessionNone(t *testing.T) {
	a, b := newDeleteApp(t)
	a.sessionID = ""

	a.handleAction("delete-session")

	if len(b.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", b.deleted)
	}
	if a.layout.TopModal() != nil {
		t.Fatal("no confirm modal expected without a session")
	}
	if a.model.StatusMsg != "no session to delete" {
		t.Fatalf("status = %q, want 'no session to delete'", a.model.StatusMsg)
	}
}
