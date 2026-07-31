package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
)

func TestPermissionMiddleware_AllowRule(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{
		Default: ActionPrompt,
		Rules:   []Rule{{Tool: "read", Action: ActionAllow}},
	}

	mw := PermissionMiddleware(mgr, nil)
	mockedAgent := &mockAgent{}
	called := false
	err := mw(toolCtx("read", map[string]any{"path": "x"}), mockedAgent, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("next should have been called for allowed tool")
	}
}

func TestPermissionMiddleware_DenyRule(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{
		Default: ActionPrompt,
		Rules: []Rule{
			{Tool: "bash", Commands: []string{"rm -rf*"}, Action: ActionDeny},
		},
	}

	mw := PermissionMiddleware(mgr, nil)
	called := false
	err := mw(toolCtx("bash", map[string]any{"command": "rm -rf /tmp"}), &mockAgent{}, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("expected denial error, got nil")
	}
	if called {
		t.Error("next should NOT have been called for denied tool")
	}
}

func TestPermissionMiddleware_PromptAllow(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{Default: ActionPrompt}

	approver := ApproverFunc(func(_ context.Context, _ string, _ map[string]any, _ string) (Decision, error) {
		return DecisionAllow, nil
	})

	mw := PermissionMiddleware(mgr, approver)
	called := false
	err := mw(toolCtx("bash", map[string]any{"command": "ls"}), &mockAgent{}, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("next should be called after user allows")
	}
}

func TestPermissionMiddleware_PromptDeny(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{Default: ActionPrompt}

	approver := ApproverFunc(func(_ context.Context, _ string, _ map[string]any, _ string) (Decision, error) {
		return DecisionDeny, nil
	})

	mw := PermissionMiddleware(mgr, approver)
	called := false
	err := mw(toolCtx("bash", map[string]any{"command": "ls"}), &mockAgent{}, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("expected denial error after user denies")
	}
	if called {
		t.Error("next should NOT be called after user denies")
	}
}

func TestPermissionMiddleware_PromptAllowAlways_Persists(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.global = &Rules{Default: ActionPrompt}

	approver := ApproverFunc(func(_ context.Context, _ string, _ map[string]any, _ string) (Decision, error) {
		return DecisionAllowAlways, nil
	})

	mw := PermissionMiddleware(mgr, approver)
	called := false
	err := mw(toolCtx("bash", map[string]any{"command": "go test"}), &mockAgent{}, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("next should be called after allow-always")
	}

	mgr2 := NewManager(dir)
	res := mgr2.Check("bash", map[string]any{"command": "go test ./..."})
	if res.Action != ActionAllow {
		t.Errorf("after allow-always, go test should be auto-allowed, got %s (reason: %s)", res.Action, res.Reason)
	}
}

func TestPermissionMiddleware_PromptNilApprover_Denies(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{Default: ActionPrompt}

	mw := PermissionMiddleware(mgr, nil)
	err := mw(toolCtx("bash", map[string]any{"command": "ls"}), &mockAgent{}, func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Fatal("nil approver should deny prompt")
	}
}

func TestPermissionMiddleware_OutOfProjectPath_Prompts(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.global = &Rules{Default: ActionAllow}

	approved := false
	approver := ApproverFunc(func(_ context.Context, _ string, _ map[string]any, _ string) (Decision, error) {
		approved = true
		return DecisionAllow, nil
	})

	mw := PermissionMiddleware(mgr, approver)
	err := mw(toolCtx("write", map[string]any{"path": "/etc/something"}), &mockAgent{}, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("out-of-project path should trigger prompt even when default=allow")
	}
}

func TestPermissionMiddleware_NoToolName_Skips(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{Default: ActionDeny}

	mw := PermissionMiddleware(mgr, nil)
	called := false
	err := mw(context.Background(), &mockAgent{}, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("next should be called when no tool name in context")
	}
}

func TestPermissionMiddleware_ApproverError(t *testing.T) {
	mgr := NewManager(t.TempDir())
	mgr.global = &Rules{Default: ActionPrompt}

	approver := ApproverFunc(func(_ context.Context, _ string, _ map[string]any, _ string) (Decision, error) {
		return 0, errors.New("UI crashed")
	})

	mw := PermissionMiddleware(mgr, approver)
	err := mw(toolCtx("bash", map[string]any{"command": "ls"}), &mockAgent{}, func(ctx context.Context) error {
		return nil
	})
	if err == nil || err.Error() == "" {
		t.Fatal("approver error should bubble up")
	}
}

type mockAgent struct{}

func (m *mockAgent) Instructions() string      { return "" }
func (m *mockAgent) ToolNames() []string       { return nil }
func (m *mockAgent) Config() agent.AgentConfig { return agent.AgentConfig{} }

func toolCtx(name string, params map[string]any) context.Context {
	return agent.WithToolContext(context.Background(), name, params)
}
