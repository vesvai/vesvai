package openrouter

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/llm/providers/openai"
)

const (
	BaseURL     = "https://openrouter.ai/api/v1"
	providerKey = "openrouter"
)

type Client struct {
	*openai.Service
}

func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}
	svc := openai.NewService(providerKey, openai.ServiceConfig{
		BaseURL: baseURL,
		APIKey:  cfg.APIKey,
		Headers: map[string]string{
			"HTTP-Referer": "https://github.com/vesvai/vesv",
			"X-Title":      "Vesva",
		},
		ModifyRequest: injectOpenRouterFields,
	})
	return &Client{Service: svc}
}

type Config struct {
	APIKey  string
	BaseURL string
}

func injectOpenRouterFields(body map[string]any) {
	body["provider"] = map[string]any{"allow_fallbacks": true}
}

func (c *Client) Name() string { return providerKey }

func RegisterHooks(hooks *hook.Hooks) {
	hooks.AddFilter(llm.HookProviderRegistry, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		registry, ok := value.(*llm.ProviderRegistry)
		if !ok {
			return value
		}

		registry.Register(providerKey, func(cfg config.LLMConfig) (llm.Provider, error) {
			if cfg.APIKey == "" {
				return nil, fmt.Errorf("openrouter API key not configured (run: vesvai provider add openrouter <api-key>)")
			}

			return NewClient(Config{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL}), nil
		})

		return registry
	}, 100)
}
