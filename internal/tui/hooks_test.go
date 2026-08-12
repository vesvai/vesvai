package tui

import (
	"testing"
	"time"
)

type fakeOwner struct {
	requests int
}

func (f *fakeOwner) RequestRender() { f.requests++ }

func TestUseStateReplaysAcrossRenders(t *testing.T) {
	h := NewHooks(&fakeOwner{})

	first, setFirst := UseState(h, 1)
	if first != 1 {
		t.Fatalf("initial state = %d, want 1", first)
	}
	setFirst(42)
	h.BeginRender()
	second, _ := UseState(h, 1)
	if second != 42 {
		t.Fatalf("state after set = %d, want 42", second)
	}
}

func TestUseStateSetterRequestsRender(t *testing.T) {
	owner := &fakeOwner{}
	h := NewHooks(owner)

	h.BeginRender()
	_, set := UseState(h, 0)
	h.EndRender(0, func() int { return owner.requests })

	if owner.requests != 0 {
		t.Fatalf("setter should not have fired during render, got %d", owner.requests)
	}
	set(1)
	if owner.requests != 1 {
		t.Fatalf("setter should request a render, got %d", owner.requests)
	}
}

func TestUseStateStableAcrossRendersWithMultipleHooks(t *testing.T) {
	h := NewHooks(&fakeOwner{})
	h.BeginRender()
	_, setA := UseState(h, "a")
	_, setB := UseState(h, "b")
	_, setC := UseState(h, 0)
	h.EndRender(0, func() int { return 0 })

	setA("A")
	setB("B")
	setC(9)

	h.BeginRender()
	a2, _ := UseState(h, "a")
	b2, _ := UseState(h, "b")
	c2, _ := UseState(h, 0)
	if a2 != "A" || b2 != "B" || c2 != 9 {
		t.Fatalf("states out of order: %q %q %d", a2, b2, c2)
	}
}

func TestUseRefStablePointer(t *testing.T) {
	h := NewHooks(&fakeOwner{})
	h.BeginRender()
	r1 := UseRef(h, 5)
	h.EndRender(0, func() int { return 0 })
	h.BeginRender()
	r2 := UseRef(h, 99)
	if r1 != r2 {
		t.Fatal("UseRef must return the same pointer across renders")
	}
	if *r2 != 5 {
		t.Fatalf("ref value = %d, want 5", *r2)
	}
}

func TestUseEffectRunsOnceWithoutDepChanges(t *testing.T) {
	h := NewHooks(&fakeOwner{})
	count := 0

	for i := 0; i < 3; i++ {
		h.BeginRender()
		UseEffect(h, []any{1}, func() func() { count++; return nil })
		h.EndRender(0, func() int { return 0 })
	}

	if count != 1 {
		t.Fatalf("effect ran %d times, want 1", count)
	}
}

func TestUseEffectRerunsOnDepChangeAndCleanup(t *testing.T) {
	h := NewHooks(&fakeOwner{})
	runs := 0
	cleanups := 0

	run := func(dep any) {
		h.BeginRender()
		UseEffect(h, []any{dep}, func() func() {
			runs++
			return func() { cleanups++ }
		})
		h.EndRender(0, func() int { return 0 })
	}

	run(1)
	run(2)
	run(2)

	if runs != 2 {
		t.Fatalf("effect ran %d times, want 2", runs)
	}
	if cleanups != 1 {
		t.Fatalf("cleanup ran %d times, want 1", cleanups)
	}
}

func TestUseTickRegistersStableCallbacks(t *testing.T) {
	h := NewHooks(&fakeOwner{})
	first := true

	h.BeginRender()
	UseTick(h, func(elapsed time.Duration) bool { return false })
	h.EndRender(0, func() int { return 0 })

	h.BeginRender()
	dirty := false
	UseTick(h, func(elapsed time.Duration) bool {
		_ = first
		return true
	})
	h.EndRender(0, func() int { return 0 })

	if !h.Tick(time.Second) {
		t.Fatal("tick should report dirty")
	}
	if dirty {
		t.Fatal("flag set outside tick scope")
	}
}

func TestUseReducer(t *testing.T) {
	h := NewHooks(&fakeOwner{})
	h.BeginRender()
	state, dispatch := UseReducer(h, func(s, a int) int { return s + a }, 0)
	h.EndRender(0, func() int { return 0 })

	if state != 0 {
		t.Fatalf("initial = %d", state)
	}
	dispatch(3)
	dispatch(4)

	h.BeginRender()
	state2, _ := UseReducer(h, func(s, a int) int { return s + a }, 0)
	if state2 != 7 {
		t.Fatalf("reduced = %d, want 7", state2)
	}
}
