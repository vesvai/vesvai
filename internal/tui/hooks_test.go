package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/core/hook"
	"github.com/vesvai/vesvai/internal/tui/components"
)

func TestOnSubmitTransform(t *testing.T) {
	submitHook = hook.NewHook[string]()
	OnSubmit(func(s string) string { return "> " + s })
	if got := dispatchSubmit("hello"); got != "> hello" {
		t.Errorf("dispatchSubmit = %q, want > hello", got)
	}
	submitHook = hook.NewHook[string]()
}

func TestOnKeyConsume(t *testing.T) {
	keyHook = hook.NewHook[KeyEvent]()
	OnKey(func(ke KeyEvent) KeyEvent {
		ke.Consumed = true
		return ke
	})
	ke := dispatchKey(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl))
	if !ke.Consumed {
		t.Error("OnKey hook should be able to consume events")
	}
	keyHook = hook.NewHook[KeyEvent]()
}

func TestOnRegisterComponent(t *testing.T) {
	componentHook = hook.NewHook[[]components.Component]()
	type fake struct{ components.Component }
	OnRegisterComponent(func(c []components.Component) []components.Component {
		return append(c, fake{})
	})
	got := componentHook.Apply(nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 component after hook, got %d", len(got))
	}
}

func TestResolveGlobal(t *testing.T) {
	cases := []struct {
		key  tcell.Key
		mod  tcell.ModMask
		want keyAction
	}{
		{tcell.KeyCtrlC, tcell.ModCtrl, ActionQuit},
		{tcell.KeyCtrlQ, tcell.ModCtrl, ActionQuit},
		{tcell.KeyCtrlT, tcell.ModCtrl, ActionThemeNext},
		{tcell.KeyEnter, 0, ActionNone},
		{tcell.KeyCtrlC, tcell.ModCtrl | tcell.ModShift, ActionQuit},
	}
	for _, c := range cases {
		ev := tcell.NewEventKey(c.key, 0, c.mod)
		if got := resolveGlobal(ev); got != c.want {
			t.Errorf("resolveGlobal(%v) = %v, want %v", c.key, got, c.want)
		}
	}
}
