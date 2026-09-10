package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/llm"
)

func TestJudgeBuiltLazilyAndCached(t *testing.T) {
	mgr, bus := newJudgeTestManager(t)
	syncJudgeProvider(t, mgr, bus, "judge-lazy", []llm.Model{{ID: "m1"}})

	m := &Middleware{llm: mgr}
	if m.judgeAgent != nil {
		t.Fatal("judge must not be resolved at construction")
	}
	j1 := m.judge()
	j2 := m.judge()
	if j1 == nil || j1 != j2 {
		t.Fatal("judge must be built lazily and cached")
	}
	if j1.Model.ID != "m1" {
		t.Fatalf("judge model = %q, want m1", j1.Model.ID)
	}
}

func TestJudgeUnavailableWithoutManager(t *testing.T) {
	m := New(Deps{})
	if m.judge() != nil {
		t.Fatal("expected no judge agent without an LLM manager")
	}
}

func TestJudgeResolutionFailureCached(t *testing.T) {
	mgr, _ := newJudgeTestManager(t)
	m := &Middleware{llm: mgr}
	if m.judge() != nil {
		t.Fatal("expected nil judge agent on resolution failure")
	}
	if m.judge() != nil {
		t.Fatal("resolution failure must be cached")
	}
}

func TestJudgeDeniesOnResolutionFailure(t *testing.T) {
	mgr, _ := newJudgeTestManager(t)
	m := New(Deps{
		Config: &config.PermissionConfig{Default: "judge"},
		LLM:    mgr,
	})
	a, _ := testAgent(t)
	ctx := agent.WithAgent(context.Background(), a)

	var runs int
	_, err := m.InvokeTool(ctx, llm.ToolCall{Function: llm.Function{Name: "read", Arguments: `{"filePath": "a.txt"}`}}, func(ctx context.Context, call llm.ToolCall) (string, error) {
		runs++
		return "content", nil
	})
	var denied *DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("expected DeniedError, got %v", err)
	}
	if denied.Reason != "judge provider unavailable" {
		t.Fatalf("reason = %q", denied.Reason)
	}
	if runs != 0 {
		t.Fatalf("runs = %d, want 0", runs)
	}
}
