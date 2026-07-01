package hook

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
)

func TestCallback_IsAction(t *testing.T) {
	cb := &Callback{
		action: func(ctx context.Context, args ...interface{}) error { return nil },
	}
	if !cb.IsAction() {
		t.Error("IsAction() = false, want true")
	}
}

func TestCallback_IsFilter(t *testing.T) {
	cb := &Callback{
		filter: func(ctx context.Context, value interface{}, args ...interface{}) interface{} { return value },
	}
	if !cb.IsFilter() {
		t.Error("IsFilter() = false, want true")
	}
}

func TestCallback_DisableEnable(t *testing.T) {
	cb := &Callback{}

	if cb.IsDisabled() {
		t.Error("IsDisabled() = true, want false initially")
	}

	cb.Disable()
	if !cb.IsDisabled() {
		t.Error("IsDisabled() = false after Disable()")
	}

	cb.Enable()
	if cb.IsDisabled() {
		t.Error("IsDisabled() = true after Enable()")
	}
}

func TestCallback_RunCount(t *testing.T) {
	cb := &Callback{}
	if cb.RunCount() != 0 {
		t.Errorf("RunCount() = %d, want 0", cb.RunCount())
	}

	cb.runCount.Add(1)
	cb.runCount.Add(1)
	if cb.RunCount() != 2 {
		t.Errorf("RunCount() = %d, want 2", cb.RunCount())
	}
}

func TestNew(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	h := New(bus)
	if h == nil {
		t.Fatal("New() returned nil")
	}
	if h.eventBus != bus {
		t.Error("eventBus not set correctly")
	}
}

func TestNew_NilEventBus(t *testing.T) {
	h := New(nil)
	if h == nil {
		t.Fatal("New(nil) returned nil")
	}
}

func TestHooks_AddAction(t *testing.T) {
	h := New(nil)

	cb := h.AddAction("test:action", func(ctx context.Context, args ...interface{}) error {
		return nil
	}, 10)

	if cb == nil {
		t.Fatal("AddAction() returned nil")
	}
	if cb.Hook != "test:action" {
		t.Errorf("Hook = %q, want %q", cb.Hook, "test:action")
	}
	if cb.Priority != 10 {
		t.Errorf("Priority = %d, want 10", cb.Priority)
	}
	if cb.Once {
		t.Error("Once = true, want false")
	}
	if !cb.IsAction() {
		t.Error("IsAction() = false, want true")
	}
	if !h.HasAction("test:action") {
		t.Error("HasAction() = false after AddAction()")
	}
}

func TestHooks_AddActionOnce(t *testing.T) {
	h := New(nil)

	callCount := 0
	h.AddActionOnce("once:action", func(ctx context.Context, args ...interface{}) error {
		callCount++
		return nil
	}, 10)

	h.DoAction(context.Background(), "once:action")
	h.DoAction(context.Background(), "once:action")

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (once)", callCount)
	}
}

func TestHooks_AddFilter(t *testing.T) {
	h := New(nil)

	cb := h.AddFilter("test:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return "filtered"
	}, 20)

	if cb == nil {
		t.Fatal("AddFilter() returned nil")
	}
	if cb.Hook != "test:filter" {
		t.Errorf("Hook = %q, want %q", cb.Hook, "test:filter")
	}
	if cb.Priority != 20 {
		t.Errorf("Priority = %d, want 20", cb.Priority)
	}
	if !cb.IsFilter() {
		t.Error("IsFilter() = false, want true")
	}
	if !h.HasFilter("test:filter") {
		t.Error("HasFilter() = false after AddFilter()")
	}
}

func TestHooks_AddFilterOnce(t *testing.T) {
	h := New(nil)

	callCount := 0
	h.AddFilterOnce("once:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		callCount++
		return "filtered"
	}, 10)

	h.ApplyFilter(context.Background(), "once:filter", "original")
	h.ApplyFilter(context.Background(), "once:filter", "original")

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (once)", callCount)
	}
}

func TestHooks_DoAction(t *testing.T) {
	h := New(nil)

	var receivedArgs []interface{}
	h.AddAction("do:action", func(ctx context.Context, args ...interface{}) error {
		receivedArgs = args
		return nil
	}, 10)

	h.DoAction(context.Background(), "do:action", "arg1", "arg2")

	if len(receivedArgs) != 2 {
		t.Fatalf("receivedArgs len = %d, want 2", len(receivedArgs))
	}
	if receivedArgs[0] != "arg1" || receivedArgs[1] != "arg2" {
		t.Errorf("receivedArgs = %v, want [arg1 arg2]", receivedArgs)
	}
}

func TestHooks_DoAction_NoSubscribers(t *testing.T) {
	h := New(nil)
	// Should not panic
	h.DoAction(context.Background(), "nonexistent")
}

func TestHooks_DoAction_DisabledCallback(t *testing.T) {
	h := New(nil)

	callCount := 0
	cb := h.AddAction("disabled:action", func(ctx context.Context, args ...interface{}) error {
		callCount++
		return nil
	}, 10)

	cb.Disable()
	h.DoAction(context.Background(), "disabled:action")

	if callCount != 0 {
		t.Errorf("callCount = %d, want 0 (disabled)", callCount)
	}
}

func TestHooks_DoAction_Error(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	h := New(bus)

	wantErr := errors.New("action failed")
	h.AddAction("error:action", func(ctx context.Context, args ...interface{}) error {
		return wantErr
	}, 10)

	// Should not panic, error is published to event bus
	h.DoAction(context.Background(), "error:action")
}

func TestHooks_DoAction_Priority(t *testing.T) {
	h := New(nil)

	var order []int
	h.AddAction("prio:action", func(ctx context.Context, args ...interface{}) error {
		order = append(order, 1)
		return nil
	}, 10)

	h.AddAction("prio:action", func(ctx context.Context, args ...interface{}) error {
		order = append(order, 2)
		return nil
	}, 20)

	h.AddAction("prio:action", func(ctx context.Context, args ...interface{}) error {
		order = append(order, 3)
		return nil
	}, 5)

	h.DoAction(context.Background(), "prio:action")

	if len(order) != 3 {
		t.Fatalf("order len = %d, want 3", len(order))
	}
	// Higher priority first: 20, 10, 5
	if order[0] != 2 || order[1] != 1 || order[2] != 3 {
		t.Errorf("order = %v, want [2 1 3]", order)
	}
}

func TestHooks_ApplyFilter(t *testing.T) {
	h := New(nil)

	h.AddFilter("test:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return value.(string) + "_filtered"
	}, 10)

	result := h.ApplyFilter(context.Background(), "test:filter", "original")
	if result != "original_filtered" {
		t.Errorf("result = %q, want %q", result, "original_filtered")
	}
}

func TestHooks_ApplyFilter_NoFilters(t *testing.T) {
	h := New(nil)

	result := h.ApplyFilter(context.Background(), "nonexistent", "value")
	if result != "value" {
		t.Errorf("result = %v, want original value", result)
	}
}

func TestHooks_ApplyFilter_Chain(t *testing.T) {
	h := New(nil)

	h.AddFilter("chain:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return value.(int) + 1
	}, 10)

	h.AddFilter("chain:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		return value.(int) * 2
	}, 20)

	result := h.ApplyFilter(context.Background(), "chain:filter", 5)
	// Priority 20 first: 5*2=10, then 10+1=11
	if result != 11 {
		t.Errorf("result = %v, want 11", result)
	}
}

func TestHooks_RemoveCallback(t *testing.T) {
	h := New(nil)

	callCount := 0
	cb := h.AddAction("remove:action", func(ctx context.Context, args ...interface{}) error {
		callCount++
		return nil
	}, 10)

	h.DoAction(context.Background(), "remove:action")
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	h.RemoveCallback(cb.ID)
	h.DoAction(context.Background(), "remove:action")
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 after remove", callCount)
	}
}

func TestHooks_RemoveAll(t *testing.T) {
	h := New(nil)

	h.AddAction("remove:all", func(ctx context.Context, args ...interface{}) error { return nil }, 10)
	h.AddFilter("remove:all", func(ctx context.Context, value interface{}, args ...interface{}) interface{} { return value }, 10)

	h.RemoveAll("remove:all")

	if h.HasAction("remove:all") {
		t.Error("HasAction() = true after RemoveAll()")
	}
	if h.HasFilter("remove:all") {
		t.Error("HasFilter() = true after RemoveAll()")
	}
}

func TestHooks_GetActions(t *testing.T) {
	h := New(nil)

	h.AddAction("get:actions", func(ctx context.Context, args ...interface{}) error { return nil }, 10)
	h.AddAction("get:actions", func(ctx context.Context, args ...interface{}) error { return nil }, 20)

	cbs := h.GetActions("get:actions")
	if len(cbs) != 2 {
		t.Errorf("GetActions() len = %d, want 2", len(cbs))
	}
}

func TestHooks_GetFilters(t *testing.T) {
	h := New(nil)

	h.AddFilter("get:filters", func(ctx context.Context, value interface{}, args ...interface{}) interface{} { return value }, 10)
	h.AddFilter("get:filters", func(ctx context.Context, value interface{}, args ...interface{}) interface{} { return value }, 20)

	cbs := h.GetFilters("get:filters")
	if len(cbs) != 2 {
		t.Errorf("GetFilters() len = %d, want 2", len(cbs))
	}
}

func TestHooks_GetAllHooks(t *testing.T) {
	h := New(nil)

	h.AddAction("all:action", func(ctx context.Context, args ...interface{}) error { return nil }, 10)
	h.AddFilter("all:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} { return value }, 10)

	actions, filters := h.GetAllHooks()
	if len(actions) != 1 {
		t.Errorf("actions len = %d, want 1", len(actions))
	}
	if len(filters) != 1 {
		t.Errorf("filters len = %d, want 1", len(filters))
	}
}

func TestHooks_Stats(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	h := New(bus)

	h.AddAction("stats:action", func(ctx context.Context, args ...interface{}) error { return nil }, 10)
	h.AddFilter("stats:filter", func(ctx context.Context, value interface{}, args ...interface{}) interface{} { return value }, 10)

	h.DoAction(context.Background(), "stats:action")
	h.ApplyFilter(context.Background(), "stats:filter", "val")

	registered, actionsTriggered, filtersTriggered := h.Stats()
	if registered != 2 {
		t.Errorf("registered = %d, want 2", registered)
	}
	if actionsTriggered != 1 {
		t.Errorf("actionsTriggered = %d, want 1", actionsTriggered)
	}
	if filtersTriggered != 1 {
		t.Errorf("filtersTriggered = %d, want 1", filtersTriggered)
	}
}

func TestHooks_SetContext(t *testing.T) {
	h := New(nil)

	ctx := context.WithValue(context.Background(), "key", "value")
	h.SetContext(ctx)

	if h.globalCtx != ctx {
		t.Error("SetContext() did not update globalCtx")
	}
}

func TestHooks_EventBus(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	h := New(bus)
	if h.EventBus() != bus {
		t.Error("EventBus() returned wrong bus")
	}
}

func TestHooks_ConcurrentAccess(t *testing.T) {
	h := New(nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.AddAction("concurrent:action", func(ctx context.Context, args ...interface{}) error { return nil }, 10)
			h.DoAction(context.Background(), "concurrent:action")
		}()
	}
	wg.Wait()
}

func TestHooks_DoActionAsync(t *testing.T) {
	h := New(nil)

	var count atomic.Int32
	h.AddAction("async:action", func(ctx context.Context, args ...interface{}) error {
		count.Add(1)
		return nil
	}, 10)

	h.DoActionAsync("async:action")
	h.DoActionAsync("async:action")

	// Wait for goroutines to execute
	time.Sleep(50 * time.Millisecond)

	if count.Load() < 2 {
		t.Errorf("count = %d, want >= 2", count.Load())
	}
}
