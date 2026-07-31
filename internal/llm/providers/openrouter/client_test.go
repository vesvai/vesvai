package openrouter

import (
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
)

func TestClient_Name(t *testing.T) {
	c := NewClient(Config{APIKey: "test"})
	if c.Name() != "openrouter" {
		t.Errorf("Name() = %q, want openrouter", c.Name())
	}
}

func TestClient_SatisfiesProviderInterface(t *testing.T) {
	var _ llm.Provider = (*Client)(nil)
}

func TestInjectOpenRouterFields_AllowFallbacks(t *testing.T) {
	body := map[string]any{"model": "x"}
	injectOpenRouterFields(body)
	p, ok := body["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider field missing or wrong type: %T", body["provider"])
	}
	if p["allow_fallbacks"] != true {
		t.Errorf("allow_fallbacks = %v, want true", p["allow_fallbacks"])
	}
	if body["model"] != "x" {
		t.Errorf("existing fields should be preserved, model = %v", body["model"])
	}
}