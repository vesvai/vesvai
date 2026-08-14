package compat

import (
	"context"
	"testing"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm"
)

func newRegistry(t *testing.T, hooks *hook.Hooks) *llm.ProviderRegistry {
	t.Helper()
	registry := llm.NewProviderRegistry()
	result := hooks.ApplyFilter(context.Background(), llm.HookProviderRegistry, registry)
	got, ok := result.(*llm.ProviderRegistry)
	if !ok {
		t.Fatal("filter did not return registry")
	}
	return got
}

func TestRegisterBasic(t *testing.T) {
	hooks := hook.New(nil)
	Register(hooks, "acme", "https://api.acme.test/v1", Options{})

	registry := newRegistry(t, hooks)

	p, err := registry.Create(config.LLMConfig{Provider: "acme", APIKey: "k"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if p.Name() != "acme" {
		t.Fatalf("Name() = %q, want acme", p.Name())
	}
}

func TestRegisterMissingKey(t *testing.T) {
	hooks := hook.New(nil)
	Register(hooks, "acme", "https://api.acme.test/v1", Options{})

	registry := newRegistry(t, hooks)

	_, err := registry.Create(config.LLMConfig{Provider: "acme"})
	if err == nil {
		t.Fatal("Create() without key should error")
	}
}

func TestRegisterAllowNoKey(t *testing.T) {
	hooks := hook.New(nil)
	Register(hooks, "local", "http://localhost:8080/v1", Options{AllowNoKey: true})

	registry := newRegistry(t, hooks)

	p, err := registry.Create(config.LLMConfig{Provider: "local"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if p.Name() != "local" {
		t.Fatalf("Name() = %q, want local", p.Name())
	}
}

func TestRegisterRequiresBaseURL(t *testing.T) {
	hooks := hook.New(nil)
	Register(hooks, "variable", "", Options{ExampleBaseURL: "https://example.test"})

	registry := newRegistry(t, hooks)

	_, err := registry.Create(config.LLMConfig{Provider: "variable", APIKey: "k"})
	if err == nil {
		t.Fatal("Create() without base_url should error")
	}

	p, err := registry.Create(config.LLMConfig{Provider: "variable", APIKey: "k", BaseURL: "https://custom.test"})
	if err != nil {
		t.Fatalf("Create() with base_url error = %v", err)
	}
	if p.Name() != "variable" {
		t.Fatalf("Name() = %q, want variable", p.Name())
	}
}

func TestRegisterKeyHeader(t *testing.T) {
	hooks := hook.New(nil)
	Register(hooks, "headerkey", "https://api.acme.test/v1", Options{KeyHeader: "api-key"})

	registry := newRegistry(t, hooks)

	if _, err := registry.Create(config.LLMConfig{Provider: "headerkey", APIKey: "k"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}
