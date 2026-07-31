package agent

import (
	"context"
	"errors"
	"testing"
)

func TestMiddlewareChain_Empty(t *testing.T) {
	chain := NewMiddlewareChain()
	called := false

	err := chain.Execute(context.Background(), nil, func(ctx context.Context) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !called {
		t.Error("final handler not called")
	}
}

func TestMiddlewareChain_Single(t *testing.T) {
	var order []string

	mw := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		order = append(order, "mw:before")
		err := next(ctx)
		order = append(order, "mw:after")
		return err
	}

	chain := NewMiddlewareChain(mw)
	chain.Execute(context.Background(), nil, func(ctx context.Context) error {
		order = append(order, "handler")
		return nil
	})

	expected := []string{"mw:before", "handler", "mw:after"}
	if len(order) != len(expected) {
		t.Fatalf("order len = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestMiddlewareChain_Multiple_ExecutionOrder(t *testing.T) {
	var order []string

	mw1 := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		order = append(order, "1:before")
		err := next(ctx)
		order = append(order, "1:after")
		return err
	}
	mw2 := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		order = append(order, "2:before")
		err := next(ctx)
		order = append(order, "2:after")
		return err
	}
	mw3 := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		order = append(order, "3:before")
		err := next(ctx)
		order = append(order, "3:after")
		return err
	}

	chain := NewMiddlewareChain(mw1, mw2, mw3)
	chain.Execute(context.Background(), nil, func(ctx context.Context) error {
		order = append(order, "handler")
		return nil
	})

	expected := []string{
		"1:before", "2:before", "3:before",
		"handler",
		"3:after", "2:after", "1:after",
	}
	if len(order) != len(expected) {
		t.Fatalf("order len = %d, want %d\norder = %v", len(order), len(expected), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestMiddlewareChain_Error_Propagates(t *testing.T) {
	expectedErr := errors.New("middleware error")

	mw := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		return expectedErr
	}

	chain := NewMiddlewareChain(mw)
	err := chain.Execute(context.Background(), nil, func(ctx context.Context) error {
		t.Error("handler should not be called")
		return nil
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestMiddlewareChain_HandlerError_Propagates(t *testing.T) {
	expectedErr := errors.New("handler error")

	mw := func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		return next(ctx)
	}

	chain := NewMiddlewareChain(mw)
	err := chain.Execute(context.Background(), nil, func(ctx context.Context) error {
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestMiddlewareChain_Use(t *testing.T) {
	chain := NewMiddlewareChain()
	var called bool

	chain.Use(func(ctx context.Context, agent Agent, next MiddlewareFunc) error {
		called = true
		return next(ctx)
	})

	chain.Execute(context.Background(), nil, func(ctx context.Context) error {
		return nil
	})

	if !called {
		t.Error("Use() middleware not executed")
	}
}

func TestMiddlewareChain_AgentPassed(t *testing.T) {
	agent := &mockAgent{name: "test-agent"}
	var receivedAgent Agent

	mw := func(ctx context.Context, a Agent, next MiddlewareFunc) error {
		receivedAgent = a
		return next(ctx)
	}

	chain := NewMiddlewareChain(mw)
	chain.Execute(context.Background(), agent, func(ctx context.Context) error {
		return nil
	})

	if receivedAgent != agent {
		t.Errorf("middleware received wrong agent: %v, want %v", receivedAgent, agent)
	}
}

func TestPreToolCallMiddleware(t *testing.T) {
	mw := PreToolCallMiddleware(func(ctx context.Context, toolName string, params map[string]any) error {
		return nil
	})

	nextCalled := false
	err := mw(context.Background(), nil, func(ctx context.Context) error {
		nextCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("PreToolCallMiddleware() error = %v", err)
	}
	if !nextCalled {
		t.Error("next handler not called")
	}
}

func TestPostToolCallMiddleware(t *testing.T) {
	mw := PostToolCallMiddleware(func(ctx context.Context, toolName string, result string, err error) (string, error) {
		return result, err
	})

	nextCalled := false
	err := mw(context.Background(), nil, func(ctx context.Context) error {
		nextCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("PostToolCallMiddleware() error = %v", err)
	}
	if !nextCalled {
		t.Error("next handler not called")
	}
}
