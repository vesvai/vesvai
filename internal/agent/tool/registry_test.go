package tool

import (
	"context"
	"errors"
	"testing"
)

type stubTool struct {
	name string
	exec func(ctx context.Context, args string) (string, error)
}

func (s stubTool) Name() string        { return s.name }
func (s stubTool) Description() string { return "desc-" + s.name }
func (s stubTool) Parameters() any     { return map[string]any{"type": "object"} }
func (s stubTool) Execute(ctx context.Context, args string) (string, error) {
	if s.exec != nil {
		return s.exec(ctx, args)
	}
	return "ok:" + args, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stubTool{name: "a"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get("a"); !ok {
		t.Fatal("expected tool to be registered")
	}
	if err := r.Register(stubTool{name: "a"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	if err := r.Register(stubTool{name: ""}); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("want ErrEmptyName, got %v", err)
	}
	if err := r.Register(nil); !errors.Is(err, ErrNilTool) {
		t.Fatalf("want ErrNilTool, got %v", err)
	}
}

func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()
	r.Register(stubTool{name: "a"})
	if !r.Unregister("a") {
		t.Fatal("expected unregister to succeed")
	}
	if r.Unregister("a") {
		t.Fatal("expected second unregister to fail")
	}
	if _, ok := r.Get("a"); ok {
		t.Fatal("tool should be gone")
	}
}

func TestRegistryListSorted(t *testing.T) {
	r := NewRegistry()
	r.Register(stubTool{name: "zeta"})
	r.Register(stubTool{name: "alpha"})
	r.Register(stubTool{name: "mid"})

	names := r.Names()
	if len(names) != 3 || names[0] != "alpha" || names[1] != "mid" || names[2] != "zeta" {
		t.Fatalf("unexpected names: %v", names)
	}
	list := r.List()
	for i, want := range names {
		if list[i].Name() != want {
			t.Fatalf("list[%d] = %q, want %q", i, list[i].Name(), want)
		}
	}
}

func TestRegistryLLMTools(t *testing.T) {
	r := NewRegistry()
	r.Register(stubTool{name: "zeta"})
	r.Register(stubTool{name: "alpha"})

	tools := r.LLMTools()
	if len(tools) != 2 {
		t.Fatalf("len = %d, want 2", len(tools))
	}
	if tools[0].Type != "function" || tools[0].Function.Name != "alpha" {
		t.Fatalf("unexpected first tool: %+v", tools[0])
	}
	if tools[0].Function.Description != "desc-alpha" {
		t.Fatalf("unexpected description: %+v", tools[0])
	}
	if tools[0].Function.Parameters == nil {
		t.Fatal("parameters should be carried over")
	}
	if tools[1].Function.Name != "zeta" {
		t.Fatalf("unexpected second tool: %+v", tools[1])
	}
}

func TestSpecExecute(t *testing.T) {
	var gotArgs string
	params := map[string]any{"type": "object"}
	s := NewSpec("echo", "desc", params, func(_ context.Context, args string) (string, error) {
		gotArgs = args
		return "echo:" + args, nil
	})
	if s.Name() != "echo" || s.Description() != "desc" {
		t.Fatalf("spec metadata mismatch: %+v", s)
	}
	if p, ok := s.Parameters().(map[string]any); !ok || p["type"] != "object" {
		t.Fatalf("parameters mismatch: %+v", s.Parameters())
	}
	out, err := s.Execute(context.Background(), `{"x":1}`)
	if err != nil || out != "echo:{\"x\":1}" || gotArgs != `{"x":1}` {
		t.Fatalf("out=%q err=%v gotArgs=%q", out, err, gotArgs)
	}

	empty := NewSpec("empty", "", nil, nil)
	if _, err := empty.Execute(context.Background(), ""); !errors.Is(err, ErrNilTool) {
		t.Fatalf("want ErrNilTool, got %v", err)
	}
}
