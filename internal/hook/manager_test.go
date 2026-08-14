package hook

import (
	"context"
	"testing"
)

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

func TestHookType_Constants(t *testing.T) {
	if HookTypeAction != "action" {
		t.Errorf("HookTypeAction = %q, want %q", HookTypeAction, "action")
	}
	if HookTypeFilter != "filter" {
		t.Errorf("HookTypeFilter = %q, want %q", HookTypeFilter, "filter")
	}
}
