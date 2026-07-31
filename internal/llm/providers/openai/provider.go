package openai

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm"
)

const (
	DefaultBaseURL = "https://api.openai.com/v1"
	ProviderName   = "openai"
)

type Provider struct {
	*Service
}

func NewProvider(cfg Config) *Provider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	svc := NewService(ProviderName, ServiceConfig{
		BaseURL: baseURL,
		APIKey:  cfg.APIKey,
		Headers: cfg.Headers,
	})
	return &Provider{Service: svc}
}

func (p *Provider) Name() string { return ProviderName }

type Config struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

func RegisterHooks(hooks *hook.Hooks) {
	hooks.AddFilter(llm.HookProviderRegistry, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		registry, ok := value.(*llm.ProviderRegistry)
		if !ok {
			return value
		}

		registry.Register(ProviderName, func(cfg config.LLMConfig) (llm.Provider, error) {
			if cfg.APIKey == "" {
				return nil, fmt.Errorf("openai API key not configured (run: vesvai provider add openai <api-key>)")
			}

			return NewProvider(Config{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL,
				Headers: cfg.Headers,
			}), nil
		})

		return registry
	}, 100)
}
