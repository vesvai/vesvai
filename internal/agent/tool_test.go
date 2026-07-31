package agent

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

func TestFuncTool_Name(t *testing.T) {
	tool := NewFuncTool("search", "Search the web", nil, func(ctx context.Context, params map[string]any) (string, error) {
		return "result", nil
	})

	if tool.Name() != "search" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "search")
	}
}

func TestFuncTool_Description(t *testing.T) {
	tool := NewFuncTool("search", "Search the web", nil, func(ctx context.Context, params map[string]any) (string, error) {
		return "", nil
	})

	if tool.Description() != "Search the web" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Search the web")
	}
}

func TestFuncTool_Schema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}
	tool := NewFuncTool("search", "desc", schema, func(ctx context.Context, params map[string]any) (string, error) {
		return "", nil
	})

	got := tool.Schema()
	if got["type"] != "object" {
		t.Errorf("Schema().type = %v, want object", got["type"])
	}
	props, ok := got["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema().properties is not map[string]any")
	}
	if _, exists := props["query"]; !exists {
		t.Error("Schema().properties missing 'query'")
	}
}

func TestFuncTool_Handle(t *testing.T) {
	called := false
	var receivedParams map[string]any
	tool := NewFuncTool("echo", "echo", nil, func(ctx context.Context, params map[string]any) (string, error) {
		called = true
		receivedParams = params
		return "echoed", nil
	})

	result, err := tool.Handle(context.Background(), map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !called {
		t.Error("handler not called")
	}
	if receivedParams["key"] != "val" {
		t.Errorf("params[key] = %v, want val", receivedParams["key"])
	}
	if result != "echoed" {
		t.Errorf("Handle() = %q, want %q", result, "echoed")
	}
}

func TestFuncTool_Handle_Error(t *testing.T) {
	expectedErr := errors.New("handler failed")
	tool := NewFuncTool("fail", "fail", nil, func(ctx context.Context, params map[string]any) (string, error) {
		return "", expectedErr
	})

	_, err := tool.Handle(context.Background(), nil)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Handle() error = %v, want %v", err, expectedErr)
	}
}

func TestParseToolArgs_Empty(t *testing.T) {
	tests := []struct {
		name string
		args string
	}{
		{"empty string", ""},
		{"empty object", "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ParseToolArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseToolArgs(%q) error = %v", tt.args, err)
			}
			if params != nil {
				t.Errorf("ParseToolArgs(%q) = %v, want nil", tt.args, params)
			}
		})
	}
}

func TestParseToolArgs_Valid(t *testing.T) {
	input := `{"query":"hello","count":5}`
	params, err := ParseToolArgs(input)
	if err != nil {
		t.Fatalf("ParseToolArgs() error = %v", err)
	}
	if params["query"] != "hello" {
		t.Errorf("query = %v, want hello", params["query"])
	}
	if params["count"] != float64(5) {
		t.Errorf("count = %v, want 5", params["count"])
	}
}

func TestParseToolArgs_Invalid(t *testing.T) {
	_, err := ParseToolArgs("{invalid")
	if err == nil {
		t.Error("ParseToolArgs({invalid) should return error")
	}
}

func TestToLLMTool(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q": map[string]any{"type": "string"},
		},
	}
	tool := NewFuncTool("search", "Search", schema, nil)
	llmTool := ToLLMTool(tool)

	if llmTool.Type != "function" {
		t.Errorf("Type = %q, want %q", llmTool.Type, "function")
	}
	if llmTool.Function.Name != "search" {
		t.Errorf("Function.Name = %q, want %q", llmTool.Function.Name, "search")
	}
	if llmTool.Function.Description != "Search" {
		t.Errorf("Function.Description = %q, want %q", llmTool.Function.Description, "Search")
	}
}

func TestToolRegistry_RegisterAndGet(t *testing.T) {
	reg := NewToolRegistry()
	tool := NewFuncTool("test", "test tool", nil, nil)

	reg.Register(tool)

	got, ok := reg.Get("test")
	if !ok {
		t.Fatal("Get() ok = false")
	}
	if got.Name() != "test" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "test")
	}
}

func TestToolRegistry_Get_NotFound(t *testing.T) {
	reg := NewToolRegistry()

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Get() ok = true for nonexistent tool")
	}
}

func TestToolRegistry_Has(t *testing.T) {
	reg := NewToolRegistry()
	tool := NewFuncTool("exists", "desc", nil, nil)

	reg.Register(tool)

	if !reg.Has("exists") {
		t.Error("Has() = false for registered tool")
	}
	if reg.Has("missing") {
		t.Error("Has() = true for unregistered tool")
	}
}

func TestToolRegistry_Register_DuplicatePanics(t *testing.T) {
	reg := NewToolRegistry()
	tool1 := NewFuncTool("dup", "first", nil, nil)
	tool2 := NewFuncTool("dup", "second", nil, nil)

	reg.Register(tool1)

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Register() duplicate should panic")
		}
	}()

	reg.Register(tool2)
}

func TestToolRegistry_All_PreservesOrder(t *testing.T) {
	reg := NewToolRegistry()
	names := []string{"alpha", "bravo", "charlie", "delta"}

	for _, n := range names {
		reg.Register(NewFuncTool(n, "desc", nil, nil))
	}

	all := reg.All()
	if len(all) != len(names) {
		t.Fatalf("All() len = %d, want %d", len(all), len(names))
	}
	for i, tool := range all {
		if tool.Name() != names[i] {
			t.Errorf("All()[%d].Name() = %q, want %q", i, tool.Name(), names[i])
		}
	}
}

func TestToolRegistry_Names(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(NewFuncTool("a", "desc", nil, nil))
	reg.Register(NewFuncTool("b", "desc", nil, nil))

	names := reg.Names()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("Names() = %v, want [a b]", names)
	}
}

func TestToolRegistry_Count(t *testing.T) {
	reg := NewToolRegistry()
	if reg.Count() != 0 {
		t.Errorf("Count() = %d, want 0", reg.Count())
	}

	reg.Register(NewFuncTool("a", "desc", nil, nil))
	reg.Register(NewFuncTool("b", "desc", nil, nil))
	if reg.Count() != 2 {
		t.Errorf("Count() = %d, want 2", reg.Count())
	}
}

func TestToolRegistry_ToLLMTools(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(NewFuncTool("a", "desc a", nil, nil))
	reg.Register(NewFuncTool("b", "desc b", nil, nil))

	llmTools := reg.ToLLMTools()
	if len(llmTools) != 2 {
		t.Fatalf("ToLLMTools() len = %d, want 2", len(llmTools))
	}
	if llmTools[0].Function.Name != "a" {
		t.Errorf("ToLLMTools()[0].Function.Name = %q, want a", llmTools[0].Function.Name)
	}
}

func TestToolRegistry_Merge(t *testing.T) {
	reg1 := NewToolRegistry()
	reg1.Register(NewFuncTool("a", "desc", nil, nil))

	reg2 := NewToolRegistry()
	reg2.Register(NewFuncTool("b", "desc", nil, nil))

	reg1.Merge(reg2)

	if reg1.Count() != 2 {
		t.Errorf("Count() after merge = %d, want 2", reg1.Count())
	}
	if !reg1.Has("a") || !reg1.Has("b") {
		t.Error("Merge() missing tools")
	}
}

func TestToolRegistry_Merge_DuplicatePanics(t *testing.T) {
	reg1 := NewToolRegistry()
	reg1.Register(NewFuncTool("same", "desc", nil, nil))

	reg2 := NewToolRegistry()
	reg2.Register(NewFuncTool("same", "desc", nil, nil))

	defer func() {
		if r := recover(); r == nil {
			t.Error("Merge() duplicate should panic")
		}
	}()

	reg1.Merge(reg2)
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewToolRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := "tool_" + string(rune('a'+n%26)) + "_" + string(rune('0'+n/26))
			reg.Register(NewFuncTool(name, "desc", nil, nil))
		}(i)
	}

	wg.Wait()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := "tool_" + string(rune('a'+n%26)) + "_" + string(rune('0'+n/26))
			reg.Has(name)
			reg.Get(name)
			reg.Names()
		}(i)
	}

	wg.Wait()
}

func TestParseToolArgs_NestedJSON(t *testing.T) {
	nested := map[string]any{
		"config": map[string]any{
			"timeout": float64(30),
			"retries": float64(3),
		},
		"items": []any{"a", "b", "c"},
	}
	b, _ := json.Marshal(nested)

	params, err := ParseToolArgs(string(b))
	if err != nil {
		t.Fatalf("ParseToolArgs() error = %v", err)
	}

	config, ok := params["config"].(map[string]any)
	if !ok {
		t.Fatal("config is not map")
	}
	if config["timeout"] != float64(30) {
		t.Errorf("config.timeout = %v, want 30", config["timeout"])
	}

	items, ok := params["items"].([]any)
	if !ok {
		t.Fatal("items is not slice")
	}
	if len(items) != 3 {
		t.Errorf("items len = %d, want 3", len(items))
	}
}
