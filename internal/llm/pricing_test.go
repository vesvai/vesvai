package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/vesvai/vesvai/internal/core/config"
)

func TestLookupModelConfig(t *testing.T) {
	prices := map[string]ModelConfig{
		"openai/gpt-4o":                        {Mode: "chat", LitellmProvider: "openai", MaxInputTokens: 128000},
		"gpt-4o-mini":                          {Mode: "chat", LitellmProvider: "openai"},
		"groq/llama-3.3-70b":                   {Mode: "chat", LitellmProvider: "groq"},
		"anthropic/claude-3-5-sonnet-20241022": {Mode: "chat", LitellmProvider: "anthropic"},
	}

	t.Run("provider/model exact", func(t *testing.T) {
		cfg, err := lookupModelConfig(prices, "openai", "gpt-4o")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxInputTokens != 128000 {
			t.Fatalf("cfg = %+v", cfg)
		}
	})

	t.Run("model exact", func(t *testing.T) {
		cfg, err := lookupModelConfig(prices, "openai", "gpt-4o-mini")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.LitellmProvider != "openai" {
			t.Fatalf("cfg = %+v", cfg)
		}
	})

	t.Run("contains fallback", func(t *testing.T) {
		cfg, err := lookupModelConfig(prices, "openai", "claude-3-5-sonnet")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.LitellmProvider != "anthropic" {
			t.Fatalf("cfg = %+v", cfg)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := lookupModelConfig(prices, "openai", "nope-model"); !errors.Is(err, ErrModelConfigNotFound) {
			t.Fatalf("want ErrModelConfigNotFound, got %v", err)
		}
	})
}

func TestReasoningEffortsFor(t *testing.T) {
	cases := []struct {
		key  string
		want []string
	}{
		{"o1", []string{"low", "medium", "high"}},
		{"openai/o1", []string{"low", "medium", "high"}},
		{"gpt-5.2-2025-12-11", []string{"none", "low", "medium", "high", "xhigh"}},
		{"openai/gpt-5.2-codex", []string{"low", "medium", "high", "xhigh"}},
		{"gemini-3.5-flash", []string{"minimal", "low", "medium", "high"}},
		{"gemini-3-pro-preview", []string{"low", "high"}},
		{"gpt-4o", nil},
		{"random-model", nil},
	}
	for _, tc := range cases {
		got := reasoningEffortsFor(tc.key)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("reasoningEffortsFor(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func TestFetchPricesFrom(t *testing.T) {
	body := `{
		"openai/gpt-4o": {"mode": "chat", "litellm_provider": "openai", "max_input_tokens": 128000, "input_cost_per_token": 0.0000025},
		"openai/o1": {"mode": "chat", "litellm_provider": "openai"},
		"text-embedding-3-small": {"mode": "embedding", "litellm_provider": "openai"},
		"dall-e-3": {"mode": "image_generation", "litellm_provider": "openai"}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	prices, err := fetchPricesFrom(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 {
		t.Fatalf("prices = %+v, want only chat entries", prices)
	}
	cfg, ok := prices["openai/gpt-4o"]
	if !ok || cfg.MaxInputTokens != 128000 || cfg.InputCostPerToken != 0.0000025 {
		t.Fatalf("gpt-4o cfg = %+v", cfg)
	}
	o1, ok := prices["openai/o1"]
	if !ok {
		t.Fatal("o1 missing")
	}
	if !reflect.DeepEqual(o1.ReasoningEfforts, []string{"low", "medium", "high"}) {
		t.Fatalf("o1 efforts = %v", o1.ReasoningEfforts)
	}

	if _, err := fetchPricesFrom(context.Background(), "http://127.0.0.1:1/nope"); err == nil {
		t.Fatal("want error for unreachable url")
	}
}

func TestFetchPricesFromBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"openai/gpt-4o": {`))
	}))
	defer srv.Close()

	if _, err := fetchPricesFrom(context.Background(), srv.URL); err == nil {
		t.Fatal("want error for malformed json")
	}
}

func TestManagerPricesCachedAndLookup(t *testing.T) {
	mgr, _, c := newTestManagerWithCache(t)
	prices := map[string]ModelConfig{
		"openai/gpt-4o":                {Mode: "chat", LitellmProvider: "openai", MaxInputTokens: 128000},
		"groq/llama-3.3-70b-versatile": {Mode: "chat", LitellmProvider: "groq"},
	}
	data, err := json.Marshal(prices)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set(PricesCacheKey, data); err != nil {
		t.Fatal(err)
	}

	if err := mgr.EnsurePricesCached(context.Background()); err != nil {
		t.Fatal(err)
	}

	cfg, err := mgr.ModelConfigFor("openai", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxInputTokens != 128000 {
		t.Fatalf("cfg = %+v", cfg)
	}
	if _, err := mgr.ModelConfigFor("openai", "nope"); !errors.Is(err, ErrModelConfigNotFound) {
		t.Fatalf("want ErrModelConfigNotFound, got %v", err)
	}
}

func TestManagerEnrichesModelsWithConfig(t *testing.T) {
	mgr, _, c := newTestManagerWithCache(t)
	prices := map[string]ModelConfig{
		"openai/gpt-4o": {Mode: "chat", LitellmProvider: "openai", MaxInputTokens: 128000},
		"openai/o1":     {Mode: "chat", LitellmProvider: "openai", SupportsReasoning: true},
	}
	data, _ := json.Marshal(prices)
	if err := c.Set(PricesCacheKey, data); err != nil {
		t.Fatal(err)
	}

	registerMockProvider(t, "enrich-prov", []Model{{ID: "gpt-4o"}, {ID: "o1"}, {ID: "unknown-model"}})
	mgr.Sync(context.Background(), []config.LLMConfig{{Provider: "enrich-prov"}})

	models, err := mgr.Models("enrich-prov")
	if err != nil {
		t.Fatal(err)
	}
	if models[0].Config == nil || models[0].Config.MaxInputTokens != 128000 {
		t.Fatalf("models[0] = %+v", models[0])
	}
	if models[1].Config == nil || !models[1].Config.SupportsReasoning {
		t.Fatalf("models[1] = %+v", models[1])
	}
	if models[2].Config != nil {
		t.Fatalf("models[2] = %+v, want no config", models[2])
	}

	reloaded, ok := mgr.loadCachedModels("enrich-prov")
	if !ok || len(reloaded) != 3 || reloaded[0].Config == nil {
		t.Fatalf("reloaded = %+v", reloaded)
	}
}
