package tui

import (
	"reflect"
	"time"
)

type renderRequester interface {
	RequestRender()
}

type effectEntry struct {
	deps     []any
	prevDeps []any
	effect   func() func()
	cleanup  func()
	ran      bool
}

type tickEntry struct {
	fn func(elapsed time.Duration) bool
}

type Hooks struct {
	owner renderRequester

	states []any
	idx    int

	effects []*effectEntry
	effIdx  int

	ticks   []tickEntry
	tickIdx int
}

func NewHooks(owner renderRequester) *Hooks {
	return &Hooks{owner: owner}
}

func (h *Hooks) BeginRender() {
	h.idx = 0
	h.effIdx = 0
	h.tickIdx = 0
}

func UseState[T any](h *Hooks, initial T) (T, func(T)) {
	slot := h.slot()
	if slot == nil {
		h.states = append(h.states, initial)
		slot = &h.states[len(h.states)-1]
	}
	idx := h.idx
	h.idx++
	v, _ := (*slot).(T)
	return v, func(next T) {
		h.states[idx] = next
		if h.owner != nil {
			h.owner.RequestRender()
		}
	}
}

func UseRef[T any](h *Hooks, initial T) *T {
	slot := h.slot()
	if slot == nil {
		ref := &initial
		h.states = append(h.states, ref)
		slot = &h.states[len(h.states)-1]
	}
	h.idx++
	ref, _ := (*slot).(*T)
	return ref
}

func UseReducer[S, A any](h *Hooks, reducer func(S, A) S, initial S) (S, func(A)) {
	ref := UseRef(h, initial)
	dispatch := func(action A) {
		*ref = reducer(*ref, action)
		if h.owner != nil {
			h.owner.RequestRender()
		}
	}
	return *ref, dispatch
}

func UseEffect(h *Hooks, deps []any, effect func() func()) {
	var entry *effectEntry
	if h.effIdx < len(h.effects) {
		entry = h.effects[h.effIdx]
	} else {
		entry = &effectEntry{}
		h.effects = append(h.effects, entry)
	}
	entry.deps = deps
	entry.effect = effect
	h.effIdx++
}

func UseTick(h *Hooks, fn func(elapsed time.Duration) bool) {
	for h.tickIdx >= len(h.ticks) {
		h.ticks = append(h.ticks, tickEntry{})
	}
	h.ticks[h.tickIdx].fn = fn
	h.tickIdx++
}

func (h *Hooks) slot() *any {
	if h.idx >= len(h.states) {
		return nil
	}
	return &h.states[h.idx]
}

func (h *Hooks) EndRender(requestsBefore int, requestsNow func() int) bool {
	for _, e := range h.effects {
		if !e.ran || !depsEqual(e.deps, e.prevDeps) {
			if e.cleanup != nil {
				e.cleanup()
			}
			e.cleanup = nil
			if e.effect != nil {
				if cl := e.effect(); cl != nil {
					e.cleanup = cl
				}
			}
			e.prevDeps = append(e.prevDeps[:0], e.deps...)
			e.ran = true
		}
	}
	return requestsNow() > requestsBefore
}

func (h *Hooks) Tick(elapsed time.Duration) bool {
	dirty := false
	for i := range h.ticks {
		if h.ticks[i].fn != nil && h.ticks[i].fn(elapsed) {
			dirty = true
		}
	}
	return dirty
}

func depsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
