package compat

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/llm/providers/openai"
)

type Options struct {
	Headers        map[string]string
	ModifyRequest  func(body map[string]any)
	AllowNoKey     bool
	KeyHeader      string
	PathFor        func(endpoint, model string) string
	ExampleBaseURL string
}

func Register(hooks *hook.Hooks, name, defaultBaseURL string, opts Options) {
	hooks.AddFilter(llm.HookProviderRegistry, func(ctx context.Context, value interface{}, args ...interface{}) interface{} {
		registry, ok := value.(*llm.ProviderRegistry)
		if !ok {
			return value
		}

		registry.Register(name, func(cfg config.LLMConfig) (llm.Provider, error) {
			baseURL := cfg.BaseURL
			if baseURL == "" {
				if defaultBaseURL == "" {
					hint := opts.ExampleBaseURL
					if hint == "" {
						hint = "https://<your-resource>.example.com"
					}
					return nil, fmt.Errorf("%s: base_url must be configured (e.g. %s)", name, hint)
				}
				baseURL = defaultBaseURL
			}

			if cfg.APIKey == "" && !opts.AllowNoKey {
				return nil, fmt.Errorf("%s API key not configured (run: vesvai provider add %s <api-key>)", name, name)
			}

			headers := make(map[string]string, len(opts.Headers)+len(cfg.Headers))
			for k, v := range opts.Headers {
				headers[k] = v
			}
			for k, v := range cfg.Headers {
				headers[k] = v
			}

			apiKey := cfg.APIKey
			if opts.KeyHeader != "" {
				if cfg.APIKey != "" {
					headers[opts.KeyHeader] = cfg.APIKey
				}
				apiKey = ""
			}

			svc := openai.NewService(name, openai.ServiceConfig{
				BaseURL:       baseURL,
				APIKey:        apiKey,
				Headers:       headers,
				ModifyRequest: opts.ModifyRequest,
				PathFor:       opts.PathFor,
			})

			return svc, nil
		})

		return registry
	}, 100)
}
