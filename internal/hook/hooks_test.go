package hook

import (
	"context"
	"testing"
)

func TestNewHookContext(t *testing.T) {
	ctx := context.Background()
	hc := NewHookContext(ctx)

	if hc == nil {
		t.Fatal("NewHookContext() returned nil")
	}
	if hc.Context != ctx {
		t.Error("Context not set correctly")
	}
	if hc.Data == nil {
		t.Error("Data map not initialized")
	}
}

func TestHookContext_WithSession(t *testing.T) {
	hc := NewHookContext(context.Background())
	result := hc.WithSession("session-123", "test-session")

	if hc.Session == nil {
		t.Fatal("Session not set")
	}
	if hc.Session.ID != "session-123" {
		t.Errorf("Session.ID = %q, want %q", hc.Session.ID, "session-123")
	}
	if hc.Session.Name != "test-session" {
		t.Errorf("Session.Name = %q, want %q", hc.Session.Name, "test-session")
	}
	if result != hc {
		t.Error("WithSession() should return self for chaining")
	}
}

func TestHookContext_Set(t *testing.T) {
	hc := NewHookContext(context.Background())
	hc.Set("key1", "value1")
	hc.Set("key2", 42)

	if len(hc.Data) != 2 {
		t.Errorf("Data len = %d, want 2", len(hc.Data))
	}
	if hc.Data["key1"] != "value1" {
		t.Errorf("Data[key1] = %v, want %q", hc.Data["key1"], "value1")
	}
	if hc.Data["key2"] != 42 {
		t.Errorf("Data[key2] = %v, want 42", hc.Data["key2"])
	}
}

func TestHookContext_Get(t *testing.T) {
	hc := NewHookContext(context.Background())
	hc.Set("exists", "value")

	val, ok := hc.Get("exists")
	if !ok {
		t.Error("Get() ok = false for existing key")
	}
	if val != "value" {
		t.Errorf("Get() = %v, want %q", val, "value")
	}

	_, ok = hc.Get("nonexistent")
	if ok {
		t.Error("Get() ok = true for nonexistent key")
	}
}

func TestHookConstants(t *testing.T) {
	if HookSystemInit != "system:init" {
		t.Errorf("HookSystemInit = %q, want %q", HookSystemInit, "system:init")
	}
	if HookSystemReady != "system:ready" {
		t.Errorf("HookSystemReady = %q, want %q", HookSystemReady, "system:ready")
	}
	if HookSystemShutdown != "system:shutdown" {
		t.Errorf("HookSystemShutdown = %q, want %q", HookSystemShutdown, "system:shutdown")
	}
	if HookError != "error" {
		t.Errorf("HookError = %q, want %q", HookError, "error")
	}
	if HookErrorRecovery != "error:recovery" {
		t.Errorf("HookErrorRecovery = %q, want %q", HookErrorRecovery, "error:recovery")
	}
}

func TestSessionInfo(t *testing.T) {
	si := &SessionInfo{
		ID:   "id-1",
		Name: "session-name",
	}
	if si.ID != "id-1" {
		t.Errorf("ID = %q, want %q", si.ID, "id-1")
	}
	if si.Name != "session-name" {
		t.Errorf("Name = %q, want %q", si.Name, "session-name")
	}
}
