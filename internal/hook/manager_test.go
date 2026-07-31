package hook

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
)

func TestDefault(t *testing.T) {
	hooksOnce = sync.Once{}
	defaultHooks = nil

	h1 := Default()
	h2 := Default()

	if h1 == nil {
		t.Fatal("Default() returned nil")
	}
	if h1 != h2 {
		t.Error("Default() should return same instance")
	}
}

func TestInit(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	h := Init(bus)
	if h == nil {
		t.Fatal("Init() returned nil")
	}
	if h.EventBus() != bus {
		t.Error("Init() did not set event bus")
	}
}

func TestHookBuilder_Do(t *testing.T) {
	h := New(nil)

	called := false
	cb := h.On("builder:action").Priority(75).Do(func(ctx context.Context, args ...interface{}) error {
		called = true
		return nil
	})

	if cb == nil {
		t.Fatal("Do() returned nil")
	}
	if cb.Priority != 75 {
		t.Errorf("Priority = %d, want 75", cb.Priority)
	}

	h.DoAction(context.Background(), "builder:action")
	if !called {
		t.Error("callback not called")
	}
}

func TestHookBuilder_Do_Once(t *testing.T) {
	h := New(nil)

	callCount := 0
	h.On("builder:once").Once().Do(func(ctx context.Context, args ...interface{}) error {
		callCount++
		return nil
	})

	h.DoAction(context.Background(), "builder:once")
	h.DoAction(context.Background(), "builder:once")

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestHookBuilder_Filter(t *testing.T) {
	h := New(nil)

	cb := h.On("builder:filter").Priority(25).Filter(func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return "filtered"
	})

	if cb == nil {
		t.Fatal("Filter() returned nil")
	}
	if cb.Priority != 25 {
		t.Errorf("Priority = %d, want 25", cb.Priority)
	}

	result := h.ApplyFilter(context.Background(), "builder:filter", "original")
	if result != "filtered" {
		t.Errorf("result = %q, want %q", result, "filtered")
	}
}

func TestHookBuilder_Filter_Once(t *testing.T) {
	h := New(nil)

	callCount := 0
	h.On("builder:filter:once").Once().Filter(func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		callCount++
		return value
	})

	h.ApplyFilter(context.Background(), "builder:filter:once", "val")
	h.ApplyFilter(context.Background(), "builder:filter:once", "val")

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestHookBuilder_DoAsync(t *testing.T) {
	h := New(nil)

	var count atomic.Int32
	h.On("builder:async").DoAsync(func(ctx context.Context, args ...interface{}) error {
		count.Add(1)
		return nil
	})

	h.DoActionAsync("builder:async")
	time.Sleep(50 * time.Millisecond)

	if count.Load() < 1 {
		t.Errorf("count = %d, want >= 1", count.Load())
	}
}

func TestHookManager_NewManager(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(bus)
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.Active() == nil {
		t.Error("Active() returned nil")
	}
}

func TestHookManager_RegisterNamespace(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(bus)

	h1 := m.RegisterNamespace("ns1")
	h2 := m.RegisterNamespace("ns1")

	if h1 == nil {
		t.Fatal("RegisterNamespace() returned nil")
	}
	if h1 != h2 {
		t.Error("RegisterNamespace() should return same instance for same name")
	}
}

func TestHookManager_GetHooks(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(bus)
	m.RegisterNamespace("ns")

	h := m.GetHooks("ns")
	if h == nil {
		t.Error("GetHooks() returned nil for registered namespace")
	}

	h = m.GetHooks("nonexistent")
	if h != nil {
		t.Error("GetHooks() returned nil for nonexistent namespace")
	}
}

func TestHookManager_RemoveNamespace(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(bus)
	m.RegisterNamespace("ns")
	m.RemoveNamespace("ns")

	h := m.GetHooks("ns")
	if h != nil {
		t.Error("GetHooks() returned nil after RemoveNamespace()")
	}
}

func TestHookCollection(t *testing.T) {
	c := NewCollection()
	if c == nil {
		t.Fatal("NewCollection() returned nil")
	}
}

func TestHookCollection_DoAction(t *testing.T) {
	c := NewCollection()

	h1 := New(nil)
	h2 := New(nil)

	var order []int
	h1.AddAction("coll:action", func(ctx context.Context, args ...interface{}) error {
		order = append(order, 1)
		return nil
	}, 10)
	h2.AddAction("coll:action", func(ctx context.Context, args ...interface{}) error {
		order = append(order, 2)
		return nil
	}, 10)

	c.Add(h1)
	c.Add(h2)
	c.DoAction(context.Background(), "coll:action")

	if len(order) != 2 {
		t.Fatalf("order len = %d, want 2", len(order))
	}
}

func TestHookCollection_ApplyFilter(t *testing.T) {
	c := NewCollection()

	h1 := New(nil)
	h2 := New(nil)

	h1.AddFilter("coll:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return value.(string) + "_h1"
	}, 10)
	h2.AddFilter("coll:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return value.(string) + "_h2"
	}, 10)

	c.Add(h1)
	c.Add(h2)
	result := c.ApplyFilter(context.Background(), "coll:filter", "start")

	if result != "start_h1_h2" {
		t.Errorf("result = %q, want %q", result, "start_h1_h2")
	}
}

func TestHookType_Constants(t *testing.T) {
	if HookTypeAction != "action" {
		t.Errorf("HookTypeAction = %q, want %q", HookTypeAction, "action")
	}
	if HookTypeFilter != "filter" {
		t.Errorf("HookTypeFilter = %q, want %q", HookTypeFilter, "filter")
	}
}
