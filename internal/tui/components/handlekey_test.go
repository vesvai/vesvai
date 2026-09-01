package components

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestHandleKeyShiftArrows(t *testing.T) {
	in := NewInput()
	in.Focus()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	if !in.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift)) {
		t.Error("Shift+Right not consumed")
	}
	if !in.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift)) {
		t.Error("Shift+Right not consumed")
	}
	if in.SelectedText() != "he" {
		t.Errorf("SelectedText = %q, want he", in.SelectedText())
	}
}

func TestHandleKeyCtrlShiftWordSelect(t *testing.T) {
	in := NewInput()
	in.Focus()
	for _, r := range "one two three" {
		in.InsertRune(r)
	}
	in.Home()
	in.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl|tcell.ModShift))
	if in.SelectedText() != "one " {
		t.Errorf("Ctrl+Shift+Right SelectedText = %q, want one ", in.SelectedText())
	}
}

func TestHandleKeyCtrlShiftKDeleteLine(t *testing.T) {
	in := NewInput()
	in.Focus()
	for _, r := range "aaa" {
		in.InsertRune(r)
	}
	in.Newline()
	for _, r := range "bbb" {
		in.InsertRune(r)
	}
	in.HandleKey(tcell.NewEventKey(tcell.KeyCtrlK, 0, tcell.ModCtrl|tcell.ModShift))
	if in.Value() != "aaa" {
		t.Errorf("Ctrl+Shift+K Value = %q, want aaa", in.Value())
	}
}

func TestHandleKeyCtrlASelectAll(t *testing.T) {
	in := NewInput()
	in.Focus()
	for _, r := range "abc" {
		in.InsertRune(r)
	}
	in.HandleKey(tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModCtrl))
	if in.SelectedText() != "abc" {
		t.Errorf("Ctrl+A SelectedText = %q, want abc", in.SelectedText())
	}
	in.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
	if in.Value() != "x" {
		t.Errorf("after typing over selection Value = %q, want x", in.Value())
	}
}

func TestHandleKeyPlainMoveClearsSelection(t *testing.T) {
	in := NewInput()
	in.Focus()
	for _, r := range "hello" {
		in.InsertRune(r)
	}
	in.Home()
	in.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift))
	if !in.HasSelection() {
		t.Fatal("expected selection")
	}
	in.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if in.HasSelection() {
		t.Error("plain movement should clear selection")
	}
}

func TestHandleKeyIgnoresWhenBlurred(t *testing.T) {
	in := NewInput()
	if in.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)) {
		t.Error("blurred input should not consume keys")
	}
	if in.Value() != "" {
		t.Error("blurred input should not accept input")
	}
}

func TestCursorSolidWhileActive(t *testing.T) {
	in := NewInput()
	now := time.Now()
	in.lastActivity = now
	in.blinkOn = false
	if !in.CursorVisible(now) {
		t.Error("cursor should be solid immediately after activity")
	}
	if !in.CursorVisible(now.Add(CursorIdleDelay - time.Millisecond)) {
		t.Error("cursor should stay solid just under the idle threshold")
	}
	if in.CursorVisible(now.Add(CursorIdleDelay + time.Millisecond)) {
		t.Error("cursor should resume blinking once idle")
	}
	in.blinkOn = true
	if !in.CursorVisible(now.Add(2 * time.Hour)) {
		t.Error("idle cursor should be visible when blinkOn is true")
	}
}

func TestCursorBlinkOnFreshInput(t *testing.T) {
	in := NewInput()
	in.blinkOn = false
	if in.CursorVisible(time.Now()) {
		t.Error("fresh input should follow blink when idle")
	}
	in.blinkOn = true
	if !in.CursorVisible(time.Now()) {
		t.Error("fresh input cursor visible when blinkOn")
	}
}

func TestHandleKeyMarksActivity(t *testing.T) {
	in := NewInput()
	in.Focus()
	in.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if time.Since(in.lastActivity) > time.Second {
		t.Error("HandleKey should mark activity so the cursor stays solid")
	}
	in.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl|tcell.ModShift))
	if time.Since(in.lastActivity) > time.Second {
		t.Error("ctrl+shift shortcut should mark activity")
	}
}
