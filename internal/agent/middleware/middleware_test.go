package middleware

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
)

type recorder struct {
	BaseMiddleware
	calls []string
}

func (r *recorder) BeforeRun(_ context.Context, _, _ string) error {
	r.calls = append(r.calls, "BeforeRun")
	return nil
}

func (r *recorder) AfterRun(_ context.Context, _ string, _ *Result, _ error) error {
	r.calls = append(r.calls, "AfterRun")
	return nil
}

func (r *recorder) BeforeLLM(_ context.Context, _ *llm.Request) error {
	r.calls = append(r.calls, "BeforeLLM")
	return nil
}

func (r *recorder) AfterLLM(_ context.Context, _ *llm.Request, _ *llm.Response) error {
	r.calls = append(r.calls, "AfterLLM")
	return nil
}

func (r *recorder) BeforeTool(_ context.Context, _ llm.ToolCall) error {
	r.calls = append(r.calls, "BeforeTool")
	return nil
}

func (r *recorder) AfterTool(_ context.Context, _ llm.ToolCall, _ string, _ error) error {
	r.calls = append(r.calls, "AfterTool")
	return nil
}

func (r *recorder) OnError(_ context.Context, _ error) error {
	r.calls = append(r.calls, "OnError")
	return nil
}

type blocker struct {
	BaseMiddleware
}

func (blocker) BeforeLLM(context.Context, *llm.Request) error {
	return errors.New("blocked")
}

func TestChainPhaseOrder(t *testing.T) {
	r := &recorder{}
	c := NewChain(r)
	ctx := context.Background()

	_ = c.BeforeRun(ctx, "a", "in")
	req := llm.NewRequest("m", nil)
	_ = c.BeforeLLM(ctx, req)
	_ = c.AfterLLM(ctx, req, &llm.Response{})
	_ = c.BeforeTool(ctx, llm.ToolCall{ID: "c1"})
	_ = c.AfterTool(ctx, llm.ToolCall{ID: "c1"}, "out", nil)
	_ = c.AfterRun(ctx, "a", &Result{}, nil)

	want := []string{"BeforeRun", "BeforeLLM", "AfterLLM", "BeforeTool", "AfterTool", "AfterRun"}
	if !slices.Equal(r.calls, want) {
		t.Fatalf("calls = %v, want %v", r.calls, want)
	}
}

func TestChainStopsOnFirstError(t *testing.T) {
	r := &recorder{}
	c := NewChain(r, blocker{})
	ctx := context.Background()

	_ = c.BeforeRun(ctx, "a", "in")
	if err := c.BeforeLLM(ctx, llm.NewRequest("m", nil)); err == nil {
		t.Fatal("expected error from blocker")
	}
	want := []string{"BeforeRun", "BeforeLLM"}
	if !slices.Equal(r.calls, want) {
		t.Fatalf("calls = %v, want %v", r.calls, want)
	}
}

func TestChainOnErrorInvokesAll(t *testing.T) {
	r1 := &recorder{}
	r2 := &recorder{}
	c := NewChain(r1, r2)

	if err := c.OnError(context.Background(), errors.New("boom")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r1.calls) != 1 || r1.calls[0] != "OnError" {
		t.Fatalf("recorder1 calls = %v", r1.calls)
	}
	if len(r2.calls) != 1 || r2.calls[0] != "OnError" {
		t.Fatalf("recorder2 calls = %v", r2.calls)
	}
}

func TestChainAppend(t *testing.T) {
	r1 := &recorder{}
	r2 := &recorder{}
	c := NewChain(r1)
	c.Append(r2)

	if err := c.BeforeRun(context.Background(), "a", "in"); err != nil {
		t.Fatal(err)
	}
	if len(r1.calls) != 1 || len(r2.calls) != 1 {
		t.Fatalf("expected both middleware called: r1=%v r2=%v", r1.calls, r2.calls)
	}
}

func TestChainAppendEmpty(t *testing.T) {
	c := NewChain()
	c.Append()
	c.Append(nil)
	if err := c.BeforeRun(context.Background(), "a", "in"); err != nil {
		t.Fatal(err)
	}
}

type phaseFailer struct {
	BaseMiddleware
	phase string
}

func (f phaseFailer) BeforeRun(context.Context, string, string) error {
	if f.phase == "BeforeRun" {
		return errors.New("phase fail")
	}
	return nil
}

func (f phaseFailer) AfterRun(context.Context, string, *Result, error) error {
	if f.phase == "AfterRun" {
		return errors.New("phase fail")
	}
	return nil
}

func (f phaseFailer) BeforeLLM(context.Context, *llm.Request) error {
	if f.phase == "BeforeLLM" {
		return errors.New("phase fail")
	}
	return nil
}

func (f phaseFailer) AfterLLM(context.Context, *llm.Request, *llm.Response) error {
	if f.phase == "AfterLLM" {
		return errors.New("phase fail")
	}
	return nil
}

func (f phaseFailer) BeforeTool(context.Context, llm.ToolCall) error {
	if f.phase == "BeforeTool" {
		return errors.New("phase fail")
	}
	return nil
}

func (f phaseFailer) AfterTool(context.Context, llm.ToolCall, string, error) error {
	if f.phase == "AfterTool" {
		return errors.New("phase fail")
	}
	return nil
}

func TestChainPhaseErrorPropagation(t *testing.T) {
	ctx := context.Background()
	req := llm.NewRequest("m", nil)
	cases := []struct {
		phase string
		call  func(c *Chain) error
	}{
		{"BeforeRun", func(c *Chain) error { return c.BeforeRun(ctx, "a", "in") }},
		{"AfterRun", func(c *Chain) error { return c.AfterRun(ctx, "a", &Result{}, nil) }},
		{"BeforeLLM", func(c *Chain) error { return c.BeforeLLM(ctx, req) }},
		{"AfterLLM", func(c *Chain) error { return c.AfterLLM(ctx, req, &llm.Response{}) }},
		{"BeforeTool", func(c *Chain) error { return c.BeforeTool(ctx, llm.ToolCall{}) }},
		{"AfterTool", func(c *Chain) error { return c.AfterTool(ctx, llm.ToolCall{}, "", nil) }},
	}
	for _, tc := range cases {
		c := NewChain(phaseFailer{phase: tc.phase})
		if err := tc.call(c); err == nil {
			t.Fatalf("phase %s: expected error", tc.phase)
		}
	}
}

type errOnError struct {
	BaseMiddleware
}

func (errOnError) OnError(context.Context, error) error {
	return errors.New("onerr fail")
}

func TestChainOnErrorPropagatesMiddlewareError(t *testing.T) {
	c := NewChain(errOnError{})
	if err := c.OnError(context.Background(), errors.New("boom")); err == nil || err.Error() != "onerr fail" {
		t.Fatalf("want onerr fail, got %v", err)
	}
}

func TestBaseMiddlewareNoop(t *testing.T) {
	var m Middleware = BaseMiddleware{}
	ctx := context.Background()
	req := llm.NewRequest("m", nil)
	if err := m.BeforeRun(ctx, "a", "in"); err != nil {
		t.Fatal(err)
	}
	if err := m.AfterRun(ctx, "a", &Result{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := m.BeforeLLM(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := m.AfterLLM(ctx, req, &llm.Response{}); err != nil {
		t.Fatal(err)
	}
	if err := m.BeforeTool(ctx, llm.ToolCall{}); err != nil {
		t.Fatal(err)
	}
	if err := m.AfterTool(ctx, llm.ToolCall{}, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.OnError(ctx, errors.New("x")); err != nil {
		t.Fatal(err)
	}
}
